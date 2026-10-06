package mongo

import (
	"context"
	"log/slog"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/luongtran/luna/backend/internal/auth"
	"github.com/luongtran/luna/backend/internal/export"
	"github.com/luongtran/luna/backend/internal/grammar"
	"github.com/luongtran/luna/backend/internal/job"
	"github.com/luongtran/luna/backend/internal/lesson"
	"github.com/luongtran/luna/backend/internal/progress"
	"github.com/luongtran/luna/backend/internal/settings"
	"github.com/luongtran/luna/backend/internal/storage"
	"github.com/luongtran/luna/backend/internal/topic"
	"github.com/luongtran/luna/backend/internal/vocab"
	"github.com/luongtran/luna/backend/internal/writing"
)

var _ storage.Store = (*Store)(nil)

// Store is the MongoDB implementation of storage.Store: one client and one database, and a
// repository for each domain built on that database.
type Store struct {
	client *Client
	db     *mongo.Database
}

// Open creates a Store for uri and the named database. Like Connect it does not contact the
// server; only an invalid URI is an error.
func Open(uri, database string) (*Store, error) {
	c, err := Connect(uri)
	if err != nil {
		return nil, err
	}
	return &Store{client: c, db: c.Database(database)}, nil
}

// Ping checks that the primary answers before ctx expires.
func (s *Store) Ping(ctx context.Context) error { return s.client.Ping(ctx) }

// Prepare runs the migrations, the seeding and the index creation in the background.
func (s *Store) Prepare(ctx context.Context, interval time.Duration, log *slog.Logger) {
	PrepareInBackground(ctx, s.db, interval, log)
}

// Close closes every connection in the pool.
func (s *Store) Close(ctx context.Context) error { return s.client.Disconnect(ctx) }

func (s *Store) Users() auth.UserRepository       { return NewUsers(s.db) }
func (s *Store) Sessions() auth.SessionRepository { return NewSessions(s.db) }
func (s *Store) Settings() settings.Repository    { return NewSettings(s.db) }
func (s *Store) Export() export.Repository        { return NewExport(s.db) }
func (s *Store) Jobs() job.Repository             { return NewJobs(s.db) }

func (s *Store) Lessons() storage.LessonStore                   { return NewLessons(s.db) }
func (s *Store) Topics() topic.Repository                       { return NewTopics(s.db) }
func (s *Store) ReadingAnswers() lesson.AnswerRepository        { return NewReadingAnswers(s.db) }
func (s *Store) AILookups() lesson.AskRepository                { return NewAILookups(s.db) }
func (s *Store) Writings() writing.Repository                   { return NewWritings(s.db) }
func (s *Store) WordImages() lesson.ImageRepository             { return NewWordImages(s.db) }
func (s *Store) Cards() vocab.Repository                        { return NewCards(s.db) }
func (s *Store) ReviewLogs() vocab.ReviewLogRepository          { return NewReviewLogs(s.db) }
func (s *Store) DictationResults() progress.DictationRepository { return NewDictationResults(s.db) }
func (s *Store) Goals() progress.GoalRepository                 { return NewGoals(s.db) }
func (s *Store) LessonProgress() progress.ProgressRepository    { return NewLessonProgress(s.db) }
func (s *Store) StudyDays() progress.DayRepository              { return NewStudyDays(s.db) }

func (s *Store) GrammarLessons() grammar.LessonRepository    { return NewGrammarLessons(s.db) }
func (s *Store) GrammarProgress() grammar.ProgressRepository { return NewGrammarProgress(s.db) }

func (s *Store) GrammarReports() grammar.ReportRepository { return NewGrammarReports(s.db) }
