package lesson

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/luongtran/luna/backend/internal/platform/httpx"
)

func newReadingAPI(t *testing.T) (*http.ServeMux, string) {
	t.Helper()
	r, l := newReaderEnv(t)
	mux := http.NewServeMux()
	NewReadingHandler(r, slog.New(slog.DiscardHandler)).Register(mux, httpx.RequireAuth(resolver), allowAll)
	return mux, l.ID
}

// allowAll is a guard that lets every request through.
func allowAll(next http.Handler) http.Handler { return next }

func TestReadingRoutesAreGuarded(t *testing.T) {
	t.Parallel()
	r, l := newReaderEnv(t)
	mux := http.NewServeMux()
	deny := func(http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusForbidden) })
	}
	NewReadingHandler(r, slog.New(slog.DiscardHandler)).Register(mux, httpx.RequireAuth(resolver), deny)
	for _, p := range []string{"", "/lookup?q=went&sentence=0", "/vocabulary"} {
		if rec := get(t, mux, "/api/lessons/"+l.ID+p, "learner"); rec.Code != http.StatusForbidden {
			t.Errorf("%s: %d", p, rec.Code)
		}
	}
}

func get(t *testing.T, mux http.Handler, path, token string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, http.NoBody)
	if token != "" {
		req.AddCookie(&http.Cookie{Name: httpx.SessionCookieName, Value: token})
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestReadingLessonEndpoint(t *testing.T) {
	t.Parallel()
	mux, id := newReadingAPI(t)

	rec := get(t, mux, "/api/lessons/"+id, "learner")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d %s", rec.Code, rec.Body)
	}
	body := rec.Body.String()
	for _, want := range []string{`"paragraphs":[[0,1],[2,3]]`, `"went":"go"`, `"phrases":[{"text":"gave up","lemma":"give up"}]`, `"audioUrl":null`, `"topic":"Family"`, `"level":"A1"`} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing %s: %s", want, body)
		}
	}
	for _, leak := range []string{"annotations", "meaningVi", "secret", "revision"} {
		if strings.Contains(body, leak) {
			t.Errorf("body leaks %q: %s", leak, body)
		}
	}

	if rec := get(t, mux, "/api/lessons/missing", "learner"); rec.Code != http.StatusNotFound {
		t.Errorf("missing: %d", rec.Code)
	}
	if rec := get(t, mux, "/api/lessons/"+id, ""); rec.Code != http.StatusUnauthorized {
		t.Errorf("anonymous: %d", rec.Code)
	}
}

func TestLookupEndpoint(t *testing.T) {
	t.Parallel()
	mux, id := newReadingAPI(t)
	lookup := func(q, sentence string) *httptest.ResponseRecorder {
		v := url.Values{"q": {q}}
		if sentence != "" {
			v.Set("sentence", sentence)
		}
		return get(t, mux, "/api/lessons/"+id+"/lookup?"+v.Encode(), "learner")
	}

	rec := lookup("went", "0")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"source":"ai"`) || !strings.Contains(rec.Body.String(), `"lemma":"go"`) {
		t.Fatalf("went: %d %s", rec.Code, rec.Body)
	}
	rec = lookup("studies", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"source":"dictionary"`) ||
		!strings.Contains(rec.Body.String(), `"meanings":[{"pos":"V","text":"Học."}]`) {
		t.Fatalf("studies: %d %s", rec.Code, rec.Body)
	}

	if rec := lookup("smoking", "1"); rec.Code != http.StatusNotFound || !strings.Contains(rec.Body.String(), `"not_found"`) {
		t.Errorf("unknown word: %d %s", rec.Code, rec.Body)
	}
	if rec := get(t, mux, "/api/lessons/missing/lookup?q=went", "learner"); rec.Code != http.StatusNotFound ||
		!strings.Contains(rec.Body.String(), `"lesson_not_found"`) {
		t.Errorf("missing lesson: %d %s", rec.Code, rec.Body)
	}

	for name, rec := range map[string]*httptest.ResponseRecorder{
		"empty":        lookup("  ", ""),
		"too long":     lookup(strings.Repeat("a", 101), ""),
		"seven words":  lookup("a b c d e f g", ""),
		"bad sentence": lookup("went", "x"),
	} {
		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s: %d, want 400", name, rec.Code)
		}
	}
	if rec := get(t, mux, "/api/lessons/"+id+"/lookup?q=went", ""); rec.Code != http.StatusUnauthorized {
		t.Errorf("anonymous: %d", rec.Code)
	}
}
