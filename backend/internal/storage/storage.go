// Package storage is where the database is chosen. A Store is an abstract factory: it hands out
// the repositories of every domain for one kind of database (MongoDB, MySQL...), so the services
// never know which one is behind them. A Registry maps a driver name, from DB_DRIVER, to the
// function that opens that kind of Store.
//
// To add a database: write a package under internal/storage/<name> with a type that implements
// Store, and register it in internal/storage/factory. Nothing in the domains changes.
package storage

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/luongtran/luna/backend/internal/auth"
	"github.com/luongtran/luna/backend/internal/export"
	"github.com/luongtran/luna/backend/internal/grammar"
	"github.com/luongtran/luna/backend/internal/job"
	"github.com/luongtran/luna/backend/internal/lesson"
	"github.com/luongtran/luna/backend/internal/progress"
	"github.com/luongtran/luna/backend/internal/settings"
	"github.com/luongtran/luna/backend/internal/topic"
	"github.com/luongtran/luna/backend/internal/vocab"
	"github.com/luongtran/luna/backend/internal/writing"
)

// LessonStore is the lesson repository plus the lesson queries the topic domain needs
// (topic.Lessons); both are answered from the same lessons data.
type LessonStore interface {
	lesson.Repository
	CountByTopic(ctx context.Context) (map[string]map[string]int, error)
	PlaceOf(ctx context.Context, ids []string) (map[string]topic.Place, error)
	TopicTexts(ctx context.Context, topicIDs []string) (map[string][]topic.LessonText, error)
}

// Store is one opened database with all the repositories of the app. Each call returns a
// repository bound to that database; implementations may build it on each call, so keep the result
// instead of calling again in a loop.
type Store interface {
	// Ping checks that the database answers before ctx expires (GET /api/health).
	Ping(ctx context.Context) error
	// Prepare creates what the database needs before use (indexes, tables), runs the one-time data
	// migrations and the seeding, in the background: it returns at once and retries every interval
	// while the database is unreachable.
	Prepare(ctx context.Context, interval time.Duration, log *slog.Logger)
	// Close releases every connection.
	Close(ctx context.Context) error

	Users() auth.UserRepository
	Sessions() auth.SessionRepository
	Settings() settings.Repository
	Export() export.Repository
	Jobs() job.Repository

	Lessons() LessonStore
	Topics() topic.Repository
	ReadingAnswers() lesson.AnswerRepository
	AILookups() lesson.AskRepository
	Writings() writing.Repository
	// WordImages keeps the picture settings of lessons and their word pictures (F23).
	WordImages() lesson.ImageRepository

	Cards() vocab.Repository
	ReviewLogs() vocab.ReviewLogRepository

	DictationResults() progress.DictationRepository
	Goals() progress.GoalRepository
	LessonProgress() progress.ProgressRepository
	StudyDays() progress.DayRepository

	GrammarLessons() grammar.LessonRepository
	GrammarProgress() grammar.ProgressRepository
	GrammarReports() grammar.ReportRepository
}

// Opener opens a Store. It captures its own settings (a URI, a DSN) when it is registered, so the
// registry knows nothing about any one database. It should not block on the network: the backend
// starts while the database is still down.
type Opener func() (Store, error)

// ErrUnknownDriver is returned by Registry.Open for a name nobody registered.
var ErrUnknownDriver = errors.New("storage: unknown database driver")

// Registry maps driver names to Openers. Build one in main and fill it explicitly; there is no
// global registry, so a test can make its own.
type Registry struct {
	openers map[string]Opener
}

// NewRegistry returns an empty Registry.
func NewRegistry() *Registry {
	return &Registry{openers: map[string]Opener{}}
}

// Register adds a driver. The name is case-insensitive. It panics on an empty name, a nil Opener
// or a name registered twice, since these are mistakes in the program, not in the environment.
func (r *Registry) Register(name string, open Opener) {
	key := normalize(name)
	if key == "" || open == nil {
		panic("storage: Register needs a name and an Opener")
	}
	if _, dup := r.openers[key]; dup {
		panic("storage: driver registered twice: " + key)
	}
	r.openers[key] = open
}

// Drivers lists the registered names, sorted.
func (r *Registry) Drivers() []string {
	names := make([]string, 0, len(r.openers))
	for n := range r.openers {
		names = append(names, n)
	}
	slices.Sort(names)
	return names
}

// Open opens the Store of the named driver. An unknown name is ErrUnknownDriver and the message
// lists the known ones.
func (r *Registry) Open(name string) (Store, error) {
	open, ok := r.openers[normalize(name)]
	if !ok {
		return nil, fmt.Errorf("%w %q (known: %s)", ErrUnknownDriver, name, strings.Join(r.Drivers(), ", "))
	}
	s, err := open()
	if err != nil {
		return nil, fmt.Errorf("storage: open %s: %w", normalize(name), err)
	}
	return s, nil
}

func normalize(name string) string { return strings.ToLower(strings.TrimSpace(name)) }
