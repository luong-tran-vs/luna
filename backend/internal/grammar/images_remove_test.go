package grammar

import (
	"errors"
	"testing"
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
	if err != nil || got != (ImagesRemoved{Pictures: 3, Points: 1}) {
		t.Fatalf("dry run = %+v, %v", got, err)
	}
	if keys, _ := store.Keys(t.Context(), "a1-to-be"); len(keys) != 3 {
		t.Fatalf("dry run removed: %v", keys)
	}

	got, err = e.svc.RemoveImages(t.Context(), []string{"a1-to-be"}, false)
	if err != nil || got != (ImagesRemoved{Pictures: 3, Points: 1}) {
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
