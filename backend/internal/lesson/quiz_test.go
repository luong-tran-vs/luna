package lesson

import (
	"context"
	"errors"
	"testing"
)

var quizQuestions = []Question{
	{Prompt: "Where did we go?", Options: []string{"Park", "Home", "School", "Work"}, AnswerIndex: 0, ExplanationVi: "Câu 1."},
	{Prompt: "What did he give up?", Options: []string{"Tea", "Smoking", "Sport", "Work"}, AnswerIndex: 1, ExplanationVi: "Câu 2."},
	{Prompt: "Who studies every day?", Options: []string{"He", "We", "She", "They"}, AnswerIndex: 2, ExplanationVi: "Câu 3."},
}

// newQuizEnv returns a reader over a lesson with three questions at QuizVersion 2.
func newQuizEnv(t *testing.T) (*Reader, *fakeLessons, *fakeAnswers, Lesson) {
	t.Helper()
	lessons := newFakeLessons()
	answers := newFakeAnswers()
	l, _ := lessons.Create(context.Background(), Lesson{
		Title: "Park", Level: "A1", TopicID: "topic-a1", Content: readingContent, Revision: 1,
		Sentences:   toSentences(SplitSentences(readingContent)),
		Extras:      Extras{Questions: quizQuestions, GrammarNote: &GrammarNote{Title: "Quá khứ đơn", BodyVi: "b", Examples: []string{"We went home."}}, WritingPrompt: "secret prompt"},
		QuizVersion: 2,
	})
	return NewReader(lessons, readingDict, newFakeTopics(), answers, newFakeAsks(), &fakeAI{}, ""), lessons, answers, l
}

func answer(t *testing.T, r *Reader, user, lessonID string, version, index, choice int) (AnswerResult, error) {
	t.Helper()
	return r.Answer(t.Context(), user, lessonID, AnswerInput{Version: version, QuestionIndex: index, Choice: choice})
}

func TestQuizViewHidesAnswers(t *testing.T) {
	t.Parallel()
	r, _, answers, l := newQuizEnv(t)
	_ = answers.Insert(t.Context(), Answer{UserID: "u1", LessonID: l.ID, QuizVersion: 2, QuestionIndex: 1, Choice: 3})
	_ = answers.Insert(t.Context(), Answer{UserID: "u1", LessonID: l.ID, QuizVersion: 1, QuestionIndex: 0, Choice: 0, Correct: true})
	_ = answers.Insert(t.Context(), Answer{UserID: "u2", LessonID: l.ID, QuizVersion: 2, QuestionIndex: 0, Choice: 0, Correct: true})

	v, err := r.View(t.Context(), "u1", l.ID)
	if err != nil {
		t.Fatal(err)
	}
	q := v.Quiz
	if q == nil || q.Version != 2 || len(q.Questions) != 3 || q.Questions[1].Prompt != "What did he give up?" ||
		len(q.Questions[1].Options) != 4 {
		t.Fatalf("quiz = %+v", q)
	}
	if len(q.Answers) != 1 {
		t.Fatalf("answers = %+v, want only u1's answer at version 2", q.Answers)
	}
	a := q.Answers[0]
	if a.QuestionIndex != 1 || a.Choice != 3 || a.Correct || a.AnswerIndex != 1 || a.ExplanationVi != "Câu 2." {
		t.Fatalf("answer = %+v", a)
	}
	if v.GrammarNote == nil || v.GrammarNote.Title != "Quá khứ đơn" {
		t.Fatalf("grammar note = %+v", v.GrammarNote)
	}
}

func TestQuizViewWithoutQuestions(t *testing.T) {
	t.Parallel()
	r, l := newReaderEnv(t)
	v, err := r.View(t.Context(), "u1", l.ID)
	if err != nil {
		t.Fatal(err)
	}
	if v.Quiz != nil || v.GrammarNote != nil {
		t.Fatalf("quiz %+v note %+v, want none", v.Quiz, v.GrammarNote)
	}
}

func TestAnswer(t *testing.T) {
	t.Parallel()
	r, _, _, l := newQuizEnv(t)

	res, err := answer(t, r, "u1", l.ID, 2, 0, 2)
	if err != nil {
		t.Fatal(err)
	}
	if res.Answer.Correct || res.Answer.AnswerIndex != 0 || res.Answer.ExplanationVi != "Câu 1." || res.Answer.Choice != 2 ||
		res.Answered != 1 || res.Total != 3 || res.Correct != 0 {
		t.Fatalf("wrong answer result = %+v", res)
	}

	res, err = answer(t, r, "u1", l.ID, 2, 1, 1)
	if err != nil || !res.Answer.Correct || res.Answered != 2 || res.Correct != 1 {
		t.Fatalf("right answer result = %+v, %v", res, err)
	}

	// The first answer stays.
	res, err = answer(t, r, "u1", l.ID, 2, 0, 0)
	var already *AlreadyAnsweredError
	if !errors.As(err, &already) || !errors.Is(err, ErrAlreadyAnswered) {
		t.Fatalf("second answer err = %v", err)
	}
	if res.Answer.Choice != 2 || res.Answer.Correct || res.Answered != 2 {
		t.Fatalf("second answer result = %+v, want the stored one", res)
	}

	// Another learner answers on their own.
	if res, err := answer(t, r, "u2", l.ID, 2, 0, 0); err != nil || !res.Answer.Correct || res.Answered != 1 {
		t.Fatalf("u2 = %+v, %v", res, err)
	}

	if q, a, err := r.QuizStatus(t.Context(), "u1", l.ID); err != nil || q != 3 || a != 2 {
		t.Fatalf("status = %d/%d, %v", a, q, err)
	}
	if n, c, err := r.Totals(t.Context(), "u1"); err != nil || n != 2 || c != 1 {
		t.Fatalf("totals = %d/%d, %v", c, n, err)
	}
}

func TestAnswerErrors(t *testing.T) {
	t.Parallel()
	r, lessons, _, l := newQuizEnv(t)

	if _, err := answer(t, r, "u1", l.ID, 1, 0, 0); !errors.Is(err, ErrQuizChanged) {
		t.Errorf("old version: %v", err)
	}
	var verr *ValidationError
	if _, err := answer(t, r, "u1", l.ID, 2, 3, 0); !errors.As(err, &verr) || verr.Fields["questionIndex"] == "" {
		t.Errorf("index out of range: %v", err)
	}
	if _, err := answer(t, r, "u1", l.ID, 2, 0, 4); !errors.As(err, &verr) || verr.Fields["choice"] == "" {
		t.Errorf("choice out of range: %v", err)
	}
	if _, err := answer(t, r, "u1", "missing", 2, 0, 0); !errors.Is(err, ErrNotFound) {
		t.Errorf("missing lesson: %v", err)
	}

	plain, _ := lessons.Create(t.Context(), Lesson{Title: "x", Content: "Hi.", Sentences: toSentences([]string{"Hi."})})
	if _, err := answer(t, r, "u1", plain.ID, 0, 0, 0); !errors.Is(err, ErrNoQuiz) {
		t.Errorf("no questions: %v", err)
	}
	if q, a, err := r.QuizStatus(t.Context(), "u1", plain.ID); err != nil || q != 0 || a != 0 {
		t.Errorf("status without questions = %d/%d, %v", a, q, err)
	}
	if q, _, err := r.QuizStatus(t.Context(), "u1", "missing"); err != nil || q != 0 {
		t.Errorf("status of a deleted lesson = %d, %v", q, err)
	}
}
