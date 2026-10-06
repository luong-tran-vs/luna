package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/luongtran/luna/backend/internal/lesson"
)

// WordImages implements lesson.ImageRepository (F23): the picture settings of each lesson in
// "lesson_image_settings" and the pictures, as binary data, in "word_images".
type WordImages struct {
	db *sql.DB
}

// NewWordImages returns the WordImages repository.
func NewWordImages(db *sql.DB) *WordImages { return &WordImages{db: db} }

var _ lesson.ImageRepository = (*WordImages)(nil)

// wordImagesSchema is the DDL of this domain (see migrate.go). A picture is a JPEG of tens of
// kilobytes; MEDIUMBLOB (16 MB) leaves room for any other format.
func wordImagesSchema() []string {
	return []string{
		`CREATE TABLE IF NOT EXISTS lesson_image_settings (
	lesson_id  CHAR(24)     NOT NULL PRIMARY KEY,
	enabled    TINYINT(1)   NOT NULL,
	style      TEXT         NOT NULL,
	status     VARCHAR(16)  NOT NULL,
	error      TEXT         NOT NULL,
	updated_at DATETIME(6)  NULL
) ` + tableOptions,
		`CREATE TABLE IF NOT EXISTS word_images (
	lesson_id  CHAR(24)     NOT NULL,
	lemma      VARCHAR(191) NOT NULL,
	mime       VARCHAR(64)  NOT NULL,
	data       MEDIUMBLOB   NOT NULL,
	created_at DATETIME(6)  NOT NULL,
	PRIMARY KEY (lesson_id, lemma)
) ` + tableOptions,
	}
}

// Settings returns the zero settings for a lesson without any (or a malformed id).
func (r *WordImages) Settings(ctx context.Context, lessonID string) (lesson.ImageSettings, error) {
	if !lessonValidID(lessonID) {
		return lesson.ImageSettings{}, nil
	}
	var (
		s       lesson.ImageSettings
		status  string
		updated sql.NullTime
	)
	err := r.db.QueryRowContext(ctx,
		`SELECT enabled, style, status, error, updated_at FROM lesson_image_settings WHERE lesson_id = ?`, lessonID).
		Scan(&s.Enabled, &s.Style, &status, &s.Error, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return lesson.ImageSettings{}, nil
	}
	if err != nil {
		return lesson.ImageSettings{}, fmt.Errorf("mysql find image settings: %w", err)
	}
	s.Status, s.UpdatedAt = lesson.Status(status), timeOf(updated)
	return s, nil
}

// SaveSettings replaces the settings of a lesson.
func (r *WordImages) SaveSettings(ctx context.Context, lessonID string, s lesson.ImageSettings) error {
	if !lessonValidID(lessonID) {
		return lesson.ErrNotFound
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO lesson_image_settings (lesson_id, enabled, style, status, error, updated_at)
		VALUES (?,?,?,?,?,?)
		ON DUPLICATE KEY UPDATE enabled = VALUES(enabled), style = VALUES(style), status = VALUES(status),
			error = VALUES(error), updated_at = VALUES(updated_at)`,
		lessonID, s.Enabled, s.Style, string(s.Status), s.Error, nullTime(s.UpdatedAt))
	if err != nil {
		return fmt.Errorf("mysql save image settings: %w", err)
	}
	return nil
}

// Lemmas lists the words of a lesson that have a picture.
func (r *WordImages) Lemmas(ctx context.Context, lessonID string) ([]string, error) {
	out := []string{}
	if !lessonValidID(lessonID) {
		return out, nil
	}
	rows, err := r.db.QueryContext(ctx, `SELECT lemma FROM word_images WHERE lesson_id = ?`, lessonID)
	if err != nil {
		return nil, fmt.Errorf("mysql find word images: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var lemma string
		if err := rows.Scan(&lemma); err != nil {
			return nil, fmt.Errorf("mysql read word images: %w", err)
		}
		out = append(out, lemma)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mysql read word images: %w", err)
	}
	return out, nil
}

// Get returns lesson.ErrImageNotFound when the word has no picture.
func (r *WordImages) Get(ctx context.Context, lessonID, lemma string) (lesson.WordImage, error) {
	if !lessonValidID(lessonID) {
		return lesson.WordImage{}, lesson.ErrImageNotFound
	}
	img := lesson.WordImage{LessonID: lessonID, Lemma: lemma}
	err := r.db.QueryRowContext(ctx,
		`SELECT mime, data, created_at FROM word_images WHERE lesson_id = ? AND lemma = ?`, lessonID, lemma).
		Scan(&img.MIME, &img.Data, &img.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return lesson.WordImage{}, lesson.ErrImageNotFound
	}
	if err != nil {
		return lesson.WordImage{}, fmt.Errorf("mysql find word image: %w", err)
	}
	img.CreatedAt = img.CreatedAt.UTC()
	return img, nil
}

// Save stores a picture, replacing the word's previous one.
func (r *WordImages) Save(ctx context.Context, img lesson.WordImage) error {
	if !lessonValidID(img.LessonID) {
		return lesson.ErrNotFound
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO word_images (lesson_id, lemma, mime, data, created_at) VALUES (?,?,?,?,?)
		ON DUPLICATE KEY UPDATE mime = VALUES(mime), data = VALUES(data), created_at = VALUES(created_at)`,
		img.LessonID, img.Lemma, img.MIME, img.Data, utc(img.CreatedAt))
	if err != nil {
		return fmt.Errorf("mysql save word image: %w", err)
	}
	return nil
}

// Delete removes the picture of one word.
func (r *WordImages) Delete(ctx context.Context, lessonID, lemma string) error {
	if !lessonValidID(lessonID) {
		return nil
	}
	if _, err := r.db.ExecContext(ctx, `DELETE FROM word_images WHERE lesson_id = ? AND lemma = ?`, lessonID, lemma); err != nil {
		return fmt.Errorf("mysql delete word image: %w", err)
	}
	return nil
}

// DeleteForLesson removes the settings and every picture of a lesson.
func (r *WordImages) DeleteForLesson(ctx context.Context, lessonID string) error {
	if !lessonValidID(lessonID) {
		return nil
	}
	if _, err := r.db.ExecContext(ctx, `DELETE FROM word_images WHERE lesson_id = ?`, lessonID); err != nil {
		return fmt.Errorf("mysql delete word images: %w", err)
	}
	if _, err := r.db.ExecContext(ctx, `DELETE FROM lesson_image_settings WHERE lesson_id = ?`, lessonID); err != nil {
		return fmt.Errorf("mysql delete image settings: %w", err)
	}
	return nil
}
