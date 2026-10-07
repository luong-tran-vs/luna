package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/luongtran/luna/backend/internal/topic"
)

// mergeSharedTopics turns the topics of one level into topics shared by every level (2026-10-07):
// topics of the same name are merged (topic.PlanMerge), each old roadmap becoming the roadmap of its
// level, and the lessons, goals and lesson progress of the merged topics move to the kept one. A
// legacy topic is one whose level column is set; once none is left it only makes sure names are
// unique. Each group is merged in one transaction, so a run that stops half way is finished by the
// next start.
func mergeSharedTopics(ctx context.Context, db *sql.DB, log *slog.Logger) error {
	var legacy int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM topics WHERE level <> ''").Scan(&legacy); err != nil {
		return fmt.Errorf("mysql count legacy topics: %w", err)
	}
	if legacy > 0 {
		sources, err := readMergeSources(ctx, db)
		if err != nil {
			return err
		}
		merged, removed := 0, 0
		for _, g := range topic.PlanMerge(sources) {
			if !g.Changed {
				continue
			}
			if err := inTx(ctx, db, func(tx *sql.Tx) error { return applyMerge(ctx, tx, g) }); err != nil {
				return err
			}
			merged++
			removed += len(g.Remove)
		}
		log.InfoContext(ctx, "shared topics migration done", slog.Int("topics", merged), slog.Int("merged away", removed))
	}
	_, err := db.ExecContext(ctx, "ALTER TABLE topics ADD UNIQUE KEY topics_name_key (name_key)")
	if err != nil && !ignorable(err, []uint16{errDuplicateKeyName}) {
		return fmt.Errorf("mysql topics name key: %w", err)
	}
	return nil
}

func readMergeSources(ctx context.Context, db *sql.DB) ([]topic.MergeSource, error) {
	rows, err := db.QueryContext(ctx,
		"SELECT id, name, description, level, lesson_ids, roadmaps, words, words_seeded, created_at FROM topics ORDER BY id")
	if err != nil {
		return nil, fmt.Errorf("mysql read topics to merge: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []topic.MergeSource
	for rows.Next() {
		var (
			s                        topic.MergeSource
			lessons, roadmaps, words []byte
		)
		if err := rows.Scan(&s.ID, &s.Name, &s.Description, &s.Level, &lessons, &roadmaps, &words, &s.WordsSeeded, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("mysql read topics to merge: %w", err)
		}
		if err := fromJSON(lessons, &s.LessonIDs); err != nil {
			return nil, err
		}
		if s.Roadmaps, err = decodeRoadmaps(roadmaps); err != nil {
			return nil, err
		}
		if s.Words, err = decodeWords(words); err != nil {
			return nil, err
		}
		s.CreatedAt = s.CreatedAt.UTC()
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mysql read topics to merge: %w", err)
	}
	return out, nil
}

// applyMerge writes the kept topic, moves what pointed at the merged ones, then deletes them.
func applyMerge(ctx context.Context, tx *sql.Tx, g topic.MergeGroup) error {
	roadmaps, err := encodeRoadmaps(g.Keep.Roadmaps)
	if err != nil {
		return err
	}
	words, err := encodeWords(g.Keep.Words)
	if err != nil {
		return err
	}
	if len(g.Remove) > 0 {
		in := "(" + strings.TrimSuffix(strings.Repeat("?,", len(g.Remove)), ",") + ")"
		args := make([]any, 0, len(g.Remove)+1)
		args = append(args, g.Keep.ID)
		for _, id := range g.Remove {
			args = append(args, id)
		}
		for _, table := range []string{"lessons", "goals", "lesson_progress"} {
			if _, err := tx.ExecContext(ctx, "UPDATE "+table+" SET topic_id = ? WHERE topic_id IN "+in, args...); err != nil {
				return fmt.Errorf("mysql move %s to merged topic: %w", table, err)
			}
		}
		// Deleted before the kept row takes the shared name, which the old rows still hold.
		if _, err := tx.ExecContext(ctx, "DELETE FROM topics WHERE id IN "+in, args[1:]...); err != nil {
			return fmt.Errorf("mysql delete merged topics: %w", err)
		}
	}
	_, err = tx.ExecContext(ctx,
		`UPDATE topics SET name = ?, name_key = ?, description = ?, level = '', lesson_ids = '[]', roadmaps = ?, words = ?,
		 words_seeded = ?, created_at = ?, updated_at = ? WHERE id = ?`,
		g.Keep.Name, topic.NameKey(g.Keep.Name), g.Keep.Description, roadmaps, words, g.Keep.WordsSeeded,
		utc(g.Keep.CreatedAt), utc(time.Now()), g.Keep.ID)
	if err != nil {
		return fmt.Errorf("mysql write merged topic %s: %w", g.Keep.Name, err)
	}
	return nil
}
