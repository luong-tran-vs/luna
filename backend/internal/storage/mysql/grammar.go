package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/luongtran/luna/backend/internal/grammar"
)

// grammarSchema is the DDL of this domain (see migrate.go): the grammar lessons (one per syllabus
// point, content read and written whole so a JSON column) and the learner progress on each point.
func grammarSchema() []string {
	return []string{
		`CREATE TABLE IF NOT EXISTS grammar_lessons (
  point_id VARCHAR(64) NOT NULL PRIMARY KEY,
  status VARCHAR(16) NOT NULL,
  edited TINYINT(1) NOT NULL DEFAULT 0,
  content JSON NOT NULL,
  checks JSON NOT NULL,
  checked_at DATETIME(6) NULL,
  verified_at DATETIME(6) NULL,
  created_at DATETIME(6) NOT NULL,
  updated_at DATETIME(6) NOT NULL,
  published_at DATETIME(6) NULL
) ` + tableOptions,
		`CREATE TABLE IF NOT EXISTS grammar_progress (
  user_id CHAR(24) NOT NULL,
  point_id VARCHAR(64) NOT NULL,
  status VARCHAR(16) NOT NULL,
  practice_attempts INT NOT NULL DEFAULT 0,
  last_practice INT NOT NULL DEFAULT 0,
  mastery_attempts INT NOT NULL DEFAULT 0,
  best_mastery INT NOT NULL DEFAULT 0,
  mastered_at DATETIME(6) NULL,
  weak JSON NOT NULL,
  updated_at DATETIME(6) NOT NULL,
  PRIMARY KEY (user_id, point_id)
) ` + tableOptions,
	}
}

// --- grammar_lessons ---

type grammarExerciseJSON struct {
	ID            string   `json:"id"`
	Kind          string   `json:"kind"`
	PromptVi      string   `json:"promptVi,omitempty"`
	Text          string   `json:"text,omitempty"`
	Options       []string `json:"options,omitempty"`
	AnswerIndex   int      `json:"answerIndex,omitempty"`
	Answers       []string `json:"answers,omitempty"`
	Words         []string `json:"words,omitempty"`
	Sentence      string   `json:"sentence,omitempty"`
	ExplanationVi string   `json:"explanationVi,omitempty"`
}

type grammarStructureJSON struct {
	Label   string `json:"label"`
	Pattern string `json:"pattern"`
	Example string `json:"example"`
}

type grammarExampleJSON struct {
	En string `json:"en"`
	Vi string `json:"vi"`
}

type grammarMistakeJSON struct {
	Wrong  string `json:"wrong"`
	Right  string `json:"right"`
	NoteVi string `json:"noteVi"`
}

type grammarContentJSON struct {
	Objective   string                 `json:"objective"`
	Explanation []string               `json:"explanation"`
	Usage       []string               `json:"usage"`
	Structures  []grammarStructureJSON `json:"structures"`
	Examples    []grammarExampleJSON   `json:"examples"`
	Mistakes    []grammarMistakeJSON   `json:"mistakes"`
	Practice    []grammarExerciseJSON  `json:"practice"`
	Mastery     []grammarExerciseJSON  `json:"mastery"`
}

// grammarStrs returns s, or an empty non-nil slice, so a list is "[]" in JSON and never nil.
func grammarStrs(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func grammarContentToJSON(c grammar.Content) grammarContentJSON {
	out := grammarContentJSON{
		Objective: c.Objective, Explanation: grammarStrs(c.Explanation), Usage: grammarStrs(c.Usage),
		Structures: make([]grammarStructureJSON, len(c.Structures)),
		Examples:   make([]grammarExampleJSON, len(c.Examples)),
		Mistakes:   make([]grammarMistakeJSON, len(c.Mistakes)),
		Practice:   make([]grammarExerciseJSON, len(c.Practice)),
		Mastery:    make([]grammarExerciseJSON, len(c.Mastery)),
	}
	for i, s := range c.Structures {
		out.Structures[i] = grammarStructureJSON(s)
	}
	for i, e := range c.Examples {
		out.Examples[i] = grammarExampleJSON(e)
	}
	for i, m := range c.Mistakes {
		out.Mistakes[i] = grammarMistakeJSON(m)
	}
	for i, e := range c.Practice {
		out.Practice[i] = grammarExerciseToJSON(e)
	}
	for i, e := range c.Mastery {
		out.Mastery[i] = grammarExerciseToJSON(e)
	}
	return out
}

func grammarExerciseToJSON(e grammar.Exercise) grammarExerciseJSON {
	return grammarExerciseJSON{
		ID: e.ID, Kind: string(e.Kind), PromptVi: e.PromptVi, Text: e.Text, Options: e.Options,
		AnswerIndex: e.AnswerIndex, Answers: e.Answers, Words: e.Words, Sentence: e.Sentence,
		ExplanationVi: e.ExplanationVi,
	}
}

func (e grammarExerciseJSON) toExercise() grammar.Exercise {
	return grammar.Exercise{
		ID: e.ID, Kind: grammar.Kind(e.Kind), PromptVi: e.PromptVi, Text: e.Text, Options: e.Options,
		AnswerIndex: e.AnswerIndex, Answers: e.Answers, Words: e.Words, Sentence: e.Sentence,
		ExplanationVi: e.ExplanationVi,
	}
}

func (c grammarContentJSON) toContent() grammar.Content {
	out := grammar.Content{
		Objective: c.Objective, Explanation: grammarStrs(c.Explanation), Usage: grammarStrs(c.Usage),
		Structures: make([]grammar.Structure, len(c.Structures)),
		Examples:   make([]grammar.Example, len(c.Examples)),
		Mistakes:   make([]grammar.Mistake, len(c.Mistakes)),
		Practice:   make([]grammar.Exercise, len(c.Practice)),
		Mastery:    make([]grammar.Exercise, len(c.Mastery)),
	}
	for i, s := range c.Structures {
		out.Structures[i] = grammar.Structure(s)
	}
	for i, e := range c.Examples {
		out.Examples[i] = grammar.Example(e)
	}
	for i, m := range c.Mistakes {
		out.Mistakes[i] = grammar.Mistake(m)
	}
	for i, e := range c.Practice {
		out.Practice[i] = e.toExercise()
	}
	for i, e := range c.Mastery {
		out.Mastery[i] = e.toExercise()
	}
	return out
}

type grammarCheckJSON struct {
	ExerciseID string `json:"exerciseId"`
	Kind       string `json:"kind"`
	NoteVi     string `json:"noteVi"`
	Confirmed  bool   `json:"confirmed"`
}

func grammarChecksToJSON(in []grammar.Check) []grammarCheckJSON {
	out := make([]grammarCheckJSON, len(in))
	for i, c := range in {
		out[i] = grammarCheckJSON{ExerciseID: c.ExerciseID, Kind: string(c.Kind), NoteVi: c.NoteVi, Confirmed: c.Confirmed}
	}
	return out
}

func grammarChecksFromJSON(raw []byte) ([]grammar.Check, error) {
	var in []grammarCheckJSON
	if err := fromJSON(raw, &in); err != nil {
		return nil, err
	}
	out := make([]grammar.Check, len(in))
	for i, c := range in {
		out[i] = grammar.Check{ExerciseID: c.ExerciseID, Kind: grammar.CheckKind(c.Kind), NoteVi: c.NoteVi, Confirmed: c.Confirmed}
	}
	return out, nil
}

// GrammarLessons implements grammar.LessonRepository on "grammar_lessons".
type GrammarLessons struct {
	db *sql.DB
}

// NewGrammarLessons returns the grammar lesson repository.
func NewGrammarLessons(db *sql.DB) *GrammarLessons { return &GrammarLessons{db: db} }

var _ grammar.LessonRepository = (*GrammarLessons)(nil)

// Get returns the lesson of a point.
func (r *GrammarLessons) Get(ctx context.Context, pointID string) (grammar.Lesson, error) {
	var (
		l          grammar.Lesson
		status     string
		raw        []byte
		checksRaw  []byte
		checkedAt  sql.NullTime
		verifiedAt sql.NullTime
		published  sql.NullTime
	)
	err := r.db.QueryRowContext(ctx,
		"SELECT point_id, status, edited, content, checks, checked_at, verified_at, created_at, updated_at, published_at FROM grammar_lessons WHERE point_id = ?",
		pointID).Scan(&l.PointID, &status, &l.Edited, &raw, &checksRaw, &checkedAt, &verifiedAt, &l.CreatedAt, &l.UpdatedAt, &published)
	if errors.Is(err, sql.ErrNoRows) {
		return grammar.Lesson{}, grammar.ErrNotFound
	}
	if err != nil {
		return grammar.Lesson{}, fmt.Errorf("mysql find grammar lesson: %w", err)
	}
	var c grammarContentJSON
	if err := fromJSON(raw, &c); err != nil {
		return grammar.Lesson{}, err
	}
	if l.Checks, err = grammarChecksFromJSON(checksRaw); err != nil {
		return grammar.Lesson{}, err
	}
	l.Status, l.Content, l.CheckedAt, l.VerifiedAt = grammar.Status(status), c.toContent(), timeOf(checkedAt), timeOf(verifiedAt)
	l.CreatedAt, l.UpdatedAt, l.PublishedAt = l.CreatedAt.UTC(), l.UpdatedAt.UTC(), timeOf(published)
	return l, nil
}

// Save inserts or replaces the whole lesson.
func (r *GrammarLessons) Save(ctx context.Context, l grammar.Lesson) error {
	raw, err := toJSON(grammarContentToJSON(l.Content))
	if err != nil {
		return err
	}
	checks, err := toJSON(grammarChecksToJSON(l.Checks))
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO grammar_lessons (point_id, status, edited, content, checks, checked_at, verified_at, created_at, updated_at, published_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE status = VALUES(status), edited = VALUES(edited), content = VALUES(content),
  checks = VALUES(checks), checked_at = VALUES(checked_at), verified_at = VALUES(verified_at),
  created_at = VALUES(created_at), updated_at = VALUES(updated_at), published_at = VALUES(published_at)`,
		l.PointID, string(l.Status), l.Edited, string(raw), string(checks), nullTime(l.CheckedAt), nullTime(l.VerifiedAt), utc(l.CreatedAt), utc(l.UpdatedAt), nullTime(l.PublishedAt))
	if err != nil {
		return fmt.Errorf("mysql save grammar lesson: %w", err)
	}
	return nil
}

// List returns a summary of every lesson, without content.
func (r *GrammarLessons) List(ctx context.Context) ([]grammar.LessonSummary, error) {
	rows, err := r.db.QueryContext(ctx,
		"SELECT point_id, status, edited, checks, checked_at, verified_at, updated_at, published_at FROM grammar_lessons ORDER BY point_id")
	if err != nil {
		return nil, fmt.Errorf("mysql find grammar lessons: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := []grammar.LessonSummary{}
	for rows.Next() {
		var (
			s          grammar.LessonSummary
			status     string
			published  sql.NullTime
			checksRaw  []byte
			checkedAt  sql.NullTime
			verifiedAt sql.NullTime
		)
		if err := rows.Scan(&s.PointID, &status, &s.Edited, &checksRaw, &checkedAt, &verifiedAt, &s.UpdatedAt, &published); err != nil {
			return nil, fmt.Errorf("mysql read grammar lessons: %w", err)
		}
		s.Status, s.UpdatedAt, s.PublishedAt = grammar.Status(status), s.UpdatedAt.UTC(), timeOf(published)
		checks, err := grammarChecksFromJSON(checksRaw)
		if err != nil {
			return nil, err
		}
		for _, c := range checks {
			if !c.Confirmed {
				s.Flags++
			}
		}
		s.Checked, s.Verified = !timeOf(checkedAt).IsZero(), !timeOf(verifiedAt).IsZero()
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mysql read grammar lessons: %w", err)
	}
	return out, nil
}

// --- grammar_progress ---

// GrammarProgress implements grammar.ProgressRepository on "grammar_progress".
type GrammarProgress struct {
	db *sql.DB
}

// NewGrammarProgress returns the grammar progress repository.
func NewGrammarProgress(db *sql.DB) *GrammarProgress { return &GrammarProgress{db: db} }

var _ grammar.ProgressRepository = (*GrammarProgress)(nil)

const grammarProgressCols = `user_id, point_id, status, practice_attempts, last_practice, mastery_attempts,
  best_mastery, mastered_at, weak, updated_at`

type grammarRowScanner interface{ Scan(dest ...any) error }

func grammarProgressScan(s grammarRowScanner) (grammar.Progress, error) {
	var (
		p        grammar.Progress
		status   string
		mastered sql.NullTime
		raw      []byte
	)
	if err := s.Scan(&p.UserID, &p.PointID, &status, &p.PracticeAttempts, &p.LastPractice, &p.MasteryAttempts,
		&p.BestMastery, &mastered, &raw, &p.UpdatedAt); err != nil {
		return grammar.Progress{}, err
	}
	var weak []string
	if err := fromJSON(raw, &weak); err != nil {
		return grammar.Progress{}, err
	}
	p.Status, p.MasteredAt, p.Weak, p.UpdatedAt = grammar.ProgressStatus(status), timeOf(mastered), grammarStrs(weak), p.UpdatedAt.UTC()
	return p, nil
}

// Get returns the learner's record on a point.
func (r *GrammarProgress) Get(ctx context.Context, userID, pointID string) (grammar.Progress, error) {
	p, err := grammarProgressScan(r.db.QueryRowContext(ctx,
		"SELECT "+grammarProgressCols+" FROM grammar_progress WHERE user_id = ? AND point_id = ?", userID, pointID))
	if errors.Is(err, sql.ErrNoRows) {
		return grammar.Progress{}, grammar.ErrNotFound
	}
	if err != nil {
		return grammar.Progress{}, fmt.Errorf("mysql find grammar progress: %w", err)
	}
	return p, nil
}

// List returns every record of the learner, ordered by point.
func (r *GrammarProgress) List(ctx context.Context, userID string) ([]grammar.Progress, error) {
	rows, err := r.db.QueryContext(ctx,
		"SELECT "+grammarProgressCols+" FROM grammar_progress WHERE user_id = ? ORDER BY point_id", userID)
	if err != nil {
		return nil, fmt.Errorf("mysql find grammar progress: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := []grammar.Progress{}
	for rows.Next() {
		p, err := grammarProgressScan(rows)
		if err != nil {
			return nil, fmt.Errorf("mysql read grammar progress: %w", err)
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mysql read grammar progress: %w", err)
	}
	return out, nil
}

// Save inserts or replaces the record of the learner and point.
func (r *GrammarProgress) Save(ctx context.Context, p grammar.Progress) error {
	weak, err := toJSON(grammarStrs(p.Weak))
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO grammar_progress (`+grammarProgressCols+`)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE status = VALUES(status), practice_attempts = VALUES(practice_attempts),
  last_practice = VALUES(last_practice), mastery_attempts = VALUES(mastery_attempts),
  best_mastery = VALUES(best_mastery), mastered_at = VALUES(mastered_at), weak = VALUES(weak),
  updated_at = VALUES(updated_at)`,
		p.UserID, p.PointID, string(p.Status), p.PracticeAttempts, p.LastPractice, p.MasteryAttempts,
		p.BestMastery, nullTime(p.MasteredAt), string(weak), utc(p.UpdatedAt))
	if err != nil {
		return fmt.Errorf("mysql save grammar progress: %w", err)
	}
	return nil
}
