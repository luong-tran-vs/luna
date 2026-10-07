package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/luongtran/luna/backend/internal/topic"
)

// addExtraTopics creates, once, the topics added on 2026-10-07 (topic.LoadExtraTopics) that no
// topic has the name of yet, with their words; a row in schema_migrations marks it done, so a
// topic deleted later is not created again. Names taken meanwhile are skipped.
func addExtraTopics(ctx context.Context, db *sql.DB, log *slog.Logger) error {
	var done int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM schema_migrations WHERE id = ?", topic.ExtraTopicsID).Scan(&done); err != nil {
		return fmt.Errorf("mysql check extra topics: %w", err)
	}
	if done > 0 {
		return nil
	}

	now := time.Now().UTC()
	extra, err := topic.LoadExtraTopics(now)
	if err != nil {
		return err
	}
	rows, err := db.QueryContext(ctx, "SELECT name FROM topics")
	if err != nil {
		return fmt.Errorf("mysql read topic names: %w", err)
	}
	var names []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			_ = rows.Close()
			return fmt.Errorf("mysql read topic names: %w", err)
		}
		names = append(names, n)
	}
	err = rows.Err()
	_ = rows.Close()
	if err != nil {
		return fmt.Errorf("mysql read topic names: %w", err)
	}

	added := 0
	for _, t := range topic.PlanExtraTopics(names, extra) {
		words, err := encodeWords(t.Words)
		if err != nil {
			return err
		}
		_, err = db.ExecContext(ctx,
			"INSERT INTO topics (id, name, name_key, level, description, lesson_ids, roadmaps, words, words_seeded, created_at, updated_at) VALUES (?,?,?,'',?,'[]','{}',?,1,?,?)",
			newID(), t.Name, topic.NameKey(t.Name), t.Description, words, now, now)
		if err != nil {
			if isDuplicate(err) {
				continue
			}
			return fmt.Errorf("mysql insert topic %s: %w", t.Name, err)
		}
		added++
	}
	if _, err := db.ExecContext(ctx, "INSERT INTO schema_migrations (id) VALUES (?)", topic.ExtraTopicsID); err != nil {
		return fmt.Errorf("mysql mark extra topics: %w", err)
	}
	log.InfoContext(ctx, "extra topics added", slog.Int("topics", added))
	return nil
}
