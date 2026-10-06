package mysql

import (
	"errors"
	"testing"
	"time"

	"github.com/luongtran/luna/backend/internal/lesson"
)

func TestWordImages(t *testing.T) {
	t.Parallel()
	r := NewWordImages(testDB(t))
	ctx := t.Context()
	id := newID()
	at := time.Date(2026, 10, 6, 8, 0, 0, 0, time.UTC)

	if s, err := r.Settings(ctx, id); err != nil || s != (lesson.ImageSettings{}) {
		t.Fatalf("no settings = %+v, %v", s, err)
	}
	want := lesson.ImageSettings{Enabled: true, Style: "màu nước", Status: lesson.StatusRunning, UpdatedAt: at}
	if err := r.SaveSettings(ctx, id, want); err != nil {
		t.Fatal(err)
	}
	want.Status, want.Error = lesson.StatusFailed, "lỗi"
	if err := r.SaveSettings(ctx, id, want); err != nil {
		t.Fatal(err)
	}
	if got, err := r.Settings(ctx, id); err != nil || got != want {
		t.Fatalf("settings = %+v, %v", got, err)
	}

	img := lesson.WordImage{LessonID: id, Lemma: "give up", MIME: "image/jpeg", Data: []byte{0xff, 0xd8, 0x00, 0x01}, CreatedAt: at}
	if err := r.Save(ctx, img); err != nil {
		t.Fatal(err)
	}
	img.Data = []byte{0xff, 0xd8, 0x02}
	if err := r.Save(ctx, img); err != nil {
		t.Fatal(err)
	}
	got, err := r.Get(ctx, id, "give up")
	if err != nil || string(got.Data) != string(img.Data) || got.MIME != "image/jpeg" || !got.CreatedAt.Equal(at) {
		t.Fatalf("image = %+v, %v", got, err)
	}
	if lemmas, err := r.Lemmas(ctx, id); err != nil || len(lemmas) != 1 || lemmas[0] != "give up" {
		t.Fatalf("lemmas = %v, %v", lemmas, err)
	}
	if _, err := r.Get(ctx, id, "go"); !errors.Is(err, lesson.ErrImageNotFound) {
		t.Fatalf("missing: %v", err)
	}

	if err := r.Save(ctx, lesson.WordImage{LessonID: id, Lemma: "go", MIME: "image/jpeg", Data: []byte{1}, CreatedAt: at}); err != nil {
		t.Fatal(err)
	}
	if err := r.Delete(ctx, id, "go"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Get(ctx, id, "go"); !errors.Is(err, lesson.ErrImageNotFound) {
		t.Fatalf("after Delete: %v", err)
	}

	if err := r.DeleteForLesson(ctx, id); err != nil {
		t.Fatal(err)
	}
	if lemmas, _ := r.Lemmas(ctx, id); len(lemmas) != 0 {
		t.Fatalf("after delete: %v", lemmas)
	}
	if s, _ := r.Settings(ctx, id); s.Enabled {
		t.Fatalf("settings after delete: %+v", s)
	}
}
