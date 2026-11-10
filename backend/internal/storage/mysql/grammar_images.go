package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/luongtran/luna/backend/internal/grammar"
)

// GrammarImages implements grammar.ImageRepository: the pictures of the examples of grammar
// lessons in "grammar_images" (added 2026-11-06).
type GrammarImages struct {
	db *sql.DB
}

// NewGrammarImages returns the GrammarImages repository.
func NewGrammarImages(db *sql.DB) *GrammarImages { return &GrammarImages{db: db} }

var _ grammar.ImageRepository = (*GrammarImages)(nil)

// grammarImagesSchema is the DDL of the example pictures (see migrate.go).
func grammarImagesSchema() []string {
	return []string{
		`CREATE TABLE IF NOT EXISTS grammar_images (
	point_id   VARCHAR(64)  NOT NULL,
	image_key  VARCHAR(32)  NOT NULL,
	mime       VARCHAR(64)  NOT NULL,
	data       MEDIUMBLOB   NOT NULL,
	created_at DATETIME(6)  NOT NULL,
	PRIMARY KEY (point_id, image_key)
) ` + tableOptions,
	}
}

// Keys lists the keys of the point's pictures.
func (r *GrammarImages) Keys(ctx context.Context, pointID string) ([]string, error) {
	out := []string{}
	rows, err := r.db.QueryContext(ctx, `SELECT image_key FROM grammar_images WHERE point_id = ?`, pointID)
	if err != nil {
		return nil, fmt.Errorf("mysql find grammar images: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, fmt.Errorf("mysql read grammar images: %w", err)
		}
		out = append(out, k)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mysql read grammar images: %w", err)
	}
	return out, nil
}

// KeysOf lists the keys of the pictures of every point among pointIDs, in one query.
func (r *GrammarImages) KeysOf(ctx context.Context, pointIDs []string) (map[string][]string, error) {
	out := map[string][]string{}
	if len(pointIDs) == 0 {
		return out, nil
	}
	rows, err := r.db.QueryContext(ctx,
		`SELECT point_id, image_key FROM grammar_images WHERE point_id IN (`+placeholders(len(pointIDs))+`)`,
		args(pointIDs)...)
	if err != nil {
		return nil, fmt.Errorf("mysql find grammar images: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var id, k string
		if err := rows.Scan(&id, &k); err != nil {
			return nil, fmt.Errorf("mysql read grammar images: %w", err)
		}
		out[id] = append(out[id], k)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mysql read grammar images: %w", err)
	}
	return out, nil
}

// Get returns grammar.ErrImageNotFound when the example has no picture.
func (r *GrammarImages) Get(ctx context.Context, pointID, key string) (grammar.ExampleImage, error) {
	img := grammar.ExampleImage{PointID: pointID, Key: key}
	err := r.db.QueryRowContext(ctx,
		`SELECT mime, data, created_at FROM grammar_images WHERE point_id = ? AND image_key = ?`, pointID, key).
		Scan(&img.MIME, &img.Data, &img.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return grammar.ExampleImage{}, grammar.ErrImageNotFound
	}
	if err != nil {
		return grammar.ExampleImage{}, fmt.Errorf("mysql find grammar image: %w", err)
	}
	img.CreatedAt = img.CreatedAt.UTC()
	return img, nil
}

// Save stores a picture, replacing the example's previous one.
func (r *GrammarImages) Save(ctx context.Context, img grammar.ExampleImage) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO grammar_images (point_id, image_key, mime, data, created_at) VALUES (?,?,?,?,?)
		ON DUPLICATE KEY UPDATE mime = VALUES(mime), data = VALUES(data), created_at = VALUES(created_at)`,
		img.PointID, img.Key, img.MIME, img.Data, utc(img.CreatedAt))
	if err != nil {
		return fmt.Errorf("mysql save grammar image: %w", err)
	}
	return nil
}

// Delete removes one picture.
func (r *GrammarImages) Delete(ctx context.Context, pointID, key string) error {
	if _, err := r.db.ExecContext(ctx, `DELETE FROM grammar_images WHERE point_id = ? AND image_key = ?`, pointID, key); err != nil {
		return fmt.Errorf("mysql delete grammar image: %w", err)
	}
	return nil
}

// DeleteForPoint removes every picture of a point.
func (r *GrammarImages) DeleteForPoint(ctx context.Context, pointID string) error {
	if _, err := r.db.ExecContext(ctx, `DELETE FROM grammar_images WHERE point_id = ?`, pointID); err != nil {
		return fmt.Errorf("mysql delete grammar images: %w", err)
	}
	return nil
}
