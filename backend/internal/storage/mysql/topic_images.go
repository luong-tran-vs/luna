package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/luongtran/luna/backend/internal/topicimage"
)

// TopicImages implements topicimage.Repository: the pictures of the topics of the course (added 2026-11-16).
type TopicImages struct {
	db *sql.DB
}

// NewTopicImages returns the TopicImages repository.
func NewTopicImages(db *sql.DB) *TopicImages { return &TopicImages{db: db} }

var _ topicimage.Repository = (*TopicImages)(nil)

// topicImagesSchema is the DDL of the topic pictures (see migrate.go).
func topicImagesSchema() []string {
	return []string{
		`CREATE TABLE IF NOT EXISTS topic_images (
	topic_id   CHAR(24)     NOT NULL PRIMARY KEY,
	mime       VARCHAR(64)  NOT NULL,
	data       MEDIUMBLOB   NOT NULL,
	created_at DATETIME(6)  NOT NULL
) ` + tableOptions,
	}
}

// List tells every topic that has a picture, without the pictures.
func (r *TopicImages) List(ctx context.Context) ([]topicimage.Info, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT topic_id, created_at FROM topic_images ORDER BY topic_id`)
	if err != nil {
		return nil, fmt.Errorf("mysql list topic images: %w", err)
	}
	defer rows.Close()
	out := []topicimage.Info{}
	for rows.Next() {
		var i topicimage.Info
		if err := rows.Scan(&i.TopicID, &i.CreatedAt); err != nil {
			return nil, fmt.Errorf("mysql scan topic image: %w", err)
		}
		i.CreatedAt = i.CreatedAt.UTC()
		out = append(out, i)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mysql list topic images: %w", err)
	}
	return out, nil
}

// Get returns topicimage.ErrImageNotFound when the topic has no picture.
func (r *TopicImages) Get(ctx context.Context, topicID string) (topicimage.Image, error) {
	img := topicimage.Image{TopicID: topicID}
	err := r.db.QueryRowContext(ctx, `SELECT mime, data, created_at FROM topic_images WHERE topic_id = ?`, topicID).
		Scan(&img.MIME, &img.Data, &img.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return topicimage.Image{}, topicimage.ErrImageNotFound
	}
	if err != nil {
		return topicimage.Image{}, fmt.Errorf("mysql find topic image: %w", err)
	}
	img.CreatedAt = img.CreatedAt.UTC()
	return img, nil
}

// Save stores a picture, replacing the topic's previous one.
func (r *TopicImages) Save(ctx context.Context, img topicimage.Image) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO topic_images (topic_id, mime, data, created_at) VALUES (?,?,?,?)
		ON DUPLICATE KEY UPDATE mime = VALUES(mime), data = VALUES(data), created_at = VALUES(created_at)`,
		img.TopicID, img.MIME, img.Data, utc(img.CreatedAt))
	if err != nil {
		return fmt.Errorf("mysql save topic image: %w", err)
	}
	return nil
}

// Delete removes a topic's picture.
func (r *TopicImages) Delete(ctx context.Context, topicID string) error {
	if _, err := r.db.ExecContext(ctx, `DELETE FROM topic_images WHERE topic_id = ?`, topicID); err != nil {
		return fmt.Errorf("mysql delete topic image: %w", err)
	}
	return nil
}
