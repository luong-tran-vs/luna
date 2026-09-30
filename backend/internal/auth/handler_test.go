package auth

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/luongtran/luna/backend/internal/platform/httpx"
)

type api struct {
	env *testEnv
	mux http.Handler
}

func newAPI(t *testing.T) *api {
	t.Helper()
	env := newTestEnv(t)
	h := NewHandler(env.svc, false, slog.New(slog.DiscardHandler))
	requireAuth := httpx.RequireAuth(h.ResolvePrincipal)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/auth/register", h.Register)
	mux.HandleFunc("POST /api/auth/login", h.Login)
	mux.HandleFunc("POST /api/auth/logout", h.Logout)
	mux.Handle("GET /api/auth/me", requireAuth(http.HandlerFunc(h.Me)))
	return &api{env: env, mux: mux}
}

func (a *api) do(t *testing.T, method, path, body string, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	a.mux.ServeHTTP(rec, req)
	return rec
}

func sessionCookie(t *testing.T, rec *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	for _, c := range rec.Result().Cookies() {
		if c.Name == httpx.SessionCookieName {
			return c
		}
	}
	t.Fatalf("no %s cookie in response (headers %v)", httpx.SessionCookieName, rec.Header())
	return nil
}

func decode(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("body %q is not JSON: %v", rec.Body.String(), err)
	}
	return out
}

const validRegister = `{"email":"An@Example.com","password":"matkhau123","timezone":"Europe/Paris"}`

// --- Me (foundation) ---

func TestMe(t *testing.T) {
	t.Parallel()
	a := newAPI(t)

	if rec := a.do(t, http.MethodGet, "/api/auth/me", "", nil); rec.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous /me: status %d, want 401", rec.Code)
	}

	cookie := sessionCookie(t, a.do(t, http.MethodPost, "/api/auth/register", validRegister, nil))
	rec := a.do(t, http.MethodGet, "/api/auth/me", "", cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("/me: status %d body %s", rec.Code, rec.Body)
	}
	user := decode(t, rec)["user"].(map[string]any)
	if user["email"] != "an@example.com" || user["role"] != "admin" || user["timezone"] != "Europe/Paris" {
		t.Fatalf("/me user = %v", user)
	}
}

// --- Register (US1) ---

func TestRegisterEndpoint(t *testing.T) {
	t.Parallel()
	a := newAPI(t)

	rec := a.do(t, http.MethodPost, "/api/auth/register", validRegister, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d body %s", rec.Code, rec.Body)
	}
	if strings.Contains(rec.Body.String(), "passwordHash") || strings.Contains(rec.Body.String(), "argon2") {
		t.Fatalf("response leaks password hash: %s", rec.Body)
	}
	c := sessionCookie(t, rec)
	if !c.HttpOnly || c.SameSite != http.SameSiteStrictMode || c.MaxAge != 2592000 || c.Path != "/" || c.Secure {
		t.Fatalf("cookie flags = %+v", c)
	}
}

func TestRegisterEndpointErrors(t *testing.T) {
	t.Parallel()
	a := newAPI(t)
	a.do(t, http.MethodPost, "/api/auth/register", validRegister, nil)

	tests := []struct {
		name        string
		body        string
		contentType string
		wantStatus  int
		wantCode    string
		wantFields  []string
	}{
		{name: "bad email and short password", body: `{"email":"x","password":"123"}`, wantStatus: 400, wantCode: "validation_failed", wantFields: []string{"email", "password"}},
		{name: "duplicate email other case", body: `{"email":"AN@example.com","password":"matkhau123"}`, wantStatus: 409, wantCode: "email_taken"},
		{name: "unknown field", body: `{"email":"b@example.com","password":"matkhau123","role":"admin"}`, wantStatus: 400, wantCode: "invalid_body"},
		{name: "not json", body: `email=b@example.com`, contentType: "application/x-www-form-urlencoded", wantStatus: 415, wantCode: "unsupported_media_type"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/auth/register", strings.NewReader(tt.body))
			ct := tt.contentType
			if ct == "" {
				ct = "application/json"
			}
			req.Header.Set("Content-Type", ct)
			rec := httptest.NewRecorder()
			a.mux.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status %d body %s, want %d", rec.Code, rec.Body, tt.wantStatus)
			}
			body := decode(t, rec)
			if body["error"] != tt.wantCode {
				t.Fatalf("error = %v, want %s", body["error"], tt.wantCode)
			}
			for _, f := range tt.wantFields {
				if fields, _ := body["fields"].(map[string]any); fields[f] == nil {
					t.Errorf("missing field error %q in %v", f, body)
				}
			}
		})
	}
}

// --- Login / Logout (US2) ---

func TestLoginEndpoint(t *testing.T) {
	t.Parallel()
	a := newAPI(t)
	a.do(t, http.MethodPost, "/api/auth/register", validRegister, nil)

	rec := a.do(t, http.MethodPost, "/api/auth/login", `{"email":"an@example.com","password":"matkhau123"}`, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("login: status %d body %s", rec.Code, rec.Body)
	}
	sessionCookie(t, rec)

	if rec := a.do(t, http.MethodPost, "/api/auth/login", `{"email":"","password":""}`, nil); rec.Code != http.StatusBadRequest {
		t.Fatalf("missing fields: status %d, want 400", rec.Code)
	}
}

func TestLoginFailuresAreIndistinguishable(t *testing.T) {
	t.Parallel()
	a := newAPI(t)
	a.do(t, http.MethodPost, "/api/auth/register", validRegister, nil)

	wrongPassword := a.do(t, http.MethodPost, "/api/auth/login", `{"email":"an@example.com","password":"sai-mat-khau"}`, nil)
	unknownEmail := a.do(t, http.MethodPost, "/api/auth/login", `{"email":"khong-co@example.com","password":"sai-mat-khau"}`, nil)

	if wrongPassword.Code != http.StatusUnauthorized || unknownEmail.Code != http.StatusUnauthorized {
		t.Fatalf("statuses %d / %d, want 401 / 401", wrongPassword.Code, unknownEmail.Code)
	}
	if wrongPassword.Body.String() != unknownEmail.Body.String() {
		t.Fatalf("bodies differ:\n%s\n%s", wrongPassword.Body, unknownEmail.Body)
	}
	if decode(t, wrongPassword)["error"] != "invalid_credentials" {
		t.Fatalf("body = %s", wrongPassword.Body)
	}
}

func TestLogoutEndpoint(t *testing.T) {
	t.Parallel()
	a := newAPI(t)
	cookie := sessionCookie(t, a.do(t, http.MethodPost, "/api/auth/register", validRegister, nil))

	rec := a.do(t, http.MethodPost, "/api/auth/logout", "", cookie)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("logout: status %d", rec.Code)
	}
	if cleared := sessionCookie(t, rec); cleared.MaxAge >= 0 || cleared.Value != "" {
		t.Fatalf("cookie not cleared: %+v", cleared)
	}
	if rec := a.do(t, http.MethodGet, "/api/auth/me", "", cookie); rec.Code != http.StatusUnauthorized {
		t.Fatalf("/me with old cookie: status %d, want 401", rec.Code)
	}
	if rec := a.do(t, http.MethodPost, "/api/auth/logout", "", nil); rec.Code != http.StatusNoContent {
		t.Fatalf("logout without cookie: status %d, want 204", rec.Code)
	}
}

// --- Lockout (US3) ---

func TestLoginLockedEndpoint(t *testing.T) {
	t.Parallel()
	a := newAPI(t)
	a.do(t, http.MethodPost, "/api/auth/register", validRegister, nil)

	var rec *httptest.ResponseRecorder
	for range MaxFailedLogins {
		rec = a.do(t, http.MethodPost, "/api/auth/login", `{"email":"an@example.com","password":"sai"}`, nil)
	}
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("5th failure: status %d, want 429", rec.Code)
	}
	if got := rec.Header().Get("Retry-After"); got != "900" {
		t.Errorf("Retry-After = %q, want 900", got)
	}
	body := decode(t, rec)
	if body["error"] != "account_locked" || body["retryAfterSeconds"] != float64(900) {
		t.Errorf("body = %v", body)
	}
	if msg, _ := body["message"].(string); !strings.Contains(msg, "15 phút") {
		t.Errorf("message = %q, want minutes", msg)
	}
}
