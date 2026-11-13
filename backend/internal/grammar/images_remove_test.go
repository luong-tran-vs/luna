package grammar

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/luongtran/luna/backend/internal/platform/httpx"
)

func TestRemoveImagesTakesEveryPictureOfTheLessonsAway(t *testing.T) {
	t.Parallel()
	e, store, _ := newImageEnv(t)
	for _, k := range []string{"k1", "k2", "picture"} {
		_ = store.Save(t.Context(), ExampleImage{PointID: "a1-to-be", Key: k, MIME: "image/jpeg", Data: []byte("x")})
	}
	_ = store.Save(t.Context(), ExampleImage{PointID: "other", Key: "k1", MIME: "image/jpeg", Data: []byte("x")})

	// A dry run only counts.
	got, err := e.svc.RemoveImages(t.Context(), []string{"a1-to-be", "nope", "a1-to-be"}, true)
	if err != nil || got.Pictures != 3 || got.Points != 1 || len(got.ByPoint) != 1 || got.ByPoint[0] != (PointPictures{PointID: "a1-to-be", Pictures: 3}) {
		t.Fatalf("dry run = %+v, %v", got, err)
	}
	if keys, _ := store.Keys(t.Context(), "a1-to-be"); len(keys) != 3 {
		t.Fatalf("dry run removed: %v", keys)
	}

	got, err = e.svc.RemoveImages(t.Context(), []string{"a1-to-be"}, false)
	if err != nil || got.Pictures != 3 || got.Points != 1 || len(got.ByPoint) != 1 || got.ByPoint[0] != (PointPictures{PointID: "a1-to-be", Pictures: 3}) {
		t.Fatalf("removed = %+v, %v", got, err)
	}
	if keys, _ := store.Keys(t.Context(), "a1-to-be"); len(keys) != 0 {
		t.Fatalf("left = %v", keys)
	}
	if keys, _ := store.Keys(t.Context(), "other"); len(keys) != 1 {
		t.Fatalf("another lesson lost its pictures: %v", keys)
	}
	var verr *ValidationError
	if _, err := e.svc.RemoveImages(t.Context(), []string{" "}, false); !errors.As(err, &verr) {
		t.Fatalf("no lessons: %v", err)
	}
}

func TestRemoveImagesAPI(t *testing.T) {
	t.Parallel()
	e, store, _ := newImageEnv(t)
	mux := http.NewServeMux()
	NewHandler(e.svc, slog.New(slog.DiscardHandler)).Register(mux, httpx.RequireAuth(resolver))
	a := &api{env: e, mux: mux}
	_ = store.Save(t.Context(), ExampleImage{PointID: "a1-to-be", Key: "k1", MIME: "image/jpeg", Data: []byte("x")})

	rec := a.do(t, http.MethodPost, "/api/admin/grammar-images/remove", "admin", `{"pointIds":["a1-to-be","nope"],"dryRun":true}`)
	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != `{"pictures":1,"points":1,"byPoint":[{"pointId":"a1-to-be","pictures":1}]}` {
		t.Fatalf("dry run = %d %s", rec.Code, rec.Body)
	}
	if rec := a.do(t, http.MethodPost, "/api/admin/grammar-images/remove", "learner", `{"pointIds":["a1-to-be"]}`); rec.Code != http.StatusForbidden {
		t.Fatalf("learner = %d", rec.Code)
	}
}
