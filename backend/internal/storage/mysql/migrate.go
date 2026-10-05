package mysql

import (
	"context"
	"database/sql"
	"fmt"
)

// migration is a named, ordered group of DDL statements. Statements must be idempotent
// (CREATE TABLE IF NOT EXISTS ...), since MySQL cannot roll back DDL: a run that stops half way is
// simply run again.
type migration struct {
	ID    string
	Stmts []string
}

// migrations lists the schema of every domain, in the order they are applied. To change a table
// after release, add a new migration at the end; never edit one that has shipped.
func migrations() []migration {
	return []migration{
		{ID: "001_users", Stmts: usersSchema()},
		{ID: "002_sessions", Stmts: sessionsSchema()},
		{ID: "003_settings", Stmts: settingsSchema()},
		{ID: "004_jobs", Stmts: jobsSchema()},
		{ID: "005_lessons", Stmts: lessonsSchema()},
		{ID: "006_topics", Stmts: topicsSchema()},
		{ID: "007_reading_answers", Stmts: readingAnswersSchema()},
		{ID: "008_ai_lookups", Stmts: aiLookupsSchema()},
		{ID: "009_writings", Stmts: writingsSchema()},
		{ID: "010_cards", Stmts: cardsSchema()},
		{ID: "011_review_logs", Stmts: reviewLogsSchema()},
		{ID: "012_dictation", Stmts: dictationSchema()},
		{ID: "013_study", Stmts: studySchema()},
		{ID: "014_grammar", Stmts: grammarSchema()},
		{ID: "015_grammar_reports", Stmts: grammarReportsSchema()},
	}
}

// migrateLock names the server-side lock that keeps two backends from migrating at once.
const migrateLock = "luna_schema_migrate"

// migrate applies the migrations not yet recorded in schema_migrations. It holds a named lock on
// one connection for the whole run, so concurrent starts take turns.
func migrate(ctx context.Context, db *sql.DB, list []migration) error {
	conn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("mysql migrate: connect: %w", err)
	}
	defer func() { _ = conn.Close() }()

	var got int
	if err := conn.QueryRowContext(ctx, "SELECT GET_LOCK(?, 30)", migrateLock).Scan(&got); err != nil || got != 1 {
		return fmt.Errorf("mysql migrate: lock (got %d): %w", got, err)
	}
	defer func() {
		_, _ = conn.ExecContext(context.WithoutCancel(ctx), "SELECT RELEASE_LOCK(?)", migrateLock)
	}()

	if _, err := conn.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		id         VARCHAR(64) NOT NULL PRIMARY KEY,
		applied_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6)
	) `+tableOptions); err != nil {
		return fmt.Errorf("mysql migrate: schema_migrations: %w", err)
	}

	for _, m := range list {
		var done int
		if err := conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM schema_migrations WHERE id = ?", m.ID).Scan(&done); err != nil {
			return fmt.Errorf("mysql migrate: check %s: %w", m.ID, err)
		}
		if done > 0 {
			continue
		}
		for i, stmt := range m.Stmts {
			if _, err := conn.ExecContext(ctx, stmt); err != nil {
				return fmt.Errorf("mysql migrate: %s statement %d: %w", m.ID, i+1, err)
			}
		}
		if _, err := conn.ExecContext(ctx, "INSERT INTO schema_migrations (id) VALUES (?)", m.ID); err != nil {
			return fmt.Errorf("mysql migrate: record %s: %w", m.ID, err)
		}
	}
	return nil
}
