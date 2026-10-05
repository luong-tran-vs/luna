package mysql

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/luongtran/luna/backend/internal/lesson"
)

// ReadingAnswers implements lesson.AnswerRepository on "reading_answers" (F15). The unique
// (user_id, lesson_id, quiz_version, question_index) key makes each answer final.
type ReadingAnswers struct {
	db *sql.DB
}

// NewReadingAnswers returns the ReadingAnswers repository.
func NewReadingAnswers(db *sql.DB) *ReadingAnswers { return &ReadingAnswers{db: db} }

var _ lesson.AnswerRepository = (*ReadingAnswers)(nil)

// readingAnswersSchema is the DDL of this domain (see migrate.go).
func readingAnswersSchema() []string {
	return []string{`CREATE TABLE IF NOT EXISTS reading_answers (
		id             CHAR(24)    NOT NULL PRIMARY KEY,
		user_id        CHAR(24)    NOT NULL,
		lesson_id      CHAR(24)    NOT NULL,
		quiz_version   INT         NOT NULL,
		question_index INT         NOT NULL,
		choice         INT         NOT NULL,
		correct        TINYINT(1)  NOT NULL,
		answered_at    DATETIME(6) NOT NULL,
		UNIQUE KEY reading_answers_question (user_id, lesson_id, quiz_version, question_index)
	) ` + tableOptions}
}

const readingAnswerCols = "user_id, lesson_id, quiz_version, question_index, choice, correct, answered_at"

func scanReadingAnswer(s interface{ Scan(dest ...any) error }) (lesson.Answer, error) {
	var a lesson.Answer
	if err := s.Scan(&a.UserID, &a.LessonID, &a.QuizVersion, &a.QuestionIndex, &a.Choice, &a.Correct, &a.AnsweredAt); err != nil {
		return lesson.Answer{}, err
	}
	a.AnsweredAt = a.AnsweredAt.UTC()
	return a, nil
}

// Insert stores a new answer; a second answer to the same question returns the first one in
// a *lesson.AlreadyAnsweredError.
func (r *ReadingAnswers) Insert(ctx context.Context, a lesson.Answer) error {
	_, err := r.db.ExecContext(ctx, "INSERT INTO reading_answers (id, "+readingAnswerCols+") VALUES (?,?,?,?,?,?,?,?)",
		newID(), a.UserID, a.LessonID, a.QuizVersion, a.QuestionIndex, a.Choice, a.Correct, utc(a.AnsweredAt))
	switch {
	case err == nil:
		return nil
	case !isDuplicate(err):
		return fmt.Errorf("mysql insert reading answer: %w", err)
	}
	first, err := scanReadingAnswer(r.db.QueryRowContext(ctx,
		"SELECT "+readingAnswerCols+" FROM reading_answers WHERE user_id = ? AND lesson_id = ? AND quiz_version = ? AND question_index = ?",
		a.UserID, a.LessonID, a.QuizVersion, a.QuestionIndex))
	if err != nil {
		return fmt.Errorf("mysql find existing reading answer: %w", err)
	}
	return &lesson.AlreadyAnsweredError{Answer: first}
}

// List returns a learner's answers to one question-set version of a lesson, by question.
func (r *ReadingAnswers) List(ctx context.Context, userID, lessonID string, version int) ([]lesson.Answer, error) {
	rows, err := r.db.QueryContext(ctx,
		"SELECT "+readingAnswerCols+" FROM reading_answers WHERE user_id = ? AND lesson_id = ? AND quiz_version = ? ORDER BY question_index",
		userID, lessonID, version)
	if err != nil {
		return nil, fmt.Errorf("mysql find reading answers: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := []lesson.Answer{}
	for rows.Next() {
		a, err := scanReadingAnswer(rows)
		if err != nil {
			return nil, fmt.Errorf("mysql read reading answers: %w", err)
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mysql read reading answers: %w", err)
	}
	return out, nil
}

// Totals counts every answer of the learner and the correct ones.
func (r *ReadingAnswers) Totals(ctx context.Context, userID string) (answered, correct int, err error) {
	err = r.db.QueryRowContext(ctx,
		"SELECT COUNT(*), CAST(COALESCE(SUM(correct), 0) AS SIGNED) FROM reading_answers WHERE user_id = ?", userID).
		Scan(&answered, &correct)
	if err != nil {
		return 0, 0, fmt.Errorf("mysql reading answer totals: %w", err)
	}
	return answered, correct, nil
}
