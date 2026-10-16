package mongo

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/luongtran/luna/backend/internal/aiusage"
	"github.com/luongtran/luna/backend/internal/auth"
	"github.com/luongtran/luna/backend/internal/export"
	"github.com/luongtran/luna/backend/internal/grammar"
	"github.com/luongtran/luna/backend/internal/job"
	"github.com/luongtran/luna/backend/internal/lesson"
	"github.com/luongtran/luna/backend/internal/progress"
	"github.com/luongtran/luna/backend/internal/settings"
	"github.com/luongtran/luna/backend/internal/storage"
	"github.com/luongtran/luna/backend/internal/topic"
	"github.com/luongtran/luna/backend/internal/verbs"
	"github.com/luongtran/luna/backend/internal/vocab"
	"github.com/luongtran/luna/backend/internal/wordbank"
	"github.com/luongtran/luna/backend/internal/writing"
)

var _ storage.Store = (*Store)(nil)

// Store is the MongoDB implementation of storage.Store: one client and one database, and a
// repository for each domain built on that database. The word pictures can live in a database of
// their own (imagesDB), on another cluster (imagesClient), to spare the main one's storage.
type Store struct {
	client       *Client
	db           *mongo.Database
	imagesClient *Client // nil when the pictures use the main client
	imagesDB     *mongo.Database
}

// Open creates a Store for uri and the named database, the word pictures going to imagesURI and
// imagesDatabase (each empty: the main one). Like Connect it does not contact the server; only an
// invalid URI is an error.
func Open(uri, database, imagesURI, imagesDatabase string) (*Store, error) {
	c, err := Connect(uri)
	if err != nil {
		return nil, err
	}
	s := &Store{client: c, db: c.Database(database)}
	if imagesDatabase == "" {
		imagesDatabase = database
	}
	switch {
	case imagesURI != "" && imagesURI != uri:
		ic, err := Connect(imagesURI)
		if err != nil {
			_ = c.Disconnect(context.Background())
			return nil, fmt.Errorf("images: %w", err)
		}
		s.imagesClient, s.imagesDB = ic, ic.Database(imagesDatabase)
	case imagesDatabase != database:
		s.imagesDB = c.Database(imagesDatabase)
	default:
		s.imagesDB = s.db
	}
	return s, nil
}

// Ping checks that the primary of each cluster answers before ctx expires.
func (s *Store) Ping(ctx context.Context) error {
	if err := s.client.Ping(ctx); err != nil {
		return err
	}
	if s.imagesClient != nil {
		if err := s.imagesClient.Ping(ctx); err != nil {
			return fmt.Errorf("images: %w", err)
		}
	}
	return nil
}

// Prepare runs the migrations, the seeding and the index creation in the background.
func (s *Store) Prepare(ctx context.Context, interval time.Duration, log *slog.Logger) {
	PrepareInBackground(ctx, s.db, s.imagesDB, interval, log)
}

// Close closes every connection in the pools.
func (s *Store) Close(ctx context.Context) error {
	err := s.client.Disconnect(ctx)
	if s.imagesClient != nil {
		err = errors.Join(err, s.imagesClient.Disconnect(ctx))
	}
	return err
}

func (s *Store) Users() auth.UserRepository       { return NewUsers(s.db) }
func (s *Store) Sessions() auth.SessionRepository { return NewSessions(s.db) }
func (s *Store) Settings() settings.Repository    { return NewSettings(s.db) }
func (s *Store) Export() export.Repository        { return NewExport(s.db) }
func (s *Store) Jobs() job.Repository             { return NewJobs(s.db) }

func (s *Store) Lessons() storage.LessonStore                   { return NewLessons(s.db) }
func (s *Store) Topics() topic.Repository                       { return NewTopics(s.db) }
func (s *Store) ReadingAnswers() lesson.AnswerRepository        { return NewReadingAnswers(s.db) }
func (s *Store) AILookups() lesson.AskRepository                { return NewAILookups(s.db) }
func (s *Store) AskUsage() lesson.AskUsageRepository            { return NewAskUsage(s.db) }
func (s *Store) Writings() writing.Repository                   { return NewWritings(s.db) }
func (s *Store) WordImages() lesson.ImageRepository             { return NewWordImages(s.imagesDB) }
func (s *Store) WordBank() wordbank.Repository                  { return NewWordBank(s.db) }
func (s *Store) AIUsage() aiusage.Repository                    { return NewAIUsage(s.db) }
func (s *Store) Cards() vocab.Repository                        { return NewCards(s.db) }
func (s *Store) ReviewLogs() vocab.ReviewLogRepository          { return NewReviewLogs(s.db) }
func (s *Store) LessonResults() vocab.LessonResultRepository    { return NewLessonResults(s.db) }
func (s *Store) VerbProgress() verbs.Repository                  { return NewVerbProgress(s.db) }
func (s *Store) DictationResults() progress.DictationRepository { return NewDictationResults(s.db) }
func (s *Store) Goals() progress.GoalRepository                 { return NewGoals(s.db) }
func (s *Store) LessonProgress() progress.ProgressRepository    { return NewLessonProgress(s.db) }
func (s *Store) StudyDays() progress.DayRepository              { return NewStudyDays(s.db) }

func (s *Store) GrammarLessons() grammar.LessonRepository    { return NewGrammarLessons(s.db) }
func (s *Store) GrammarProgress() grammar.ProgressRepository { return NewGrammarProgress(s.db) }

func (s *Store) GrammarReports() grammar.ReportRepository { return NewGrammarReports(s.db) }
