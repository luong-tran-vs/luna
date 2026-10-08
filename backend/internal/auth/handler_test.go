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
	h.RegisterAdmin(mux, requireAuth)
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

// --- account management (admin) ---

func TestAccountEndpoints(t *testing.T) {
	t.Parallel()
	a := newAPI(t)
	rec := a.do(t, http.MethodPost, "/api/auth/register", validRegister, nil)
	admin := sessionCookie(t, rec)
	adminID := decode(t, rec)["user"].(map[string]any)["id"].(string)
	rec = a.do(t, http.MethodPost, "/api/auth/register", `{"email":"b@example.com","password":"matkhau123"}`, nil)
	guest := sessionCookie(t, rec)
	guestID := decode(t, rec)["user"].(map[string]any)["id"].(string)
	if role := decode(t, rec)["user"].(map[string]any)["role"]; role != "guest" {
		t.Fatalf("new account role = %v", role)
	}

	// list
	rec = a.do(t, http.MethodGet, "/api/admin/users", "", admin)
	users, _ := decode(t, rec)["users"].([]any)
	if rec.Code != http.StatusOK || len(users) != 2 || users[0].(map[string]any)["role"] != "admin" {
		t.Fatalf("list: %d %s", rec.Code, rec.Body)
	}
	if strings.Contains(rec.Body.String(), "assword") {
		t.Fatal("list shows the password")
	}
	for _, ep := range []struct{ method, path, body string }{
		{http.MethodGet, "/api/admin/users", ""},
		{http.MethodPost, "/api/admin/users", `{}`},
		{http.MethodPut, "/api/admin/users/" + guestID, `{}`},
		{http.MethodDelete, "/api/admin/users/" + guestID, ""},
	} {
		if rec := a.do(t, ep.method, ep.path, ep.body, guest); rec.Code != http.StatusForbidden {
			t.Errorf("guest %s %s: %d", ep.method, ep.path, rec.Code)
		}
	}

	// create
	rec = a.do(t, http.MethodPost, "/api/admin/users", `{"email":" New@Example.com ","password":"matkhau123","role":"member"}`, admin)
	created := decode(t, rec)["user"].(map[string]any)
	if rec.Code != http.StatusCreated || created["email"] != "new@example.com" || created["role"] != "member" {
		t.Fatalf("create: %d %s", rec.Code, rec.Body)
	}
	if rec := a.do(t, http.MethodPost, "/api/auth/login", `{"email":"new@example.com","password":"matkhau123"}`, nil); rec.Code != http.StatusOK {
		t.Fatalf("login as created: %d", rec.Code)
	}
	rec = a.do(t, http.MethodPost, "/api/admin/users", `{"email":"x","password":"1","role":"boss"}`, admin)
	if body := decode(t, rec); rec.Code != http.StatusBadRequest || len(body["fields"].(map[string]any)) != 3 {
		t.Fatalf("create invalid: %d %s", rec.Code, rec.Body)
	}
	if rec := a.do(t, http.MethodPost, "/api/admin/users", `{"email":"b@example.com","password":"matkhau123","role":"guest"}`, admin); rec.Code != http.StatusConflict {
		t.Fatalf("create taken: %d", rec.Code)
	}

	// update: role and email, password kept when empty
	rec = a.do(t, http.MethodPut, "/api/admin/users/"+guestID, `{"email":"b2@example.com","password":"","role":"member"}`, admin)
	if u := decode(t, rec)["user"].(map[string]any); rec.Code != http.StatusOK || u["role"] != "member" || u["email"] != "b2@example.com" {
		t.Fatalf("update: %d %s", rec.Code, rec.Body)
	}
	if role := decode(t, a.do(t, http.MethodGet, "/api/auth/me", "", guest))["user"].(map[string]any)["role"]; role != "member" {
		t.Fatalf("me after promotion = %v", role)
	}
	if rec := a.do(t, http.MethodPost, "/api/auth/login", `{"email":"b2@example.com","password":"matkhau123"}`, nil); rec.Code != http.StatusOK {
		t.Fatalf("password not kept: %d", rec.Code)
	}
	rec = a.do(t, http.MethodPut, "/api/admin/users/"+guestID, `{"email":"b2@example.com","password":"moimatkhau1","role":"member"}`, admin)
	if rec.Code != http.StatusOK {
		t.Fatalf("new password: %d %s", rec.Code, rec.Body)
	}
	if rec := a.do(t, http.MethodPost, "/api/auth/login", `{"email":"b2@example.com","password":"moimatkhau1"}`, nil); rec.Code != http.StatusOK {
		t.Fatalf("login with new password: %d", rec.Code)
	}
	if rec := a.do(t, http.MethodPut, "/api/admin/users/"+guestID, `{"email":"new@example.com","password":"","role":"member"}`, admin); rec.Code != http.StatusConflict {
		t.Fatalf("update taken email: %d", rec.Code)
	}
	if rec := a.do(t, http.MethodPut, "/api/admin/users/"+guestID, `{"email":"b2@example.com","password":"short","role":"member"}`, admin); rec.Code != http.StatusBadRequest {
		t.Fatalf("update short password: %d", rec.Code)
	}
	// own account: email may change, role may not
	if rec := a.do(t, http.MethodPut, "/api/admin/users/"+adminID, `{"email":"an@example.com","password":"","role":"guest"}`, admin); rec.Code != http.StatusConflict {
		t.Fatalf("own role: %d %s", rec.Code, rec.Body)
	}
	if rec := a.do(t, http.MethodPut, "/api/admin/users/"+adminID, `{"email":"boss@example.com","password":"","role":"admin"}`, admin); rec.Code != http.StatusOK {
		t.Fatalf("own email: %d %s", rec.Code, rec.Body)
	}
	if rec := a.do(t, http.MethodPut, "/api/admin/users/missing", `{"email":"z@example.com","password":"","role":"guest"}`, admin); rec.Code != http.StatusNotFound {
		t.Fatalf("update missing: %d", rec.Code)
	}

	// delete
	if rec := a.do(t, http.MethodDelete, "/api/admin/users/"+adminID, "", admin); rec.Code != http.StatusConflict {
		t.Fatalf("delete self: %d", rec.Code)
	}
	if rec := a.do(t, http.MethodDelete, "/api/admin/users/"+guestID, "", admin); rec.Code != http.StatusNoContent {
		t.Fatalf("delete: %d %s", rec.Code, rec.Body)
	}
	if rec := a.do(t, http.MethodGet, "/api/auth/me", "", guest); rec.Code != http.StatusUnauthorized {
		t.Fatalf("deleted account still logged in: %d", rec.Code)
	}
	if rec := a.do(t, http.MethodDelete, "/api/admin/users/"+guestID, "", admin); rec.Code != http.StatusNotFound {
		t.Fatalf("delete again: %d", rec.Code)
	}
}

func TestParseRole(t *testing.T) {
	t.Parallel()
	for in, want := range map[string]Role{"admin": RoleAdmin, "member": RoleMember, "guest": RoleGuest, "learner": RoleMember, "": RoleGuest} {
		if got := ParseRole(in); got != want {
			t.Errorf("ParseRole(%q) = %q, want %q", in, got, want)
		}
	}
}
