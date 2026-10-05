package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/luongtran/luna/backend/internal/topic"
)

// Topics implements topic.Repository on the "topics" table; the roadmap is the lesson_ids JSON array
// of each topic and the word list is the words JSON array (NULL until first set).
type Topics struct {
	db *sql.DB
}

// NewTopics returns the Topics repository.
func NewTopics(db *sql.DB) *Topics { return &Topics{db: db} }

var _ topic.Repository = (*Topics)(nil)

// topicsSchema is the DDL of this domain (see migrate.go).
func topicsSchema() []string {
	return []string{`CREATE TABLE IF NOT EXISTS topics (
		id           CHAR(24)     NOT NULL PRIMARY KEY,
		name         VARCHAR(255) NOT NULL,
		name_key     VARCHAR(191) NOT NULL,
		level        VARCHAR(16)  NOT NULL,
		description  TEXT         NOT NULL,
		lesson_ids   JSON         NOT NULL,
		words        JSON         NULL,
		words_seeded TINYINT(1)   NOT NULL DEFAULT 0,
		created_at   DATETIME(6)  NOT NULL,
		updated_at   DATETIME(6)  NOT NULL,
		UNIQUE KEY topics_level_name_key (level, name_key)
	) ` + tableOptions}
}

const topicCols = "id, name, level, description, lesson_ids, words, words_seeded, created_at, updated_at"

type topicScanner interface{ Scan(dest ...any) error }

func scanTopicRow(s topicScanner) (topic.Topic, error) {
	var (
		t               topic.Topic
		lessons, words  []byte
		created, update time.Time
	)
	if err := s.Scan(&t.ID, &t.Name, &t.Level, &t.Description, &lessons, &words, &t.WordsSeeded, &created, &update); err != nil {
		return topic.Topic{}, err
	}
	t.LessonIDs = []string{}
	if err := fromJSON(lessons, &t.LessonIDs); err != nil {
		return topic.Topic{}, err
	}
	if t.LessonIDs == nil {
		t.LessonIDs = []string{}
	}
	if err := fromJSON(words, &t.Words); err != nil {
		return topic.Topic{}, err
	}
	t.CreatedAt, t.UpdatedAt = created.UTC(), update.UTC()
	return t, nil
}

func topicGet(ctx context.Context, q execer, id string, forUpdate bool) (topic.Topic, error) {
	query := "SELECT " + topicCols + " FROM topics WHERE id = ?"
	if forUpdate {
		query += " FOR UPDATE"
	}
	t, err := scanTopicRow(q.QueryRowContext(ctx, query, id))
	if errors.Is(err, sql.ErrNoRows) {
		return topic.Topic{}, topic.ErrNotFound
	}
	if err != nil {
		return topic.Topic{}, fmt.Errorf("mysql find topic: %w", err)
	}
	return t, nil
}

// Create inserts t; the unique (level, name_key) key turns duplicates into ErrNameTaken.
func (r *Topics) Create(ctx context.Context, t topic.Topic) (topic.Topic, error) {
	ids := t.LessonIDs
	if ids == nil {
		ids = []string{}
	}
	raw, err := toJSON(ids)
	if err != nil {
		return topic.Topic{}, err
	}
	id := newID()
	_, err = r.db.ExecContext(ctx,
		"INSERT INTO topics (id, name, name_key, level, description, lesson_ids, words_seeded, created_at, updated_at) VALUES (?,?,?,?,?,?,0,?,?)",
		id, t.Name, topic.NameKey(t.Name), t.Level, t.Description, raw, utc(t.CreatedAt), utc(t.UpdatedAt))
	if err != nil {
		if isDuplicate(err) {
			return topic.Topic{}, topic.ErrNameTaken
		}
		return topic.Topic{}, fmt.Errorf("mysql insert topic: %w", err)
	}
	return topic.Topic{
		ID: id, Name: t.Name, Level: t.Level, Description: t.Description, LessonIDs: slices.Clone(ids),
		CreatedAt: utc(t.CreatedAt), UpdatedAt: utc(t.UpdatedAt),
	}, nil
}

// Get returns topic.ErrNotFound for unknown or malformed ids.
func (r *Topics) Get(ctx context.Context, id string) (topic.Topic, error) {
	return topicGet(ctx, r.db, id, false)
}

// List returns the topics of level ("" = all); the service sorts them. Ordered by id for stability.
func (r *Topics) List(ctx context.Context, level string) ([]topic.Topic, error) {
	query := "SELECT " + topicCols + " FROM topics"
	var a []any
	if level != "" {
		query += " WHERE level = ?"
		a = append(a, level)
	}
	query += " ORDER BY id"
	rows, err := r.db.QueryContext(ctx, query, a...)
	if err != nil {
		return nil, fmt.Errorf("mysql find topics: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []topic.Topic
	for rows.Next() {
		t, err := scanTopicRow(rows)
		if err != nil {
			return nil, fmt.Errorf("mysql read topics: %w", err)
		}
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mysql read topics: %w", err)
	}
	return out, nil
}

// Update changes name, level and description.
func (r *Topics) Update(ctx context.Context, id string, in topic.Input) (topic.Topic, error) {
	var out topic.Topic
	err := inTx(ctx, r.db, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx,
			"UPDATE topics SET name = ?, name_key = ?, level = ?, description = ?, updated_at = ? WHERE id = ?",
			in.Name, topic.NameKey(in.Name), in.Level, in.Description, utc(time.Now()), id)
		if err != nil {
			if isDuplicate(err) {
				return topic.ErrNameTaken
			}
			return fmt.Errorf("mysql update topic: %w", err)
		}
		ok, err := changed(res)
		if err != nil {
			return err
		}
		if !ok {
			return topic.ErrNotFound
		}
		out, err = topicGet(ctx, tx, id, false)
		return err
	})
	if err != nil {
		return topic.Topic{}, err
	}
	return out, nil
}

// Delete removes a topic (no error when it is already gone).
func (r *Topics) Delete(ctx context.Context, id string) error {
	if _, err := r.db.ExecContext(ctx, "DELETE FROM topics WHERE id = ?", id); err != nil {
		return fmt.Errorf("mysql delete topic: %w", err)
	}
	return nil
}

// SetLessons replaces the roadmap.
func (r *Topics) SetLessons(ctx context.Context, id string, lessonIDs []string) error {
	if lessonIDs == nil {
		lessonIDs = []string{}
	}
	raw, err := toJSON(lessonIDs)
	if err != nil {
		return err
	}
	res, err := r.db.ExecContext(ctx, "UPDATE topics SET lesson_ids = ?, updated_at = ? WHERE id = ?", raw, utc(time.Now()), id)
	if err != nil {
		return fmt.Errorf("mysql set roadmap: %w", err)
	}
	ok, err := changed(res)
	if err != nil {
		return err
	}
	if !ok {
		return topic.ErrNotFound
	}
	return nil
}

// topicRoadmapEdit reads the roadmap under a row lock, lets edit change it and writes it back when
// edit says so; it reports whether the topic exists and edit changed it.
func (r *Topics) topicRoadmapEdit(ctx context.Context, id string, edit func(ids []string) ([]string, bool)) (found, wrote bool, err error) {
	err = inTx(ctx, r.db, func(tx *sql.Tx) error {
		var raw []byte
		if err := tx.QueryRowContext(ctx, "SELECT lesson_ids FROM topics WHERE id = ? FOR UPDATE", id).Scan(&raw); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil
			}
			return fmt.Errorf("mysql read roadmap: %w", err)
		}
		found = true
		var ids []string
		if err := fromJSON(raw, &ids); err != nil {
			return err
		}
		next, write := edit(ids)
		if !write {
			return nil
		}
		if next == nil {
			next = []string{}
		}
		b, err := toJSON(next)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, "UPDATE topics SET lesson_ids = ?, updated_at = ? WHERE id = ?", b, utc(time.Now()), id); err != nil {
			return fmt.Errorf("mysql write roadmap: %w", err)
		}
		wrote = true
		return nil
	})
	return found, wrote, err
}

// RemoveLesson pulls a lesson out of the roadmap; removed is false when it was not there.
func (r *Topics) RemoveLesson(ctx context.Context, id, lessonID string) (bool, error) {
	_, wrote, err := r.topicRoadmapEdit(ctx, id, func(ids []string) ([]string, bool) {
		if !slices.Contains(ids, lessonID) {
			return nil, false
		}
		return slices.DeleteFunc(ids, func(l string) bool { return l == lessonID }), true
	})
	return wrote, err
}

// AppendLesson pushes a lesson at the end of the roadmap unless it is already there. Like Mongo's
// update with a filter, an unknown topic is not an error.
func (r *Topics) AppendLesson(ctx context.Context, id, lessonID string) error {
	_, _, err := r.topicRoadmapEdit(ctx, id, func(ids []string) ([]string, bool) {
		if slices.Contains(ids, lessonID) {
			return nil, false
		}
		return append(ids, lessonID), true
	})
	return err
}

// SetWords replaces the topic's words and marks them seeded, so the startup seed never
// overwrites an admin edit (F18).
func (r *Topics) SetWords(ctx context.Context, id string, words []string) (topic.Topic, error) {
	if words == nil {
		words = []string{}
	}
	raw, err := toJSON(words)
	if err != nil {
		return topic.Topic{}, err
	}
	var out topic.Topic
	err = inTx(ctx, r.db, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, "UPDATE topics SET words = ?, words_seeded = 1, updated_at = ? WHERE id = ?", raw, utc(time.Now()), id)
		if err != nil {
			return fmt.Errorf("mysql set topic words: %w", err)
		}
		ok, err := changed(res)
		if err != nil {
			return err
		}
		if !ok {
			return topic.ErrNotFound
		}
		out, err = topicGet(ctx, tx, id, false)
		return err
	})
	if err != nil {
		return topic.Topic{}, err
	}
	return out, nil
}

// SeedTopicWords gives every topic not seeded yet its starting vocabulary (F18): the seed list
// of the same name, or an empty list. Each write only applies while the topic is still not
// seeded, so a restart or a concurrent run never overwrites a list, even one an admin emptied.
func SeedTopicWords(ctx context.Context, db *sql.DB, seed topic.Seed, log *slog.Logger) error {
	rows, err := db.QueryContext(ctx, "SELECT id, name FROM topics WHERE words_seeded = 0 ORDER BY id")
	if err != nil {
		return fmt.Errorf("mysql find topics to seed: %w", err)
	}
	var targets []topic.SeedTarget
	for rows.Next() {
		var t topic.SeedTarget
		if err := rows.Scan(&t.ID, &t.Name); err != nil {
			_ = rows.Close()
			return fmt.Errorf("mysql decode topics to seed: %w", err)
		}
		targets = append(targets, t)
	}
	err = rows.Err()
	_ = rows.Close()
	if err != nil {
		return fmt.Errorf("mysql decode topics to seed: %w", err)
	}
	if len(targets) == 0 {
		return nil
	}

	matched, unmatched := 0, 0
	for _, w := range topic.PlanSeed(targets, seed) {
		words := w.Words
		if words == nil {
			words = []string{}
		}
		raw, err := toJSON(words)
		if err != nil {
			return err
		}
		res, err := db.ExecContext(ctx, "UPDATE topics SET words = ?, words_seeded = 1 WHERE id = ? AND words_seeded = 0", raw, w.ID)
		if err != nil {
			return fmt.Errorf("mysql seed topic words: %w", err)
		}
		ok, err := changed(res)
		if err != nil {
			return err
		}
		switch {
		case !ok: // seeded meanwhile
		case w.Matched:
			matched++
		default:
			unmatched++
		}
	}
	log.InfoContext(ctx, "topic words seeded", slog.Int("matched", matched), slog.Int("unmatched", unmatched))
	return nil
}

// seedTopicWords runs SeedTopicWords with the embedded seed.
func seedTopicWords(ctx context.Context, db *sql.DB, log *slog.Logger) error {
	seed, err := topic.LoadSeed()
	if err != nil {
		return err
	}
	return SeedTopicWords(ctx, db, seed, log)
}
