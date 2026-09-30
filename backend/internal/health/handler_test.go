package health_test

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/luongtran/luna/backend/internal/health"
)

type pingerFunc func(ctx context.Context) error

func (f pingerFunc) Ping(ctx context.Context) error { return f(ctx) }

func newServer(p health.Pinger, timeout time.Duration) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /api/health", health.NewHandler(p, timeout, slog.New(slog.DiscardHandler)))
	return mux
}

func TestHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		pinger     pingerFunc
		wantStatus int
		wantBody   health.Response
	}{
		{
			name:       "database up",
			pinger:     func(context.Context) error { return nil },
			wantStatus: http.StatusOK,
			wantBody:   health.Response{Status: "ok", Database: "up"},
		},
		{
			name:       "database down",
			pinger:     func(context.Context) error { return errors.New("connection refused") },
			wantStatus: http.StatusServiceUnavailable,
			wantBody:   health.Response{Status: "degraded", Database: "down"},
		},
		{
			name: "database slower than timeout",
			pinger: func(ctx context.Context) error {
				<-ctx.Done()
				return ctx.Err()
			},
			wantStatus: http.StatusServiceUnavailable,
			wantBody:   health.Response{Status: "degraded", Database: "down"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			const timeout = 50 * time.Millisecond
			rec := httptest.NewRecorder()
			start := time.Now()
			newServer(tt.pinger, timeout).ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/health", http.NoBody))

			if elapsed := time.Since(start); elapsed > 20*timeout {
				t.Fatalf("handler took %v, want about %v", elapsed, timeout)
			}
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", ct)
			}
			var got health.Response
			if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if got != tt.wantBody {
				t.Fatalf("body = %+v, want %+v", got, tt.wantBody)
			}
		})
	}
}

func TestHandlerRejectsOtherMethods(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	up := pingerFunc(func(context.Context) error { return nil })
	newServer(up, time.Second).ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/health", http.NoBody))

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rec.Code)
	}
}
