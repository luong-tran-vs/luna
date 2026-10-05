package grammar

import (
	"context"
	"errors"
	"time"
)

// A grammar lesson is the study page of one syllabus point (F20): an explanation, structures,
// examples, common mistakes, practice exercises and a mastery check. The AI writes it once per
// point, an admin reviews and edits it, and only a published lesson is open to learners.

// Status of a grammar lesson.
type Status string

const (
	// StatusDraft is a lesson only admins see.
	StatusDraft Status = "draft"
	// StatusPublished is open to learners.
	StatusPublished Status = "published"
)

// Kind is the form of an exercise.
type Kind string

const (
	// KindChoice is a multiple-choice question: Text is the stem, Options has 4 entries, AnswerIndex is
	// the right one.
	KindChoice Kind = "choice"
	// KindFill is a sentence with one blank written "___" in Text; Answers lists the accepted words
	// (compared ignoring case and surrounding spaces).
	KindFill Kind = "fill"
	// KindReorder asks for an English sentence: Text is its Vietnamese meaning, Sentence is the right
	// answer and Words are the tiles to arrange (the words of Sentence, plus optional distractors).
	KindReorder Kind = "reorder"
)

// Exercise is one question. Only the fields of its Kind are used.
type Exercise struct {
	// ID identifies the exercise within its lesson, e.g. "p3" (practice) or "m2" (mastery check). The
	// service assigns it when the content is saved.
	ID       string
	Kind     Kind
	PromptVi string
	Text     string
	// Options and AnswerIndex are for KindChoice.
	Options     []string
	AnswerIndex int
	// Answers is for KindFill.
	Answers []string
	// Words and Sentence are for KindReorder.
	Words         []string
	Sentence      string
	ExplanationVi string
}

// Structure is one pattern of the point, e.g. the affirmative form.
type Structure struct {
	Label   string
	Pattern string
	Example string
}

// Example is an English sentence with its Vietnamese meaning.
type Example struct {
	En string
	Vi string
}

// Mistake is a common error with the fix.
type Mistake struct {
	Wrong  string
	Right  string
	NoteVi string
}

// Content is everything a grammar lesson teaches.
type Content struct {
	// Objective is one Vietnamese sentence, "Bạn có thể ...".
	Objective string
	// Explanation is a few short Vietnamese paragraphs.
	Explanation []string
	// Usage lists in Vietnamese when the point is used.
	Usage      []string
	Structures []Structure
	Examples   []Example
	Mistakes   []Mistake
	// Practice is done as often as the learner likes, with instant feedback.
	Practice []Exercise
	// Mastery is the check that marks the point as mastered.
	Mastery []Exercise
}

// CheckKind says why an exercise was flagged by the AI check.
type CheckKind string

const (
	// CheckMismatch means a second AI, solving the exercise without the key, got a different answer.
	CheckMismatch CheckKind = "mismatch"
	// CheckAmbiguous means the second AI found more than one right answer, or none.
	CheckAmbiguous CheckKind = "ambiguous"
	// CheckUnchecked means the second AI gave no answer for the exercise, so nobody has checked it.
	CheckUnchecked CheckKind = "unchecked"
)

// Check is one exercise flagged by the AI check for the admin to look at before publishing.
type Check struct {
	ExerciseID string
	Kind       CheckKind
	// NoteVi is a short Vietnamese description of the disagreement.
	NoteVi string
	// Confirmed is true once an admin looked at the flag and kept the exercise as it is.
	Confirmed bool
}

// Lesson is the grammar lesson of a syllabus point; PointID is the syllabus id.
type Lesson struct {
	PointID string
	Status  Status
	Content Content
	// Edited is true once an admin changed what the AI wrote, so a new generation asks first.
	Edited bool
	// Checks are the exercises the AI check flagged; CheckedAt is when the check last ran (zero = never ran or
	// failed). Editing the content clears both, since the flags no longer describe it.
	Checks    []Check
	CheckedAt time.Time
	// VerifiedAt is when an admin confirmed the whole check; editing the whole content clears it.
	VerifiedAt  time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
	PublishedAt time.Time
}

// LessonSummary is a lesson without its content, for lists.
type LessonSummary struct {
	PointID string
	Status  Status
	Edited  bool
	// Flags is the number of exercises flagged by the AI check; Checked is whether the check has run.
	Flags   int
	Checked bool
	// Verified is whether an admin confirmed the check; Flags counts only unconfirmed checks.
	Verified    bool
	UpdatedAt   time.Time
	PublishedAt time.Time
}

// MasteryPercent is the score of a mastery check that marks a point as mastered.
const MasteryPercent = 80

// ProgressStatus is where a learner stands on a point; no progress at all means "new".
type ProgressStatus string

const (
	ProgressLearning ProgressStatus = "learning"
	ProgressMastered ProgressStatus = "mastered"
)

// Progress is one learner's record on one point.
type Progress struct {
	UserID  string
	PointID string
	Status  ProgressStatus
	// PracticeAttempts counts finished practice rounds; LastPractice is the last score in percent.
	PracticeAttempts int
	LastPractice     int
	// MasteryAttempts counts mastery checks; BestMastery is the best score in percent.
	MasteryAttempts int
	BestMastery     int
	MasteredAt      time.Time
	// Weak lists the ids of the exercises answered wrongly in the latest attempt of any kind.
	Weak      []string
	UpdatedAt time.Time
}

// LessonRepository stores grammar lessons, one per syllabus point. Implementations live in
// internal/storage.
type LessonRepository interface {
	// Get returns ErrNotFound when the point has no lesson.
	Get(ctx context.Context, pointID string) (Lesson, error)
	// Save inserts or replaces the whole lesson of l.PointID.
	Save(ctx context.Context, l Lesson) error
	// List returns a summary of every lesson that exists, in no particular order.
	List(ctx context.Context) ([]LessonSummary, error)
}

// ProgressRepository stores learner progress, one record per learner and point.
type ProgressRepository interface {
	// Get returns ErrNotFound when the learner has no record on the point.
	Get(ctx context.Context, userID, pointID string) (Progress, error)
	// List returns every record of the learner.
	List(ctx context.Context, userID string) ([]Progress, error)
	// Save inserts or replaces the record of p.UserID and p.PointID.
	Save(ctx context.Context, p Progress) error
}

// ErrNotFound is returned by the repositories (and the service) for a missing lesson or record.
var ErrNotFound = errors.New("grammar: not found")
