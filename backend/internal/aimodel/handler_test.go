package aimodel_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/luongtran/luna/backend/internal/ai"
	"github.com/luongtran/luna/backend/internal/aimodel"
	"github.com/luongtran/luna/backend/internal/platform/httpx"
)

func serve(t *testing.T, svc *aimodel.Service, role, method, path, body string) (int, map[string]any) {
	t.Helper()
	mux := http.NewServeMux()
	auth := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(httpx.WithPrincipal(r.Context(), httpx.Principal{UserID: "u1", Role: role})))
		})
	}
	aimodel.NewHandler(svc, slog.New(slog.DiscardHandler)).Register(mux, auth)
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	return rec.Code, out
}

func TestHandler(t *testing.T) {
	t.Parallel()
	tester := &fakeTester{}
	svc, _ := newService(&memRepo{}, tester)
	if code, _ := serve(t, svc, "learner", "GET", "/api/admin/ai-model", ""); code != http.StatusForbidden {
		t.Errorf("learner: %d", code)
	}
	code, body := serve(t, svc, "admin", "GET", "/api/admin/ai-model", "")
	if code != http.StatusOK || body["model"] != "gemini-3.5-flash-lite" || body["saved"] != "" ||
		body["updatedAt"] != nil || len(body["byOp"].(map[string]any)) != 0 || len(body["options"].([]any)) != len(aimodel.GeminiOptions)+len(aimodel.GroqOptions)+len(aimodel.OpenAIOptions) {
		t.Fatalf("get: %d %v", code, body)
	}
	if code, body := serve(t, svc, "admin", "PUT", "/api/admin/ai-model", `{"model":"gpt-5"}`); code != http.StatusBadRequest || body["fields"] == nil {
		t.Errorf("unknown model: %d %v", code, body)
	}
	code, body = serve(t, svc, "admin", "PUT", "/api/admin/ai-model", `{"model":"gemini-3.6-flash"}`)
	if code != http.StatusOK || body["model"] != "gemini-3.6-flash" || body["updatedAt"] == nil {
		t.Errorf("set: %d %v", code, body)
	}
	if code, body := serve(t, svc, "admin", "PUT", "/api/admin/ai-model", `{"model":"","byOp":{"x":"gemini-3.5-flash"}}`); code != http.StatusBadRequest || body["fields"] == nil {
		t.Errorf("unknown op: %d %v", code, body)
	}
	code, body = serve(t, svc, "admin", "PUT", "/api/admin/ai-model", `{"model":"","byOp":{"talk":"openai/gpt-oss-120b"}}`)
	if byOp, _ := body["byOp"].(map[string]any); code != http.StatusOK || byOp["talk"] != "openai/gpt-oss-120b" || len(body["ops"].([]any)) != len(aimodel.Ops) {
		t.Errorf("set by op: %d %v", code, body)
	}

	for _, c := range []struct {
		err  error
		want string
	}{
		{nil, "ok"},
		{context.DeadlineExceeded, "timeout"},
		{ai.ErrQuota, "quota"},
		{ai.ErrBusy, "busy"},
		{ai.ErrInvalidKey, "error"},
	} {
		tester.err = c.err
		code, body := serve(t, svc, "admin", "POST", "/api/admin/ai-model/test", `{"model":"gemini-3.8-flash"}`)
		if code != http.StatusOK || body["outcome"] != c.want || body["durationMs"] == nil {
			t.Errorf("test with %v: %d %v", c.err, code, body)
		}
	}
	noAI, _ := newService(&memRepo{}, nil)
	if code, _ := serve(t, noAI, "admin", "POST", "/api/admin/ai-model/test", `{"model":"gemini-3.8-flash"}`); code != http.StatusServiceUnavailable {
		t.Errorf("no provider: %d", code)
	}
}
