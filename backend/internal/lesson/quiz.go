package lesson

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// AnswerRepository stores learners' answers to comprehension questions (F15). A learner keeps
// one answer per question of a question-set version: the latest one.
type AnswerRepository interface {
	// Upsert stores a, replacing the learner's earlier answer to the same question and version.
	Upsert(ctx context.Context, a Answer) error
	// Delete removes a learner's answers to one version of a lesson's questions.
	Delete(ctx context.Context, userID, lessonID string, version int) error
	// List returns a learner's answers to one version of a lesson's questions, by question.
	List(ctx context.Context, userID, lessonID string, version int) ([]Answer, error)
	// Totals counts a learner's answers (every version) answered at or after since (nil = all)
	// and the correct ones.
	Totals(ctx context.Context, userID string, since *time.Time) (answered, correct int, err error)
}

// QuizQuestion is a question as learners see it before answering: no answer, no explanation.
type QuizQuestion struct {
	Prompt  string
	Options []string
}

// AnswerView is an answered question with its solution.
type AnswerView struct {
	QuestionIndex int
	Choice        int
	Correct       bool
	AnswerIndex   int
	ExplanationVi string
}

// QuizView is the comprehension quiz of a lesson for one learner.
type QuizView struct {
	Version   int
	Questions []QuizQuestion
	// Answers are the learner's answers to this version, by question.
	Answers []AnswerView
}

// AnswerInput is one answer sent by a learner.
type AnswerInput struct {
	Version       int
	QuestionIndex int
	Choice        int
}

// AnswerResult is the checked answer and the learner's progress on the quiz.
type AnswerResult struct {
	Answer   AnswerView
	Answered int
	Total    int
	Correct  int
}

// quiz returns the learner's quiz of l, or nil when l has no questions.
func (r *Reader) quiz(ctx context.Context, userID string, l Lesson) (*QuizView, error) {
	qs := l.Extras.Questions
	if len(qs) == 0 {
		return nil, nil
	}
	v := &QuizView{Version: l.QuizVersion, Questions: make([]QuizQuestion, len(qs)), Answers: []AnswerView{}}
	for i, q := range qs {
		v.Questions[i] = QuizQuestion{Prompt: q.Prompt, Options: q.Options}
	}
	answers, err := r.answers.List(ctx, userID, l.ID, l.QuizVersion)
	if err != nil {
		return nil, fmt.Errorf("lesson: list answers: %w", err)
	}
	for _, a := range answers {
		if a.QuestionIndex >= 0 && a.QuestionIndex < len(qs) {
			v.Answers = append(v.Answers, answerView(a, qs[a.QuestionIndex]))
		}
	}
	return v, nil
}

// Answer checks and stores a learner's answer. Answering a question again replaces the stored
// answer (the latest attempt counts); answers to an older question set are refused.
func (r *Reader) Answer(ctx context.Context, userID, lessonID string, in AnswerInput) (AnswerResult, error) {
	l, err := r.lessons.Get(ctx, lessonID)
	if err != nil {
		return AnswerResult{}, err
	}
	qs := l.Extras.Questions
	switch {
	case len(qs) == 0:
		return AnswerResult{}, ErrNoQuiz
	case in.Version != l.QuizVersion:
		return AnswerResult{}, ErrQuizChanged
	}
	fields := map[string]string{}
	if in.QuestionIndex < 0 || in.QuestionIndex >= len(qs) {
		fields["questionIndex"] = "Câu hỏi không tồn tại"
	}
	if in.Choice < 0 || in.Choice >= questionOptions {
		fields["choice"] = "Lựa chọn không hợp lệ"
	}
	if len(fields) > 0 {
		return AnswerResult{}, &ValidationError{Fields: fields}
	}

	q := qs[in.QuestionIndex]
	a := Answer{
		UserID: userID, LessonID: l.ID, QuizVersion: l.QuizVersion, QuestionIndex: in.QuestionIndex,
		Choice: in.Choice, Correct: in.Choice == q.AnswerIndex, AnsweredAt: time.Now().UTC(),
	}
	if err := r.answers.Upsert(ctx, a); err != nil {
		return AnswerResult{}, fmt.Errorf("lesson: save answer: %w", err)
	}

	res := AnswerResult{Answer: answerView(a, q), Total: len(qs)}
	answers, err := r.answers.List(ctx, userID, l.ID, l.QuizVersion)
	if err != nil {
		return AnswerResult{}, fmt.Errorf("lesson: list answers: %w", err)
	}
	for _, x := range answers {
		res.Answered++
		if x.Correct {
			res.Correct++
		}
	}
	return res, nil
}

// ResetAnswers deletes the learner's answers to the current question set of a lesson so the
// quiz can be taken again. Answers to older question sets are kept: they belong to questions
// that are no longer shown and only count in statistics. Lesson progress is not touched.
func (r *Reader) ResetAnswers(ctx context.Context, userID, lessonID string) error {
	l, err := r.lessons.Get(ctx, lessonID)
	if err != nil {
		return err
	}
	if err := r.answers.Delete(ctx, userID, l.ID, l.QuizVersion); err != nil {
		return fmt.Errorf("lesson: delete answers: %w", err)
	}
	return nil
}

// QuizStatus returns how many questions the lesson has and how many of them the learner has
// answered in the current version. A missing lesson has no questions.
func (r *Reader) QuizStatus(ctx context.Context, userID, lessonID string) (questions, answered int, err error) {
	l, err := r.lessons.Get(ctx, lessonID)
	if errors.Is(err, ErrNotFound) {
		return 0, 0, nil
	}
	if err != nil {
		return 0, 0, err
	}
	if len(l.Extras.Questions) == 0 {
		return 0, 0, nil
	}
	answers, err := r.answers.List(ctx, userID, l.ID, l.QuizVersion)
	if err != nil {
		return 0, 0, fmt.Errorf("lesson: list answers: %w", err)
	}
	return len(l.Extras.Questions), len(answers), nil
}

// Totals counts a learner's answers since `since` (nil = all) and the correct ones, for statistics.
func (r *Reader) Totals(ctx context.Context, userID string, since *time.Time) (answered, correct int, err error) {
	return r.answers.Totals(ctx, userID, since)
}

func answerView(a Answer, q Question) AnswerView {
	return AnswerView{
		QuestionIndex: a.QuestionIndex, Choice: a.Choice, Correct: a.Correct,
		AnswerIndex: q.AnswerIndex, ExplanationVi: q.ExplanationVi,
	}
}
