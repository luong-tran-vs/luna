package settings

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/luongtran/luna/backend/internal/platform/httpx"
)

// Handler serves the settings API (contracts/settings-api.md).
type Handler struct {
	svc *Service
	log *slog.Logger
}

// NewHandler returns a Handler.
func NewHandler(svc *Service, log *slog.Logger) *Handler {
	return &Handler{svc: svc, log: log}
}

// Register adds the routes behind requireAuth.
func (h *Handler) Register(mux *http.ServeMux, requireAuth httpx.Middleware) {
	mux.Handle("GET /api/settings", requireAuth(http.HandlerFunc(h.get)))
	mux.Handle("PUT /api/settings", requireAuth(http.HandlerFunc(h.update)))
}

type settingsJSON struct {
	Theme            string `json:"theme"`
	DailyReviewLimit int    `json:"dailyReviewLimit"`
	Timezone         string `json:"timezone"`
}

func toJSON(s Settings) settingsJSON {
	return settingsJSON{Theme: string(s.Theme), DailyReviewLimit: s.DailyReviewLimit, Timezone: s.Timezone}
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	p, _ := httpx.PrincipalFrom(r.Context())
	s, err := h.svc.Get(r.Context(), p.UserID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toJSON(s))
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	p, _ := httpx.PrincipalFrom(r.Context())
	var req struct {
		Theme            *string `json:"theme"`
		DailyReviewLimit *int    `json:"dailyReviewLimit"`
		Timezone         *string `json:"timezone"`
	}
	if httpx.DecodeJSON(w, r, &req) != nil {
		return
	}
	patch := Patch{DailyReviewLimit: req.DailyReviewLimit, Timezone: req.Timezone}
	if req.Theme != nil {
		t := Theme(*req.Theme)
		patch.Theme = &t
	}
	s, err := h.svc.Update(r.Context(), p.UserID, patch)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toJSON(s))
}

func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	var verr *ValidationError
	switch {
	case errors.As(err, &verr):
		httpx.WriteFieldErrors(w, verr.Fields)
	case errors.Is(err, ErrNotFound):
		// The session outlived its account.
		httpx.WriteError(w, http.StatusUnauthorized, "unauthenticated", "Vui lòng đăng nhập lại")
	default:
		h.log.ErrorContext(r.Context(), "settings request failed", slog.Any("error", err))
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "Có lỗi xảy ra, vui lòng thử lại")
	}
}
