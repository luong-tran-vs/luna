// Package games is the learner's "Trò chơi" (F26): short rounds that go over what the learner has
// already studied. A round is built from data that exists (cards, word bank pictures, lesson
// words, published grammar lessons, learned verbs); the page scores it and sends the result, which
// raises the personal best and brings what was missed back to the review queue. Only the daily
// challenge keeps one result a day. No AI is called while a learner plays; the admin may ask the AI
// for more "Bắn lỗi" sentences (spot_admin.go).
package games

import (
	"errors"
	"fmt"
	"time"
)

// Game names one game.
type Game string

const (
	GamePicture Game = "picture" // Ảnh nhanh tay (P1)
	GameVerbs   Game = "verbs"   // Đấu động từ (P2)
	GameLetters Game = "letters" // Chữ rơi (P3)
	GameMemory  Game = "memory"  // Lật thẻ ghi nhớ (P4)
	GameGrammar Game = "grammar" // Bắn lỗi ngữ pháp (P5), members only
	GameDaily   Game = "daily"   // Thử thách hằng ngày (P6)
)

// MaxScore is the highest score each game can give; a higher one is refused.
var MaxScore = map[Game]int{
	GamePicture: 30000,
	GameVerbs:   500,
	GameLetters: 20000,
	GameMemory:  10000,
	GameGrammar: 5000,
	GameDaily:   3000,
}

// released are the games shown to learners, in the order of the list.
var released = []Game{GamePicture, GameVerbs, GameLetters, GameMemory, GameGrammar}

// Released reports whether g is a game learners can play (the daily challenge included).
func Released(g Game) bool {
	if g == GameDaily {
		return true
	}
	for _, r := range released {
		if r == g {
			return true
		}
	}
	return false
}

// Limits of one result.
const (
	MaxWrongWords = 60
	MaxWrongVerbs = 50
	DailyCount    = 10
)

var (
	// ErrNotFound: no such record (a spot item, a daily result, a grammar point without a published lesson).
	ErrNotFound = errors.New("games: not found")
	// ErrExists: the record already exists (a daily result of that day).
	ErrExists = errors.New("games: already exists")
	// ErrAlreadyDone: today's challenge was already counted.
	ErrAlreadyDone = errors.New("games: daily challenge already done")
	// ErrMembersOnly: the game is for members (guests cannot play "Bắn lỗi").
	ErrMembersOnly = errors.New("games: members only")
	// ErrLessonLocked: the learner cannot open the lesson asked for.
	ErrLessonLocked = errors.New("games: lesson locked")
)

// NotEnoughError means there is too little content for the round; Code is "not_enough_words" or
// "not_enough_sentences".
type NotEnoughError struct{ Code string }

func (e *NotEnoughError) Error() string { return "games: " + e.Code }

// ErrNotEnough matches any NotEnoughError with errors.Is.
var ErrNotEnough = errors.New("games: not enough content")

// Is makes errors.Is(err, ErrNotEnough) true.
func (e *NotEnoughError) Is(target error) bool { return target == ErrNotEnough }

func notEnoughWords() error     { return &NotEnoughError{Code: "not_enough_words"} }
func notEnoughSentences() error { return &NotEnoughError{Code: "not_enough_sentences"} }

// ValidationError lists invalid input fields with Vietnamese messages.
type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string { return fmt.Sprintf("games: invalid input %v", e.Fields) }

func invalid(field, msg string) error {
	return &ValidationError{Fields: map[string]string{field: msg}}
}

// Best is a learner's personal best of one game.
type Best struct {
	UserID string
	Game   Game
	Score  int
	At     time.Time
}

// DailyAnswer is one answer of a daily challenge: its kind and whether it was right.
type DailyAnswer struct {
	Kind  string
	Right bool
}

// DailyResult is the counted result of one day's challenge.
type DailyResult struct {
	UserID     string
	Day        string // YYYY-MM-DD in the learner's timezone
	Score      int
	Answers    []DailyAnswer
	FinishedAt time.Time
}

// Daily question kinds.
const (
	KindPicture = "picture"
	KindVerb    = "verb"
	KindLetters = "letters"
	KindGrammar = "grammar"
)

func validKind(k string) bool {
	return k == KindPicture || k == KindVerb || k == KindLetters || k == KindGrammar
}

// SpotStatus is the state of an extra "Bắn lỗi" sentence.
type SpotStatus string

const (
	SpotDraft    SpotStatus = "draft"
	SpotApproved SpotStatus = "approved"
)

// SpotItem is a "Bắn lỗi" sentence written for a grammar point (by the AI, checked by an admin):
// Sentence has one mistake, Wrong, to be replaced by Right; Distractors are two other wrong fixes.
type SpotItem struct {
	ID          string
	PointID     string
	Level       string
	Sentence    string
	Wrong       string
	Right       string
	Distractors []string
	RuleVi      string
	Status      SpotStatus
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// SpotCount is how many extra sentences a point has, approved and draft.
type SpotCount struct {
	Approved int
	Draft    int
}

// PictureQuestion is one picture and four words; Answer is the index of the right one.
type PictureQuestion struct {
	Lemma     string
	ImageURL  string
	MeaningVi string
	Options   []string
	Answer    int
}

// LetterWord is a word to spell from falling (or shuffled) letters; ImageURL may be "".
type LetterWord struct {
	Lemma     string
	MeaningVi string
	ImageURL  string
}

// MemoryPair is an English word and its picture, or its Vietnamese meaning when it has none.
type MemoryPair struct {
	Lemma     string
	ImageURL  string
	MeaningVi string
}

// Span is the token range [From, To).
type Span struct{ From, To int }

// SpotQuestion is one "Bắn lỗi" sentence as tokens. Wrong is the range of the mistake, nil when the
// sentence is right; Options are the three fixes and Answer the right one.
type SpotQuestion struct {
	Tokens  []string
	Wrong   *Span
	Options []string
	Answer  int
	RuleVi  string
	TopicVi string
	PointID string
	Source  string // "choice" | "fix" | "mistake" | "spot"
}

// DailyVerb is a verb question of the daily challenge; Base "" lets the page pick a common verb
// with the day's seed.
type DailyVerb struct{ Base string }

// DailyQuestion is one question of the day's challenge; exactly one of the pointers is set.
type DailyQuestion struct {
	Kind    string
	Picture *PictureQuestion
	Verb    *DailyVerb
	Letters *LetterWord
	Grammar *SpotQuestion
}

// VerbMiss is a verb form typed wrong in a game.
type VerbMiss struct {
	Base  string
	Form  string // "past" | "participle"
	Typed string
}
