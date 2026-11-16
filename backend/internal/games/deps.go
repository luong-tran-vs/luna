package games

import (
	"context"
	"time"

	"github.com/luongtran/luna/backend/internal/ai"
)

// BestRepository stores the personal bests, one per learner and game.
type BestRepository interface {
	// List returns every best of the user.
	List(ctx context.Context, userID string) ([]Best, error)
	// Raise stores score when it beats the user's best of game; improved reports whether it did.
	Raise(ctx context.Context, userID string, game Game, score int, at time.Time) (improved bool, err error)
}

// DailyRepository stores the counted result of each day's challenge.
type DailyRepository interface {
	// Get returns ErrNotFound when the user has no result on day.
	Get(ctx context.Context, userID, day string) (DailyResult, error)
	// Save stores the first result of a day; ErrExists when the day already has one.
	Save(ctx context.Context, r DailyResult) error
	// List returns the user's results, newest day first.
	List(ctx context.Context, userID string) ([]DailyResult, error)
}

// SpotRepository stores the extra "Bắn lỗi" sentences.
type SpotRepository interface {
	ListByPoint(ctx context.Context, pointID string) ([]SpotItem, error)
	// Approved returns the approved items of the points (all points when pointIDs is nil).
	Approved(ctx context.Context, pointIDs []string) ([]SpotItem, error)
	// Counts returns the approved and draft items per point.
	Counts(ctx context.Context) (map[string]SpotCount, error)
	// Create stores the items with new ids and returns them.
	Create(ctx context.Context, items []SpotItem) ([]SpotItem, error)
	// Get returns ErrNotFound for an unknown id.
	Get(ctx context.Context, id string) (SpotItem, error)
	// Update replaces the item; ErrNotFound for an unknown id.
	Update(ctx context.Context, it SpotItem) error
	// Delete removes the item; ErrNotFound for an unknown id.
	Delete(ctx context.Context, id string) error
	// ApproveDrafts approves every draft of the points (all points when pointIDs is nil) and
	// returns how many it approved.
	ApproveDrafts(ctx context.Context, pointIDs []string, now time.Time) (int, error)
}

// Card is a word of the learner's notebook as the games see it.
type Card struct {
	Lemma     string
	Text      string
	MeaningVi string
	LessonID  string
	Due       time.Time
}

// Cards gives the learner's cards (over vocab in internal/service).
type Cards interface {
	Cards(ctx context.Context, userID string) ([]Card, error)
}

// Pictures gives the word bank picture URL of lemmas; a lemma without one is left out.
type Pictures interface {
	ImageURLs(ctx context.Context, lemmas []string) (map[string]string, error)
}

// Meanings gives the word bank meaning of lemmas, by lemma as given; unknown ones left out.
type Meanings interface {
	MeaningsOf(ctx context.Context, lemmas []string) (map[string]string, error)
}

// LessonWord is a word of a lesson's vocabulary.
type LessonWord struct {
	Lemma     string
	Text      string
	MeaningVi string
}

// LessonWords gives the annotated words of a lesson; none when it has none or is gone.
type LessonWords interface {
	LessonWords(ctx context.Context, lessonID string) ([]LessonWord, error)
}

// Study gives where the learner stands in the course (over progress in internal/service).
type Study interface {
	// StudiedLessons lists the lessons the learner studied, newest first.
	StudiedLessons(ctx context.Context, userID string) ([]string, error)
	// CurrentLesson is the lesson to study now, "" when none.
	CurrentLesson(ctx context.Context, userID string) (string, error)
	// CanOpen reports whether the user may open the lesson.
	CanOpen(ctx context.Context, userID string, isAdmin bool, lessonID string) (bool, error)
	// Level is the level the learner studies (the active goal), "" when none.
	Level(ctx context.Context, userID string) (string, error)
}

// WordMisses brings missed words back to the review queue; it returns how many cards moved.
type WordMisses interface {
	GameMisses(ctx context.Context, userID string, lemmas []string) (int, error)
}

// Verbs is the irregular verbs practice as the games see it.
type Verbs interface {
	// GameMisses records the verb forms typed wrong in a game.
	GameMisses(ctx context.Context, userID string, misses []VerbMiss) error
	// LearnedBases are the learner's learned verbs: those due now, then the others.
	LearnedBases(ctx context.Context, userID string) (due, other []string, err error)
}

// PointSource is one grammar point with a published lesson: what "Bắn lỗi" builds sentences from.
type PointSource struct {
	PointID string
	Level   string
	TitleVi string
	TitleEn string
	Pattern string
	HintVi  string
	// Examples are right sentences of the point (syllabus examples).
	Examples  []string
	Exercises []SourceExercise
	Mistakes  []PointMistake
}

// SourceExercise is a practice or mastery exercise of a grammar lesson.
type SourceExercise struct {
	ID            string
	Kind          string // "choice", "fix", others are ignored
	Text          string
	Options       []string
	AnswerIndex   int
	Sentence      string
	ExplanationVi string
}

// PointMistake is a common mistake of a point with its fix.
type PointMistake struct {
	Wrong  string
	Right  string
	NoteVi string
}

// Grammar gives the published grammar points and the learner's progress on them.
type Grammar interface {
	PublishedSources(ctx context.Context) ([]PointSource, error)
	// StartedPoints are the points the learner has progress on.
	StartedPoints(ctx context.Context, userID string) ([]string, error)
}

// SpotAI writes extra "Bắn lỗi" sentences (ai.Provider).
type SpotAI interface {
	GrammarSpotItems(ctx context.Context, req ai.GrammarSpotRequest) ([]ai.GrammarSpotItem, error)
}

// StudyDays counts today as a study day for the streak.
type StudyDays interface {
	MarkStudied(ctx context.Context, userID string) error
}

// Timezones gives a learner's timezone.
type Timezones interface {
	Location(ctx context.Context, userID string) (*time.Location, error)
}
