package mysql

import (
	"errors"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/luongtran/luna/backend/internal/job"
	"github.com/luongtran/luna/backend/internal/lesson"
	"github.com/luongtran/luna/backend/internal/topic"
)

const (
	lessonTestTopicA = "aaaaaaaaaaaaaaaaaaaaaaaa"
	lessonTestTopicB = "bbbbbbbbbbbbbbbbbbbbbbbb"
)

func lessonTestBase(title, level, topic string, created time.Time) lesson.Lesson {
	return lesson.Lesson{
		Title: title, Content: "I eat an apple. She runs fast.", Level: lesson.Level(level), TopicID: topic,
		Source: "src", License: "CC0", Revision: 1,
		Sentences:        []lesson.Sentence{{Index: 0, Text: "I eat an apple."}, {Index: 1, Text: "She runs fast."}},
		AnnotationStatus: lesson.StatusRunning,
		CreatedAt:        created, UpdatedAt: created,
	}
}

func lessonTestAnns() []lesson.Annotation {
	return []lesson.Annotation{
		{Text: "apples", Lemma: "Apple", MeaningVi: "quả táo", SentenceIndex: 0},
		{Text: "runs", Lemma: "run", MeaningVi: "chạy", SentenceIndex: 1, EditedByAdmin: true},
		{Text: "eat an", Lemma: "eat", MeaningVi: "ăn", SentenceIndex: 0},
	}
}

func lessonTestExtras() lesson.Extras {
	return lesson.Extras{
		Questions:     []lesson.Question{{Prompt: "Who runs?", Options: []string{"I", "She"}, AnswerIndex: 1, ExplanationVi: "Cô ấy"}},
		GrammarNote:   &lesson.GrammarNote{Title: "Present simple", BodyVi: "thì hiện tại", Examples: []string{"She runs."}},
		WritingPrompt: "Write about food.",
	}
}

func lessonTestCreate(t *testing.T, r *Lessons, l lesson.Lesson) lesson.Lesson {
	t.Helper()
	got, err := r.Create(t.Context(), l)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got.ID == "" {
		t.Fatal("Create returned no ID")
	}
	return got
}

func TestLessonsCreateAndGet(t *testing.T) {
	t.Parallel()
	r := NewLessons(testDB(t))
	now := time.Now().UTC().Truncate(time.Microsecond)
	l := lessonTestBase("Apples", "A1", lessonTestTopicA, now)
	l.Annotations = lessonTestAnns()
	created := lessonTestCreate(t, r, l)

	got, err := r.Get(t.Context(), created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !reflect.DeepEqual(got.Sentences, l.Sentences) || !reflect.DeepEqual(got.Annotations, l.Annotations) {
		t.Fatalf("sentences/annotations = %+v / %+v", got.Sentences, got.Annotations)
	}
	if got.Title != "Apples" || got.Level != "A1" || got.TopicID != lessonTestTopicA || got.Revision != 1 ||
		got.Source != "src" || got.License != "CC0" || got.Content != l.Content || got.AnnotationStatus != lesson.StatusRunning {
		t.Fatalf("Get = %+v", got)
	}
	if !got.CreatedAt.Equal(now) || !got.UpdatedAt.Equal(now) {
		t.Fatalf("times = %v %v, want %v", got.CreatedAt, got.UpdatedAt, now)
	}
	if got.Practice != nil || got.Extras.GrammarNote != nil || got.Extras.Questions == nil || got.PracticeStatus != lesson.StatusNone {
		t.Fatalf("defaults wrong: %+v", got)
	}

	// Unicode survives.
	l2 := lessonTestBase("Tiêu đề 🍎", "A1", "", now)
	c2 := lessonTestCreate(t, r, l2)
	g2, err := r.Get(t.Context(), c2.ID)
	if err != nil || g2.Title != "Tiêu đề 🍎" || g2.TopicID != "" {
		t.Fatalf("unicode/no topic: %+v %v", g2, err)
	}
}

func TestLessonsGetNotFound(t *testing.T) {
	t.Parallel()
	r := NewLessons(testDB(t))
	for _, id := range []string{"", "nope", "zzzzzzzzzzzzzzzzzzzzzzzz", lessonTestTopicA} {
		if _, err := r.Get(t.Context(), id); !errors.Is(err, lesson.ErrNotFound) {
			t.Fatalf("Get(%q) = %v, want ErrNotFound", id, err)
		}
	}
}

func TestLessonsListFilterAndOrder(t *testing.T) {
	t.Parallel()
	r := NewLessons(testDB(t))
	base := time.Now().UTC().Truncate(time.Second)
	a := lessonTestCreate(t, r, lessonTestBase("old A1", "A1", lessonTestTopicA, base))
	b := lessonTestCreate(t, r, lessonTestBase("mid B1", "B1", lessonTestTopicB, base.Add(time.Hour)))
	c := lessonTestCreate(t, r, lessonTestBase("new A1", "A1", lessonTestTopicB, base.Add(2*time.Hour)))

	ids := func(f lesson.Filter) []string {
		got, err := r.List(t.Context(), f)
		if err != nil {
			t.Fatalf("List(%+v): %v", f, err)
		}
		var out []string
		for _, s := range got {
			out = append(out, s.ID)
		}
		return out
	}
	if got, want := ids(lesson.Filter{}), []string{c.ID, b.ID, a.ID}; !slices.Equal(got, want) {
		t.Fatalf("all = %v, want %v", got, want)
	}
	if got, want := ids(lesson.Filter{Level: "A1"}), []string{c.ID, a.ID}; !slices.Equal(got, want) {
		t.Fatalf("A1 = %v, want %v", got, want)
	}
	if got, want := ids(lesson.Filter{TopicID: lessonTestTopicB}), []string{c.ID, b.ID}; !slices.Equal(got, want) {
		t.Fatalf("topic B = %v, want %v", got, want)
	}
	if got, want := ids(lesson.Filter{Level: "A1", TopicID: lessonTestTopicB}), []string{c.ID}; !slices.Equal(got, want) {
		t.Fatalf("A1+B = %v, want %v", got, want)
	}
	if got := ids(lesson.Filter{TopicID: "not-an-id"}); len(got) != 0 {
		t.Fatalf("malformed topic = %v", got)
	}

	sums, _ := r.List(t.Context(), lesson.Filter{TopicID: lessonTestTopicA})
	if len(sums) != 1 || sums[0].Title != "old A1" || sums[0].Level != "A1" || sums[0].TopicID != lessonTestTopicA ||
		sums[0].AnnotationStatus != lesson.StatusRunning || !sums[0].CreatedAt.Equal(base) {
		t.Fatalf("summary = %+v", sums)
	}
}

func TestLessonsSummaries(t *testing.T) {
	t.Parallel()
	r := NewLessons(testDB(t))
	now := time.Now().UTC()
	a := lessonTestCreate(t, r, lessonTestBase("a", "A1", lessonTestTopicA, now))
	b := lessonTestCreate(t, r, lessonTestBase("b", "A2", lessonTestTopicB, now))

	got, err := r.Summaries(t.Context(), []string{b.ID, "bad", lessonTestTopicA, a.ID})
	if err != nil || len(got) != 2 {
		t.Fatalf("Summaries = %+v %v", got, err)
	}
	titles := []string{got[0].Title, got[1].Title}
	slices.Sort(titles)
	if !slices.Equal(titles, []string{"a", "b"}) {
		t.Fatalf("titles = %v", titles)
	}
	if got, err := r.Summaries(t.Context(), nil); err != nil || len(got) != 0 {
		t.Fatalf("empty = %+v %v", got, err)
	}
	if got, err := r.Summaries(t.Context(), []string{"bad"}); err != nil || len(got) != 0 {
		t.Fatalf("malformed = %+v %v", got, err)
	}
}

func TestLessonsUpdateInfo(t *testing.T) {
	t.Parallel()
	r := NewLessons(testDB(t))
	l := lessonTestBase("t", "A1", lessonTestTopicA, time.Now().UTC())
	l.Annotations = lessonTestAnns()
	c := lessonTestCreate(t, r, l)

	info := lesson.Info{Title: "new", Level: "B2", TopicID: lessonTestTopicB, Source: "s2", License: "l2"}
	if err := r.UpdateInfo(t.Context(), c.ID, info); err != nil {
		t.Fatalf("UpdateInfo: %v", err)
	}
	got, _ := r.Get(t.Context(), c.ID)
	if got.Title != "new" || got.Level != "B2" || got.TopicID != lessonTestTopicB || got.Source != "s2" || got.License != "l2" {
		t.Fatalf("after = %+v", got)
	}
	if got.Content != l.Content || len(got.Annotations) != 3 || got.Revision != 1 {
		t.Fatalf("UpdateInfo touched other fields: %+v", got)
	}
	// Same values again still succeeds (matched row).
	if err := r.UpdateInfo(t.Context(), c.ID, info); err != nil {
		t.Fatalf("UpdateInfo twice: %v", err)
	}
	for _, id := range []string{"bad", lessonTestTopicA} {
		if err := r.UpdateInfo(t.Context(), id, info); !errors.Is(err, lesson.ErrNotFound) {
			t.Fatalf("UpdateInfo(%q) = %v", id, err)
		}
	}
}

func TestLessonsReplaceContent(t *testing.T) {
	t.Parallel()
	r := NewLessons(testDB(t))
	created := time.Now().UTC().Truncate(time.Microsecond)
	l := lessonTestBase("t", "A1", lessonTestTopicA, created)
	l.Annotations = lessonTestAnns()
	c := lessonTestCreate(t, r, l)

	next := c
	next.Title, next.Content, next.Revision = "t2", "New text.", 2
	next.Sentences = []lesson.Sentence{{Index: 0, Text: "New text."}}
	next.Annotations = nil
	next.AnnotationStatus = lesson.StatusRunning
	next.Extras = lessonTestExtras()
	next.ExtrasEditedByAdmin, next.QuizVersion = true, 3
	next.Practice = &lesson.Practice{ObjectiveVi: "mục tiêu"}
	next.PracticeStatus, next.PracticeError = lesson.StatusFailed, "boom"
	next.UpdatedAt = created.Add(time.Minute)
	next.CreatedAt = created.Add(time.Hour) // must not change
	if err := r.ReplaceContent(t.Context(), next); err != nil {
		t.Fatalf("ReplaceContent: %v", err)
	}
	got, _ := r.Get(t.Context(), c.ID)
	if got.Title != "t2" || got.Content != "New text." || got.Revision != 2 || len(got.Sentences) != 1 || len(got.Annotations) != 0 {
		t.Fatalf("got = %+v", got)
	}
	if !reflect.DeepEqual(got.Extras, next.Extras) || !got.ExtrasEditedByAdmin || got.QuizVersion != 3 {
		t.Fatalf("extras = %+v edited=%v quiz=%d", got.Extras, got.ExtrasEditedByAdmin, got.QuizVersion)
	}
	if got.Practice == nil || got.Practice.ObjectiveVi != "mục tiêu" || got.PracticeStatus != lesson.StatusFailed || got.PracticeError != "boom" {
		t.Fatalf("practice = %+v %q %q", got.Practice, got.PracticeStatus, got.PracticeError)
	}
	if !got.CreatedAt.Equal(created) || !got.UpdatedAt.Equal(created.Add(time.Minute)) {
		t.Fatalf("times = %v %v", got.CreatedAt, got.UpdatedAt)
	}

	missing := next
	missing.ID = lessonTestTopicA
	if err := r.ReplaceContent(t.Context(), missing); !errors.Is(err, lesson.ErrNotFound) {
		t.Fatalf("missing = %v", err)
	}
	missing.ID = "bad"
	if err := r.ReplaceContent(t.Context(), missing); !errors.Is(err, lesson.ErrNotFound) {
		t.Fatalf("malformed = %v", err)
	}
}

func TestLessonsSetStatus(t *testing.T) {
	t.Parallel()
	r := NewLessons(testDB(t))
	c := lessonTestCreate(t, r, lessonTestBase("t", "A1", lessonTestTopicA, time.Now().UTC()))

	ok, err := r.SetStatus(t.Context(), c.ID, 1, job.TypeAnnotate, lesson.StatusFailed, "ai down")
	if err != nil || !ok {
		t.Fatalf("annotate = %v %v", ok, err)
	}
	got, _ := r.Get(t.Context(), c.ID)
	if got.AnnotationStatus != lesson.StatusFailed || got.AnnotationError != "ai down" || got.PracticeStatus != lesson.StatusNone {
		t.Fatalf("after annotate: %+v", got)
	}
	ok, err = r.SetStatus(t.Context(), c.ID, 1, job.TypePractice, lesson.StatusRunning, "")
	if err != nil || !ok {
		t.Fatalf("practice = %v %v", ok, err)
	}
	got, _ = r.Get(t.Context(), c.ID)
	if got.PracticeStatus != lesson.StatusRunning || got.AnnotationStatus != lesson.StatusFailed {
		t.Fatalf("after practice: %+v", got)
	}

	// Wrong revision: nothing changes.
	ok, err = r.SetStatus(t.Context(), c.ID, 2, job.TypeAnnotate, lesson.StatusDone, "")
	if err != nil || ok {
		t.Fatalf("stale = %v %v", ok, err)
	}
	if got, _ = r.Get(t.Context(), c.ID); got.AnnotationStatus != lesson.StatusFailed {
		t.Fatalf("stale write changed status: %+v", got)
	}
	// Missing lesson: not ok; malformed id: ErrNotFound; unknown job type: error.
	if ok, err := r.SetStatus(t.Context(), lessonTestTopicA, 1, job.TypeAnnotate, lesson.StatusDone, ""); ok || err != nil {
		t.Fatalf("missing = %v %v", ok, err)
	}
	if _, err := r.SetStatus(t.Context(), "bad", 1, job.TypeAnnotate, lesson.StatusDone, ""); !errors.Is(err, lesson.ErrNotFound) {
		t.Fatalf("malformed = %v", err)
	}
	if _, err := r.SetStatus(t.Context(), c.ID, 1, job.Type("other"), lesson.StatusDone, ""); err == nil {
		t.Fatal("unknown job type accepted")
	}
}

func TestLessonsSaveAnnotations(t *testing.T) {
	t.Parallel()
	r := NewLessons(testDB(t))
	l := lessonTestBase("t", "A1", lessonTestTopicA, time.Now().UTC())
	l.QuizVersion = 2
	l.Practice = &lesson.Practice{ObjectiveVi: "old"}
	l.PracticeStatus, l.PracticeVersion = lesson.StatusDone, 4
	l.ExtrasEditedByAdmin = true
	l.AnnotationStatus, l.AnnotationError = lesson.StatusFailed, "x"
	c := lessonTestCreate(t, r, l)

	// Stale revision: nothing changes.
	ok, err := r.SaveAnnotations(t.Context(), c.ID, 9, lessonTestAnns(), lessonTestExtras())
	if err != nil || ok {
		t.Fatalf("stale = %v %v", ok, err)
	}
	if got, _ := r.Get(t.Context(), c.ID); got.QuizVersion != 2 || got.Practice == nil || len(got.Annotations) != 0 {
		t.Fatalf("stale write changed the lesson: %+v", got)
	}

	ok, err = r.SaveAnnotations(t.Context(), c.ID, 1, lessonTestAnns(), lessonTestExtras())
	if err != nil || !ok {
		t.Fatalf("SaveAnnotations = %v %v", ok, err)
	}
	got, _ := r.Get(t.Context(), c.ID)
	if !reflect.DeepEqual(got.Annotations, lessonTestAnns()) || !reflect.DeepEqual(got.Extras, lessonTestExtras()) {
		t.Fatalf("annotations/extras = %+v / %+v", got.Annotations, got.Extras)
	}
	if got.AnnotationStatus != lesson.StatusDone || got.AnnotationError != "" || got.ExtrasEditedByAdmin || got.QuizVersion != 3 {
		t.Fatalf("annotation state = %+v", got)
	}
	if got.Practice != nil || got.PracticeStatus != lesson.StatusRunning || got.PracticeError != "" || got.PracticeVersion != 4 {
		t.Fatalf("practice state = %+v %q v%d", got.Practice, got.PracticeStatus, got.PracticeVersion)
	}
	// Extras without a grammar note and with no questions.
	if ok, err := r.SaveAnnotations(t.Context(), c.ID, 1, nil, lesson.Extras{}); err != nil || !ok {
		t.Fatalf("empty = %v %v", ok, err)
	}
	got, _ = r.Get(t.Context(), c.ID)
	if got.Extras.GrammarNote != nil || len(got.Extras.Questions) != 0 || got.Annotations == nil || got.QuizVersion != 4 {
		t.Fatalf("empty save = %+v", got)
	}
	if ok, err := r.SaveAnnotations(t.Context(), lessonTestTopicA, 1, nil, lesson.Extras{}); ok || err != nil {
		t.Fatalf("missing = %v %v", ok, err)
	}
	if _, err := r.SaveAnnotations(t.Context(), "bad", 1, nil, lesson.Extras{}); !errors.Is(err, lesson.ErrNotFound) {
		t.Fatalf("malformed = %v", err)
	}
}

func TestLessonsReplaceExtras(t *testing.T) {
	t.Parallel()
	r := NewLessons(testDB(t))
	c := lessonTestCreate(t, r, lessonTestBase("t", "A1", lessonTestTopicA, time.Now().UTC()))

	if err := r.ReplaceExtras(t.Context(), c.ID, lessonTestExtras(), false); err != nil {
		t.Fatalf("ReplaceExtras: %v", err)
	}
	got, _ := r.Get(t.Context(), c.ID)
	if !reflect.DeepEqual(got.Extras, lessonTestExtras()) || !got.ExtrasEditedByAdmin || got.QuizVersion != 0 {
		t.Fatalf("no bump: %+v edited=%v quiz=%d", got.Extras, got.ExtrasEditedByAdmin, got.QuizVersion)
	}
	if err := r.ReplaceExtras(t.Context(), c.ID, lesson.Extras{WritingPrompt: "only"}, true); err != nil {
		t.Fatalf("ReplaceExtras bump: %v", err)
	}
	got, _ = r.Get(t.Context(), c.ID)
	if got.QuizVersion != 1 || got.Extras.WritingPrompt != "only" || got.Extras.GrammarNote != nil || len(got.Extras.Questions) != 0 {
		t.Fatalf("bump: %+v quiz=%d", got.Extras, got.QuizVersion)
	}
	for _, id := range []string{"bad", lessonTestTopicA} {
		if err := r.ReplaceExtras(t.Context(), id, lesson.Extras{}, true); !errors.Is(err, lesson.ErrNotFound) {
			t.Fatalf("ReplaceExtras(%q) = %v", id, err)
		}
	}
}

func TestLessonsReplaceAnnotations(t *testing.T) {
	t.Parallel()
	r := NewLessons(testDB(t))
	l := lessonTestBase("t", "A1", lessonTestTopicA, time.Now().UTC())
	l.AnnotationStatus, l.AnnotationError = lesson.StatusFailed, "x"
	l.Extras = lessonTestExtras()
	c := lessonTestCreate(t, r, l)

	if err := r.ReplaceAnnotations(t.Context(), c.ID, lessonTestAnns()); err != nil {
		t.Fatalf("ReplaceAnnotations: %v", err)
	}
	got, _ := r.Get(t.Context(), c.ID)
	if !reflect.DeepEqual(got.Annotations, lessonTestAnns()) || got.AnnotationStatus != lesson.StatusDone || got.AnnotationError != "" {
		t.Fatalf("got = %+v", got)
	}
	if got.QuizVersion != 0 || len(got.Extras.Questions) != 1 {
		t.Fatalf("touched extras: %+v", got)
	}
	for _, id := range []string{"bad", lessonTestTopicA} {
		if err := r.ReplaceAnnotations(t.Context(), id, nil); !errors.Is(err, lesson.ErrNotFound) {
			t.Fatalf("ReplaceAnnotations(%q) = %v", id, err)
		}
	}
}

func TestLessonsDelete(t *testing.T) {
	t.Parallel()
	r := NewLessons(testDB(t))
	c := lessonTestCreate(t, r, lessonTestBase("t", "A1", lessonTestTopicA, time.Now().UTC()))
	if err := r.Delete(t.Context(), c.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := r.Get(t.Context(), c.ID); !errors.Is(err, lesson.ErrNotFound) {
		t.Fatalf("after delete = %v", err)
	}
	if err := r.Delete(t.Context(), c.ID); err != nil {
		t.Fatalf("delete twice: %v", err)
	}
	if err := r.Delete(t.Context(), "bad"); !errors.Is(err, lesson.ErrNotFound) {
		t.Fatalf("malformed = %v", err)
	}
}

func TestLessonsCountByTopic(t *testing.T) {
	t.Parallel()
	r := NewLessons(testDB(t))
	now := time.Now().UTC()
	lessonTestCreate(t, r, lessonTestBase("1", "A1", lessonTestTopicA, now))
	lessonTestCreate(t, r, lessonTestBase("2", "A1", lessonTestTopicA, now))
	lessonTestCreate(t, r, lessonTestBase("5", "B2", lessonTestTopicA, now))
	lessonTestCreate(t, r, lessonTestBase("3", "A1", lessonTestTopicB, now))
	lessonTestCreate(t, r, lessonTestBase("4", "A1", "", now))
	got, err := r.CountByTopic(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]map[string]int{lessonTestTopicA: {"A1": 2, "B2": 1}, lessonTestTopicB: {"A1": 1}, "": {"A1": 1}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("counts = %v, want %v", got, want)
	}
}

func TestLessonsPlaceOf(t *testing.T) {
	t.Parallel()
	r := NewLessons(testDB(t))
	now := time.Now().UTC()
	a := lessonTestCreate(t, r, lessonTestBase("1", "A2", lessonTestTopicA, now))
	b := lessonTestCreate(t, r, lessonTestBase("2", "A1", "", now))
	got, err := r.PlaceOf(t.Context(), []string{a.ID, b.ID, "bad", lessonTestTopicB})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]topic.Place{a.ID: {TopicID: lessonTestTopicA, Level: "A2"}, b.ID: {Level: "A1"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("PlaceOf = %v, want %v", got, want)
	}
}

func TestLessonsTopicTexts(t *testing.T) {
	t.Parallel()
	r := NewLessons(testDB(t))
	now := time.Now().UTC()
	l := lessonTestBase("1", "A1", lessonTestTopicA, now)
	l.Annotations = []lesson.Annotation{
		{Text: " Apples ", Lemma: " Apple "},
		{Text: "eat an", Lemma: "eat"}, // phrase: skipped
		{Text: "ran", Lemma: ""},       // no lemma: skipped
		{Text: "Runs", Lemma: "RUN"},
	}
	lessonTestCreate(t, r, l)
	other := lessonTestBase("2", "A1", lessonTestTopicA, now.Add(time.Second))
	other.Content = "Second."
	lessonTestCreate(t, r, other)
	lessonTestCreate(t, r, lessonTestBase("3", "A1", lessonTestTopicB, now))

	got, err := r.TopicTexts(t.Context(), []string{lessonTestTopicA, "bad", "cccccccccccccccccccccccc"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || len(got[lessonTestTopicA]) != 2 {
		t.Fatalf("TopicTexts = %+v", got)
	}
	first := got[lessonTestTopicA][0]
	wantLemmas := map[string]string{"apples": "apple", "runs": "run"}
	if first.Content != l.Content || !reflect.DeepEqual(first.Lemmas, wantLemmas) {
		t.Fatalf("first = %+v", first)
	}
	if second := got[lessonTestTopicA][1]; second.Content != "Second." || second.Lemmas == nil || len(second.Lemmas) != 0 {
		t.Fatalf("second = %+v", second)
	}
	if got, err := r.TopicTexts(t.Context(), nil); err != nil || got == nil || len(got) != 0 {
		t.Fatalf("empty = %+v %v", got, err)
	}
}

func TestLessonsGrammarPoint(t *testing.T) {
	t.Parallel()
	r := NewLessons(testDB(t))
	now := time.Now().UTC().Truncate(time.Microsecond)

	l := lessonTestBase("a", "A1", lessonTestTopicA, now)
	l.GrammarPointID = "a1-to-be"
	a := lessonTestCreate(t, r, l)
	if got, _ := r.Get(t.Context(), a.ID); got.GrammarPointID != "a1-to-be" {
		t.Fatalf("Create/Get = %q", got.GrammarPointID)
	}
	l2 := lessonTestBase("b", "A1", lessonTestTopicB, now)
	l2.GrammarPointID = "a1-to-be"
	lessonTestCreate(t, r, l2)
	l3 := lessonTestBase("c", "A1", lessonTestTopicB, now)
	l3.GrammarPointID = "a1-articles"
	c := lessonTestCreate(t, r, l3)
	lessonTestCreate(t, r, lessonTestBase("none", "A1", lessonTestTopicB, now))

	all, err := r.CountByGrammarPoint(t.Context(), "")
	if err != nil || !reflect.DeepEqual(all, map[string]int{"a1-to-be": 2, "a1-articles": 1}) {
		t.Fatalf("all = %v %v", all, err)
	}
	inB, err := r.CountByGrammarPoint(t.Context(), lessonTestTopicB)
	if err != nil || !reflect.DeepEqual(inB, map[string]int{"a1-to-be": 1, "a1-articles": 1}) {
		t.Fatalf("topic B = %v %v", inB, err)
	}
	if got, err := r.CountByGrammarPoint(t.Context(), "bad"); err != nil || len(got) != 0 {
		t.Fatalf("malformed topic = %v %v", got, err)
	}

	// UpdateInfo sets and clears the point.
	info := lesson.Info{Title: "c", Level: "A1", TopicID: lessonTestTopicB, Source: "src", License: "CC0", GrammarPointID: "a1-this-that"}
	if err := r.UpdateInfo(t.Context(), c.ID, info); err != nil {
		t.Fatal(err)
	}
	if got, _ := r.Get(t.Context(), c.ID); got.GrammarPointID != "a1-this-that" {
		t.Fatalf("UpdateInfo = %q", got.GrammarPointID)
	}
	info.GrammarPointID = ""
	if err := r.UpdateInfo(t.Context(), c.ID, info); err != nil {
		t.Fatal(err)
	}
	if got, _ := r.Get(t.Context(), c.ID); got.GrammarPointID != "" {
		t.Fatalf("cleared = %q", got.GrammarPointID)
	}

	// ReplaceContent writes it.
	next, _ := r.Get(t.Context(), c.ID)
	next.GrammarPointID, next.Revision = "a1-have-has", 2
	if err := r.ReplaceContent(t.Context(), next); err != nil {
		t.Fatal(err)
	}
	if got, _ := r.Get(t.Context(), c.ID); got.GrammarPointID != "a1-have-has" {
		t.Fatalf("ReplaceContent = %q", got.GrammarPointID)
	}

}
