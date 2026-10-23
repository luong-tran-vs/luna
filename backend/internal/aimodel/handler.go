package aimodel

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/luongtran/luna/backend/internal/ai"
	"github.com/luongtran/luna/backend/internal/platform/httpx"
)

// Handler serves the admin AI model page.
type Handler struct {
	svc *Service
	log *slog.Logger
}

// NewHandler returns a Handler for svc.
func NewHandler(svc *Service, log *slog.Logger) *Handler {
	return &Handler{svc: svc, log: log}
}

// Register adds GET and PUT /api/admin/ai-model and POST /api/admin/ai-model/test, behind
// requireAuth and the admin role.
func (h *Handler) Register(mux *http.ServeMux, requireAuth httpx.Middleware) {
	admin := func(f http.HandlerFunc) http.Handler { return requireAuth(httpx.RequireAdmin(f)) }
	mux.Handle("GET /api/admin/ai-model", admin(h.get))
	mux.Handle("PUT /api/admin/ai-model", admin(h.set))
	mux.Handle("POST /api/admin/ai-model/test", admin(h.test))
}

type stateJSON struct {
	Model     string     `json:"model"`
	Default   string     `json:"default"`
	Saved     string     `json:"saved"`
	UpdatedAt *time.Time `json:"updatedAt"`
	Options   []string   `json:"options"`
	// Ops are the text requests a model can be chosen for; ByOp those given their own model.
	Ops  []string          `json:"ops"`
	ByOp map[string]string `json:"byOp"`
}

func toState(s State, options []string) stateJSON {
	out := stateJSON{Model: s.Model, Default: s.Default, Saved: s.Saved, Options: options, Ops: Ops, ByOp: s.ByOp}
	if !s.UpdatedAt.IsZero() {
		out.UpdatedAt = &s.UpdatedAt
	}
	return out
}

// get serves GET /api/admin/ai-model.
func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	s, err := h.svc.State(r.Context())
	if err != nil {
		h.fail(w, r, "read ai model failed", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toState(s, h.svc.Options()))
}

type modelBody struct {
	Model string `json:"model"`
	// ByOp is only read by PUT: the ops' own models, "" for the model of every op.
	ByOp map[string]string `json:"byOp"`
}

// set serves PUT /api/admin/ai-model: the model of every op (empty for GEMINI_MODEL) and the ops'
// own models, replacing the earlier choice.
func (h *Handler) set(w http.ResponseWriter, r *http.Request) {
	var body modelBody
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		return // DecodeJSON wrote the response
	}
	s, err := h.svc.Set(r.Context(), body.Model, body.ByOp)
	switch {
	case errors.Is(err, ErrUnknownModel):
		httpx.WriteFieldErrors(w, map[string]string{"model": "Model không có trong danh sách"})
		return
	case errors.Is(err, ErrUnknownOp):
		httpx.WriteFieldErrors(w, map[string]string{"byOp": "Việc AI không có trong danh sách"})
		return
	}
	if err != nil {
		h.fail(w, r, "save ai model failed", err)
		return
	}
	h.log.InfoContext(r.Context(), "ai model changed", slog.String("model", s.Model), slog.Int("ops", len(s.ByOp)))
	httpx.WriteJSON(w, http.StatusOK, toState(s, h.svc.Options()))
}

// Outcome of a test, said in words on the page.
const (
	outcomeOK      = "ok"
	outcomeTimeout = "timeout"
	outcomeQuota   = "quota"
	outcomeBusy    = "busy"
	outcomeError   = "error"
)

type testJSON struct {
	Outcome    string `json:"outcome"`
	DurationMs int64  `json:"durationMs"`
}

// test serves POST /api/admin/ai-model/test: one short request with the model. A model that fails
// is still an answer (200); only a model off the list or no provider is an error.
func (h *Handler) test(w http.ResponseWriter, r *http.Request) {
	// The server writes within 15 seconds; a test waits up to testTimeout for the model.
	if err := http.NewResponseController(w).SetWriteDeadline(time.Now().Add(testTimeout + 5*time.Second)); err != nil &&
		!errors.Is(err, http.ErrNotSupported) {
		h.log.WarnContext(r.Context(), "ai model: extend write deadline", slog.Any("error", err))
	}
	var body modelBody
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		return // DecodeJSON wrote the response
	}
	d, err := h.svc.Test(r.Context(), body.Model)
	switch {
	case errors.Is(err, ErrUnknownModel):
		httpx.WriteFieldErrors(w, map[string]string{"model": "Model không có trong danh sách"})
		return
	case errors.Is(err, ErrNoProvider), errors.Is(err, ai.ErrNotConfigured):
		httpx.WriteError(w, http.StatusServiceUnavailable, "ai_not_configured", "Chưa cấu hình AI")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, testJSON{Outcome: testOutcome(err), DurationMs: d.Milliseconds()})
}

func testOutcome(err error) string {
	switch {
	case err == nil:
		return outcomeOK
	case errors.Is(err, context.DeadlineExceeded):
		return outcomeTimeout
	case errors.Is(err, ai.ErrQuota):
		return outcomeQuota
	case errors.Is(err, ai.ErrBusy):
		return outcomeBusy
	default:
		return outcomeError
	}
}

func (h *Handler) fail(w http.ResponseWriter, r *http.Request, msg string, err error) {
	h.log.ErrorContext(r.Context(), msg, slog.Any("error", err))
	httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "Có lỗi xảy ra, vui lòng thử lại")
}
