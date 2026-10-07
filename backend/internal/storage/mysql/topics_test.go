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

// topicsMake creates a topic whose A1 roadmap holds lessons.
func topicsMake(t *testing.T, r *Topics, name string, lessons ...string) topic.Topic {
	t.Helper()
	roadmaps := map[string][]string{}
	if len(lessons) > 0 {
		roadmaps["A1"] = lessons
	}
	got, err := r.Create(t.Context(), topic.Topic{Name: name, Description: "d", Roadmaps: roadmaps, CreatedAt: topicsT0, UpdatedAt: topicsT0})
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
	made := topicsMake(t, r, "Gia đình", l1, l2)
	if len(made.ID) != 24 || len(made.Words) != 0 || made.WordsSeeded {
		t.Fatalf("created = %+v", made)
	}
	got, err := r.Get(ctx, made.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Gia đình" || got.Description != "d" || !slices.Equal(got.Roadmap("A1"), []string{l1, l2}) ||
		!got.CreatedAt.Equal(topicsT0) || !got.UpdatedAt.Equal(topicsT0) || len(got.Words) != 0 || got.WordsSeeded {
		t.Fatalf("got = %+v", got)
	}
	empty := topicsMake(t, r, "Rỗng")
	if g, _ := r.Get(ctx, empty.ID); g.Roadmaps == nil || len(g.Roadmap("A1")) != 0 {
		t.Fatalf("empty roadmaps = %#v", g.Roadmaps)
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

func TestTopicsNameUnique(t *testing.T) {
	t.Parallel()
	r := NewTopics(testDB(t))
	ctx := t.Context()
	a := topicsMake(t, r, "Gia đình")
	_, err := r.Create(ctx, topic.Topic{Name: "  GIA   đình ", CreatedAt: topicsT0, UpdatedAt: topicsT0})
	if !errors.Is(err, topic.ErrNameTaken) {
		t.Fatalf("duplicate create = %v", err)
	}
	b := topicsMake(t, r, "Du lịch")
	if _, err := r.Update(ctx, b.ID, topic.Input{Name: "gia đình"}); !errors.Is(err, topic.ErrNameTaken) {
		t.Fatalf("update into taken = %v", err)
	}
	// Keeping its own name is fine.
	if _, err := r.Update(ctx, a.ID, topic.Input{Name: "Gia đình", Description: "new"}); err != nil {
		t.Fatalf("update to itself: %v", err)
	}
}

func TestTopicsList(t *testing.T) {
	t.Parallel()
	r := NewTopics(testDB(t))
	topicsMake(t, r, "A")
	topicsMake(t, r, "B")
	if all, err := r.List(t.Context()); err != nil || len(all) != 2 {
		t.Fatalf("all = %d, %v", len(all), err)
	}
}

func TestTopicsUpdate(t *testing.T) {
	t.Parallel()
	r := NewTopics(testDB(t))
	ctx := t.Context()
	l := newID()
	made := topicsMake(t, r, "Cũ", l)
	got, err := r.Update(ctx, made.ID, topic.Input{Name: "Mới", Description: "mô tả"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Mới" || got.Description != "mô tả" || !slices.Equal(got.Roadmap("A1"), []string{l}) ||
		!got.CreatedAt.Equal(topicsT0) || !got.UpdatedAt.After(topicsT0) {
		t.Fatalf("updated = %+v", got)
	}
	// The name key follows the name: the old name is free again.
	topicsMake(t, r, "Cũ")
	if _, err := r.Update(ctx, newID(), topic.Input{Name: "X"}); !errors.Is(err, topic.ErrNotFound) {
		t.Fatalf("update unknown = %v", err)
	}
	if _, err := r.Update(ctx, "bad", topic.Input{Name: "X"}); !errors.Is(err, topic.ErrNotFound) {
		t.Fatalf("update malformed = %v", err)
	}
}

func TestTopicsDelete(t *testing.T) {
	t.Parallel()
	r := NewTopics(testDB(t))
	ctx := t.Context()
	made := topicsMake(t, r, "X")
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

func TestTopicsRoadmapPerLevel(t *testing.T) {
	t.Parallel()
	r := NewTopics(testDB(t))
	ctx := t.Context()
	made := topicsMake(t, r, "X")
	a, b, c := newID(), newID(), newID()

	if err := r.SetLessons(ctx, made.ID, "A1", []string{a, b}); err != nil {
		t.Fatal(err)
	}
	if err := r.AppendLesson(ctx, made.ID, "A1", c); err != nil {
		t.Fatal(err)
	}
	if err := r.AppendLesson(ctx, made.ID, "A1", a); err != nil { // already there: no change
		t.Fatal(err)
	}
	if err := r.AppendLesson(ctx, made.ID, "B2", a); err != nil { // another level is another roadmap
		t.Fatal(err)
	}
	got, _ := r.Get(ctx, made.ID)
	if !slices.Equal(got.Roadmap("A1"), []string{a, b, c}) || !slices.Equal(got.Roadmap("B2"), []string{a}) {
		t.Fatalf("roadmaps = %v", got.Roadmaps)
	}

	removed, err := r.RemoveLesson(ctx, made.ID, "A1", b)
	if err != nil || !removed {
		t.Fatalf("remove = %v, %v", removed, err)
	}
	removed, err = r.RemoveLesson(ctx, made.ID, "A1", b)
	if err != nil || removed {
		t.Fatalf("remove again = %v, %v", removed, err)
	}
	if removed, err = r.RemoveLesson(ctx, newID(), "A1", a); err != nil || removed {
		t.Fatalf("remove from unknown topic = %v, %v", removed, err)
	}
	if removed, err = r.RemoveLesson(ctx, "bad", "A1", "bad"); err != nil || removed {
		t.Fatalf("remove malformed = %v, %v", removed, err)
	}
	if removed, err = r.RemoveLesson(ctx, made.ID, "Z9", a); err != nil || removed {
		t.Fatalf("remove at unknown level = %v, %v", removed, err)
	}
	got, _ = r.Get(ctx, made.ID)
	if !slices.Equal(got.Roadmap("A1"), []string{a, c}) {
		t.Fatalf("roadmap = %v", got.Roadmaps)
	}

	if err := r.SetLessons(ctx, made.ID, "A1", nil); err != nil {
		t.Fatal(err)
	}
	if got, _ = r.Get(ctx, made.ID); len(got.Roadmap("A1")) != 0 || len(got.Roadmap("B2")) != 1 {
		t.Fatalf("emptied roadmap = %#v", got.Roadmaps)
	}
	if err := r.SetLessons(ctx, newID(), "A1", []string{a}); !errors.Is(err, topic.ErrNotFound) {
		t.Fatalf("SetLessons unknown = %v", err)
	}
	if err := r.SetLessons(ctx, "bad", "A1", nil); !errors.Is(err, topic.ErrNotFound) {
		t.Fatalf("SetLessons malformed = %v", err)
	}
}

func TestTopicsAppendKeepsEveryConcurrentLesson(t *testing.T) {
	t.Parallel()
	r := NewTopics(testDB(t))
	ctx := t.Context()
	made := topicsMake(t, r, "X")
	const n = 12
	ids := make([]string, n)
	var wg sync.WaitGroup
	for i := range n {
		ids[i] = newID()
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := r.AppendLesson(ctx, made.ID, "A1", ids[i]); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	got, _ := r.Get(ctx, made.ID)
	if len(got.Roadmap("A1")) != n {
		t.Fatalf("roadmap has %d lessons, want %d", len(got.Roadmap("A1")), n)
	}
}

func TestTopicsSetWords(t *testing.T) {
	t.Parallel()
	r := NewTopics(testDB(t))
	ctx := t.Context()
	made := topicsMake(t, r, "X", newID())
	want := []topic.Word{{Text: "Family"}, {Text: "take a shower", Level: "B1"}}
	got, err := r.SetWords(ctx, made.ID, want)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got.Words, want) || !got.WordsSeeded || len(got.Roadmap("A1")) != 1 {
		t.Fatalf("set = %+v", got)
	}
	got, err = r.SetWords(ctx, made.ID, nil)
	if err != nil || len(got.Words) != 0 || !got.WordsSeeded {
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
	seed := topic.Seed{Topics: []topic.SeedTopic{
		{Name: "Gia đình", Words: []string{"family", "mother"}},
		{Name: "Du lịch", Words: []string{"trip"}},
		{Name: "Mua sắm", Words: []string{"shop"}},
	}}

	matched := topicsMake(t, r, "gia đình")
	other := topicsMake(t, r, "Lạ")
	edited, err := r.SetWords(ctx, topicsMake(t, r, "Du lịch").ID, []topic.Word{{Text: "mine"}})
	if err != nil {
		t.Fatal(err)
	}
	emptied, err := r.SetWords(ctx, topicsMake(t, r, "Mua sắm").ID, nil)
	if err != nil {
		t.Fatal(err)
	}

	if err := SeedTopicWords(ctx, db, seed, topicsLog()); err != nil {
		t.Fatal(err)
	}
	if got := topicsWords(t, db, matched.ID); !slices.Equal(got.Words, []topic.Word{{Text: "family"}, {Text: "mother"}}) || !got.WordsSeeded {
		t.Fatalf("matched = %+v", got)
	}
	if got := topicsWords(t, db, other.ID); len(got.Words) != 0 || !got.WordsSeeded {
		t.Fatalf("unmatched = %#v", got)
	}
	if got := topicsWords(t, db, edited.ID); !slices.Equal(got.Words, []topic.Word{{Text: "mine"}}) {
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
	if got := topicsWords(t, db, matched.ID); !slices.Equal(got.Words, []topic.Word{{Text: "family"}, {Text: "mother"}}) {
		t.Fatalf("seeded twice: %+v", got)
	}
}

func TestSeedTopicWordsWithEmbeddedSeed(t *testing.T) {
	t.Parallel()
	db := testDB(t)
	topicsMake(t, NewTopics(db), "Chủ đề lạ")
	if err := seedTopicWords(t.Context(), db, topicsLog()); err != nil {
		t.Fatal(err)
	}
	if err := seedTopicWords(t.Context(), db, topicsLog()); err != nil {
		t.Fatal(err)
	}
}

// legacyTopic inserts a topic as it was stored before topics were shared by every level.
func legacyTopic(t *testing.T, db *sql.DB, name, level string, lessons ...string) string {
	t.Helper()
	id := newID()
	raw, _ := toJSON(append([]string{}, lessons...))
	_, err := db.ExecContext(t.Context(),
		"INSERT INTO topics (id, name, name_key, level, description, lesson_ids, words, words_seeded, created_at, updated_at) VALUES (?,?,?,?,'',?,'[\"Family\"]',1,?,?)",
		id, name, topic.NameKey(name)+"#"+level, level, raw, topicsT0, topicsT0)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func TestMergeSharedTopics(t *testing.T) {
	t.Parallel()
	db := testDB(t)
	ctx := t.Context()
	l1, l2, l3 := newID(), newID(), newID()
	// Legacy rows of one name get distinct keys here, since the shared unique key already exists
	// in a migrated test database.
	a1 := legacyTopic(t, db, "Gia đình", "A1", l1)
	a2 := legacyTopic(t, db, "gia đình", "A2", l2, l3)
	work := legacyTopic(t, db, "Công việc", "B1")
	if _, err := db.ExecContext(ctx, "INSERT INTO goals (id, user_id, topic_id, level, status, effective_from, started_at) VALUES (?,?,?,?,?,?,?)",
		newID(), newID(), a2, "A2", "active", "2026-09-30", topicsT0); err != nil {
		t.Fatal(err)
	}

	for range 2 { // a second run changes nothing
		if err := mergeSharedTopics(ctx, db, topicsLog()); err != nil {
			t.Fatal(err)
		}
	}
	all, err := NewTopics(db).List(ctx)
	if err != nil || len(all) != 2 {
		t.Fatalf("topics = %+v, %v", all, err)
	}
	family := topicsWords(t, db, a1)
	if family.Name != "Gia đình" || !slices.Equal(family.Roadmap("A1"), []string{l1}) || !slices.Equal(family.Roadmap("A2"), []string{l2, l3}) {
		t.Fatalf("family = %+v", family)
	}
	if !slices.Equal(family.Words, []topic.Word{{Text: "Family", Level: "A1"}}) {
		t.Fatalf("words = %+v", family.Words)
	}
	if _, err := NewTopics(db).Get(ctx, a2); !errors.Is(err, topic.ErrNotFound) {
		t.Fatalf("merged topic still there: %v", err)
	}
	var goalTopic string
	if err := db.QueryRowContext(ctx, "SELECT topic_id FROM goals").Scan(&goalTopic); err != nil || goalTopic != a1 {
		t.Fatalf("goal topic = %s, %v", goalTopic, err)
	}
	if w := topicsWords(t, db, work); len(w.Roadmap("B1")) != 0 {
		t.Fatalf("work = %+v", w)
	}
}

func TestAddExtraTopicsOnce(t *testing.T) {
	t.Parallel()
	db := testDB(t)
	ctx := t.Context()
	r := NewTopics(db)
	music := topicsMake(t, r, "âm nhạc") // already there: kept as it is
	if err := addExtraTopics(ctx, db, topicsLog()); err != nil {
		t.Fatal(err)
	}
	all, _ := r.List(ctx)
	if len(all) != 24 {
		t.Fatalf("topics = %d, want 24 (23 added + the existing one)", len(all))
	}
	if got := topicsWords(t, db, music.ID); got.Name != "âm nhạc" || len(got.Words) != 0 {
		t.Fatalf("existing topic changed: %+v", got)
	}
	// Deleted afterwards, a topic is not created again.
	_ = r.Delete(ctx, all[0].ID)
	if err := addExtraTopics(ctx, db, topicsLog()); err != nil {
		t.Fatal(err)
	}
	if again, _ := r.List(ctx); len(again) != 23 {
		t.Fatalf("topics after a second run = %d", len(again))
	}
}
