// Package talk is the conversation practice with the AI (added 2026-10-20): from the Ôn tập tab the
// learner picks a topic they studied, the AI invents a situation and talks with them, and at the
// end reviews what they said. The AI only knows the conversation in progress: every reply sends it
// the whole session, and sessions know nothing of each other.
package talk

import (
	"context"
	"errors"
	"time"
)

// MaxTurns is how many lines the learner may say in one session; the AI says goodbye after the last.
const MaxTurns = 20

// MaxTextLength bounds one line of the learner, in characters.
const MaxTextLength = 300

// Role is who said a turn.
type Role string

const (
	RoleAI      Role = "ai"
	RoleLearner Role = "learner"
)

// Status of a session: open while talking, ended once the learner finished it.
type Status string

const (
	StatusOpen  Status = "open"
	StatusEnded Status = "ended"
)

// Turn is one line of the conversation. TextVi is the Vietnamese of the AI's lines; Spoken tells
// a learner's line said into the microphone from a typed one.
type Turn struct {
	Role   Role
	Text   string
	TextVi string
	Spoken bool
	At     time.Time
}

// Good is a learner's line that was well said.
type Good struct {
	Said   string
	NoteVi string
}

// Fix is a learner's line said better, with why.
type Fix struct {
	Said   string
	Better string
	NoteVi string
}

// Review is what the AI says about the learner's lines at the end. WordsUnused are the topic's
// words the learner did not use.
type Review struct {
	OverallVi   string
	Good        []Good
	Fixes       []Fix
	WordsUsed   []string
	WordsUnused []string
}

// Session is one conversation about a topic. Level, Words and Grammar are what the learner had
// studied in the topic when it started, sent with every request.
type Session struct {
	ID         string
	UserID     string
	TopicID    string
	TopicName  string
	Level      string
	Words      []string
	Grammar    []string
	ScenarioVi string
	Turns      []Turn
	// Hint and HintVi are an answer to the AI's last line, for the Gợi ý button.
	Hint      string
	HintVi    string
	Status    Status
	Review    *Review
	CreatedAt time.Time
	UpdatedAt time.Time
	EndedAt   time.Time
}

// LearnerTurns counts the lines the learner said.
func (s Session) LearnerTurns() int {
	n := 0
	for _, t := range s.Turns {
		if t.Role == RoleLearner {
			n++
		}
	}
	return n
}

var (
	// ErrNotFound means no session with that id belongs to the user.
	ErrNotFound = errors.New("talk: session not found")
	// ErrTopicNotStudied means the learner studied no lesson of the topic.
	ErrTopicNotStudied = errors.New("talk: topic not studied")
	// ErrEnded means the session is over: nothing more can be said.
	ErrEnded = errors.New("talk: session ended")
	// ErrTurnLimit means the learner already said MaxTurns lines.
	ErrTurnLimit = errors.New("talk: turn limit reached")
	// ErrUnusableReply means the AI's answer could not be used.
	ErrUnusableReply = errors.New("talk: unusable AI reply")
)

// ValidationError lists invalid input fields with Vietnamese messages.
type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string { return "talk: invalid input" }

// Repository stores the sessions. Implementations live in internal/storage.
type Repository interface {
	// Create stores a new session and returns it with its id.
	Create(ctx context.Context, s Session) (Session, error)
	// Get returns the session; ErrNotFound when it does not exist.
	Get(ctx context.Context, id string) (Session, error)
	// Save replaces what changes during a session: turns, hint, status, review, updatedAt, endedAt.
	Save(ctx context.Context, s Session) error
	// List returns the user's sessions, newest first, at most limit.
	List(ctx context.Context, userID string, limit int) ([]Session, error)
}

// StudiedLesson is a lesson the learner studied, with what the conversation needs from it.
type StudiedLesson struct {
	ID        string
	Title     string
	Level     string
	TopicID   string
	TopicName string
}

// Material is what the learner studied in some lessons: their words and grammar points.
type Material struct {
	Words   []string
	Grammar []string
}

// Studies tells which lessons the learner studied (implemented over progress and lessons in
// internal/service).
type Studies interface {
	// Studied lists the lessons the learner completed or is studying, newest first.
	Studied(ctx context.Context, userID string) ([]StudiedLesson, error)
	// Material gathers the words and grammar points of the lessons.
	Material(ctx context.Context, lessonIDs []string) (Material, error)
}
