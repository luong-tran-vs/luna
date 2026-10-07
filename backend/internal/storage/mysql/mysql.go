// Package mysql implements storage.Store on MySQL 8 (database/sql with go-sql-driver/mysql).
//
// Conventions every repository here follows, so they can be swapped with the MongoDB ones:
//   - ids are 24-character lowercase hex strings made by newID (the same shape as a Mongo ObjectID),
//     stored in CHAR(24) columns; the domains treat them as opaque. A malformed or unknown id is just
//     "no row", so it reports the domain's own ErrNotFound, never a driver error;
//   - times are DATETIME(6) in UTC (Open forces parseTime=true and loc=UTC);
//   - tables use tableOptions (InnoDB, utf8mb4, binary collation, so comparisons are exact like Mongo's);
//     a case-insensitive match lowercases in SQL, and unique "keys" (email, topic name key) are lowercased
//     by the domain before they reach the repository;
//   - nested documents that are only ever read and written whole are JSON columns; anything a query
//     filters, sorts or must keep unique is a real column;
//   - no foreign keys between domains (each domain's migration stands alone and services own the
//     cross-domain cleanup, as with Mongo); foreign keys inside one domain are fine;
//   - "write if still at version/reps" operations are a single UPDATE ... WHERE and check RowsAffected.
package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net"
	"time"

	drv "github.com/go-sql-driver/mysql"

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
	"github.com/luongtran/luna/backend/internal/wordbank"
	"github.com/luongtran/luna/backend/internal/writing"
)

var _ storage.Store = (*Store)(nil)

// Defaults for the Options left at their zero value.
const (
	// dialTimeout bounds how long a connection attempt waits, so a stopped database fails fast
	// instead of hanging requests (the counterpart of Mongo's server selection timeout).
	dialTimeout = 2 * time.Second
	maxOpen     = 20
	maxIdle     = 5
	maxLifetime = 5 * time.Minute
)

// Options says where the MySQL server is and how the pool behaves. Fill either DSN or the
// Host/Port/User/Password/Database parts; DSN wins when both are given. Zero values take the
// defaults above (host localhost, port 3306).
type Options struct {
	// DSN is user:password@tcp(host:3306)/database, used as it is.
	DSN      string
	Host     string
	Port     string
	User     string
	Password string
	Database string

	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	DialTimeout     time.Duration
}

// driverConfig turns the options into a driver config: parsed from DSN, or built from the parts
// (the driver escapes whatever the password holds when it formats the DSN).
func (o Options) driverConfig() (*drv.Config, error) {
	if o.DSN != "" {
		cfg, err := drv.ParseDSN(o.DSN)
		if err != nil {
			return nil, fmt.Errorf("mysql: invalid MYSQL_DSN: %w", err)
		}
		return cfg, nil
	}
	host, port := o.Host, o.Port
	if host == "" {
		host = "localhost"
	}
	if port == "" {
		port = "3306"
	}
	cfg := drv.NewConfig()
	cfg.User, cfg.Passwd, cfg.DBName = o.User, o.Password, o.Database
	cfg.Net, cfg.Addr = "tcp", net.JoinHostPort(host, port)
	return cfg, nil
}

// Store is the MySQL implementation of storage.Store: one connection pool and a repository for
// each domain on top of it.
type Store struct {
	db *sql.DB
}

// Open prepares a connection pool. It does not contact the server, so the backend starts while the
// database is down; only invalid options are an error, and the message never includes the DSN or
// the password.
func Open(o Options) (*Store, error) {
	cfg, err := o.driverConfig()
	if err != nil {
		return nil, err
	}
	normalize(cfg)
	if o.DialTimeout > 0 {
		cfg.Timeout = o.DialTimeout
	}
	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return nil, fmt.Errorf("mysql: open: %w", err)
	}
	db.SetMaxOpenConns(orInt(o.MaxOpenConns, maxOpen))
	db.SetMaxIdleConns(orInt(o.MaxIdleConns, maxIdle))
	db.SetConnMaxLifetime(orDuration(o.ConnMaxLifetime, maxLifetime))
	return &Store{db: db}, nil
}

func orInt(v, fallback int) int {
	if v > 0 {
		return v
	}
	return fallback
}

func orDuration(v, fallback time.Duration) time.Duration {
	if v > 0 {
		return v
	}
	return fallback
}

// normalize forces the settings the repositories rely on, whatever the DSN says.
func normalize(cfg *drv.Config) *drv.Config {
	cfg.ParseTime = true
	cfg.Loc = time.UTC
	cfg.MultiStatements = false
	// RowsAffected counts the rows an UPDATE matched, not only those whose value changed
	// (Mongo's MatchedCount): "write if still at version" must succeed when it rewrites equal data.
	cfg.ClientFoundRows = true
	cfg.Timeout = dialTimeout
	if cfg.Params == nil {
		cfg.Params = map[string]string{}
	}
	cfg.Params["charset"] = "utf8mb4"
	return cfg
}

// Ping checks that the server answers before ctx expires.
func (s *Store) Ping(ctx context.Context) error {
	if err := s.db.PingContext(ctx); err != nil {
		return fmt.Errorf("mysql ping: %w", err)
	}
	return nil
}

// Prepare creates the tables (applying the pending migrations) and seeds the starting data, in the
// background, retrying every interval until it works or ctx ends.
func (s *Store) Prepare(ctx context.Context, interval time.Duration, log *slog.Logger) {
	go func() {
		for {
			err := migrate(ctx, s.db, migrations())
			if err == nil {
				err = seedTopicWords(ctx, s.db, log)
			}
			if err == nil {
				err = mergeSharedTopics(ctx, s.db, log)
			}
			if err == nil {
				err = addExtraTopics(ctx, s.db, log)
			}
			if err == nil {
				log.InfoContext(ctx, "mysql ready: schema and seed data")
				return
			}
			log.WarnContext(ctx, "mysql not prepared, will retry", slog.Any("error", err))
			select {
			case <-ctx.Done():
				return
			case <-time.After(interval):
			}
		}
	}()
}

// Close closes every connection in the pool.
func (s *Store) Close(context.Context) error {
	if err := s.db.Close(); err != nil {
		return fmt.Errorf("mysql close: %w", err)
	}
	return nil
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
func (s *Store) Writings() writing.Repository                   { return NewWritings(s.db) }
func (s *Store) WordImages() lesson.ImageRepository             { return NewWordImages(s.db) }
func (s *Store) WordBank() wordbank.Repository                  { return NewWordBank(s.db) }
func (s *Store) Cards() vocab.Repository                        { return NewCards(s.db) }
func (s *Store) ReviewLogs() vocab.ReviewLogRepository          { return NewReviewLogs(s.db) }
func (s *Store) DictationResults() progress.DictationRepository { return NewDictationResults(s.db) }
func (s *Store) Goals() progress.GoalRepository                 { return NewGoals(s.db) }
func (s *Store) LessonProgress() progress.ProgressRepository    { return NewLessonProgress(s.db) }
func (s *Store) StudyDays() progress.DayRepository              { return NewStudyDays(s.db) }

func (s *Store) GrammarLessons() grammar.LessonRepository    { return NewGrammarLessons(s.db) }
func (s *Store) GrammarProgress() grammar.ProgressRepository { return NewGrammarProgress(s.db) }

func (s *Store) GrammarReports() grammar.ReportRepository { return NewGrammarReports(s.db) }
