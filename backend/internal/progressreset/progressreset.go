// Package progressreset lets a learner start parts of the app over (added 2026-11-23): from the
// "Tôi" tab the learner ticks what to reset, and the progress of those parts is removed.
//   - lessons: the topic lessons' steps done and lessons completed, Reading answers, dictation
//     results and lesson review results;
//   - grammar: the grammar points' status and practice and mastery scores;
//   - vocab: the notebook's review cards go back to new and the review history is removed (the
//     saved words stay);
//   - toeic: the TOEIC attempts (tests, practice, results) and the mistakes notebook (the target
//     score stays).
//
// The study days (streak) and the settings are never touched.
package progressreset

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"slices"
	"time"

	"github.com/luongtran/luna/backend/internal/platform/httpx"
)

// Part is a part of the app whose progress can be reset.
type Part string

const (
	PartLessons Part = "lessons"
	PartGrammar Part = "grammar"
	PartVocab   Part = "vocab"
	PartToeic   Part = "toeic"
)

// Parts are every part, in the order they are reset.
var Parts = []Part{PartLessons, PartGrammar, PartVocab, PartToeic}

// Repository resets a learner's progress. Implementations live in internal/storage.
type Repository interface {
	// Reset removes the user's progress in part (see the package comment); now is when the review
	// cards become due again. A user without any progress there is not an error.
	Reset(ctx context.Context, userID string, part Part, now time.Time) error
}

// ErrNoPart means the request named no part, or one that does not exist.
var ErrNoPart = errors.New("progressreset: no valid part")

// Service resets the parts a learner chose.
type Service struct {
	repo Repository
	now  func() time.Time
}

// NewService returns a Service.
func NewService(repo Repository, now func() time.Time) *Service {
	return &Service{repo: repo, now: now}
}

// Reset resets the given parts of the user's progress, each once, in the order of Parts.
func (s *Service) Reset(ctx context.Context, userID string, parts []Part) ([]Part, error) {
	var todo []Part
	for _, p := range parts {
		if !slices.Contains(Parts, p) {
			return nil, ErrNoPart
		}
	}
	for _, p := range Parts {
		if slices.Contains(parts, p) {
			todo = append(todo, p)
		}
	}
	if len(todo) == 0 {
		return nil, ErrNoPart
	}
	now := s.now()
	for _, p := range todo {
		if err := s.repo.Reset(ctx, userID, p, now); err != nil {
			return nil, err
		}
	}
	return todo, nil
}

// Handler serves POST /api/me/reset.
type Handler struct {
	svc *Service
	log *slog.Logger
}

// NewHandler returns a Handler.
func NewHandler(svc *Service, log *slog.Logger) *Handler {
	return &Handler{svc: svc, log: log}
}

// Register adds the route behind requireAuth.
func (h *Handler) Register(mux *http.ServeMux, requireAuth httpx.Middleware) {
	mux.Handle("POST /api/me/reset", requireAuth(http.HandlerFunc(h.reset)))
}

// reset serves POST /api/me/reset with {"parts": ["lessons", "grammar", "vocab", "toeic"]}: 204 once
// the parts are reset, 400 for no part or an unknown one.
func (h *Handler) reset(w http.ResponseWriter, r *http.Request) {
	p, _ := httpx.PrincipalFrom(r.Context())
	var in struct {
		Parts []Part `json:"parts"`
	}
	if httpx.DecodeJSON(w, r, &in) != nil {
		return
	}
	done, err := h.svc.Reset(r.Context(), p.UserID, in.Parts)
	if errors.Is(err, ErrNoPart) {
		httpx.WriteError(w, http.StatusBadRequest, "validation_failed", "Hãy chọn ít nhất một phần để đặt lại.")
		return
	}
	if err != nil {
		h.log.ErrorContext(r.Context(), "progressreset: reset", slog.String("user", p.UserID), slog.Any("error", err))
		httpx.WriteError(w, http.StatusInternalServerError, "internal", "Không đặt lại được, vui lòng thử lại.")
		return
	}
	h.log.InfoContext(r.Context(), "progressreset: reset", slog.String("user", p.UserID), slog.Any("parts", done))
	w.WriteHeader(http.StatusNoContent)
}
