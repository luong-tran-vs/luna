package httpx_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/luongtran/luna/backend/internal/platform/httpx"
)

func fakeResolver(ctx context.Context, token string) (httpx.Principal, error) {
	switch token {
	case "admin-token":
		return httpx.Principal{UserID: "u1", Email: "admin@example.com", Role: "admin"}, nil
	case "learner-token":
		return httpx.Principal{UserID: "u2", Email: "hoc@example.com", Role: "learner"}, nil
	default:
		return httpx.Principal{}, errors.New("no session")
	}
}

func serve(t *testing.T, h http.Handler, token string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", http.NoBody)
	if token != "" {
		req.AddCookie(&http.Cookie{Name: httpx.SessionCookieName, Value: token})
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestRequireAuth(t *testing.T) {
	t.Parallel()

	var seen httpx.Principal
	h := httpx.RequireAuth(fakeResolver)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		seen, _ = httpx.PrincipalFrom(r.Context())
	}))

	if rec := serve(t, h, ""); rec.Code != http.StatusUnauthorized || !strings.Contains(rec.Body.String(), `"unauthenticated"`) {
		t.Errorf("no cookie: status %d body %s", rec.Code, rec.Body)
	}
	if rec := serve(t, h, "bogus"); rec.Code != http.StatusUnauthorized {
		t.Errorf("bad token: status %d, want 401", rec.Code)
	}
	if rec := serve(t, h, "learner-token"); rec.Code != http.StatusOK {
		t.Errorf("valid token: status %d, want 200", rec.Code)
	}
	if seen.Email != "hoc@example.com" || seen.Role != "learner" {
		t.Errorf("principal = %+v", seen)
	}
}

func TestRequireAdmin(t *testing.T) {
	t.Parallel()

	ok := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	chained := httpx.RequireAuth(fakeResolver)(httpx.RequireAdmin(ok))

	if rec := serve(t, chained, "admin-token"); rec.Code != http.StatusOK {
		t.Errorf("admin: status %d, want 200", rec.Code)
	}
	if rec := serve(t, chained, "learner-token"); rec.Code != http.StatusForbidden || !strings.Contains(rec.Body.String(), `"forbidden"`) {
		t.Errorf("learner: status %d body %s", rec.Code, rec.Body)
	}
	if rec := serve(t, chained, ""); rec.Code != http.StatusUnauthorized {
		t.Errorf("anonymous: status %d, want 401", rec.Code)
	}

	// RequireAdmin alone, with no principal in context, must not let the request through.
	if rec := serve(t, httpx.RequireAdmin(ok), ""); rec.Code != http.StatusUnauthorized {
		t.Errorf("missing principal: status %d, want 401", rec.Code)
	}
}
