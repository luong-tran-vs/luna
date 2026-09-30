// Package health reports whether the backend and its database are reachable.
package health

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/luongtran/luna/backend/internal/platform/httpx"
)

// Pinger checks that a dependency answers before ctx expires.
type Pinger interface {
	Ping(ctx context.Context) error
}

// Response is the body of GET /api/health.
type Response struct {
	Status   string `json:"status"`
	Database string `json:"database"`
}

// Handler serves GET /api/health.
type Handler struct {
	db      Pinger
	timeout time.Duration
	log     *slog.Logger
}

// NewHandler returns a Handler that pings db with the given timeout.
func NewHandler(db Pinger, timeout time.Duration, log *slog.Logger) *Handler {
	return &Handler{db: db, timeout: timeout, log: log}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), h.timeout)
	defer cancel()

	if err := h.db.Ping(ctx); err != nil {
		// Details stay in the log; the response never exposes internal errors.
		h.log.WarnContext(r.Context(), "database ping failed",
			slog.Any("error", err),
			slog.String("request_id", httpx.RequestIDFrom(r.Context())),
		)
		httpx.WriteJSON(w, http.StatusServiceUnavailable, Response{Status: "degraded", Database: "down"})
		return
	}

	httpx.WriteJSON(w, http.StatusOK, Response{Status: "ok", Database: "up"})
}
