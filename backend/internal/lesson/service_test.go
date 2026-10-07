package lesson

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/luongtran/luna/backend/internal/ai"
	"github.com/luongtran/luna/backend/internal/job"
)

type env struct {
	svc      *Service
	lessons  *fakeLessons
	topics   *fakeTopics
	jobs     *fakeJobs
	ai       *fakeAI
	notified int
	dir      string
	now      time.Time
}

func newEnv(t *testing.T) *env {
	t.Helper()
	e := &env{
		lessons: newFakeLessons(), topics: newFakeTopics(), jobs: &fakeJobs{},
		ai: &fakeAI{}, dir: t.TempDir(),
		now: time.Date(2026, 9, 29, 8, 0, 0, 0, time.UTC),
	}
	e.svc = NewService(Deps{
		Lessons: e.lessons, Topics: e.topics, Jobs: e.jobs, AI: e.ai,
		Notify: func() { e.notified++ },
		Now:    func() time.Time { return e.now }, Log: slog.New(slog.DiscardHandler),
	})
	return e
}

const sampleContent = "We went to the park. He gave up smoking. It was a sunny day."

func (e *env) create(t *testing.T, mutate ...func(*Input)) Lesson {
	t.Helper()
	in := Input{Title: "Park", Content: sampleContent, TopicID: "topic-b1", Level: "B1", Source: "Tự viết", License: "CC BY"}
	for _, m := range mutate {
		m(&in)
	}
	l, err := e.svc.Create(t.Context(), in)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	e.now = e.now.Add(time.Minute)
	return l
}

func jobFor(l Lesson, t job.Type) job.Job {
	return job.Job{Type: t, LessonID: l.ID, Revision: l.Revision}
}

// --- US1: create, get, list ---

func TestCreate(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	l := e.create(t, func(in *Input) { in.Title = "  Park  " })

	if l.Title != "Park" || l.Revision != 1 || len(l.Sentences) != 3 {
		t.Fatalf("lesson = %+v", l)
	}
	if l.AnnotationStatus != StatusRunning {
		t.Fatalf("annotation status = %s", l.AnnotationStatus)
	}
	jobs := e.jobs.all()
	if len(jobs) != 1 || jobs[0].Type != job.TypeAnnotate {
		t.Fatalf("jobs = %+v", jobs)
	}
	for _, j := range jobs {
		if j.LessonID != l.ID || j.Revision != 1 || j.Status != job.StatusPending {
			t.Errorf("job = %+v", j)
		}
	}
	if e.notified != 1 {
		t.Errorf("worker notified %d times", e.notified)
	}
}

func TestCreateInvalid(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	_, err := e.svc.Create(context.Background(), Input{Title: "x"})
	var verr *ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("err = %v", err)
	}
	if len(e.lessons.byID) != 0 || len(e.jobs.all()) != 0 {
		t.Fatal("stored something for invalid input")
	}
}

func TestCreateKeepsItsOwnLevel(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	// A topic is shared by every level: the lesson's level is the one given.
	l := e.create(t, func(in *Input) { in.TopicID, in.Level = "topic-a1", "c1" })
	if l.TopicID != "topic-a1" || l.Level != "C1" {
		t.Fatalf("lesson = %s %s", l.TopicID, l.Level)
	}

	_, err := e.svc.Create(context.Background(), Input{Title: "x", Content: sampleContent, TopicID: "nope", Level: "A1", Source: "s", License: "l"})
	var verr *ValidationError
	if !errors.As(err, &verr) || verr.Fields["topicId"] != "Chủ đề không tồn tại" {
		t.Fatalf("unknown topic: %v", err)
	}
}

func TestUpdateTopicMovesRoadmap(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	e := newEnv(t)
	l := e.create(t) // B1 Work
	e.topics.setRoadmap("topic-b1", l.ID)

	in := Input{Title: "Park", Content: sampleContent, TopicID: "topic-b1", Level: "B1", Source: "s", License: "l"}
	if _, err := e.svc.Update(ctx, l.ID, in); err != nil {
		t.Fatal(err)
	}
	if len(e.topics.moves) != 0 {
		t.Fatalf("same topic moved: %v", e.topics.moves)
	}

	in.TopicID, in.Level = "topic-a1", "A1"
	got, err := e.svc.Update(ctx, l.ID, in)
	if err != nil {
		t.Fatal(err)
	}
	if got.TopicID != "topic-a1" || got.Level != "A1" {
		t.Fatalf("lesson = %s %s", got.TopicID, got.Level)
	}
	if len(e.topics.moves) != 1 || e.topics.moves[0] != l.ID+":topic-b1/B1>topic-a1/A1" {
		t.Fatalf("moves = %v", e.topics.moves)
	}

	// Changing only the level moves the lesson to that level's roadmap of the same topic.
	in.Level = "A2"
	if got, err = e.svc.Update(ctx, l.ID, in); err != nil || got.Level != "A2" {
		t.Fatalf("level change = %+v, %v", got, err)
	}
	if len(e.topics.moves) != 2 || e.topics.moves[1] != l.ID+":topic-a1/A1>topic-a1/A2" {
		t.Fatalf("moves = %v", e.topics.moves)
	}
	if ids, _ := e.topics.RoadmapLessonIDs(ctx); !ids[l.ID] || len(e.topics.roadmaps["topic-a1"]) != 1 {
		t.Fatalf("roadmaps = %v", e.topics.roadmaps)
	}

	// A content change with a topic change also moves the lesson.
	in.TopicID, in.Content = "topic-b1", "Brand new text."
	if got, _ := e.svc.Update(ctx, l.ID, in); got.TopicID != "topic-b1" || len(e.topics.moves) != 3 {
		t.Fatalf("content + topic: %s %v", got.TopicID, e.topics.moves)
	}
}

func TestGetAndList(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	e := newEnv(t)
	a := e.create(t, func(in *Input) { in.TopicID, in.Level = "topic-a1", "A1" })
	b := e.create(t)
	e.topics.setRoadmap("topic-a1", a.ID)

	if _, err := e.svc.Get(ctx, "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get missing: %v", err)
	}
	all, _ := e.svc.List(ctx, Filter{})
	if len(all) != 2 || all[0].ID != b.ID || !all[1].InRoadmap || all[0].InRoadmap ||
		all[1].TopicName != "Family" || all[0].TopicName != "Work" {
		t.Fatalf("list = %+v", all)
	}
	family, _ := e.svc.List(ctx, Filter{TopicID: "topic-a1"})
	a1, _ := e.svc.List(ctx, Filter{Level: "A1"})
	both, _ := e.svc.List(ctx, Filter{Level: "B1", TopicID: "topic-a1"})
	if len(family) != 1 || len(a1) != 1 || family[0].ID != a.ID || len(both) != 0 {
		t.Fatalf("filters: %+v %+v %+v", family, a1, both)
	}
	if _, err := e.svc.List(ctx, Filter{Level: "Z9"}); err == nil {
		t.Fatal("bad level accepted")
	}
}

// --- US2: background processing ---

func TestProcessStaleRevisionIsDiscarded(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	e := newEnv(t)
	l := e.create(t)
	e.ai.result = []ai.Annotation{{Text: "went", Lemma: "go", MeaningVi: "đi"}}
	if _, err := e.svc.Update(ctx, l.ID, Input{Title: "Park", Content: "New text here.", TopicID: "topic-b1", Level: "B1", Source: "s", License: "l"}); err != nil {
		t.Fatal(err)
	}

	if err := e.svc.ProcessAnnotate(ctx, jobFor(l, job.TypeAnnotate)); err != nil { // revision 1
		t.Fatal(err)
	}
	got, _ := e.lessons.Get(ctx, l.ID)
	if got.AnnotationStatus != StatusRunning || len(got.Annotations) != 0 {
		t.Fatalf("stale results were saved: %+v", got)
	}
	if e.ai.calls != 0 {
		t.Fatalf("stale job called the AI %d times", e.ai.calls)
	}
}

func TestProcessAnnotate(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	e := newEnv(t)
	l := e.create(t)
	e.ai.result = []ai.Annotation{
		{Text: "gave up", Lemma: "give up", MeaningVi: "bỏ", SentenceIndex: 1},
		{Text: "banana", Lemma: "banana", MeaningVi: "chuối"},
	}

	if err := e.svc.ProcessAnnotate(ctx, jobFor(l, job.TypeAnnotate)); err != nil {
		t.Fatal(err)
	}
	got, _ := e.lessons.Get(ctx, l.ID)
	if e.ai.calls != 1 || got.AnnotationStatus != StatusDone || len(got.Annotations) != 1 || got.Annotations[0].Lemma != "give up" {
		t.Fatalf("calls %d lesson %+v", e.ai.calls, got)
	}
}

func TestProcessAnnotateErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		err       error
		result    []ai.Annotation
		permanent bool
	}{
		{name: "not configured", err: ai.ErrNotConfigured, permanent: true},
		{name: "invalid key", err: ai.ErrInvalidKey, permanent: true},
		{name: "quota", err: ai.ErrQuota},
		{name: "nothing valid", result: []ai.Annotation{{Text: "banana", Lemma: "b", MeaningVi: "c"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			e := newEnv(t)
			l := e.create(t)
			e.ai.err, e.ai.result = tt.err, tt.result

			err := e.svc.ProcessAnnotate(t.Context(), jobFor(l, job.TypeAnnotate))
			if err == nil || job.IsPermanent(err) != tt.permanent {
				t.Fatalf("err = %v, permanent = %v, want %v", err, job.IsPermanent(err), tt.permanent)
			}
		})
	}
}

func TestProcessDeletedLessonIsPermanent(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	err := e.svc.ProcessAnnotate(context.Background(), job.Job{Type: job.TypeAnnotate, LessonID: "gone", Revision: 1})
	if !job.IsPermanent(err) {
		t.Fatalf("err = %v, want permanent", err)
	}
}

func TestJobFailedAndRetry(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	e := newEnv(t)
	l := e.create(t)

	// F15: annotation can be retried unless it is running.
	if _, err := e.svc.Retry(ctx, l.ID, job.TypeAnnotate); !errors.Is(err, ErrAnnotationRunning) {
		t.Fatalf("retry while running: %v", err)
	}

	e.svc.JobFailed(ctx, jobFor(l, job.TypeAnnotate), ai.ErrNotConfigured)
	got, _ := e.lessons.Get(ctx, l.ID)
	if got.AnnotationStatus != StatusFailed || got.AnnotationError != "AI chưa được cấu hình" {
		t.Fatalf("after failure: %+v", got)
	}

	before := len(e.jobs.all())
	got, err := e.svc.Retry(ctx, l.ID, job.TypeAnnotate)
	if err != nil || got.AnnotationStatus != StatusRunning || got.AnnotationError != "" {
		t.Fatalf("Retry = %+v, %v", got, err)
	}
	jobs := e.jobs.all()
	if len(jobs) != before+1 || jobs[len(jobs)-1].Type != job.TypeAnnotate {
		t.Fatalf("jobs = %+v", jobs)
	}

	// A failure reported for an old revision does not touch the lesson.
	e.svc.JobFailed(ctx, job.Job{Type: job.TypeAnnotate, LessonID: l.ID, Revision: 99}, errors.New("x"))
	if got, _ := e.lessons.Get(ctx, l.ID); got.AnnotationStatus != StatusRunning {
		t.Fatalf("stale failure applied: %+v", got)
	}

	// A failed job of a removed kind (audio) changes nothing.
	e.svc.JobFailed(ctx, job.Job{Type: "tts", LessonID: l.ID, Revision: l.Revision}, errors.New("x"))
	if got, _ := e.lessons.Get(ctx, l.ID); got.AnnotationStatus != StatusRunning || got.PracticeStatus != StatusNone {
		t.Fatalf("removed job kind applied: %+v", got)
	}
}

func TestFailureMessages(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		failureMessage(job.TypeAnnotate, ai.ErrQuota):              "AI hết hạn mức, thử lại sau",
		failureMessage(job.TypeAnnotate, ErrNoValidAnnotations):    "AI không trả về chú thích hợp lệ",
		failureMessage(job.TypeAnnotate, ai.ErrInvalidKey):         "Khoá API của AI không hợp lệ",
		failureMessage(job.TypeAnnotate, errors.New("status 500")): "Không chú thích được bài",
	}
	for got, want := range cases {
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	}
}

// --- US4: annotation editing ---

func TestUpdateAnnotations(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	e := newEnv(t)
	l := e.create(t)
	e.ai.result = []ai.Annotation{
		{Text: "went", Lemma: "go", MeaningVi: "đã đi", SentenceIndex: 0},
		{Text: "gave up", Lemma: "give up", MeaningVi: "bỏ", SentenceIndex: 1},
		{Text: "sunny", Lemma: "sunny", MeaningVi: "nắng", SentenceIndex: 2},
	}
	if _, err := e.svc.UpdateAnnotations(ctx, l.ID, nil); !errors.Is(err, ErrAnnotationRunning) {
		t.Fatalf("edit while running: %v", err)
	}
	_ = e.svc.ProcessAnnotate(ctx, jobFor(l, job.TypeAnnotate))

	got, err := e.svc.UpdateAnnotations(ctx, l.ID, []AnnotationInput{
		{Text: "went", Lemma: "go", MeaningVi: "đã đi"},                 // unchanged
		{Text: "gave up", Lemma: "give up", MeaningVi: "từ bỏ"},         // meaning changed
		{Text: "sunny day", Lemma: "sunny day", MeaningVi: "ngày nắng"}, // new
	}) // "sunny" removed
	if err != nil {
		t.Fatal(err)
	}
	want := []Annotation{
		{Text: "went", Lemma: "go", MeaningVi: "đã đi", SentenceIndex: 0, EditedByAdmin: false},
		{Text: "gave up", Lemma: "give up", MeaningVi: "từ bỏ", SentenceIndex: 1, EditedByAdmin: true},
		{Text: "sunny day", Lemma: "sunny day", MeaningVi: "ngày nắng", SentenceIndex: 2, EditedByAdmin: true},
	}
	if len(got.Annotations) != len(want) {
		t.Fatalf("annotations = %+v", got.Annotations)
	}
	for i := range want {
		if got.Annotations[i] != want[i] {
			t.Errorf("item %d = %+v, want %+v", i, got.Annotations[i], want[i])
		}
	}
}

func TestUpdateAnnotationsValidation(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	e := newEnv(t)
	l := e.create(t)
	e.svc.JobFailed(ctx, jobFor(l, job.TypeAnnotate), ai.ErrQuota)

	_, err := e.svc.UpdateAnnotations(ctx, l.ID, []AnnotationInput{
		{Text: "went", Lemma: "go", MeaningVi: "đi"},
		{Text: "banana", Lemma: "banana", MeaningVi: "chuối"},
		{Text: "park", Lemma: "", MeaningVi: ""},
	})
	var verr *ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("err = %v", err)
	}
	for _, k := range []string{"annotations.1.text", "annotations.2.lemma", "annotations.2.meaningVi"} {
		if verr.Fields[k] == "" {
			t.Errorf("missing %s in %v", k, verr.Fields)
		}
	}
	if verr.Fields["annotations.1.text"] != "Cụm từ không có trong bài" {
		t.Errorf("message = %q", verr.Fields["annotations.1.text"])
	}

	many := make([]AnnotationInput, 101)
	if _, err := e.svc.UpdateAnnotations(ctx, l.ID, many); !errors.As(err, &verr) {
		t.Fatalf("101 items accepted: %v", err)
	}

	// Manual annotations on a failed lesson mark the annotation work done.
	got, err := e.svc.UpdateAnnotations(ctx, l.ID, []AnnotationInput{{Text: "went", Lemma: "go", MeaningVi: "đi"}})
	if err != nil || got.AnnotationStatus != StatusDone || !got.Annotations[0].EditedByAdmin {
		t.Fatalf("got %+v, %v", got, err)
	}
}

// --- US5: edit and delete ---

func TestUpdateInfoOnly(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	e := newEnv(t)
	l := e.create(t)
	jobsBefore := len(e.jobs.all())

	got, err := e.svc.Update(ctx, l.ID, Input{Title: "New title", Content: sampleContent, TopicID: "topic-a1", Level: "A1", Source: "S", License: "L"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "New title" || got.Level != "A1" || got.TopicID != "topic-a1" || got.Revision != 1 || len(got.Sentences) != 3 {
		t.Fatalf("got %+v", got)
	}
	if len(e.jobs.all()) != jobsBefore {
		t.Fatal("info-only edit enqueued jobs")
	}
}

func TestUpdateContent(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	e := newEnv(t)
	l := e.create(t)
	_ = e.lessons.ReplaceAnnotations(ctx, l.ID, []Annotation{{Text: "went", Lemma: "go", MeaningVi: "đi", EditedByAdmin: true}})

	got, err := e.svc.Update(ctx, l.ID, Input{Title: "Park", Content: "One. Two.", TopicID: "topic-b1", Level: "B1", Source: "s", License: "l"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Revision != 2 || len(got.Sentences) != 2 || len(got.Annotations) != 0 || got.AnnotationStatus != StatusRunning {
		t.Fatalf("got %+v", got)
	}
	jobs := e.jobs.all()
	if last := jobs[len(jobs)-1]; last.Type != job.TypeAnnotate || last.Revision != 2 {
		t.Fatalf("new job = %+v", last)
	}
	if len(e.jobs.deletedPend) != 1 || e.jobs.deletedPend[0] != l.ID {
		t.Fatalf("pending jobs not dropped: %v", e.jobs.deletedPend)
	}
}

func TestDelete(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	e := newEnv(t)
	l := e.create(t)
	e.topics.setRoadmap("topic-b1", l.ID)

	if err := e.svc.Delete(ctx, l.ID); !errors.Is(err, ErrInRoadmap) {
		t.Fatalf("delete in roadmap: %v", err)
	}
	e.topics.setRoadmap("topic-b1")
	if err := e.svc.Delete(ctx, l.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := e.lessons.Get(ctx, l.ID); !errors.Is(err, ErrNotFound) {
		t.Fatal("lesson still stored")
	}
	if len(e.jobs.deletedLesson) != 1 {
		t.Fatal("jobs not deleted")
	}
	if err := e.svc.Delete(ctx, l.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("delete twice: %v", err)
	}
}
