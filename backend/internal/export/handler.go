package export

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/luongtran/luna/backend/internal/platform/httpx"
)

// Handler serves GET /api/export (contracts/export-api.md).
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
	mux.Handle("GET /api/export", requireAuth(http.HandlerFunc(h.export)))
}

func (h *Handler) export(w http.ResponseWriter, r *http.Request) {
	p, _ := httpx.PrincipalFrom(r.Context())
	e, name, err := h.svc.Build(r.Context(), p.UserID)
	switch {
	case errors.Is(err, ErrNotFound):
		httpx.WriteError(w, http.StatusUnauthorized, "unauthenticated", "Vui lòng đăng nhập lại")
		return
	case err != nil:
		h.log.ErrorContext(r.Context(), "export failed", slog.Any("error", err))
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "Có lỗi xảy ra, vui lòng thử lại")
		return
	}
	body, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		h.log.ErrorContext(r.Context(), "export encoding failed", slog.Any("error", err))
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "Có lỗi xảy ra, vui lòng thử lại")
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`"`)
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}
