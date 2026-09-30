package httpx_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/luongtran/luna/backend/internal/platform/httpx"
)

var generatedID = regexp.MustCompile(`^[0-9a-f]{32}$`)

func discardLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

func TestRequestID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		incoming string
		wantSame bool
	}{
		{name: "no header generates id", incoming: ""},
		{name: "valid header is reused", incoming: "abc-123", wantSame: true},
		{name: "invalid characters are replaced", incoming: "bad id<script>"},
		{name: "too long header is replaced", incoming: strings.Repeat("a", 65)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var fromCtx string
			h := httpx.RequestID(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				fromCtx = httpx.RequestIDFrom(r.Context())
			}))

			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", http.NoBody)
			if tt.incoming != "" {
				req.Header.Set(httpx.RequestIDHeader, tt.incoming)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			got := rec.Header().Get(httpx.RequestIDHeader)
			if got != fromCtx {
				t.Fatalf("response header %q differs from context id %q", got, fromCtx)
			}
			if tt.wantSame {
				if got != tt.incoming {
					t.Fatalf("id = %q, want reused %q", got, tt.incoming)
				}
				return
			}
			if !generatedID.MatchString(got) {
				t.Fatalf("id = %q, want 32 hex chars", got)
			}
		})
	}
}

func TestRecover(t *testing.T) {
	t.Parallel()

	h := httpx.Recover(discardLogger())(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", http.NoBody))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["error"] != "internal_error" {
		t.Fatalf("body = %v, want error=internal_error", body)
	}
}

func TestLogger(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&buf, nil))

	h := httpx.Chain(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusTeapot)
		}),
		httpx.RequestID,
		httpx.Logger(log),
	)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/health", http.NoBody)
	req.Header.Set(httpx.RequestIDHeader, "req-1")
	h.ServeHTTP(httptest.NewRecorder(), req)

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("log line is not JSON: %v (%q)", err, buf.String())
	}
	if entry["status"] != float64(http.StatusTeapot) {
		t.Errorf("status = %v, want %d", entry["status"], http.StatusTeapot)
	}
	if entry["method"] != http.MethodGet || entry["path"] != "/api/health" {
		t.Errorf("method/path = %v %v", entry["method"], entry["path"])
	}
	if entry["request_id"] != "req-1" {
		t.Errorf("request_id = %v, want req-1", entry["request_id"])
	}
	if _, ok := entry["duration_ms"]; !ok {
		t.Error("duration_ms missing")
	}
}
