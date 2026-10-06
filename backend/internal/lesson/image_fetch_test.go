package lesson

import (
	"bytes"
	"context"
	"errors"
	"image/jpeg"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"

	"github.com/luongtran/luna/backend/internal/platform/httpx"
)

func TestPublicAddr(t *testing.T) {
	t.Parallel()
	for addr, want := range map[string]bool{
		"8.8.8.8":         true,
		"2606:4700::1111": true,
		"127.0.0.1":       false,
		"10.1.2.3":        false,
		"172.16.0.1":      false,
		"192.168.1.10":    false,
		"169.254.169.254": false, // cloud metadata
		"100.64.0.1":      false,
		"0.0.0.0":         false,
		"::1":             false,
		"fd00::1":         false,
		"fe80::1":         false,
		"224.0.0.1":       false,
	} {
		if got := publicAddr(netip.MustParseAddr(addr)); got != want {
			t.Errorf("publicAddr(%s) = %v, want %v", addr, got, want)
		}
	}
}

func TestImageFetcherRefusesLocalAndOtherSchemes(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(testPNG(10, 10))
	}))
	defer srv.Close()
	fetch := NewImageFetcher()
	for _, link := range []string{srv.URL + "/a.png", "http://localhost:1/a.png", "http://169.254.169.254/latest"} {
		if _, err := fetch(t.Context(), link); !errors.Is(err, errBlockedHost) {
			t.Errorf("%s: err = %v, want blocked", link, err)
		}
	}
	for _, link := range []string{"file:///etc/passwd", "ftp://example.com/a.png", "/a.png", "http://user:pw@example.com/a.png"} {
		if _, err := fetch(t.Context(), link); !errors.Is(err, errBadImageURL) {
			t.Errorf("%s: err = %v, want bad url", link, err)
		}
	}
}

func TestImageFetcherRedirectToLocalIsBlocked(t *testing.T) {
	t.Parallel()
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(testPNG(10, 10))
	}))
	defer target.Close()
	// The first server counts as public here; its redirect leads to a local one, which is not.
	first := httptest.NewServer(http.RedirectHandler(target.URL, http.StatusFound))
	defer first.Close()
	firstAddr := netip.MustParseAddrPort(strings.TrimPrefix(first.URL, "http://"))
	allowed := func(ap netip.AddrPort) bool { return ap == firstAddr }
	if _, err := newImageFetcher(allowed)(t.Context(), first.URL); !errors.Is(err, errBlockedHost) {
		t.Fatalf("err = %v", err)
	}
	// The target itself is reachable when allowed: the redirect is what was refused.
	both := func(ap netip.AddrPort) bool { return ap.Addr().IsLoopback() }
	if _, err := newImageFetcher(both)(t.Context(), first.URL); err != nil {
		t.Fatalf("allowed redirect: %v", err)
	}
}

func TestImageFetcherDownloads(t *testing.T) {
	t.Parallel()
	png := testPNG(20, 10)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/a.png":
			_, _ = w.Write(png)
		case "/moved":
			http.Redirect(w, r, "/a.png", http.StatusMovedPermanently)
		case "/big":
			_, _ = w.Write(make([]byte, MaxUploadBytes+1))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	fetch := newImageFetcher(func(ap netip.AddrPort) bool { return ap.Addr().IsLoopback() })

	for _, path := range []string{"/a.png", "/moved"} {
		data, err := fetch(t.Context(), srv.URL+path)
		if err != nil || !bytes.Equal(data, png) {
			t.Fatalf("%s: %d bytes, %v", path, len(data), err)
		}
	}
	if _, err := fetch(t.Context(), srv.URL+"/missing"); !errors.Is(err, errFetchStatus) {
		t.Fatalf("missing: %v", err)
	}
	if _, err := fetch(t.Context(), srv.URL+"/big"); !errors.Is(err, errFetchTooLarge) {
		t.Fatalf("big: %v", err)
	}
}

func TestImportImage(t *testing.T) {
	t.Parallel()
	e := newImageEnv(t)
	l := e.annotated(t, nil)
	var asked []string
	e.svc.FetchImage = func(_ context.Context, link string) ([]byte, error) {
		asked = append(asked, link)
		switch link {
		case "https://example.com/cup.png":
			return testPNG(1200, 900), nil
		case "https://example.com/page.html":
			return []byte("<html>not a picture</html>"), nil
		case "http://10.0.0.1/a.png":
			return nil, errBlockedHost
		case "https://example.com/big.png":
			return nil, errFetchTooLarge
		default:
			return nil, errFetchStatus
		}
	}

	if err := e.svc.ImportImage(t.Context(), l.ID, "go", "  https://example.com/cup.png  "); err != nil {
		t.Fatal(err)
	}
	img, err := e.store.Get(t.Context(), l.ID, "go")
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := jpeg.DecodeConfig(bytes.NewReader(img.Data))
	if err != nil || cfg.Width != imageSide || cfg.Height != 270 {
		t.Fatalf("stored %dx%d, %v", cfg.Width, cfg.Height, err)
	}
	if asked[0] != "https://example.com/cup.png" {
		t.Fatalf("asked = %v", asked)
	}

	for link, want := range map[string]string{
		"":                              "Vui lòng dán link ảnh bắt đầu bằng http:// hoặc https://",
		"https://example.com/page.html": "Link này không phải ảnh JPEG, PNG hoặc GIF. Hãy dùng link trỏ thẳng tới file ảnh.",
		"http://10.0.0.1/a.png":         "Không được lấy ảnh từ địa chỉ nội bộ.",
		"https://example.com/big.png":   "Ảnh ở link này lớn hơn 5 MB.",
		"https://example.com/404.png":   "Không tải được ảnh từ link này. Hãy thử link khác hoặc tải file lên.",
	} {
		var verr *ValidationError
		if err := e.svc.ImportImage(t.Context(), l.ID, "give up", link); !errors.As(err, &verr) || verr.Fields["url"] != want {
			t.Errorf("%q: err = %v", link, err)
		}
	}
	if err := e.svc.ImportImage(t.Context(), l.ID, "banana", "https://example.com/cup.png"); !errors.Is(err, ErrNotLessonWord) {
		t.Fatalf("other word: %v", err)
	}
	if _, err := e.store.Get(t.Context(), l.ID, "give up"); !errors.Is(err, ErrImageNotFound) {
		t.Fatalf("a refused link stored something: %v", err)
	}
}

func TestImportImageEndpoint(t *testing.T) {
	t.Parallel()
	e := newImageEnv(t)
	l := e.annotated(t, nil)
	e.svc.FetchImage = func(context.Context, string) ([]byte, error) { return testPNG(50, 50), nil }
	mux := http.NewServeMux()
	NewHandler(e.svc, slog.New(slog.DiscardHandler)).Register(mux, httpx.RequireAuth(resolver))

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost,
		"/api/admin/lessons/"+l.ID+"/images/go/import", strings.NewReader(`{"url":"https://example.com/a.png"}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: httpx.SessionCookieName, Value: "admin"})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `/images/go"`) {
		t.Fatalf("import: %d %s", rec.Code, rec.Body)
	}
}
