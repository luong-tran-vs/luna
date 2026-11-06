package grammar

import (
	"context"
	"crypto/sha1" //nolint:gosec // a short key of a sentence, not a security hash
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/luongtran/luna/backend/internal/ai"
	"github.com/luongtran/luna/backend/internal/imageutil"
)

// Pictures of grammar examples (added 2026-11-06): every example sentence of a lesson may have a
// picture of the situation it tells, shown beside it in "Cấu trúc & ví dụ". The AI draws them when
// the admin asks, in the background (images_batch.go), with Gemini or the Codex CLI of the server.

// exampleSide is the longest side of a stored example picture, in pixels.
const exampleSide = 480

// ErrImageNotFound means the example has no picture.
var ErrImageNotFound = errors.New("grammar: image not found")

// ErrImagesUnavailable means the server keeps no example pictures, or has no AI to draw them.
var ErrImagesUnavailable = errors.New("grammar: example images are not available")

var (
	// ErrCodexUnavailable means the admin chose Codex but this server does not run it.
	ErrCodexUnavailable = errors.New("grammar: codex images not enabled on this server")
	// ErrCodexNotLoggedIn means codex runs but is not logged in to ChatGPT.
	ErrCodexNotLoggedIn = errors.New("grammar: codex not logged in")
	// ErrDrawFailed means the AI answered without a usable picture.
	ErrDrawFailed = errors.New("grammar: drawing failed")
)

// ImageEngine is what draws a picture: Gemini (the default) or the Codex CLI of the server.
type ImageEngine string

const (
	EngineGemini ImageEngine = "gemini"
	EngineCodex  ImageEngine = "codex"
)

// ExampleImage is the picture of one example sentence of a point's lesson. Key is ExampleKey of
// the sentence: a picture follows its sentence when the examples are reordered, and an edited
// sentence has none until drawn again.
type ExampleImage struct {
	PointID   string
	Key       string
	MIME      string
	Data      []byte
	CreatedAt time.Time
}

// ImageRepository stores the example pictures. Implementations live in internal/storage.
type ImageRepository interface {
	// Keys lists the keys of the point's pictures.
	Keys(ctx context.Context, pointID string) ([]string, error)
	// Get returns ErrImageNotFound when the example has no picture.
	Get(ctx context.Context, pointID, key string) (ExampleImage, error)
	// Save stores a picture, replacing the example's previous one.
	Save(ctx context.Context, img ExampleImage) error
	// Delete removes one picture; a missing one is not an error.
	Delete(ctx context.Context, pointID, key string) error
	// DeleteForPoint removes every picture of a point.
	DeleteForPoint(ctx context.Context, pointID string) error
}

// ExampleKey is the key of an example's picture: a short hash of its sentence, ignoring case and spacing.
func ExampleKey(en string) string {
	norm := strings.ToLower(strings.Join(strings.Fields(en), " "))
	sum := sha1.Sum([]byte(norm)) //nolint:gosec // see the import
	return hex.EncodeToString(sum[:])[:16]
}

// ImageDeps are what drawing example pictures needs.
type ImageDeps struct {
	Store ImageRepository
	// Gemini draws with GEMINI_IMAGE_MODEL; Codex with the Codex CLI, nil when the server runs none.
	Gemini ai.ImageProvider
	Codex  ai.ImageProvider
}

// WithImages lets the service keep and draw example pictures.
func (s *Service) WithImages(d ImageDeps) *Service {
	s.images = d
	return s
}

// CodexAvailable tells whether this server can draw with the Codex CLI.
func (s *Service) CodexAvailable() bool { return s.images.Codex != nil }

// ExamplePicture is an example of a lesson and whether it has a picture.
type ExamplePicture struct {
	Key      string
	En       string
	Vi       string
	HasImage bool
}

// ExamplePictures lists the examples of a point's lesson with whether each has a picture.
func (s *Service) ExamplePictures(ctx context.Context, pointID string) ([]ExamplePicture, error) {
	l, err := s.lessons.Get(ctx, pointID)
	if err != nil {
		return nil, err
	}
	have, err := s.imageKeys(ctx, pointID)
	if err != nil {
		return nil, err
	}
	out := make([]ExamplePicture, len(l.Content.Examples))
	for i, e := range l.Content.Examples {
		k := ExampleKey(e.En)
		out[i] = ExamplePicture{Key: k, En: e.En, Vi: e.Vi, HasImage: have[k]}
	}
	return out, nil
}

// ExampleImageKeys gives, for each example of content in order, the key of its picture, or "" for
// one without (what the lesson pages show beside the examples).
func (s *Service) ExampleImageKeys(ctx context.Context, pointID string, c Content) ([]string, error) {
	have, err := s.imageKeys(ctx, pointID)
	if err != nil {
		return nil, err
	}
	out := make([]string, len(c.Examples))
	for i, e := range c.Examples {
		if k := ExampleKey(e.En); have[k] {
			out[i] = k
		}
	}
	return out, nil
}

func (s *Service) imageKeys(ctx context.Context, pointID string) (map[string]bool, error) {
	out := map[string]bool{}
	if s.images.Store == nil {
		return out, nil
	}
	keys, err := s.images.Store.Keys(ctx, pointID)
	if err != nil {
		return nil, fmt.Errorf("grammar: image keys: %w", err)
	}
	for _, k := range keys {
		out[k] = true
	}
	return out, nil
}

// Image returns the picture of one example. A learner sees only those of a published lesson.
func (s *Service) Image(ctx context.Context, pointID, key string, admin bool) (ExampleImage, error) {
	if s.images.Store == nil {
		return ExampleImage{}, ErrImageNotFound
	}
	if !admin {
		l, err := s.lessons.Get(ctx, pointID)
		if errors.Is(err, ErrNotFound) || (err == nil && l.Status != StatusPublished) {
			return ExampleImage{}, ErrImageNotFound
		}
		if err != nil {
			return ExampleImage{}, err
		}
	}
	return s.images.Store.Get(ctx, pointID, key)
}

// DeleteImage removes the picture of one example.
func (s *Service) DeleteImage(ctx context.Context, pointID, key string) error {
	if s.images.Store == nil {
		return ErrImagesUnavailable
	}
	if _, err := s.lessons.Get(ctx, pointID); err != nil {
		return err
	}
	return s.images.Store.Delete(ctx, pointID, key)
}

// imageDrawer is the AI that engine names.
func (s *Service) imageDrawer(engine ImageEngine) (ai.ImageProvider, error) {
	switch engine {
	case "", EngineGemini:
		if s.images.Gemini == nil {
			return nil, ErrImagesUnavailable
		}
		return s.images.Gemini, nil
	case EngineCodex:
		if s.images.Codex == nil {
			return nil, ErrCodexUnavailable
		}
		return s.images.Codex, nil
	default:
		return nil, &ValidationError{Fields: map[string]string{"engine": "Cách sinh ảnh không hợp lệ"}}
	}
}

// drawExample draws the picture of one example and stores it, replacing the one it has.
func (s *Service) drawExample(ctx context.Context, drawer ai.ImageProvider, engine ImageEngine, item batchItem) error {
	brief := &ai.ExampleBrief{En: item.En, Vi: item.Vi, Point: item.Point}
	if item.Picture != "" {
		brief = &ai.ExampleBrief{En: item.Picture, Point: item.Point, Exercise: true}
	}
	img, err := drawer.GenerateImage(ctx, ai.ImageRequest{Example: brief})
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
	data, err := imageutil.Shrink(img.Data, exampleSide)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrDrawFailed, err)
	}
	if err := s.images.Store.Save(ctx, ExampleImage{PointID: item.PointID, Key: item.Key, MIME: "image/jpeg", Data: data, CreatedAt: s.now()}); err != nil {
		return fmt.Errorf("grammar: save example image: %w", err)
	}
	return nil
}

// PictureKey is the key of the picture of a picture exercise: a short hash of what it shows, so
// the picture follows its description and an edited description has none until drawn again.
func PictureKey(e Exercise) string { return ExampleKey("picture: " + e.Picture) }

// learnerContent is c as a learner gets it (added 2026-11-06): a picture exercise is kept only once
// its picture is drawn, and without its description, which would tell the answer. pictures gives
// the key of the picture of each picture exercise kept, by exercise id.
func (s *Service) learnerContent(ctx context.Context, pointID string, c Content) (out Content, pictures map[string]string, err error) {
	pictures = map[string]string{}
	if !hasPictureExercises(c) {
		return c, pictures, nil
	}
	have, err := s.imageKeys(ctx, pointID)
	if err != nil {
		return Content{}, nil, err
	}
	keep := func(in []Exercise) []Exercise {
		kept := make([]Exercise, 0, len(in))
		for _, e := range in {
			if e.Kind == KindPicture {
				k := PictureKey(e)
				if !have[k] {
					continue
				}
				pictures[e.ID] = k
				e.Picture = ""
			}
			kept = append(kept, e)
		}
		return kept
	}
	c.Practice, c.Mastery = keep(c.Practice), keep(c.Mastery)
	return c, pictures, nil
}

func hasPictureExercises(c Content) bool {
	for _, list := range [][]Exercise{c.Practice, c.Mastery} {
		for _, e := range list {
			if e.Kind == KindPicture {
				return true
			}
		}
	}
	return false
}

// PictureImageKeys gives, for each picture exercise of a stored content that has its picture, the
// key of it by exercise id (the admin page).
func (s *Service) PictureImageKeys(ctx context.Context, pointID string, c Content) (map[string]string, error) {
	_, pictures, err := s.learnerContent(ctx, pointID, c)
	return pictures, err
}
