package mysql

import (
	"reflect"
	"testing"
	"time"

	"github.com/luongtran/luna/backend/internal/job"
	"github.com/luongtran/luna/backend/internal/lesson"
)

func lessonTestReview() *lesson.Review {
	t := time.Date(2026, 10, 5, 8, 0, 0, 0, time.UTC)
	return &lesson.Review{
		CheckedAt: t,
		Flags: []lesson.Flag{
			{Area: lesson.AreaSentence, Index: 2, Kind: lesson.FlagWrong, NoteVi: "Câu \"sai\"."},
			{Area: lesson.AreaQuestion, Index: 0, Kind: lesson.FlagMismatch, NoteVi: "AI chọn khác.", Confirmed: true},
		},
	}
}

func TestLessonReviewJSON(t *testing.T) {
	t.Parallel()
	for _, in := range []*lesson.Review{lessonTestReview(), {CheckedAt: time.Now().UTC().Truncate(time.Second), VerifiedAt: time.Now().UTC().Truncate(time.Second), Flags: []lesson.Flag{}}} {
		v, err := lessonReviewValue(in)
		if err != nil {
			t.Fatal(err)
		}
		got, err := lessonReviewFrom([]byte(v.(string)))
		if err != nil || !reflect.DeepEqual(got, in) {
			t.Fatalf("round trip = %+v, %v; want %+v", got, err, in)
		}
	}
	if v, err := lessonReviewValue(nil); v != nil || err != nil {
		t.Errorf("nil review = %v, %v", v, err)
	}
	for _, raw := range [][]byte{nil, []byte("null")} {
		if got, err := lessonReviewFrom(raw); got != nil || err != nil {
			t.Errorf("%q = %+v, %v", raw, got, err)
		}
	}
}

func TestLessonsSaveReview(t *testing.T) {
	t.Parallel()
	r := NewLessons(testDB(t))
	l := lessonTestBase("t", "A1", lessonTestTopicA, time.Now().UTC())
	c := lessonTestCreate(t, r, l)
	if c.Review != nil {
		t.Fatalf("new lesson review = %+v", c.Review)
	}
	if got, _ := r.Get(t.Context(), c.ID); got.Review != nil {
		t.Fatalf("stored review = %+v", got.Review)
	}

	before, _ := r.Get(t.Context(), c.ID)
	in := lessonTestReview()
	ok, err := r.SaveReview(t.Context(), c.ID, c.Revision, in)
	if err != nil || !ok {
		t.Fatalf("SaveReview = %v, %v", ok, err)
	}
	got, _ := r.Get(t.Context(), c.ID)
	if !reflect.DeepEqual(got.Review, in) || !got.UpdatedAt.Equal(before.UpdatedAt) {
		t.Fatalf("review = %+v, updatedAt %v (was %v)", got.Review, got.UpdatedAt, before.UpdatedAt)
	}

	// Saving the same review again still counts as saved.
	if ok, err := r.SaveReview(t.Context(), c.ID, c.Revision, in); err != nil || !ok {
		t.Fatalf("same review = %v, %v", ok, err)
	}
	verified := lessonTestReview()
	verified.VerifiedAt = in.CheckedAt.Add(time.Hour)
	verified.Flags = []lesson.Flag{}
	if ok, err := r.SaveReview(t.Context(), c.ID, c.Revision, verified); err != nil || !ok {
		t.Fatalf("verified = %v, %v", ok, err)
	}
	got, _ = r.Get(t.Context(), c.ID)
	if !reflect.DeepEqual(got.Review, verified) {
		t.Fatalf("verified review = %+v", got.Review)
	}

	// A lesson at another revision is left alone.
	if ok, err := r.SaveReview(t.Context(), c.ID, c.Revision+1, in); err != nil || ok {
		t.Fatalf("stale revision = %v, %v", ok, err)
	}
	if got, _ := r.Get(t.Context(), c.ID); !reflect.DeepEqual(got.Review, verified) {
		t.Fatalf("stale write changed the review: %+v", got.Review)
	}

	// nil clears it.
	if ok, err := r.SaveReview(t.Context(), c.ID, c.Revision, nil); err != nil || !ok {
		t.Fatalf("clear = %v, %v", ok, err)
	}
	if got, _ := r.Get(t.Context(), c.ID); got.Review != nil {
		t.Fatalf("cleared review = %+v", got.Review)
	}

	if _, err := r.SaveReview(t.Context(), "bad", 1, in); err == nil {
		t.Error("malformed id: want an error")
	}
	if ok, err := r.SaveReview(t.Context(), lessonTestTopicA, 1, in); err != nil || ok {
		t.Errorf("missing lesson = %v, %v", ok, err)
	}
}

func TestLessonsCreateKeepsReview(t *testing.T) {
	t.Parallel()
	r := NewLessons(testDB(t))
	l := lessonTestBase("t", "A1", lessonTestTopicA, time.Now().UTC())
	l.Review = lessonTestReview()
	c := lessonTestCreate(t, r, l)
	if got, _ := r.Get(t.Context(), c.ID); !reflect.DeepEqual(got.Review, l.Review) {
		t.Fatalf("review = %+v", got.Review)
	}
}

func TestLessonsContentWritesDropTheReview(t *testing.T) {
	t.Parallel()
	writes := map[string]func(r *Lessons, c lesson.Lesson) error{
		"ReplaceContent": func(r *Lessons, c lesson.Lesson) error {
			next := c
			next.Content, next.Revision = "New text.", c.Revision+1
			next.Review = lessonTestReview() // ReplaceContent never keeps one
			return r.ReplaceContent(t.Context(), next)
		},
		"SaveAnnotations": func(r *Lessons, c lesson.Lesson) error {
			_, err := r.SaveAnnotations(t.Context(), c.ID, c.Revision, lessonTestAnns(), lessonTestExtras())
			return err
		},
		"SavePractice": func(r *Lessons, c lesson.Lesson) error {
			_, err := r.SavePractice(t.Context(), c.ID, c.Revision, c.PracticeVersion, lessonTestPractice())
			return err
		},
		"ReplaceExtras": func(r *Lessons, c lesson.Lesson) error {
			return r.ReplaceExtras(t.Context(), c.ID, lessonTestExtras(), false)
		},
		"ReplaceAnnotations": func(r *Lessons, c lesson.Lesson) error {
			return r.ReplaceAnnotations(t.Context(), c.ID, lessonTestAnns())
		},
	}
	for name, write := range writes {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			r := NewLessons(testDB(t))
			c := lessonTestCreate(t, r, lessonTestBase("t", "A1", lessonTestTopicA, time.Now().UTC()))
			if ok, err := r.SaveReview(t.Context(), c.ID, c.Revision, lessonTestReview()); err != nil || !ok {
				t.Fatalf("SaveReview = %v, %v", ok, err)
			}
			if err := write(r, c); err != nil {
				t.Fatal(err)
			}
			if got, _ := r.Get(t.Context(), c.ID); got.Review != nil {
				t.Fatalf("review kept: %+v", got.Review)
			}
		})
	}
}

func TestLessonsInfoAndStatusKeepTheReview(t *testing.T) {
	t.Parallel()
	r := NewLessons(testDB(t))
	c := lessonTestCreate(t, r, lessonTestBase("t", "A1", lessonTestTopicA, time.Now().UTC()))
	in := lessonTestReview()
	if ok, err := r.SaveReview(t.Context(), c.ID, c.Revision, in); err != nil || !ok {
		t.Fatalf("SaveReview = %v, %v", ok, err)
	}
	if err := r.UpdateInfo(t.Context(), c.ID, lesson.Info{Title: "t2", Level: "A1", TopicID: lessonTestTopicA, Source: "s", License: "l"}); err != nil {
		t.Fatal(err)
	}
	if ok, err := r.SetStatus(t.Context(), c.ID, c.Revision, job.TypePractice, lesson.StatusFailed, "x"); err != nil || !ok {
		t.Fatalf("SetStatus = %v, %v", ok, err)
	}
	if got, _ := r.Get(t.Context(), c.ID); got.Title != "t2" || !reflect.DeepEqual(got.Review, in) {
		t.Fatalf("lesson = %+v", got)
	}
}

func TestLessonsSummariesCountOpenFlags(t *testing.T) {
	t.Parallel()
	r := NewLessons(testDB(t))
	now := time.Now().UTC()
	plain := lessonTestCreate(t, r, lessonTestBase("plain", "A1", lessonTestTopicA, now))
	open := lessonTestCreate(t, r, lessonTestBase("open", "A1", lessonTestTopicA, now.Add(time.Second)))
	done := lessonTestCreate(t, r, lessonTestBase("done", "A1", lessonTestTopicA, now.Add(2*time.Second)))
	if ok, err := r.SaveReview(t.Context(), open.ID, open.Revision, lessonTestReview()); err != nil || !ok {
		t.Fatalf("SaveReview open = %v, %v", ok, err)
	}
	verified := lessonTestReview()
	verified.VerifiedAt = verified.CheckedAt.Add(time.Hour)
	verified.Flags[0].Confirmed = true
	if ok, err := r.SaveReview(t.Context(), done.ID, done.Revision, verified); err != nil || !ok {
		t.Fatalf("SaveReview done = %v, %v", ok, err)
	}

	list, err := r.List(t.Context(), lesson.Filter{})
	if err != nil || len(list) != 3 {
		t.Fatalf("List = %v, %v", list, err)
	}
	type row struct {
		flags             int
		checked, verified bool
	}
	got := map[string]row{}
	for _, s := range list {
		got[s.ID] = row{s.Flags, s.Checked, s.Verified}
	}
	want := map[string]row{plain.ID: {0, false, false}, open.ID: {1, true, false}, done.ID: {0, true, true}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("summaries = %v, want %v", got, want)
	}
	sums, err := r.Summaries(t.Context(), []string{open.ID})
	if err != nil || len(sums) != 1 || sums[0].Flags != 1 || !sums[0].Checked {
		t.Fatalf("Summaries = %+v, %v", sums, err)
	}
}

// The migration can run again after it stopped before it was recorded.
func TestLessonReviewMigrationIsRepeatable(t *testing.T) {
	t.Parallel()
	db := testDB(t)
	if _, err := db.ExecContext(t.Context(), "DELETE FROM schema_migrations WHERE id = '016_lesson_review'"); err != nil {
		t.Fatal(err)
	}
	if err := migrate(t.Context(), db, migrations()); err != nil {
		t.Fatalf("migrate again: %v", err)
	}
	var n int
	if err := db.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM schema_migrations WHERE id = '016_lesson_review'").Scan(&n); err != nil || n != 1 {
		t.Fatalf("recorded = %d, %v", n, err)
	}
}
