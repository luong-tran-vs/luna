package verbs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/luongtran/luna/backend/internal/ai"
	"github.com/luongtran/luna/backend/internal/imageutil"
)

// Pictures of the irregular verbs (added 2026-11-10): one picture per verb, of its sense as a verb
// (ring = reo, not a ring on a finger), shown on the cards of the "Động từ" tab. They are kept apart
// from the word bank's pictures, which a noun spelt the same way may already use. The verb list
// itself lives in the frontend (irregular-verbs.data.ts): the admin page sends the meaning and the
// example sentence of each verb it asks to draw.

// imageSide is the longest side of a stored verb picture, in pixels: the card shows it about 300px
// wide, sharp on a phone screen.
const imageSide = 480

// MaxUploadBytes bounds a picture the admin uploads.
const MaxUploadBytes = 5 << 20

var (
	// ErrImageNotFound means the verb has no picture.
	ErrImageNotFound = errors.New("verbs: image not found")
	// ErrImagesUnavailable means the server keeps no verb pictures, or has no AI to draw them.
	ErrImagesUnavailable = errors.New("verbs: images are not available")
	// ErrCodexUnavailable means the admin chose Codex but this server does not run it.
	ErrCodexUnavailable = errors.New("verbs: codex images not enabled on this server")
	// ErrCodexNotLoggedIn means codex runs but is not logged in to ChatGPT.
	ErrCodexNotLoggedIn = errors.New("verbs: codex not logged in")
	// ErrDrawFailed means the AI answered without a usable picture.
	ErrDrawFailed = errors.New("verbs: drawing failed")
	// ErrBadImage means an uploaded file is not a picture the server can read.
	ErrBadImage = errors.New("verbs: not a readable image")
	// ErrNoBankImage means the word bank has no picture for the verb.
	ErrNoBankImage = errors.New("verbs: no word bank image")
)

// ImageEngine is what draws a picture: Gemini or the Codex CLI of the server. There is no default:
// the admin chooses each time.
type ImageEngine string

const (
	EngineGemini ImageEngine = "gemini"
	EngineCodex  ImageEngine = "codex"
)

// VerbImage is the picture of one verb, by its base form.
type VerbImage struct {
	Base      string
	MIME      string
	Data      []byte
	CreatedAt time.Time
}

// ImageInfo tells that a verb has a picture, and since when (the URL changes with it).
type ImageInfo struct {
	Base      string
	CreatedAt time.Time
}

// ImageRepository stores the verb pictures. Implementations live in internal/storage.
type ImageRepository interface {
	// List tells every verb that has a picture.
	List(ctx context.Context) ([]ImageInfo, error)
	// Get returns ErrImageNotFound when the verb has no picture.
	Get(ctx context.Context, base string) (VerbImage, error)
	// Save stores a picture, replacing the verb's previous one.
	Save(ctx context.Context, img VerbImage) error
	// Delete removes a verb's picture; a missing one is not an error.
	Delete(ctx context.Context, base string) error
}

// BankPicture is a picture of the word bank for a word spelt like a verb.
type BankPicture struct {
	MeaningVi string
	CreatedAt time.Time
}

// Bank reads the word bank's pictures (internal/service adapts wordbank.Service).
type Bank interface {
	// Pictures tells which of the words have a picture in the bank, with the bank's meaning.
	Pictures(ctx context.Context, words []string) (map[string]BankPicture, error)
	// Image returns the bank's picture of a word; ErrNoBankImage when it has none.
	Image(ctx context.Context, word string) (mime string, data []byte, err error)
}

// ImageDeps are what the verb pictures need.
type ImageDeps struct {
	Store ImageRepository
	// Gemini draws with GEMINI_IMAGE_MODEL; Codex with the Codex CLI, nil when the server runs none.
	Gemini ai.ImageProvider
	Codex  ai.ImageProvider
	// CodexConcurrency is how many pictures Codex draws at once.
	CodexConcurrency int
	// Bank, when set, lets the admin reuse a word bank picture.
	Bank Bank
	Log  *slog.Logger
	Now  func() time.Time
}

// Images keeps, draws and serves the verb pictures.
type Images struct {
	d   ImageDeps
	mu  sync.Mutex
	run *imageRun
}

// NewImages returns the verb picture service.
func NewImages(d ImageDeps) *Images {
	if d.Log == nil {
		d.Log = slog.Default()
	}
	if d.Now == nil {
		d.Now = time.Now
	}
	return &Images{d: d}
}

// CodexAvailable tells whether this server can draw with the Codex CLI.
func (s *Images) CodexAvailable() bool { return s.d.Codex != nil }

// baseRe is the shape of a verb's base form: lower-case letters, maybe with a hyphen.
var baseRe = regexp.MustCompile(`^[a-z][a-z-]{0,29}$`)

// checkBase normalises a base form and checks its shape.
func checkBase(base string) (string, error) {
	b := strings.ToLower(strings.TrimSpace(base))
	if !baseRe.MatchString(b) {
		return "", &ValidationError{Fields: map[string]string{"base": "Động từ không hợp lệ"}}
	}
	return b, nil
}

// List tells every verb that has a picture.
func (s *Images) List(ctx context.Context) ([]ImageInfo, error) {
	if s.d.Store == nil {
		return []ImageInfo{}, nil
	}
	out, err := s.d.Store.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("verbs: list images: %w", err)
	}
	return out, nil
}

// BankPictures tells which of the verbs have a picture in the word bank (none without a bank).
func (s *Images) BankPictures(ctx context.Context, bases []string) (map[string]BankPicture, error) {
	if s.d.Bank == nil || len(bases) == 0 {
		return map[string]BankPicture{}, nil
	}
	clean := make([]string, 0, len(bases))
	for _, b := range bases {
		if c, err := checkBase(b); err == nil {
			clean = append(clean, c)
		}
	}
	out, err := s.d.Bank.Pictures(ctx, clean)
	if err != nil {
		return nil, fmt.Errorf("verbs: bank pictures: %w", err)
	}
	return out, nil
}

// Image returns the picture of a verb.
func (s *Images) Image(ctx context.Context, base string) (VerbImage, error) {
	b, err := checkBase(base)
	if err != nil {
		return VerbImage{}, ErrImageNotFound
	}
	if s.d.Store == nil {
		return VerbImage{}, ErrImageNotFound
	}
	return s.d.Store.Get(ctx, b)
}

// Upload stores a picture the admin chose for a verb, made smaller first.
func (s *Images) Upload(ctx context.Context, base string, data []byte) (ImageInfo, error) {
	b, err := checkBase(base)
	if err != nil {
		return ImageInfo{}, err
	}
	small, err := imageutil.Shrink(data, imageSide)
	if err != nil {
		return ImageInfo{}, fmt.Errorf("%w: %w", ErrBadImage, err)
	}
	return s.save(ctx, b, small)
}

// FromBank copies the word bank's picture of a word spelt like the verb, when it shows the verb's sense.
func (s *Images) FromBank(ctx context.Context, base string) (ImageInfo, error) {
	b, err := checkBase(base)
	if err != nil {
		return ImageInfo{}, err
	}
	if s.d.Bank == nil {
		return ImageInfo{}, ErrNoBankImage
	}
	_, data, err := s.d.Bank.Image(ctx, b)
	if err != nil {
		return ImageInfo{}, err
	}
	small, err := imageutil.Shrink(data, imageSide)
	if err != nil {
		return ImageInfo{}, fmt.Errorf("%w: %w", ErrBadImage, err)
	}
	return s.save(ctx, b, small)
}

// Delete removes a verb's picture.
func (s *Images) Delete(ctx context.Context, base string) error {
	b, err := checkBase(base)
	if err != nil {
		return err
	}
	if s.d.Store == nil {
		return ErrImagesUnavailable
	}
	if err := s.d.Store.Delete(ctx, b); err != nil {
		return fmt.Errorf("verbs: delete image: %w", err)
	}
	return nil
}

func (s *Images) save(ctx context.Context, base string, data []byte) (ImageInfo, error) {
	if s.d.Store == nil {
		return ImageInfo{}, ErrImagesUnavailable
	}
	img := VerbImage{Base: base, MIME: "image/jpeg", Data: data, CreatedAt: s.d.Now()}
	if err := s.d.Store.Save(ctx, img); err != nil {
		return ImageInfo{}, fmt.Errorf("verbs: save image: %w", err)
	}
	return ImageInfo{Base: base, CreatedAt: img.CreatedAt}, nil
}

// drawer is the AI that engine names, and how many pictures it may draw at once.
func (s *Images) drawer(engine ImageEngine) (ai.ImageProvider, int, error) {
	switch engine {
	case "":
		// No default (asked 2026-11-10): the admin picks Gemini or Codex each time.
		return nil, 0, &ValidationError{Fields: map[string]string{"engine": "Hãy chọn sinh ảnh bằng Gemini hay Codex CLI"}}
	case EngineGemini:
		if s.d.Gemini == nil {
			return nil, 0, ErrImagesUnavailable
		}
		return s.d.Gemini, 1, nil
	case EngineCodex:
		if s.d.Codex == nil {
			return nil, 0, ErrCodexUnavailable
		}
		return s.d.Codex, max(1, s.d.CodexConcurrency), nil
	default:
		return nil, 0, &ValidationError{Fields: map[string]string{"engine": "Cách sinh ảnh không hợp lệ"}}
	}
}

// DrawItem is a verb to draw: its base form, Vietnamese meaning and an example sentence, which keep
// the picture on the verb's sense.
type DrawItem struct {
	Base      string
	MeaningVi string
	Sentence  string
}

// draw asks drawer for the picture of a verb and stores it, replacing the one it has.
func (s *Images) draw(ctx context.Context, drawer ai.ImageProvider, engine ImageEngine, it DrawItem, style string) error {
	img, err := drawer.GenerateImage(ctx, ai.ImageRequest{
		// "to ring", not "ring": the word is a verb, whatever noun is spelt alike.
		Word: "to " + it.Base, MeaningVi: it.MeaningVi, Sentence: it.Sentence, Style: style,
	})
	if engine == EngineCodex && errors.Is(err, ai.ErrNotConfigured) {
		return fmt.Errorf("%w: %w", ErrCodexNotLoggedIn, err)
	}
	switch {
	case errors.Is(err, ai.ErrNotConfigured), errors.Is(err, ai.ErrInvalidKey), errors.Is(err, ai.ErrQuota),
		errors.Is(err, ai.ErrNoImageTool), errors.Is(err, context.DeadlineExceeded):
		return err
	case err != nil:
		return fmt.Errorf("%w: %w", ErrDrawFailed, err)
	}
	data, err := imageutil.Shrink(img.Data, imageSide)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrDrawFailed, err)
	}
	_, err = s.save(ctx, it.Base, data)
	return err
}
