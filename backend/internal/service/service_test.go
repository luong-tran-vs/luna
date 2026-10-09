package service_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/luongtran/luna/backend/internal/platform/config"
	"github.com/luongtran/luna/backend/internal/service"
	"github.com/luongtran/luna/backend/internal/storage/factory"
)

// newContainer builds every service on a Mongo store whose server does not exist. Opening is lazy
// and nothing here needs data, so this checks the wiring alone.
func newContainer(t *testing.T) *service.Container {
	t.Helper()
	cfg := config.Config{
		DBDriver: "mongo", MongoURI: "mongodb://unreachable.invalid:27017", MongoDatabase: "luna",
		HTTPAddr: ":0", LogLevel: "error", AIProvider: "none", DictionaryPath: "does-not-exist.db",
	}
	store, err := factory.Open(cfg)
	if err != nil {
		t.Fatalf("factory.Open: %v", err)
	}
	t.Cleanup(func() { _ = store.Close(context.Background()) })

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	c, err := service.Init(t.Context(), cfg, log, store)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

// apiRoutes is every route the API registers (one line per "METHOD /path"). A route that goes
// missing in a refactor shows up here as a 404.
var apiRoutes = strings.Fields(`
	DELETE_/api/admin/lessons/{id}
	DELETE_/api/admin/topics/{id}
	DELETE_/api/vocab/cards/{id}
	GET_/api/admin/lessons
	GET_/api/admin/lessons/{id}
	GET_/api/admin/topics
	GET_/api/admin/topics/{id}/roadmap
	GET_/api/admin/topics/{id}/word-plan
	GET_/api/admin/topics/{id}/words
	GET_/api/auth/me
	GET_/api/dashboard
	GET_/api/export
	GET_/api/goals
	GET_/api/health
	GET_/api/lessons/mine
	GET_/api/lessons/{id}
	GET_/api/lessons/{id}/dictation/summary
	GET_/api/lessons/{id}/lookup
	GET_/api/lessons/{id}/practice
	GET_/api/lessons/{id}/study
	GET_/api/lessons/{id}/vocabulary
	GET_/api/lessons/{id}/writing
	GET_/api/settings
	GET_/api/stats
	GET_/api/topics
	GET_/api/vocab/cards
	GET_/api/vocab/lessons
	GET_/api/vocab/review/due
	GET_/api/vocab/words
	GET_/api/writings
	GET_/api/writings/unseen-count
	GET_/api/writings/{id}
	PATCH_/api/vocab/cards/{id}
	POST_/api/admin/lessons
	POST_/api/admin/lessons/{id}/practice/regenerate
	POST_/api/admin/lessons/{id}/retry
	POST_/api/admin/topics
	POST_/api/admin/topics/{id}/generate
	POST_/api/admin/topics/{id}/words/suggest
	POST_/api/auth/login
	POST_/api/auth/logout
	POST_/api/auth/register
	POST_/api/goals
	POST_/api/lessons/{id}/answers
	POST_/api/lessons/{id}/ask
	POST_/api/lessons/{id}/dictation
	POST_/api/lessons/{id}/steps/write/skip
	POST_/api/lessons/{id}/steps/{step}/complete
	POST_/api/lessons/{id}/writing/submit
	POST_/api/vocab/cards
	POST_/api/vocab/cards/bulk
	POST_/api/vocab/cards/{id}/review
	POST_/api/vocab/practice-misses
	POST_/api/writings/{id}/regrade
	POST_/api/writings/{id}/seen
	PUT_/api/admin/lessons/{id}
	PUT_/api/admin/lessons/{id}/annotations
	PUT_/api/admin/lessons/{id}/extras
	PUT_/api/admin/topics/{id}
	PUT_/api/admin/topics/{id}/roadmap
	PUT_/api/admin/topics/{id}/words
	PUT_/api/lessons/{id}/position
	PUT_/api/lessons/{id}/writing
	PUT_/api/settings
`)

func TestEveryRouteIsRegistered(t *testing.T) {
	t.Parallel()
	h := newContainer(t).Handler()

	for _, route := range apiRoutes {
		method, path, _ := strings.Cut(route, "_")
		path = strings.NewReplacer("{id}", "0123456789abcdef01234567", "{step}", "read").Replace(path)
		t.Run(route, func(t *testing.T) {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(method, path, strings.NewReader("{}")))
			// Anything but 404 (no such route) or 405 (the path exists but not for this method).
			if rec.Code == http.StatusNotFound || rec.Code == http.StatusMethodNotAllowed {
				t.Fatalf("%s %s = %d: route is not registered", method, path, rec.Code)
			}
		})
	}
}

func TestUnknownRouteIs404(t *testing.T) {
	t.Parallel()
	rec := httptest.NewRecorder()
	newContainer(t).Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/nope", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestProtectedRoutesNeedASession(t *testing.T) {
	t.Parallel()
	h := newContainer(t).Handler()
	for _, path := range []string{"/api/auth/me", "/api/dashboard", "/api/export", "/api/vocab/cards", "/api/admin/topics", "/api/settings"} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("GET %s without a session = %d, want 401", path, rec.Code)
		}
	}
}

func TestLiveTextNeedsNothing(t *testing.T) {
	t.Parallel()
	rec := httptest.NewRecorder()
	newContainer(t).Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/test", nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Luna API đang chạy") {
		t.Fatalf("/api/test = %d %s", rec.Code, rec.Body)
	}
}

func TestHealthReportsTheDatabase(t *testing.T) {
	t.Parallel()
	rec := httptest.NewRecorder()
	newContainer(t).Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/health", nil))
	if rec.Code != http.StatusServiceUnavailable || !strings.Contains(rec.Body.String(), `"database":"down"`) {
		t.Fatalf("health = %d %s", rec.Code, rec.Body)
	}
	if rec.Header().Get("X-Request-Id") == "" {
		t.Error("the request-id middleware is not applied")
	}
}

func TestWorkerStopsWhenItsContextEnds(t *testing.T) {
	t.Parallel()
	c := newContainer(t)
	ctx, cancel := context.WithCancel(t.Context())
	wait := c.StartWorker(ctx)
	cancel()

	done := make(chan struct{})
	go func() { wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("the worker did not stop")
	}
	if err := c.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}
