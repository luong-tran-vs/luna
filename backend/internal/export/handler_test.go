package export

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/luongtran/luna/backend/internal/platform/httpx"
)

func resolver(_ context.Context, token string) (httpx.Principal, error) {
	switch token {
	case "an":
		return httpx.Principal{UserID: "u1", Role: "learner"}, nil
	case "ghost":
		return httpx.Principal{UserID: "gone", Role: "learner"}, nil
	}
	return httpx.Principal{}, errors.New("no session")
}

func get(t *testing.T, repo *fakeRepo, token string) *httptest.ResponseRecorder {
	t.Helper()
	mux := http.NewServeMux()
	now := time.Date(2026, 9, 30, 7, 0, 0, 0, time.UTC)
	NewHandler(newSvc(repo, now), slog.New(slog.DiscardHandler)).Register(mux, httpx.RequireAuth(resolver))
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/export", nil)
	if token != "" {
		req.AddCookie(&http.Cookie{Name: httpx.SessionCookieName, Value: token})
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestExportEndpoint(t *testing.T) {
	t.Parallel()
	rec := get(t, seed(), "an")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
	h := rec.Header()
	if h.Get("Content-Type") != "application/json; charset=utf-8" ||
		h.Get("Content-Disposition") != `attachment; filename="luna-export-20260930.json"` ||
		h.Get("Cache-Control") != "no-store" {
		t.Fatalf("headers = %v", h)
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{
		"version", "exportedAt", "account", "settings", "cards", "reviewLogs", "goals", "lessonProgress", "studyDays",
		"dictationResults", "lessons",
	} {
		if _, ok := body[key]; !ok {
			t.Errorf("missing %s", key)
		}
	}
}

func TestExportErrors(t *testing.T) {
	t.Parallel()
	if rec := get(t, seed(), ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("no session: %d", rec.Code)
	}
	if rec := get(t, seed(), "ghost"); rec.Code != http.StatusUnauthorized {
		t.Fatalf("account gone: %d", rec.Code)
	}
	r := seed()
	r.fail = true
	if rec := get(t, r, "an"); rec.Code != http.StatusInternalServerError {
		t.Fatalf("db error: %d", rec.Code)
	}
}
