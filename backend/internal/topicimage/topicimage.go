// Package topicimage keeps the picture of each topic of the course (added 2026-11-16): one banner
// per topic, shared by every level, shown on the "Luyện nói" topics and the topic summary. The admin
// draws them with Gemini or the Codex CLI, one topic or every topic lacking one, in a background
// batch like the verb pictures.
package topicimage

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/luongtran/luna/backend/internal/ai"
	"github.com/luongtran/luna/backend/internal/imageutil"
)

// imageSide is the longest side of a stored topic picture, in pixels: a banner about 600px wide.
const imageSide = 960

var (
	// ErrImageNotFound means the topic has no picture.
	ErrImageNotFound = errors.New("topicimage: image not found")
	// ErrTopicNotFound means there is no such topic.
	ErrTopicNotFound = errors.New("topicimage: topic not found")
	// ErrImagesUnavailable means the server keeps no topic pictures, or has no AI to draw them.
	ErrImagesUnavailable = errors.New("topicimage: images are not available")
	// ErrCodexUnavailable means the admin chose Codex but this server does not run it.
	ErrCodexUnavailable = errors.New("topicimage: codex images not enabled on this server")
	// ErrCodexNotLoggedIn means codex runs but is not logged in to ChatGPT.
	ErrCodexNotLoggedIn = errors.New("topicimage: codex not logged in")
	// ErrDrawFailed means the AI answered without a usable picture.
	ErrDrawFailed = errors.New("topicimage: drawing failed")
	// ErrBatchRunning means a topic picture batch is running already; one runs at a time.
	ErrBatchRunning = errors.New("topicimage: an image batch is running")
)

// ValidationError lists the fields of a request that are wrong, with a Vietnamese message.
type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string { return fmt.Sprintf("topicimage: invalid input %v", e.Fields) }

// Engine is what draws a picture: Gemini or the Codex CLI of the server, chosen each time.
type Engine string

const (
	EngineGemini Engine = "gemini"
	EngineCodex  Engine = "codex"
)

// Image is the picture of one topic.
type Image struct {
	TopicID   string
	MIME      string
	Data      []byte
	CreatedAt time.Time
}

// Info tells that a topic has a picture, and since when (the URL changes with it).
type Info struct {
	TopicID   string
	CreatedAt time.Time
}

// Repository stores the topic pictures. Implementations live in internal/storage.
type Repository interface {
	// List tells every topic that has a picture.
	List(ctx context.Context) ([]Info, error)
	// Get returns ErrImageNotFound when the topic has no picture.
	Get(ctx context.Context, topicID string) (Image, error)
	// Save stores a picture, replacing the topic's previous one.
	Save(ctx context.Context, img Image) error
	// Delete removes a topic's picture; a missing one is not an error.
	Delete(ctx context.Context, topicID string) error
}

// Topic is what a picture is drawn from.
type Topic struct {
	ID          string
	Name        string
	Description string
	Words       []string
}

// Topics gives the topics to draw (internal/service adapts topic.Service).
type Topics interface {
	// Get returns ErrTopicNotFound when there is no such topic.
	Get(ctx context.Context, id string) (Topic, error)
}

// Deps are what the topic pictures need.
type Deps struct {
	Store  Repository
	Topics Topics
	// Gemini draws with GEMINI_IMAGE_MODEL; Codex with the Codex CLI, nil when the server runs none.
	Gemini ai.ImageProvider
	Codex  ai.ImageProvider
	// CodexConcurrency is how many pictures Codex draws at once.
	CodexConcurrency int
	Log              *slog.Logger
	Now              func() time.Time
}

// Service keeps, draws and serves the topic pictures.
type Service struct {
	d   Deps
	mu  sync.Mutex
	run *batchRun
}

// NewService returns the topic picture service.
func NewService(d Deps) *Service {
	if d.Log == nil {
		d.Log = slog.Default()
	}
	if d.Now == nil {
		d.Now = time.Now
	}
	return &Service{d: d}
}

// CodexAvailable tells whether this server can draw with the Codex CLI.
func (s *Service) CodexAvailable() bool { return s.d.Codex != nil }

// List tells every topic that has a picture.
func (s *Service) List(ctx context.Context) ([]Info, error) {
	if s.d.Store == nil {
		return []Info{}, nil
	}
	list, err := s.d.Store.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("topicimage: list: %w", err)
	}
	return list, nil
}

// Image returns the picture of a topic.
func (s *Service) Image(ctx context.Context, topicID string) (Image, error) {
	if s.d.Store == nil {
		return Image{}, ErrImageNotFound
	}
	return s.d.Store.Get(ctx, topicID)
}

// Delete removes a topic's picture.
func (s *Service) Delete(ctx context.Context, topicID string) error {
	if s.d.Store == nil {
		return ErrImagesUnavailable
	}
	if err := s.d.Store.Delete(ctx, topicID); err != nil {
		return fmt.Errorf("topicimage: delete: %w", err)
	}
	return nil
}

// drawer is the AI that engine names, and how many pictures it may draw at once.
func (s *Service) drawer(engine Engine) (ai.ImageProvider, int, error) {
	switch engine {
	case "":
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

// draw asks drawer for the picture of a topic and stores it, replacing the one it has.
func (s *Service) draw(ctx context.Context, drawer ai.ImageProvider, engine Engine, t Topic, style string) error {
	img, err := drawer.GenerateImage(ctx, ai.ImageRequest{
		Topic: &ai.TopicBrief{Name: t.Name, Description: t.Description, Words: t.Words}, Style: style,
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
	if err := s.d.Store.Save(ctx, Image{TopicID: t.ID, MIME: "image/jpeg", Data: data, CreatedAt: s.d.Now()}); err != nil {
		return fmt.Errorf("topicimage: save: %w", err)
	}
	return nil
}
