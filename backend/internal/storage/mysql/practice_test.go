package mysql

import (
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/luongtran/luna/backend/internal/job"
	"github.com/luongtran/luna/backend/internal/lesson"
)

func lessonTestPractice() lesson.Practice {
	return lesson.Practice{
		ObjectiveVi: "Học từ về táo",
		Examples:    []lesson.Example{{Lemma: "apple", Sentence: "I eat an apple."}},
		Dialogue: &lesson.Dialogue{
			Speakers: []string{"An", "Binh"},
			Turns:    []lesson.Turn{{Speaker: 0, Text: "Hi", MeaningVi: "Chào"}, {Speaker: 1, Text: "Hello", MeaningVi: "Xin chào"}},
		},
		GrammarTipVi: "Dùng a/an",
		Translations: []lesson.Translation{{Vi: "Tôi ăn táo", En: "I eat an apple", Distractors: []string{"I eats", "Me eat"}}},
	}
}

func TestLessonsSavePractice(t *testing.T) {
	t.Parallel()
	r := NewLessons(testDB(t))
	l := lessonTestBase("t", "A1", lessonTestTopicA, time.Now().UTC())
	l.PracticeStatus = lesson.StatusRunning
	c := lessonTestCreate(t, r, l)

	// Wrong revision and wrong previous version change nothing.
	if ok, err := r.SavePractice(t.Context(), c.ID, 2, 0, lessonTestPractice()); ok || err != nil {
		t.Fatalf("stale revision = %v %v", ok, err)
	}
	if ok, err := r.SavePractice(t.Context(), c.ID, 1, 5, lessonTestPractice()); ok || err != nil {
		t.Fatalf("stale version = %v %v", ok, err)
	}
	if got, _ := r.Get(t.Context(), c.ID); got.Practice != nil || got.PracticeVersion != 0 || got.PracticeStatus != lesson.StatusRunning {
		t.Fatalf("stale write changed the lesson: %+v", got)
	}

	ok, err := r.SavePractice(t.Context(), c.ID, 1, 0, lessonTestPractice())
	if err != nil || !ok {
		t.Fatalf("SavePractice = %v %v", ok, err)
	}
	got, _ := r.Get(t.Context(), c.ID)
	if !reflect.DeepEqual(got.Practice, lessonTestPtr(lessonTestPractice())) || got.PracticeStatus != lesson.StatusDone ||
		got.PracticeError != "" || got.PracticeVersion != 1 {
		t.Fatalf("after save = %+v v%d %q", got.Practice, got.PracticeVersion, got.PracticeStatus)
	}

	// The old version can no longer overwrite; the new one can.
	if ok, _ := r.SavePractice(t.Context(), c.ID, 1, 0, lesson.Practice{ObjectiveVi: "late"}); ok {
		t.Fatal("old practice job overwrote a newer practice")
	}
	if ok, err := r.SavePractice(t.Context(), c.ID, 1, 1, lesson.Practice{ObjectiveVi: "v2"}); !ok || err != nil {
		t.Fatalf("second save = %v %v", ok, err)
	}
	got, _ = r.Get(t.Context(), c.ID)
	if got.Practice.ObjectiveVi != "v2" || got.PracticeVersion != 2 || got.Practice.Dialogue != nil ||
		got.Practice.Examples == nil || got.Practice.Translations == nil {
		t.Fatalf("second = %+v v%d", got.Practice, got.PracticeVersion)
	}

	if ok, err := r.SavePractice(t.Context(), lessonTestTopicA, 1, 0, lesson.Practice{}); ok || err != nil {
		t.Fatalf("missing = %v %v", ok, err)
	}
	if _, err := r.SavePractice(t.Context(), "bad", 1, 0, lesson.Practice{}); err != lesson.ErrNotFound {
		t.Fatalf("malformed = %v", err)
	}
}

func TestLessonsPracticeResetBySaveAnnotations(t *testing.T) {
	t.Parallel()
	r := NewLessons(testDB(t))
	c := lessonTestCreate(t, r, lessonTestBase("t", "A1", lessonTestTopicA, time.Now().UTC()))
	if ok, _ := r.SavePractice(t.Context(), c.ID, 1, 0, lessonTestPractice()); !ok {
		t.Fatal("SavePractice failed")
	}
	if ok, err := r.SaveAnnotations(t.Context(), c.ID, 1, lessonTestAnns(), lesson.Extras{}); !ok || err != nil {
		t.Fatalf("SaveAnnotations = %v %v", ok, err)
	}
	got, _ := r.Get(t.Context(), c.ID)
	// The practice is dropped and queued again, but its version keeps counting up.
	if got.Practice != nil || got.PracticeStatus != lesson.StatusRunning || got.PracticeVersion != 1 {
		t.Fatalf("after reset = %+v %q v%d", got.Practice, got.PracticeStatus, got.PracticeVersion)
	}
	if ok, _ := r.SavePractice(t.Context(), c.ID, 1, 1, lessonTestPractice()); !ok {
		t.Fatal("new practice at the current version was refused")
	}
}

func TestLessonsWithoutPractice(t *testing.T) {
	t.Parallel()
	r := NewLessons(testDB(t))
	now := time.Now().UTC()
	mk := func(title string, ann, prac lesson.Status, rev int) lesson.Lesson {
		l := lessonTestBase(title, "A1", lessonTestTopicA, now)
		l.AnnotationStatus, l.PracticeStatus, l.Revision = ann, prac, rev
		return lessonTestCreate(t, r, l)
	}
	want := mk("legacy", lesson.StatusDone, lesson.StatusNone, 3)
	mk("annotating", lesson.StatusRunning, lesson.StatusNone, 1)
	mk("failed annotations", lesson.StatusFailed, lesson.StatusNone, 1)
	mk("practice done", lesson.StatusDone, lesson.StatusDone, 1)
	mk("practice running", lesson.StatusDone, lesson.StatusRunning, 1)
	mk("practice failed", lesson.StatusDone, lesson.StatusFailed, 1)

	got, err := r.WithoutPractice(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got, []lesson.RevisionRef{{ID: want.ID, Revision: 3}}) {
		t.Fatalf("WithoutPractice = %+v", got)
	}

	// Once the practice is queued, the lesson drops out.
	if ok, _ := r.SetStatus(t.Context(), want.ID, 3, job.TypePractice, lesson.StatusRunning, ""); !ok {
		t.Fatal("SetStatus failed")
	}
	if got, _ := r.WithoutPractice(t.Context()); len(got) != 0 {
		t.Fatalf("after queue = %+v", got)
	}
}

func lessonTestPtr[T any](v T) *T { return &v }
