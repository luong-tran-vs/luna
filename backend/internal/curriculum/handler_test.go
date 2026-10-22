package curriculum

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/luongtran/luna/backend/internal/lesson"
	"github.com/luongtran/luna/backend/internal/platform/httpx"
)

func serve(t *testing.T, s *Service, role, method, path string) (int, map[string]any) {
	t.Helper()
	mux := http.NewServeMux()
	auth := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(httpx.WithPrincipal(r.Context(), httpx.Principal{UserID: "u1", Role: role})))
		})
	}
	NewHandler(s, slog.New(slog.NewTextHandler(io.Discard, nil))).Register(mux, auth)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(method, path, nil))
	var body map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	return rec.Code, body
}

func TestHandler(t *testing.T) {
	s, _, fl := newService()
	if code, _ := serve(t, s, "learner", "GET", "/api/admin/curriculum"); code != http.StatusForbidden {
		t.Errorf("learner: %d", code)
	}
	code, body := serve(t, s, "admin", "GET", "/api/admin/curriculum?level=A1")
	if code != http.StatusOK || body["main"] != 36.0 || body["applied"] != false {
		t.Fatalf("board: %d %v", code, body["main"])
	}
	music := body["topics"].([]any)[8].(map[string]any)
	slot := music["slots"].([]any)[1].(map[string]any)
	if music["number"] != 9.0 || slot["lesson"].(map[string]any)["draft"] != true || slot["checks"] == nil {
		t.Errorf("music: %v", music)
	}
	if code, body := serve(t, s, "admin", "POST", "/api/admin/curriculum/slots/a1-01-1/generate"); code != http.StatusConflict || body["error"] != "not_applied" {
		t.Errorf("not applied: %d %v", code, body)
	}
	if code, _ := serve(t, s, "admin", "GET", "/api/admin/curriculum?level=C2"); code != http.StatusNotFound {
		t.Errorf("C2: %d", code)
	}
	if code, body := serve(t, s, "admin", "POST", "/api/admin/curriculum/apply?level=A1"); code != http.StatusOK || body["applied"] != true {
		t.Errorf("apply: %d %v", code, body["applied"])
	}
	fl.drafts = []lesson.Draft{{Content: "Hello! I am Minh."}}
	code, body = serve(t, s, "admin", "POST", "/api/admin/curriculum/slots/a1-01-1/generate")
	if code != http.StatusOK || body["title"] != "Hello! I'm Minh" || body["lesson"] == nil {
		t.Errorf("generate: %d %v", code, body)
	}
	code, body = serve(t, s, "admin", "GET", "/api/admin/curriculum/checks/l1")
	if code != http.StatusOK || len(body["items"].([]any)) == 0 {
		t.Errorf("checks: %d %v", code, body)
	}
	if code, body := serve(t, s, "admin", "POST", "/api/admin/curriculum/lessons/l4/remove"); code != http.StatusOK || body["deleted"] != true {
		t.Errorf("remove a draft: %d %v", code, body)
	}
	if code, _ := serve(t, s, "admin", "GET", "/api/admin/curriculum/checks/nope"); code != http.StatusNotFound {
		t.Errorf("checks of no lesson: %d", code)
	}
}
