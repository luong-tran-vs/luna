package httpx_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/luongtran/luna/backend/internal/platform/httpx"
)

func TestCORS(t *testing.T) {
	t.Parallel()

	const allowed = "http://localhost:4200"

	tests := []struct {
		name        string
		origins     []string
		method      string
		origin      string
		preflight   bool
		wantStatus  int
		wantAllowed string
		wantNext    bool
	}{
		{name: "allowed origin", origins: []string{allowed}, method: http.MethodGet, origin: allowed,
			wantStatus: http.StatusTeapot, wantAllowed: allowed, wantNext: true},
		{name: "preflight of an allowed origin stops here", origins: []string{allowed}, method: http.MethodOptions,
			origin: allowed, preflight: true, wantStatus: http.StatusNoContent, wantAllowed: allowed},
		{name: "unknown origin gets no CORS headers", origins: []string{allowed}, method: http.MethodGet,
			origin: "http://evil.example", wantStatus: http.StatusTeapot, wantNext: true},
		{name: "same-origin request has no Origin", origins: []string{allowed}, method: http.MethodGet,
			wantStatus: http.StatusTeapot, wantNext: true},
		{name: "* allows any origin", origins: []string{"*"}, method: http.MethodGet,
			origin: "http://192.168.1.5:4200", wantStatus: http.StatusTeapot,
			wantAllowed: "http://192.168.1.5:4200", wantNext: true},
		{name: "no origins configured", method: http.MethodGet, origin: allowed,
			wantStatus: http.StatusTeapot, wantNext: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			called := false
			h := httpx.CORS(tt.origins)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				called = true
				w.WriteHeader(http.StatusTeapot)
			}))

			req := httptest.NewRequest(tt.method, "/api/health", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			if tt.preflight {
				req.Header.Set("Access-Control-Request-Method", http.MethodPost)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if called != tt.wantNext {
				t.Fatalf("next called = %v, want %v", called, tt.wantNext)
			}
			if got := rec.Header().Get("Access-Control-Allow-Origin"); got != tt.wantAllowed {
				t.Fatalf("Access-Control-Allow-Origin = %q, want %q", got, tt.wantAllowed)
			}
			wantCreds := ""
			if tt.wantAllowed != "" {
				wantCreds = "true"
			}
			if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != wantCreds {
				t.Fatalf("Access-Control-Allow-Credentials = %q, want %q", got, wantCreds)
			}
		})
	}
}
