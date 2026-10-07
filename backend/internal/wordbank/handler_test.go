package wordbank

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/luongtran/luna/backend/internal/platform/httpx"
)

// newAPI serves the handler; the caller is an admin unless the role header says "learner".
func newAPI(t *testing.T) (*http.ServeMux, *testEnv) {
	t.Helper()
	e := newEnv()
	mux := http.NewServeMux()
	auth := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role := "admin"
			if r.Header.Get("X-Role") != "" {
				role = r.Header.Get("X-Role")
			}
			next.ServeHTTP(w, r.WithContext(httpx.WithPrincipal(r.Context(), httpx.Principal{UserID: "u1", Role: role})))
		})
	}
	NewHandler(e.svc, slog.New(slog.DiscardHandler)).Register(mux, auth)
	return mux, e
}

func do(mux http.Handler, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestWordsAPI(t *testing.T) {
	t.Parallel()
	mux, _ := newAPI(t)

	rec := do(mux, http.MethodPost, "/api/admin/words", `{"lemma":"House"}`)
	if rec.Code != http.StatusCreated || !strings.Contains(rec.Body.String(), `"ipa":"/haʊs/"`) {
		t.Fatalf("add = %d %s", rec.Code, rec.Body)
	}
	if rec = do(mux, http.MethodPost, "/api/admin/words", `{"lemma":"house"}`); rec.Code != http.StatusBadRequest ||
		!strings.Contains(rec.Body.String(), "đã có trong kho") {
		t.Fatalf("duplicate = %d %s", rec.Code, rec.Body)
	}
	if rec = do(mux, http.MethodPost, "/api/admin/words/import", ``); rec.Code != http.StatusOK || rec.Body.String() != "{\"added\":2}\n" {
		t.Fatalf("import = %d %s", rec.Code, rec.Body)
	}

	rec = do(mux, http.MethodGet, "/api/admin/words?missing=ipa", "")
	var page struct {
		Words []struct {
			Lemma    string `json:"lemma"`
			ImageURL string `json:"imageUrl"`
		} `json:"words"`
		Total   int  `json:"total"`
		HasMore bool `json:"hasMore"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil || page.Total != 1 || page.Words[0].Lemma != "table" {
		t.Fatalf("list = %d %s", rec.Code, rec.Body)
	}
	if rec = do(mux, http.MethodGet, "/api/admin/words?page=x", ""); rec.Code != http.StatusBadRequest {
		t.Fatalf("bad page = %d", rec.Code)
	}

	if rec = do(mux, http.MethodPatch, "/api/admin/words/good%20morning", `{"meaningVi":"chào buổi sáng"}`); rec.Code != http.StatusOK ||
		!strings.Contains(rec.Body.String(), "chào buổi sáng") {
		t.Fatalf("update = %d %s", rec.Code, rec.Body)
	}
	if rec = do(mux, http.MethodPatch, "/api/admin/words/nope", `{}`); rec.Code != http.StatusNotFound {
		t.Fatalf("update missing = %d", rec.Code)
	}
	if rec = do(mux, http.MethodDelete, "/api/admin/words/table", ""); rec.Code != http.StatusNoContent {
		t.Fatalf("delete = %d", rec.Code)
	}

	// Learners are refused.
	req := httptest.NewRequest(http.MethodGet, "/api/admin/words", nil)
	req.Header.Set("X-Role", "learner")
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("learner = %d", rec.Code)
	}
}

func TestImageAPI(t *testing.T) {
	t.Parallel()
	mux, e := newAPI(t)
	if _, err := e.svc.Add(t.Context(), Input{Lemma: "house"}); err != nil {
		t.Fatal(err)
	}
	if rec := do(mux, http.MethodGet, "/api/admin/words/house/image", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("no image = %d", rec.Code)
	}

	req := httptest.NewRequest(http.MethodPut, "/api/admin/words/house/image", bytes.NewReader(testPNG(50, 50)))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	var w struct {
		ImageURL string `json:"imageUrl"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &w); err != nil || rec.Code != http.StatusOK ||
		!strings.HasPrefix(w.ImageURL, "/api/admin/words/house/image?v=") {
		t.Fatalf("upload = %d %s", rec.Code, rec.Body)
	}
	if rec = do(mux, http.MethodGet, w.ImageURL, ""); rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "image/jpeg" {
		t.Fatalf("get image = %d %s", rec.Code, rec.Header().Get("Content-Type"))
	}

	if rec = do(mux, http.MethodPost, "/api/admin/words/house/image/generate", `{"style":"tranh vẽ"}`); rec.Code != http.StatusOK {
		t.Fatalf("generate = %d %s", rec.Code, rec.Body)
	}
	e.draw.err = errFake
	if rec = do(mux, http.MethodPost, "/api/admin/words/house/image/generate", `{}`); rec.Code != http.StatusBadGateway {
		t.Fatalf("failed draw = %d %s", rec.Code, rec.Body)
	}
	if rec = do(mux, http.MethodPost, "/api/admin/words/house/image/import", `{"url":"ftp://x"}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("bad link = %d %s", rec.Code, rec.Body)
	}
	if rec = do(mux, http.MethodDelete, "/api/admin/words/house/image", ""); rec.Code != http.StatusOK ||
		!strings.Contains(rec.Body.String(), `"imageUrl":""`) {
		t.Fatalf("delete image = %d %s", rec.Code, rec.Body)
	}
}
