package settings

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/luongtran/luna/backend/internal/platform/httpx"
)

func resolver(_ context.Context, token string) (httpx.Principal, error) {
	switch token {
	case "an":
		return httpx.Principal{UserID: "u1", Role: "learner"}, nil
	case "binh":
		return httpx.Principal{UserID: "u2", Role: "admin"}, nil
	}
	return httpx.Principal{}, errors.New("no session")
}

func newAPI() *http.ServeMux {
	mux := http.NewServeMux()
	NewHandler(NewService(newFakeRepo("u1", "u2")), slog.New(slog.DiscardHandler)).Register(mux, httpx.RequireAuth(resolver))
	return mux
}

func call(t *testing.T, mux http.Handler, method, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), method, "/api/settings", strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.AddCookie(&http.Cookie{Name: httpx.SessionCookieName, Value: token})
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestGetSettings(t *testing.T) {
	t.Parallel()
	mux := newAPI()
	rec := call(t, mux, http.MethodGet, "an", "")
	if rec.Code != http.StatusOK || rec.Body.String() != `{"theme":"system","dailyReviewLimit":30,"timezone":"Asia/Ho_Chi_Minh"}`+"\n" {
		t.Fatalf("get: %d %s", rec.Code, rec.Body)
	}
	if rec := call(t, mux, http.MethodGet, "", ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("no session: %d", rec.Code)
	}
}

func TestPutSettings(t *testing.T) {
	t.Parallel()
	mux := newAPI()

	rec := call(t, mux, http.MethodPut, "an", `{"theme":"dark"}`)
	if rec.Code != http.StatusOK || rec.Body.String() != `{"theme":"dark","dailyReviewLimit":30,"timezone":"Asia/Ho_Chi_Minh"}`+"\n" {
		t.Fatalf("theme: %d %s", rec.Code, rec.Body)
	}
	rec = call(t, mux, http.MethodPut, "an", `{"dailyReviewLimit":20,"timezone":"Europe/London"}`)
	if rec.Code != http.StatusOK || rec.Body.String() != `{"theme":"dark","dailyReviewLimit":20,"timezone":"Europe/London"}`+"\n" {
		t.Fatalf("limit + timezone: %d %s", rec.Code, rec.Body)
	}
	if rec := call(t, mux, http.MethodPut, "an", `{}`); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"dailyReviewLimit":20`) {
		t.Fatalf("empty: %d %s", rec.Code, rec.Body)
	}

	// The other user still has the defaults.
	if rec := call(t, mux, http.MethodGet, "binh", ""); !strings.Contains(rec.Body.String(), `"theme":"system"`) {
		t.Fatalf("u2: %s", rec.Body)
	}

	rec = call(t, mux, http.MethodPut, "an", `{"theme":"blue","dailyReviewLimit":201,"timezone":"Mars/Olympus"}`)
	var body struct {
		Fields map[string]string `json:"fields"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || rec.Code != http.StatusBadRequest ||
		body.Fields["theme"] != "Chế độ giao diện không hợp lệ" || body.Fields["dailyReviewLimit"] != "Số thẻ từ 5 đến 200" ||
		body.Fields["timezone"] != "Múi giờ không hợp lệ" {
		t.Fatalf("invalid: %d %s", rec.Code, rec.Body)
	}

	for _, bad := range []string{`{"dailyReviewLimit":"10"}`, `{"dailyReviewLimit":10.5}`, `{"color":"red"}`, `{`} {
		if rec := call(t, mux, http.MethodPut, "an", bad); rec.Code != http.StatusBadRequest {
			t.Errorf("%s: %d", bad, rec.Code)
		}
	}
	if rec := call(t, mux, http.MethodGet, "an", ""); rec.Body.String() != `{"theme":"dark","dailyReviewLimit":20,"timezone":"Europe/London"}`+"\n" {
		t.Fatalf("unchanged after errors: %s", rec.Body)
	}
	if rec := call(t, mux, http.MethodPut, "", `{"theme":"dark"}`); rec.Code != http.StatusUnauthorized {
		t.Fatalf("no session: %d", rec.Code)
	}
}
