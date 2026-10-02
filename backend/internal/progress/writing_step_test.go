package progress

import (
	"errors"
	"testing"
)

// --- F8: the Write step after Listen ---

// toWriteStep studies u1's current lesson up to the Write step.
func (e *studyEnv) toWriteStep(t *testing.T) LessonStudyView {
	t.Helper()
	id := e.current(t)
	e.complete(t, StepRead)
	e.finishDictation(t)
	v, err := e.svc.CompleteStep(t.Context(), "u1", id, StepListen)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	return v
}

func TestWriteStepAfterListen(t *testing.T) {
	t.Parallel()
	e := newStudyEnv()
	e.setGoal(t, "family")
	v := e.toWriteStep(t)
	if v.Status != LessonStudying || v.CurrentStep != StepWrite || v.Steps[StepWrite] != StateCurrent || v.Goal.CompletedLessons != 0 {
		t.Fatalf("after listen = %+v", v)
	}
	if keys, _ := e.days.CompletedKeys(t.Context(), "u1"); len(keys) != 0 {
		t.Fatal("day completed before writing")
	}

	if _, err := e.svc.CompleteStep(t.Context(), "u1", "f1", StepWrite); !errors.Is(err, ErrWriteIncomplete) {
		t.Fatalf("write before submit: %v", err)
	}
	e.writings.submit("u1", "f1")
	v, err := e.svc.CompleteStep(t.Context(), "u1", "f1", StepWrite)
	if err != nil || v.Status != LessonCompleted || v.Streak != 1 || v.Goal.CompletedLessons != 1 {
		t.Fatalf("after submit = %+v, %v", v, err)
	}
}

func TestLessonDoneBeforeWritingExistedStaysDone(t *testing.T) {
	t.Parallel()
	e := newStudyEnv()
	e.setGoal(t, "family")
	now := e.clock.now()
	_ = e.progress.Upsert(t.Context(), LessonProgress{
		UserID: "u1", LessonID: "f1", TopicID: "family", DayKey: "2026-09-30",
		Done:        map[Step]bool{"review": true, StepRead: true, StepListen: true},
		CurrentStep: StepDone, StartedAt: now, CompletedAt: now,
	})
	e.days.put("u1", "2026-09-30")

	v, err := e.svc.LessonStudy(t.Context(), "u1", "f1")
	if err != nil || v.Status != LessonCompleted || v.CurrentStep != StepDone || v.Steps[StepWrite] != StateDone ||
		v.Goal.CompletedLessons != 1 || v.Streak != 1 {
		t.Fatalf("old lesson = %+v, %v", v, err)
	}
	if id := e.current(t); id != "f2" {
		t.Fatalf("current = %s", id)
	}
}

func TestCanWrite(t *testing.T) {
	t.Parallel()
	e := newStudyEnv()
	e.setGoal(t, "family")
	if ok, _ := e.svc.CanWrite(t.Context(), "u1", "f1"); ok {
		t.Fatal("can write before the write step")
	}
	e.toWriteStep(t)
	if ok, err := e.svc.CanWrite(t.Context(), "u1", "f1"); err != nil || !ok {
		t.Fatalf("at the write step: %v %v", ok, err)
	}
	if ok, _ := e.svc.CanWrite(t.Context(), "u1", "f2"); ok {
		t.Fatal("can write another lesson")
	}
	if ok, _ := e.svc.CanWrite(t.Context(), "u2", "f1"); ok {
		t.Fatal("another learner can write")
	}
	e.writings.submit("u1", "f1")
	e.complete(t, StepWrite)
	if ok, _ := e.svc.CanWrite(t.Context(), "u1", "f1"); ok {
		t.Fatal("can write after the lesson is done")
	}
}

func TestStatsWriting(t *testing.T) {
	t.Parallel()
	e := newStudyEnv()
	if v, err := e.svc.Stats(t.Context(), "u1"); err != nil || v.Writing.Submitted != 0 || v.Writing.AverageScore != nil {
		t.Fatalf("no writing = %+v, %v", v.Writing, err)
	}
	avg := 3.8
	e.writings.count, e.writings.average = 2, &avg
	v, _ := e.svc.Stats(t.Context(), "u1")
	if v.Writing.Submitted != 2 || v.Writing.AverageScore == nil || *v.Writing.AverageScore != 3.8 {
		t.Fatalf("writing = %+v", v.Writing)
	}
}

func TestSkipWriteFinishesTheLesson(t *testing.T) {
	t.Parallel()
	e := newStudyEnv()
	e.setGoal(t, "family")

	// Only the Write step can be skipped, and only when it is the next step.
	if _, err := e.svc.SkipWrite(t.Context(), "u1", "f1"); !errors.Is(err, ErrStepLocked) {
		t.Fatalf("skip before listen: %v", err)
	}
	e.toWriteStep(t)
	v, err := e.svc.SkipWrite(t.Context(), "u1", "f1")
	if err != nil || v.Status != LessonCompleted || v.Streak != 1 || v.Goal.CompletedLessons != 1 || v.Next == nil || v.Next.ID != "f2" {
		t.Fatalf("after skip = %+v, %v", v, err)
	}
	if keys, _ := e.days.CompletedKeys(t.Context(), "u1"); len(keys) != 1 || keys[0] != "2026-09-30" {
		t.Fatalf("days = %v", keys)
	}
	// Skipping again on a finished lesson changes nothing.
	if v, err := e.svc.SkipWrite(t.Context(), "u1", "f1"); err != nil || v.Status != LessonCompleted || v.Streak != 1 {
		t.Fatalf("skip again = %+v, %v", v, err)
	}
}
