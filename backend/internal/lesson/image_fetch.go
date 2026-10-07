package lesson

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"syscall"
	"time"
)

const (
	// fetchTimeout bounds downloading one picture from a link, redirects included.
	fetchTimeout = 10 * time.Second
	// maxFetchRedirects is how many redirects a picture link may follow.
	maxFetchRedirects = 3
	// maxImageURL is the longest link an admin may paste.
	maxImageURL = 2048
)

// Reasons a picture link cannot be used; ImportImage turns them into Vietnamese field errors.
var (
	errBadImageURL   = errors.New("lesson: not an http(s) link")
	errBlockedHost   = errors.New("lesson: link points to a private or local address")
	errFetchStatus   = errors.New("lesson: the link did not return the picture")
	errFetchTooLarge = errors.New("lesson: the picture behind the link is too large")
)

// ImageFetcher downloads the picture behind a link, at most MaxUploadBytes.
type ImageFetcher func(ctx context.Context, rawURL string) ([]byte, error)

// NewImageFetcher returns the fetcher used for picture links. It only follows http and https, and
// refuses to connect to loopback, private, link-local and other non-public addresses, checked on
// the address actually dialled (so a DNS answer or a redirect cannot reach inside the server's
// network). No proxy from the environment is used, since it would bypass that check.
func NewImageFetcher() ImageFetcher {
	return newImageFetcher(func(ap netip.AddrPort) bool { return publicAddr(ap.Addr()) })
}

// newImageFetcher is NewImageFetcher with the address check given, so tests can reach a local server.
func newImageFetcher(allowed func(netip.AddrPort) bool) ImageFetcher {
	dialer := &net.Dialer{
		Timeout: 5 * time.Second,
		Control: func(_, address string, _ syscall.RawConn) error {
			ap, err := netip.ParseAddrPort(address)
			if err != nil || !allowed(netip.AddrPortFrom(ap.Addr().Unmap(), ap.Port())) {
				return errBlockedHost
			}
			return nil
		},
	}
	client := &http.Client{
		Timeout: fetchTimeout,
		Transport: &http.Transport{
			Proxy:                 nil,
			DialContext:           dialer.DialContext,
			TLSHandshakeTimeout:   5 * time.Second,
			ResponseHeaderTimeout: fetchTimeout,
			MaxIdleConns:          4,
			IdleConnTimeout:       30 * time.Second,
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxFetchRedirects {
				return errFetchStatus
			}
			return checkImageURL(req.URL)
		},
	}
	return func(ctx context.Context, rawURL string) ([]byte, error) {
		u, err := url.Parse(strings.TrimSpace(rawURL))
		if err != nil {
			return nil, errBadImageURL
		}
		if err := checkImageURL(u); err != nil {
			return nil, err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), http.NoBody)
		if err != nil {
			return nil, errBadImageURL
		}
		req.Header.Set("Accept", "image/jpeg,image/png,image/gif,image/*;q=0.8")
		req.Header.Set("User-Agent", "LunaImageFetcher/1.0")
		resp, err := client.Do(req)
		if err != nil {
			if errors.Is(err, errBlockedHost) || errors.Is(err, errBadImageURL) {
				return nil, errBlockedHost
			}
			return nil, fmt.Errorf("%w: %w", errFetchStatus, err)
		}
		defer resp.Body.Close() //nolint:errcheck // read below
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("%w: status %d", errFetchStatus, resp.StatusCode)
		}
		data, err := io.ReadAll(io.LimitReader(resp.Body, MaxUploadBytes+1))
		if err != nil {
			return nil, fmt.Errorf("%w: %w", errFetchStatus, err)
		}
		if len(data) > MaxUploadBytes {
			return nil, errFetchTooLarge
		}
		return data, nil
	}
}

// checkImageURL accepts absolute http(s) links without credentials.
func checkImageURL(u *url.URL) error {
	if (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil {
		return errBadImageURL
	}
	return nil
}

// cgnat is the shared address space of carrier-grade NAT (100.64.0.0/10), not public either.
var cgnat = netip.MustParsePrefix("100.64.0.0/10")

// publicAddr reports whether a is a public unicast address.
func publicAddr(a netip.Addr) bool {
	return a.IsValid() && a.IsGlobalUnicast() && !a.IsPrivate() && !a.IsLoopback() &&
		!a.IsLinkLocalUnicast() && !cgnat.Contains(a)
}

// ImportImage downloads the picture behind a link and stores it for one vocabulary word, like an
// uploaded one (scaled down, kept as JPEG). Only the picture is kept, not the link.
func (s *Service) ImportImage(ctx context.Context, id, lemma, rawURL string) error {
	if s.ImageStore == nil || s.FetchImage == nil {
		return ErrImagesUnavailable
	}
	l, err := s.Lessons.Get(ctx, id)
	if err != nil {
		return err
	}
	if _, err := lessonWord(l, lemma); err != nil {
		return err
	}
	data, err := FetchImageLink(ctx, s.FetchImage, rawURL)
	if err != nil {
		var verr *ValidationError
		if !errors.As(err, &verr) {
			return err
		}
		if cause := errors.Unwrap(err); cause != nil {
			s.Log.InfoContext(ctx, "lesson: fetch word image", "lesson_id", id, "error", cause)
		}
		return err
	}
	var verr *ValidationError
	if err := s.UploadImage(ctx, id, lemma, data); errors.As(err, &verr) {
		return urlError(NotAnImageMessage)
	} else if err != nil {
		return err
	}
	return nil
}

// NotAnImageMessage tells the admin a link gave something that is not a picture.
const NotAnImageMessage = "Link này không phải ảnh JPEG, PNG hoặc GIF. Hãy dùng link trỏ thẳng tới file ảnh."

// FetchImageLink downloads the picture behind an admin's link with fetch. A link that cannot be
// used is a *ValidationError on field "url" with a Vietnamese message (wrapping the download error,
// if any, for the log). The data is not checked to be a picture.
func FetchImageLink(ctx context.Context, fetch ImageFetcher, rawURL string) ([]byte, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" || len(rawURL) > maxImageURL {
		return nil, urlError("Vui lòng dán link ảnh bắt đầu bằng http:// hoặc https://")
	}
	data, err := fetch(ctx, rawURL)
	switch {
	case errors.Is(err, errBadImageURL):
		return nil, urlError("Vui lòng dán link ảnh bắt đầu bằng http:// hoặc https://")
	case errors.Is(err, errBlockedHost):
		return nil, urlError("Không được lấy ảnh từ địa chỉ nội bộ.")
	case errors.Is(err, errFetchTooLarge):
		return nil, urlError(fmt.Sprintf("Ảnh ở link này lớn hơn %d MB.", MaxUploadBytes>>20))
	case err != nil:
		return nil, &ValidationError{
			Fields: map[string]string{"url": "Không tải được ảnh từ link này. Hãy thử link khác hoặc tải file lên."},
			cause:  err,
		}
	}
	return data, nil
}

func urlError(msg string) error {
	return &ValidationError{Fields: map[string]string{"url": msg}}
}
