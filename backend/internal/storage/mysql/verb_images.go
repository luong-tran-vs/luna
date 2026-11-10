package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/luongtran/luna/backend/internal/verbs"
)

// VerbImages implements verbs.ImageRepository: the pictures of the irregular verbs (added 2026-11-10).
type VerbImages struct {
	db *sql.DB
}

// NewVerbImages returns the VerbImages repository.
func NewVerbImages(db *sql.DB) *VerbImages { return &VerbImages{db: db} }

var _ verbs.ImageRepository = (*VerbImages)(nil)

// verbImagesSchema is the DDL of the verb pictures (see migrate.go).
func verbImagesSchema() []string {
	return []string{
		`CREATE TABLE IF NOT EXISTS verb_images (
	base       VARCHAR(32)  NOT NULL PRIMARY KEY,
	mime       VARCHAR(64)  NOT NULL,
	data       MEDIUMBLOB   NOT NULL,
	created_at DATETIME(6)  NOT NULL
) ` + tableOptions,
	}
}

// List tells every verb that has a picture, without the pictures.
func (r *VerbImages) List(ctx context.Context) ([]verbs.ImageInfo, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT base, created_at FROM verb_images ORDER BY base`)
	if err != nil {
		return nil, fmt.Errorf("mysql list verb images: %w", err)
	}
	defer rows.Close()
	out := []verbs.ImageInfo{}
	for rows.Next() {
		var i verbs.ImageInfo
		if err := rows.Scan(&i.Base, &i.CreatedAt); err != nil {
			return nil, fmt.Errorf("mysql scan verb image: %w", err)
		}
		i.CreatedAt = i.CreatedAt.UTC()
		out = append(out, i)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mysql list verb images: %w", err)
	}
	return out, nil
}

// Get returns verbs.ErrImageNotFound when the verb has no picture.
func (r *VerbImages) Get(ctx context.Context, base string) (verbs.VerbImage, error) {
	img := verbs.VerbImage{Base: base}
	err := r.db.QueryRowContext(ctx, `SELECT mime, data, created_at FROM verb_images WHERE base = ?`, base).
		Scan(&img.MIME, &img.Data, &img.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return verbs.VerbImage{}, verbs.ErrImageNotFound
	}
	if err != nil {
		return verbs.VerbImage{}, fmt.Errorf("mysql find verb image: %w", err)
	}
	img.CreatedAt = img.CreatedAt.UTC()
	return img, nil
}

// Save stores a picture, replacing the verb's previous one.
func (r *VerbImages) Save(ctx context.Context, img verbs.VerbImage) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO verb_images (base, mime, data, created_at) VALUES (?,?,?,?)
		ON DUPLICATE KEY UPDATE mime = VALUES(mime), data = VALUES(data), created_at = VALUES(created_at)`,
		img.Base, img.MIME, img.Data, utc(img.CreatedAt))
	if err != nil {
		return fmt.Errorf("mysql save verb image: %w", err)
	}
	return nil
}

// Delete removes a verb's picture.
func (r *VerbImages) Delete(ctx context.Context, base string) error {
	if _, err := r.db.ExecContext(ctx, `DELETE FROM verb_images WHERE base = ?`, base); err != nil {
		return fmt.Errorf("mysql delete verb image: %w", err)
	}
	return nil
}
