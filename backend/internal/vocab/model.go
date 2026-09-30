// Package vocab is the learner's word notebook: cards saved while reading (F3), added by hand,
// and reviewed on an FSRS schedule (F5).
package vocab

import (
	"errors"
	"fmt"
	"time"
)

// Source says where a card's meaning came from.
type Source string

const (
	SourceAI         Source = "ai"
	SourceDictionary Source = "dictionary"
	SourceManual     Source = "manual"
)

// State is the FSRS learning stage of a card.
type State int

const (
	StateNew State = iota
	StateLearning
	StateReview
	StateRelearning
)

// Schedule is a card's FSRS state. A zero Schedule (cards saved before F5) means a new card
// due at the start of the day after it was saved.
type Schedule struct {
	Due           time.Time
	Stability     float64
	Difficulty    float64
	ElapsedDays   uint64
	ScheduledDays uint64
	Reps          uint64
	Lapses        uint64
	State         State
	LastReview    time.Time
}

// Card is one word or phrase in a learner's notebook. Lemma is unique per user.
// LessonID is empty for cards added by hand.
type Card struct {
	ID              string
	UserID          string
	Text            string
	Lemma           string
	IPA             string
	MeaningVi       string
	ContextSentence string
	LessonID        string
	Source          Source
	CreatedAt       time.Time
	Schedule        Schedule
}

// WordRef identifies a saved word for highlighting.
type WordRef struct {
	Lemma string
	Text  string
}

// Rating is the learner's answer after a card: 1 Again, 2 Hard, 3 Good, 4 Easy.
type Rating int

const (
	Again Rating = iota + 1
	Hard
	Good
	Easy
)

// Mode is how a card was reviewed.
type Mode string

const (
	ModeFlip   Mode = "flip"   // see the word, guess the meaning
	ModeListen Mode = "listen" // hear the word, type it
)

// Context is where a review happened: the daily lesson's review step (counted against the
// daily limit, L) or free review.
type Context string

const (
	ContextDaily Context = "daily"
	ContextFree  Context = "free"
)

// ReviewLog records one review with the schedule the card had before it.
type ReviewLog struct {
	UserID     string
	CardID     string
	Rating     Rating
	Mode       Mode
	Context    Context
	ReviewedAt time.Time
	Before     Schedule
}

// Intervals is the time until the next review for each rating.
type Intervals struct {
	Again, Hard, Good, Easy time.Duration
}

// DueCard is a card to review with the interval each rating would give.
type DueCard struct {
	Card
	Intervals Intervals
}

// DueList is the review queue of a learner.
type DueList struct {
	Cards []DueCard
	Total int
	// NextDue is the earliest due time of a card not yet due; zero when there is none.
	NextDue time.Time
}

// ManualLesson is the lesson filter value for cards added by hand.
const ManualLesson = "manual"

// ListQuery filters the notebook. LessonID may be ManualLesson.
type ListQuery struct {
	Q        string
	LessonID string
	Page     int
}

// DayCard is a card with the day it was saved (YYYY-MM-DD in the learner's timezone).
type DayCard struct {
	Card
	Day string
}

// Page is one page of the notebook, newest first.
type Page struct {
	Cards     []DayCard
	HasMore   bool
	Today     string
	Yesterday string
}

// Details are the fields of a card a learner can edit; nil fields are unchanged.
type Details struct {
	MeaningVi       *string
	IPA             *string
	ContextSentence *string
}

// LessonCount is a lesson that has cards in the notebook.
type LessonCount struct {
	ID    string
	Title string
	Count int
}

// VocabItem is one annotated word or phrase of a lesson, by base form.
type VocabItem struct {
	Lemma         string
	Text          string
	MeaningVi     string
	IPA           string
	SentenceIndex int
	Sentence      string
}

var (
	// ErrExists is returned by repositories when the user already has a card with that lemma.
	ErrExists   = errors.New("vocab: card already exists")
	ErrNotFound = errors.New("vocab: card not found")
	// ErrLessonNotFound is returned when a lesson given by id does not exist.
	ErrLessonNotFound = errors.New("vocab: lesson not found")
)

// ExistsError is returned by Service.Save for a duplicate, with the card already saved.
type ExistsError struct {
	Card Card
}

func (e *ExistsError) Error() string { return fmt.Sprintf("vocab: %q already saved", e.Card.Lemma) }

// ConflictError is returned by Service.Review when the card was reviewed since the client
// loaded it; Card is its current state.
type ConflictError struct {
	Card DueCard
}

func (e *ConflictError) Error() string {
	return fmt.Sprintf("vocab: card %s already reviewed", e.Card.ID)
}

// ValidationError lists invalid input fields with Vietnamese messages.
type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string { return fmt.Sprintf("vocab: invalid input %v", e.Fields) }
