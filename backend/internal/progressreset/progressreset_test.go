package progressreset

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/luongtran/luna/backend/internal/platform/httpx"
)

type fakeRepo struct {
	calls []string
	fail  Part
}

func (f *fakeRepo) Reset(_ context.Context, userID string, part Part, _ time.Time) error {
	if part == f.fail {
		return errors.New("boom")
	}
	f.calls = append(f.calls, userID+":"+string(part))
	return nil
}

func serve(t *testing.T, repo *fakeRepo, body string) *httptest.ResponseRecorder {
	t.Helper()
	mux := http.NewServeMux()
	auth := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(httpx.WithPrincipal(r.Context(), httpx.Principal{UserID: "u1", Role: "learner"})))
		})
	}
	NewHandler(NewService(repo, time.Now), slog.New(slog.NewTextHandler(io.Discard, nil))).Register(mux, auth)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/me/reset", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)
	return rec
}

func TestResetsTheTickedPartsOnceInOrder(t *testing.T) {
	t.Parallel()
	repo := &fakeRepo{}
	rec := serve(t, repo, `{"parts":["toeic","lessons","toeic"]}`)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d %s", rec.Code, rec.Body)
	}
	if want := []string{"u1:lessons", "u1:toeic"}; !slices.Equal(repo.calls, want) {
		t.Errorf("calls = %v, want %v", repo.calls, want)
	}

	all := &fakeRepo{}
	if rec := serve(t, all, `{"parts":["vocab","grammar","toeic","lessons"]}`); rec.Code != http.StatusNoContent {
		t.Fatalf("all: %d", rec.Code)
	}
	if want := []string{"u1:lessons", "u1:grammar", "u1:vocab", "u1:toeic"}; !slices.Equal(all.calls, want) {
		t.Errorf("all calls = %v", all.calls)
	}
}

func TestRefusesNoOrUnknownParts(t *testing.T) {
	t.Parallel()
	for _, body := range []string{`{"parts":[]}`, `{}`, `{"parts":["lessons","settings"]}`} {
		repo := &fakeRepo{}
		if rec := serve(t, repo, body); rec.Code != http.StatusBadRequest {
			t.Errorf("%s: status = %d", body, rec.Code)
		}
		if len(repo.calls) != 0 {
			t.Errorf("%s: reset %v", body, repo.calls)
		}
	}
}

func TestAFailedPartIsAServerError(t *testing.T) {
	t.Parallel()
	if rec := serve(t, &fakeRepo{fail: PartGrammar}, `{"parts":["grammar"]}`); rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d", rec.Code)
	}
}
