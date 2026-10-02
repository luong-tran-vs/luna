package lesson

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"testing"

	"github.com/luongtran/luna/backend/internal/ai"
	"github.com/luongtran/luna/backend/internal/job"
)

// sampleAnnotations match sampleContent.
var sampleAnnotations = []ai.Annotation{
	{Text: "went", Lemma: "go", MeaningVi: "đã đi", SentenceIndex: 0},
	{Text: "gave up", Lemma: "Give  up", MeaningVi: "từ bỏ", SentenceIndex: 1},
	{Text: "sunny", Lemma: "sunny", MeaningVi: "nắng", SentenceIndex: 2},
}

func samplePractice() ai.Practice {
	return ai.Practice{
		ObjectiveVi: "Bạn có thể kể về một ngày đi chơi.",
		Examples: []ai.Example{
			{Lemma: "go", Sentence: "I go to school."},
			{Lemma: "give up", Sentence: "Never give up."},
		},
		Dialogue: ai.Dialogue{Speakers: []string{"Minh", "Anna"}, Turns: []ai.Turn{
			{Speaker: 0, Text: "We went to the park.", MeaningVi: "Bọn mình đã đi công viên."},
			{Speaker: 1, Text: "Did you give up?", MeaningVi: "Bạn bỏ cuộc à?"},
			{Speaker: 0, Text: "No, I never gave up.", MeaningVi: "Không, mình không bỏ cuộc."},
			{Speaker: 1, Text: "It was a sunny day.", MeaningVi: "Hôm đó trời nắng."},
		}},
		GrammarTipVi: "Dùng quá khứ đơn để kể chuyện.",
		Translations: []ai.Translation{{Vi: "Tôi đi công viên.", En: "I go to the park.", Distractors: []string{"run", "walk"}}},
	}
}

// annotated returns a lesson whose annotations are done, as after ProcessAnnotate.
func (e *env) annotated(t *testing.T) Lesson {
	t.Helper()
	l := e.create(t)
	e.ai.result = sampleAnnotations
	if err := e.svc.ProcessAnnotate(t.Context(), jobFor(l, job.TypeAnnotate)); err != nil {
		t.Fatalf("ProcessAnnotate: %v", err)
	}
	got, _ := e.lessons.Get(t.Context(), l.ID)
	return got
}

// withPractice returns a lesson with a saved practice, as after ProcessPractice.
func (e *env) withPractice(t *testing.T) Lesson {
	t.Helper()
	l := e.annotated(t)
	e.ai.practice = samplePractice()
	if err := e.svc.ProcessPractice(t.Context(), jobFor(l, job.TypePractice)); err != nil {
		t.Fatalf("ProcessPractice: %v", err)
	}
	got, _ := e.lessons.Get(t.Context(), l.ID)
	return got
}

func jobsOf(e *env, t job.Type) []job.Job {
	var out []job.Job
	for _, j := range e.jobs.all() {
		if j.Type == t {
			out = append(out, j)
		}
	}
	return out
}

func TestAnnotateQueuesPractice(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	l := e.withPractice(t)

	// Annotating again drops the old practice and queues a new one.
	if err := e.svc.ProcessAnnotate(t.Context(), jobFor(l, job.TypeAnnotate)); err != nil {
		t.Fatal(err)
	}
	got, _ := e.lessons.Get(t.Context(), l.ID)
	if got.Practice != nil || got.PracticeStatus != StatusRunning {
		t.Fatalf("practice = %+v, status %q", got.Practice, got.PracticeStatus)
	}
	queued := jobsOf(e, job.TypePractice)
	if len(queued) != 2 || queued[1].Revision != l.Revision || queued[1].LessonID != l.ID {
		t.Fatalf("practice jobs = %+v", queued)
	}
}

func TestAnnotateFailureQueuesNoPractice(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	l := e.create(t)
	e.ai.err = ai.ErrQuota
	if err := e.svc.ProcessAnnotate(t.Context(), jobFor(l, job.TypeAnnotate)); err == nil {
		t.Fatal("want error")
	}
	if n := len(jobsOf(e, job.TypePractice)); n != 0 {
		t.Fatalf("practice jobs = %d", n)
	}
}

func TestProcessPractice(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	l := e.withPractice(t)

	calls, req := e.ai.practices()
	if calls != 1 {
		t.Fatalf("AI calls = %d", calls)
	}
	wantWords := []ai.PracticeWord{
		{Lemma: "go", Text: "went", MeaningVi: "đã đi"},
		{Lemma: "give up", Text: "gave up", MeaningVi: "từ bỏ"},
		{Lemma: "sunny", Text: "sunny", MeaningVi: "nắng"},
	}
	if !slices.Equal(req.Words, wantWords) || req.Level != "B1" || req.Title != "Park" || len(req.Sentences) != 3 {
		t.Fatalf("request = %+v", req)
	}
	if l.PracticeStatus != StatusDone || l.PracticeVersion != 1 || l.Practice == nil {
		t.Fatalf("lesson = %+v", l)
	}
	if len(l.Practice.Examples) != 2 || len(l.Practice.Dialogue.Turns) != 4 || len(l.Practice.Translations) != 1 {
		t.Fatalf("practice = %+v", l.Practice)
	}
	audio := jobsOf(e, job.TypePracticeAudio)
	if len(audio) != 1 || audio[0].TargetID != "" || audio[0].Revision != l.Revision {
		t.Fatalf("audio jobs = %+v", audio)
	}
	// Annotations and extras are untouched.
	if len(l.Annotations) != len(sampleAnnotations) || l.AnnotationStatus != StatusDone {
		t.Fatalf("annotations = %+v", l.Annotations)
	}
}

func TestProcessPracticeErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		err       error
		practice  ai.Practice
		permanent bool
		message   string
	}{
		{name: "not configured", err: ai.ErrNotConfigured, permanent: true, message: "AI chưa được cấu hình"},
		{name: "invalid key", err: ai.ErrInvalidKey, permanent: true, message: "Khoá API của AI không hợp lệ"},
		{name: "quota", err: ai.ErrQuota, message: "AI hết hạn mức, thử lại sau"},
		{name: "other", err: errors.New("boom"), message: "Không tạo được phần luyện tập"},
		{name: "nothing valid", practice: ai.Practice{ObjectiveVi: "x"}, permanent: true, message: "AI không trả về phần luyện tập hợp lệ"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			e := newEnv(t)
			l := e.annotated(t)
			e.ai.practice, e.ai.practiceErr = tt.practice, tt.err

			j := jobFor(l, job.TypePractice)
			err := e.svc.ProcessPractice(t.Context(), j)
			if err == nil || job.IsPermanent(err) != tt.permanent {
				t.Fatalf("err = %v, permanent = %v, want %v", err, job.IsPermanent(err), tt.permanent)
			}
			if calls, _ := e.ai.practices(); calls != 1 {
				t.Fatalf("AI calls = %d", calls)
			}
			e.svc.JobFailed(t.Context(), j, err)
			got, _ := e.lessons.Get(t.Context(), l.ID)
			if got.PracticeStatus != StatusFailed || got.PracticeError != tt.message || got.Practice != nil {
				t.Fatalf("status %q error %q practice %+v", got.PracticeStatus, got.PracticeError, got.Practice)
			}
			if got.AnnotationStatus != StatusDone || len(got.Annotations) != len(l.Annotations) ||
				got.QuizVersion != l.QuizVersion {
				t.Fatalf("annotations changed: %+v", got)
			}
			if n := len(jobsOf(e, job.TypePracticeAudio)); n != 0 {
				t.Fatalf("audio jobs = %d", n)
			}
		})
	}
}

func TestProcessPracticeStaleOrNotAnnotated(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	l := e.annotated(t)

	stale := jobFor(l, job.TypePractice)
	stale.Revision--
	if err := e.svc.ProcessPractice(t.Context(), stale); err != nil {
		t.Fatalf("stale: %v", err)
	}

	bare := e.create(t)
	if err := e.svc.ProcessPractice(t.Context(), jobFor(bare, job.TypePractice)); !job.IsPermanent(err) ||
		!errors.Is(err, ErrAnnotationNotDone) {
		t.Fatalf("not annotated: %v", err)
	}
	if calls, _ := e.ai.practices(); calls != 0 {
		t.Fatalf("AI calls = %d", calls)
	}
}

func TestUpdateContentDropsPractice(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	l := e.withPractice(t)

	got, err := e.svc.Update(t.Context(), l.ID, Input{
		Title: l.Title, Content: "A new text. With two sentences.", TopicID: l.TopicID, Source: l.Source, License: l.License,
	})
	if err != nil {
		t.Fatal(err)
	}
	stored, _ := e.lessons.Get(t.Context(), l.ID)
	for _, x := range []Lesson{got, stored} {
		if x.Practice != nil || x.PracticeStatus != StatusNone || x.PracticeError != "" {
			t.Fatalf("practice kept: %+v / %q", x.Practice, x.PracticeStatus)
		}
	}

	// A practice job of the old revision saves nothing.
	if err := e.svc.ProcessPractice(t.Context(), jobFor(l, job.TypePractice)); err != nil {
		t.Fatal(err)
	}
	if stored, _ = e.lessons.Get(t.Context(), l.ID); stored.Practice != nil {
		t.Fatal("stale practice saved")
	}
}

func practiceFile(e *env, l Lesson, version int, name string) string {
	return filepath.Join(e.dir, l.ID, strconv.Itoa(l.Revision), "practice", strconv.Itoa(version), name+".mp3")
}

func audioJob(l Lesson) job.Job { return jobFor(l, job.TypePracticeAudio) }

func TestProcessPracticeAudio(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	l := e.withPractice(t)

	if err := e.svc.ProcessPracticeAudio(t.Context(), audioJob(l)); err != nil {
		t.Fatal(err)
	}
	names := []string{"example-0", "example-1", "turn-0", "turn-1", "turn-2", "turn-3", "answer-0"}
	for _, n := range names {
		if b, err := os.ReadFile(practiceFile(e, l, 1, n)); err != nil || len(b) == 0 {
			t.Fatalf("%s: %v", n, err)
		}
	}
	if b, _ := os.ReadFile(practiceFile(e, l, 1, "answer-0")); string(b) != "mp3:I go to the park." {
		t.Fatalf("answer audio = %q", b)
	}
	if e.tts.calls != len(names) {
		t.Fatalf("TTS calls = %d", e.tts.calls)
	}

	// Running again only makes missing files.
	_ = os.Remove(practiceFile(e, l, 1, "turn-2"))
	if err := e.svc.ProcessPracticeAudio(t.Context(), audioJob(l)); err != nil {
		t.Fatal(err)
	}
	if e.tts.calls != len(names)+1 {
		t.Fatalf("TTS calls = %d", e.tts.calls)
	}
}

func TestProcessPracticeAudioVersions(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	l := e.withPractice(t)
	old := practiceFile(e, l, 0, "turn-0")
	if err := os.MkdirAll(filepath.Dir(old), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(old, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}

	stale := audioJob(l)
	stale.Revision--
	if err := e.svc.ProcessPracticeAudio(t.Context(), stale); err != nil || e.tts.calls != 0 {
		t.Fatalf("stale revision: %v, calls %d", err, e.tts.calls)
	}

	if err := e.svc.ProcessPracticeAudio(t.Context(), audioJob(l)); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Dir(old)); !os.IsNotExist(err) {
		t.Fatalf("old version dir kept: %v", err)
	}
}

func TestProcessPracticeAudioError(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	l := e.withPractice(t)
	e.tts.err = errors.New("tts down")

	j := audioJob(l)
	err := e.svc.ProcessPracticeAudio(t.Context(), j)
	if err == nil || job.IsPermanent(err) {
		t.Fatalf("err = %v, want retryable", err)
	}
	e.svc.JobFailed(context.Background(), j, err)
	got, _ := e.lessons.Get(t.Context(), l.ID)
	if got.PracticeStatus != StatusDone || got.PracticeError != "" || got.AnnotationStatus != StatusDone {
		t.Fatalf("statuses changed: practice %q annotation %q", got.PracticeStatus, got.AnnotationStatus)
	}
}

func TestRegeneratePractice(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	l := e.withPractice(t)
	before := len(jobsOf(e, job.TypePractice))

	got, err := e.svc.RegeneratePractice(t.Context(), l.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.PracticeStatus != StatusRunning || got.Practice == nil {
		t.Fatalf("status %q practice %+v", got.PracticeStatus, got.Practice)
	}
	if n := len(jobsOf(e, job.TypePractice)); n != before+1 {
		t.Fatalf("practice jobs = %d, want %d", n, before+1)
	}
	if _, err := e.svc.RegeneratePractice(t.Context(), l.ID); !errors.Is(err, ErrPracticeRunning) {
		t.Fatalf("running: %v", err)
	}

	// The new practice replaces the old one once saved.
	e.ai.practice.ObjectiveVi = "Bạn có thể nói về thời tiết."
	if err := e.svc.ProcessPractice(t.Context(), jobFor(l, job.TypePractice)); err != nil {
		t.Fatal(err)
	}
	got, _ = e.lessons.Get(t.Context(), l.ID)
	if got.PracticeStatus != StatusDone || got.PracticeVersion != 2 || got.Practice.ObjectiveVi != "Bạn có thể nói về thời tiết." {
		t.Fatalf("lesson = %+v", got)
	}
}

func TestRegeneratePracticeNeedsAnnotations(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	l := e.create(t)
	if _, err := e.svc.RegeneratePractice(t.Context(), l.ID); !errors.Is(err, ErrAnnotationNotDone) {
		t.Fatalf("err = %v", err)
	}
	if _, err := e.svc.RegeneratePractice(t.Context(), "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing: %v", err)
	}
	if n := len(jobsOf(e, job.TypePractice)); n != 0 {
		t.Fatalf("practice jobs = %d", n)
	}
}

func TestProcessPracticeKeepsSavedPracticeWhenAudioCannotBeQueued(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	l := e.annotated(t)
	e.ai.practice = samplePractice()
	e.jobs.enqueueErr = errors.New("db down")

	// The practice is saved, so the job must succeed: a retry would call the AI again.
	if err := e.svc.ProcessPractice(t.Context(), jobFor(l, job.TypePractice)); err != nil {
		t.Fatalf("err = %v", err)
	}
	got, _ := e.lessons.Get(t.Context(), l.ID)
	if got.Practice == nil || got.PracticeStatus != StatusDone {
		t.Fatalf("practice not saved: %+v", got)
	}
}

func TestQueueMissingPractice(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	old := e.annotated(t)
	// A lesson annotated before F17 has no practice status.
	if _, err := e.lessons.update(old.ID, func(l *Lesson) bool { l.PracticeStatus = StatusNone; return true }); err != nil {
		t.Fatal(err)
	}
	e.withPractice(t) // already has one
	e.create(t)       // not annotated yet
	before := len(jobsOf(e, job.TypePractice))

	n, err := e.svc.QueueMissingPractice(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	jobs := jobsOf(e, job.TypePractice)
	if n != 1 || len(jobs) != before+1 || jobs[len(jobs)-1].LessonID != old.ID {
		t.Fatalf("queued %d, jobs %+v", n, jobs)
	}
	got, _ := e.lessons.Get(t.Context(), old.ID)
	if got.PracticeStatus != StatusRunning {
		t.Fatalf("status = %q", got.PracticeStatus)
	}

	// Queued once only.
	if n, err := e.svc.QueueMissingPractice(t.Context()); err != nil || n != 0 {
		t.Fatalf("second run: %d, %v", n, err)
	}
}
