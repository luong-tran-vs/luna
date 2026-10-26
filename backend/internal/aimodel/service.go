// Package aimodel keeps the model that writes text, chosen by the admin (added 2026-10-23): the
// model of GEMINI_MODEL is only the default, so a slow or busy model can be swapped without a new
// deployment. Each op (what the text is for: a learner's question, a lesson…) can have its own
// model, so the learner can get a fast one while content is written by a stronger one; which op
// gets which is the admin's choice, not the code's. Development and production share one
// database, so the choice applies to both.
package aimodel

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"sync"
	"time"
)

// GeminiOptions are the Gemini models the admin can choose, from the least to the most capable.
var GeminiOptions = []string{
	"gemini-3.5-flash-lite",
	"gemini-3.5-flash",
	"gemini-3.6-flash",
	"gemini-3.7-flash",
	"gemini-3.8-flash",
}

// GroqOptions are the Groq models offered when GROQ_API_KEY is set (added 2026-10-23), the best first.
var GroqOptions = []string{
	"openai/gpt-oss-120b",
	"openai/gpt-oss-20b",
	"qwen/qwen3.8-27b",
}

// OpenAIOptions are the ChatGPT models offered when OPENAI_API_KEY is set (added 2026-10-26), the
// best first.
var OpenAIOptions = []string{
	"gpt-6-sol",
	"gpt-6-luna",
	// Added 2026-10-26 at the admin's request.
	"gpt-6-astra",
	"gpt-5.5-pro",
	"gpt-5.5",
	"gpt-5.4-mini",
}

// Ops are the text requests a model can be chosen for: the op names the AI client records
// (internal/ai/llm). Pictures ("image") always use GEMINI_IMAGE_MODEL.
var Ops = []string{
	"explain", "talk", "talk_review", "speak_check", "grade",
	"generate", "annotate", "practice", "glosses", "lesson_review", "lesson_fix", "lesson_simplify",
	"curriculum_draft", "curriculum_draft_topic", "grammar_lesson", "grammar_solve",
	"suggest_words", "word_meanings",
}

var (
	// ErrUnknownModel means the model is not one of Options.
	ErrUnknownModel = errors.New("aimodel: unknown model")
	// ErrUnknownOp means the op is not one of Ops.
	ErrUnknownOp = errors.New("aimodel: unknown op")
)

// Choice is what the admin chose: the model of every op (empty for GEMINI_MODEL) and the model of
// some ops in particular (added 2026-10-23).
type Choice struct {
	Model     string
	ByOp      map[string]string
	UpdatedAt time.Time
}

// Repository keeps the one Choice.
type Repository interface {
	// Get returns the saved choice, or a zero Choice when none was saved.
	Get(ctx context.Context) (Choice, error)
	Save(ctx context.Context, c Choice) error
}

// Tester sends one short request with a model, to see whether it answers and how fast.
type Tester interface {
	TestModel(ctx context.Context, model string) error
}

// cacheTTL is how long the choice read from the database is used: every AI request asks for it.
// A choice made on the other server (development or production) shows here within this time.
const cacheTTL = 30 * time.Second

// testTimeout bounds a test: a model slower than this cannot write a lesson within the request
// timeout anyway.
const testTimeout = 60 * time.Second

// Service reads and changes the choice.
type Service struct {
	repo     Repository
	fallback string
	options  []string
	tester   Tester
	now      func() time.Time
	log      *slog.Logger

	mu     sync.Mutex
	cached Choice
	at     time.Time
}

// NewService returns a Service; fallback is the model of GEMINI_MODEL, groq and openai offer the
// models of Groq and OpenAI too, and tester may be nil when no AI provider is set.
func NewService(repo Repository, fallback string, groq, openai bool, tester Tester, now func() time.Time,
	log *slog.Logger,
) *Service {
	options := slices.Clone(GeminiOptions)
	if groq {
		options = append(options, GroqOptions...)
	}
	if openai {
		options = append(options, OpenAIOptions...)
	}
	return &Service{repo: repo, fallback: fallback, options: options, tester: tester, now: now, log: log}
}

// Options are the models the admin can choose.
func (s *Service) Options() []string { return s.options }

// ModelFor returns the model to send a text request of op to: the op's own model, else the model
// of every op, else GEMINI_MODEL. A model no longer offered (Groq or OpenAI without its key) is skipped. When
// the database cannot be read it keeps the last choice it knew.
func (s *Service) ModelFor(ctx context.Context, op string) string {
	c := s.choice(ctx)
	if m := c.ByOp[op]; m != "" && slices.Contains(s.options, m) {
		return m
	}
	return s.effective(c)
}

// choice is the saved choice, read from the database at most once per cacheTTL.
func (s *Service) choice(ctx context.Context) Choice {
	now := s.now()
	s.mu.Lock()
	if !s.at.IsZero() && now.Sub(s.at) < cacheTTL {
		c := s.cached
		s.mu.Unlock()
		return c
	}
	s.mu.Unlock()

	c, err := s.repo.Get(ctx)
	s.mu.Lock()
	defer s.mu.Unlock()
	if err != nil {
		s.log.ErrorContext(ctx, "read ai model failed", slog.Any("error", err))
		return s.cached
	}
	s.cached, s.at = c, now
	return c
}

func (s *Service) effective(c Choice) string {
	if c.Model == "" || !slices.Contains(s.options, c.Model) {
		return s.fallback
	}
	return c.Model
}

// State is what the admin page shows.
type State struct {
	// Model is the model in use; Default the one of GEMINI_MODEL.
	Model   string
	Default string
	// Saved is empty when the admin chose nothing.
	Saved string
	// ByOp are the ops given their own model.
	ByOp      map[string]string
	UpdatedAt time.Time
}

func (s *Service) state(c Choice) State {
	byOp := map[string]string{}
	for op, m := range c.ByOp {
		byOp[op] = m
	}
	return State{Model: s.effective(c), Default: s.fallback, Saved: c.Model, ByOp: byOp, UpdatedAt: c.UpdatedAt}
}

// State returns the current choice, read from the database.
func (s *Service) State(ctx context.Context) (State, error) {
	c, err := s.repo.Get(ctx)
	if err != nil {
		return State{}, err
	}
	return s.state(c), nil
}

// Set saves model as the model of every op (empty for GEMINI_MODEL) and byOp as the ops' own
// models, replacing the earlier choice; an op mapped to "" follows model.
func (s *Service) Set(ctx context.Context, model string, byOp map[string]string) (State, error) {
	if model != "" && !slices.Contains(s.options, model) {
		return State{}, ErrUnknownModel
	}
	kept := map[string]string{}
	for op, m := range byOp {
		if !slices.Contains(Ops, op) {
			return State{}, ErrUnknownOp
		}
		if m == "" {
			continue
		}
		if !slices.Contains(s.options, m) {
			return State{}, ErrUnknownModel
		}
		kept[op] = m
	}
	c := Choice{Model: model, ByOp: kept, UpdatedAt: s.now()}
	if err := s.repo.Save(ctx, c); err != nil {
		return State{}, err
	}
	s.mu.Lock()
	s.cached, s.at = c, c.UpdatedAt
	s.mu.Unlock()
	return s.state(c), nil
}

// Test sends one short request with model and returns how long it took.
func (s *Service) Test(ctx context.Context, model string) (time.Duration, error) {
	if !slices.Contains(s.options, model) && model != s.fallback {
		return 0, ErrUnknownModel
	}
	if s.tester == nil {
		return 0, ErrNoProvider
	}
	ctx, cancel := context.WithTimeout(ctx, testTimeout)
	defer cancel()
	start := s.now()
	err := s.tester.TestModel(ctx, model)
	return s.now().Sub(start), err
}

// ErrNoProvider means no AI provider is set, so a model cannot be tested.
var ErrNoProvider = errors.New("aimodel: no AI provider")
