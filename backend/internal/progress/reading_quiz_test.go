package progress

import (
	"errors"
	"net/http"
	"strings"
	"testing"
)

// --- F15: the Reading step completes once every question is answered ---

func TestReadStepWaitsForQuestions(t *testing.T) {
	t.Parallel()
	e := newStudyEnv()
	e.setGoal(t, "family")
	e.quiz.set("u1", "f1", 4, 3)

	if _, err := e.svc.CompleteStep(t.Context(), "u1", "f1", StepRead); !errors.Is(err, ErrReadIncomplete) {
		t.Fatalf("3/4 answered: %v", err)
	}
	if v, _ := e.svc.LessonStudy(t.Context(), "u1", "f1"); v.CurrentStep != StepRead {
		t.Fatalf("step = %s, want read", v.CurrentStep)
	}

	// Answering the last one (right or wrong) is enough.
	e.quiz.set("u1", "f1", 4, 4)
	if v, err := e.svc.CompleteStep(t.Context(), "u1", "f1", StepRead); err != nil || v.CurrentStep != StepListen {
		t.Fatalf("all answered: %+v, %v", v, err)
	}
}

func TestReadStepWithoutQuestions(t *testing.T) {
	t.Parallel()
	e := newStudyEnv()
	e.setGoal(t, "family")
	if v, err := e.svc.CompleteStep(t.Context(), "u1", "f1", StepRead); err != nil || v.CurrentStep != StepListen {
		t.Fatalf("no questions: %+v, %v", v, err)
	}
}

func TestReadStepQuizError(t *testing.T) {
	t.Parallel()
	e := newStudyEnv()
	e.setGoal(t, "family")
	e.quiz.err = errors.New("db down")
	if _, err := e.svc.CompleteStep(t.Context(), "u1", "f1", StepRead); err == nil || errors.Is(err, ErrReadIncomplete) {
		t.Fatalf("err = %v, want the quiz error", err)
	}
}

func TestReadIncompleteEndpoint(t *testing.T) {
	t.Parallel()
	mux, e := newStudyAPI(t)
	do(t, mux, http.MethodPost, "/api/goals", "an", `{"topicId":"family"}`)
	e.quiz.set("u1", "f1", 3, 0)
	r := do(t, mux, http.MethodPost, "/api/lessons/f1/steps/read/complete", "an", "")
	if r.code != http.StatusConflict || !strings.Contains(r.text, `"read_incomplete"`) {
		t.Fatalf("read early: %d %s", r.code, r.text)
	}
}

func TestStatsReadingAccuracy(t *testing.T) {
	t.Parallel()
	e := newStudyEnv()
	v, err := e.svc.Stats(t.Context(), "u1")
	if err != nil || v.Reading.Answered != 0 || v.ReadingRate != nil {
		t.Fatalf("no answers = %+v, %v", v, err)
	}
	e.quiz.totals = [2]int{12, 9}
	v, _ = e.svc.Stats(t.Context(), "u1")
	if v.Reading.Answered != 12 || v.Reading.Correct != 9 || v.ReadingRate == nil || *v.ReadingRate != 0.75 {
		t.Fatalf("12/9 = %+v", v)
	}
}

func TestStatsEndpointReading(t *testing.T) {
	t.Parallel()
	mux, e := newStudyAPI(t)
	if r := do(t, mux, http.MethodGet, "/api/stats", "an", ""); !strings.Contains(r.text, `"reading":{"answered":0,"correct":0,"rate":null}`) {
		t.Fatalf("empty: %s", r.text)
	}
	e.quiz.totals = [2]int{4, 3}
	if r := do(t, mux, http.MethodGet, "/api/stats", "an", ""); !strings.Contains(r.text, `"reading":{"answered":4,"correct":3,"rate":0.75}`) {
		t.Fatalf("4/3: %s", r.text)
	}
}
