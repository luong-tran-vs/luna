package vocab

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
		return httpx.Principal{UserID: "u2", Role: "learner"}, nil
	}
	return httpx.Principal{}, errors.New("no session")
}

func newAPI(t *testing.T) *http.ServeMux {
	t.Helper()
	svc, _ := newTestService()
	mux := http.NewServeMux()
	NewHandler(svc, slog.New(slog.DiscardHandler)).Register(mux, httpx.RequireAuth(resolver))
	return mux
}

func call(t *testing.T, mux http.Handler, method, path, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), method, path, strings.NewReader(body))
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

const cardBody = `{"text":"went","lemma":"go","ipa":"/ɡəʊ/","meaningVi":"đã đi","contextSentence":"We went to the park.","lessonId":"lesson1","source":"ai"}`

func TestSaveCardEndpoint(t *testing.T) {
	t.Parallel()
	mux := newAPI(t)

	rec := call(t, mux, http.MethodPost, "/api/vocab/cards", "an", cardBody)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", rec.Code, rec.Body)
	}
	var created struct{ Card map[string]any }
	_ = json.Unmarshal(rec.Body.Bytes(), &created)
	if created.Card["lemma"] != "go" || created.Card["contextSentence"] != "We went to the park." {
		t.Fatalf("card = %v", created.Card)
	}
	if _, leaked := created.Card["userId"]; leaked {
		t.Fatal("response leaks userId")
	}

	dup := strings.Replace(cardBody, `"lemma":"go"`, `"lemma":" GO "`, 1)
	rec = call(t, mux, http.MethodPost, "/api/vocab/cards", "an", dup)
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), `"card_exists"`) || !strings.Contains(rec.Body.String(), `"card":{`) {
		t.Fatalf("duplicate: %d %s", rec.Code, rec.Body)
	}

	// The other learner saves the same word into their own notebook.
	if rec := call(t, mux, http.MethodPost, "/api/vocab/cards", "binh", cardBody); rec.Code != http.StatusCreated {
		t.Fatalf("other user: %d", rec.Code)
	}
}

func TestSaveCardErrors(t *testing.T) {
	t.Parallel()
	mux := newAPI(t)

	rec := call(t, mux, http.MethodPost, "/api/vocab/cards", "an", `{"text":"","lemma":"","meaningVi":"","lessonId":"x","source":"gpt"}`)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), `"meaningVi"`) {
		t.Fatalf("invalid: %d %s", rec.Code, rec.Body)
	}
	// A client cannot choose whose notebook to write to.
	withUser := strings.Replace(cardBody, `{"text"`, `{"userId":"u2","text"`, 1)
	if rec := call(t, mux, http.MethodPost, "/api/vocab/cards", "an", withUser); rec.Code != http.StatusBadRequest {
		t.Fatalf("userId in body: %d", rec.Code)
	}
	if rec := call(t, mux, http.MethodPost, "/api/vocab/cards", "", cardBody); rec.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous: %d", rec.Code)
	}
}

func TestWordsEndpoint(t *testing.T) {
	t.Parallel()
	mux := newAPI(t)
	call(t, mux, http.MethodPost, "/api/vocab/cards", "an", cardBody)

	rec := call(t, mux, http.MethodGet, "/api/vocab/words", "an", "")
	if rec.Code != http.StatusOK || rec.Body.String() != "{\"words\":[{\"lemma\":\"go\",\"text\":\"went\"}]}\n" {
		t.Fatalf("an: %d %q", rec.Code, rec.Body)
	}
	rec = call(t, mux, http.MethodGet, "/api/vocab/words", "binh", "")
	if rec.Code != http.StatusOK || rec.Body.String() != "{\"words\":[]}\n" {
		t.Fatalf("binh sees: %q", rec.Body)
	}
	if rec := call(t, mux, http.MethodGet, "/api/vocab/words", "", ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous: %d", rec.Code)
	}
}
