package mysql

import (
	"database/sql"
	"errors"
	"io"
	"log/slog"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/luongtran/luna/backend/internal/topic"
)

var topicsT0 = time.Date(2026, 9, 30, 8, 0, 0, 0, time.UTC)

func topicsMake(t *testing.T, r *Topics, name, level string, lessons ...string) topic.Topic {
	t.Helper()
	got, err := r.Create(t.Context(), topic.Topic{Name: name, Level: level, Description: "d", LessonIDs: lessons, CreatedAt: topicsT0, UpdatedAt: topicsT0})
	if err != nil {
		t.Fatalf("Create %q: %v", name, err)
	}
	return got
}

func TestTopicsCreateGet(t *testing.T) {
	t.Parallel()
	r := NewTopics(testDB(t))
	ctx := t.Context()
	l1, l2 := newID(), newID()
	made := topicsMake(t, r, "Gia đình", "A1", l1, l2)
	if len(made.ID) != 24 || made.Words != nil || made.WordsSeeded {
		t.Fatalf("created = %+v", made)
	}
	got, err := r.Get(ctx, made.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Gia đình" || got.Level != "A1" || got.Description != "d" || !slices.Equal(got.LessonIDs, []string{l1, l2}) ||
		!got.CreatedAt.Equal(topicsT0) || !got.UpdatedAt.Equal(topicsT0) || len(got.Words) != 0 || got.WordsSeeded {
		t.Fatalf("got = %+v", got)
	}
	empty := topicsMake(t, r, "Rỗng", "A1")
	if g, _ := r.Get(ctx, empty.ID); g.LessonIDs == nil || len(g.LessonIDs) != 0 {
		t.Fatalf("empty roadmap = %#v", g.LessonIDs)
	}
}

func TestTopicsGetNotFound(t *testing.T) {
	t.Parallel()
	r := NewTopics(testDB(t))
	for _, id := range []string{newID(), "not-an-id", ""} {
		if _, err := r.Get(t.Context(), id); !errors.Is(err, topic.ErrNotFound) {
			t.Fatalf("Get(%q) = %v", id, err)
		}
	}
}

func TestTopicsNameUniquePerLevel(t *testing.T) {
	t.Parallel()
	r := NewTopics(testDB(t))
	ctx := t.Context()
	a := topicsMake(t, r, "Gia đình", "A1")
	_, err := r.Create(ctx, topic.Topic{Name: "  GIA   đình ", Level: "A1", CreatedAt: topicsT0, UpdatedAt: topicsT0})
	if !errors.Is(err, topic.ErrNameTaken) {
		t.Fatalf("duplicate create = %v", err)
	}
	b := topicsMake(t, r, "Gia đình", "A2") // another level is fine
	// Update into a taken name.
	if _, err := r.Update(ctx, b.ID, topic.Input{Name: "gia đình", Level: "A1"}); !errors.Is(err, topic.ErrNameTaken) {
		t.Fatalf("update into taken = %v", err)
	}
	// Keeping its own name is fine.
	if _, err := r.Update(ctx, a.ID, topic.Input{Name: "Gia đình", Level: "A1", Description: "new"}); err != nil {
		t.Fatalf("update to itself: %v", err)
	}
}

func TestTopicsList(t *testing.T) {
	t.Parallel()
	r := NewTopics(testDB(t))
	ctx := t.Context()
	topicsMake(t, r, "A", "A1")
	topicsMake(t, r, "B", "A2")
	topicsMake(t, r, "C", "A1")
	all, err := r.List(ctx, "")
	if err != nil || len(all) != 3 {
		t.Fatalf("all = %d, %v", len(all), err)
	}
	a1, err := r.List(ctx, "A1")
	if err != nil || len(a1) != 2 {
		t.Fatalf("A1 = %d, %v", len(a1), err)
	}
	if none, err := r.List(ctx, "C2"); err != nil || len(none) != 0 {
		t.Fatalf("C2 = %d, %v", len(none), err)
	}
}

func TestTopicsUpdate(t *testing.T) {
	t.Parallel()
	r := NewTopics(testDB(t))
	ctx := t.Context()
	l := newID()
	made := topicsMake(t, r, "Cũ", "A1", l)
	got, err := r.Update(ctx, made.ID, topic.Input{Name: "Mới", Level: "B1", Description: "mô tả"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Mới" || got.Level != "B1" || got.Description != "mô tả" || !slices.Equal(got.LessonIDs, []string{l}) ||
		!got.CreatedAt.Equal(topicsT0) || !got.UpdatedAt.After(topicsT0) {
		t.Fatalf("updated = %+v", got)
	}
	// The name key follows the name: the old name is free again.
	topicsMake(t, r, "Cũ", "A1")
	if _, err := r.Update(ctx, newID(), topic.Input{Name: "X", Level: "A1"}); !errors.Is(err, topic.ErrNotFound) {
		t.Fatalf("update unknown = %v", err)
	}
	if _, err := r.Update(ctx, "bad", topic.Input{Name: "X", Level: "A1"}); !errors.Is(err, topic.ErrNotFound) {
		t.Fatalf("update malformed = %v", err)
	}
}

func TestTopicsDelete(t *testing.T) {
	t.Parallel()
	r := NewTopics(testDB(t))
	ctx := t.Context()
	made := topicsMake(t, r, "X", "A1")
	if err := r.Delete(ctx, made.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Get(ctx, made.ID); !errors.Is(err, topic.ErrNotFound) {
		t.Fatalf("after delete = %v", err)
	}
	if err := r.Delete(ctx, made.ID); err != nil {
		t.Fatalf("second delete = %v", err)
	}
	if err := r.Delete(ctx, "bad"); err != nil {
		t.Fatalf("malformed delete = %v", err)
	}
}

func TestTopicsRoadmap(t *testing.T) {
	t.Parallel()
	r := NewTopics(testDB(t))
	ctx := t.Context()
	made := topicsMake(t, r, "X", "A1")
	a, b, c := newID(), newID(), newID()

	if err := r.SetLessons(ctx, made.ID, []string{a, b}); err != nil {
		t.Fatal(err)
	}
	if err := r.AppendLesson(ctx, made.ID, c); err != nil {
		t.Fatal(err)
	}
	if err := r.AppendLesson(ctx, made.ID, a); err != nil { // already there: no change
		t.Fatal(err)
	}
	got, _ := r.Get(ctx, made.ID)
	if !slices.Equal(got.LessonIDs, []string{a, b, c}) {
		t.Fatalf("roadmap = %v", got.LessonIDs)
	}

	removed, err := r.RemoveLesson(ctx, made.ID, b)
	if err != nil || !removed {
		t.Fatalf("remove = %v, %v", removed, err)
	}
	removed, err = r.RemoveLesson(ctx, made.ID, b)
	if err != nil || removed {
		t.Fatalf("remove again = %v, %v", removed, err)
	}
	if removed, err = r.RemoveLesson(ctx, newID(), a); err != nil || removed {
		t.Fatalf("remove from unknown topic = %v, %v", removed, err)
	}
	if removed, err = r.RemoveLesson(ctx, "bad", "bad"); err != nil || removed {
		t.Fatalf("remove malformed = %v, %v", removed, err)
	}
	got, _ = r.Get(ctx, made.ID)
	if !slices.Equal(got.LessonIDs, []string{a, c}) {
		t.Fatalf("roadmap = %v", got.LessonIDs)
	}

	if err := r.SetLessons(ctx, made.ID, nil); err != nil {
		t.Fatal(err)
	}
	if got, _ = r.Get(ctx, made.ID); got.LessonIDs == nil || len(got.LessonIDs) != 0 {
		t.Fatalf("emptied roadmap = %#v", got.LessonIDs)
	}
	if err := r.SetLessons(ctx, newID(), []string{a}); !errors.Is(err, topic.ErrNotFound) {
		t.Fatalf("SetLessons unknown = %v", err)
	}
	if err := r.SetLessons(ctx, "bad", nil); !errors.Is(err, topic.ErrNotFound) {
		t.Fatalf("SetLessons malformed = %v", err)
	}
}

func TestTopicsAppendKeepsEveryConcurrentLesson(t *testing.T) {
	t.Parallel()
	r := NewTopics(testDB(t))
	ctx := t.Context()
	made := topicsMake(t, r, "X", "A1")
	const n = 12
	ids := make([]string, n)
	var wg sync.WaitGroup
	for i := range n {
		ids[i] = newID()
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := r.AppendLesson(ctx, made.ID, ids[i]); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	got, _ := r.Get(ctx, made.ID)
	if len(got.LessonIDs) != n {
		t.Fatalf("roadmap has %d lessons, want %d", len(got.LessonIDs), n)
	}
}

func TestTopicsSetWords(t *testing.T) {
	t.Parallel()
	r := NewTopics(testDB(t))
	ctx := t.Context()
	made := topicsMake(t, r, "X", "A1", newID())
	got, err := r.SetWords(ctx, made.ID, []string{"Family", "take a shower"})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got.Words, []string{"Family", "take a shower"}) || !got.WordsSeeded || len(got.LessonIDs) != 1 {
		t.Fatalf("set = %+v", got)
	}
	got, err = r.SetWords(ctx, made.ID, nil)
	if err != nil || got.Words == nil || len(got.Words) != 0 || !got.WordsSeeded {
		t.Fatalf("emptied = %#v, %v", got, err)
	}
	if _, err := r.SetWords(ctx, newID(), nil); !errors.Is(err, topic.ErrNotFound) {
		t.Fatalf("unknown = %v", err)
	}
	if _, err := r.SetWords(ctx, "bad", nil); !errors.Is(err, topic.ErrNotFound) {
		t.Fatalf("malformed = %v", err)
	}
}

func topicsLog() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func topicsWords(t *testing.T, db *sql.DB, id string) topic.Topic {
	t.Helper()
	got, err := NewTopics(db).Get(t.Context(), id)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func TestSeedTopicWords(t *testing.T) {
	t.Parallel()
	db := testDB(t)
	r := NewTopics(db)
	ctx := t.Context()
	seed := topic.Seed{Topics: []topic.SeedTopic{{Name: "Gia đình", Words: []string{"family", "mother"}}}}

	matched := topicsMake(t, r, "gia đình", "A1")
	other := topicsMake(t, r, "Lạ", "A1")
	edited, err := r.SetWords(ctx, topicsMake(t, r, "Gia đình", "A2").ID, []string{"mine"})
	if err != nil {
		t.Fatal(err)
	}
	emptied, err := r.SetWords(ctx, topicsMake(t, r, "Gia đình", "B1").ID, nil)
	if err != nil {
		t.Fatal(err)
	}

	if err := SeedTopicWords(ctx, db, seed, topicsLog()); err != nil {
		t.Fatal(err)
	}
	if got := topicsWords(t, db, matched.ID); !slices.Equal(got.Words, []string{"family", "mother"}) || !got.WordsSeeded {
		t.Fatalf("matched = %+v", got)
	}
	if got := topicsWords(t, db, other.ID); got.Words == nil || len(got.Words) != 0 || !got.WordsSeeded {
		t.Fatalf("unmatched = %#v", got)
	}
	if got := topicsWords(t, db, edited.ID); !slices.Equal(got.Words, []string{"mine"}) {
		t.Fatalf("edited was overwritten: %+v", got)
	}
	if got := topicsWords(t, db, emptied.ID); len(got.Words) != 0 {
		t.Fatalf("emptied was overwritten: %+v", got)
	}

	// A second run, even with another seed, changes nothing.
	again := topic.Seed{Topics: []topic.SeedTopic{{Name: "Lạ", Words: []string{"x"}}, {Name: "Gia đình", Words: []string{"zzz"}}}}
	if err := SeedTopicWords(ctx, db, again, topicsLog()); err != nil {
		t.Fatal(err)
	}
	if got := topicsWords(t, db, other.ID); len(got.Words) != 0 {
		t.Fatalf("seeded twice: %+v", got)
	}
	if got := topicsWords(t, db, matched.ID); !slices.Equal(got.Words, []string{"family", "mother"}) {
		t.Fatalf("seeded twice: %+v", got)
	}
}

func TestSeedTopicWordsWithEmbeddedSeed(t *testing.T) {
	t.Parallel()
	db := testDB(t)
	topicsMake(t, NewTopics(db), "Chủ đề lạ", "A1")
	if err := seedTopicWords(t.Context(), db, topicsLog()); err != nil {
		t.Fatal(err)
	}
	if err := seedTopicWords(t.Context(), db, topicsLog()); err != nil {
		t.Fatal(err)
	}
}
