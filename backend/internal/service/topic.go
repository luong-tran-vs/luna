package service

import (
	"context"
	"fmt"
	"time"

	"github.com/luongtran/luna/backend/internal/storage"
	"github.com/luongtran/luna/backend/internal/topic"
)

// initTopic builds the topic service: topics, their roadmaps and word lists (F14, F18).
func (c *Container) initTopic() {
	c.topic = topic.NewService(c.store.Topics(), topicLessons{c.lessons}, c.ai, time.Now)
}

// topicLessons adapts the lesson repository to topic.Lessons.
type topicLessons struct {
	repo storage.LessonStore
}

func (t topicLessons) CountByTopic(ctx context.Context) (map[string]map[string]int, error) {
	return t.repo.CountByTopic(ctx)
}

func (t topicLessons) PlaceOf(ctx context.Context, ids []string) (map[string]topic.Place, error) {
	return t.repo.PlaceOf(ctx, ids)
}

func (t topicLessons) Texts(ctx context.Context, topicIDs []string) (map[string][]topic.LessonText, error) {
	return t.repo.TopicTexts(ctx, topicIDs)
}

func (t topicLessons) Refs(ctx context.Context, ids []string) ([]topic.LessonRef, error) {
	sums, err := t.repo.Summaries(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("lesson summaries: %w", err)
	}
	out := make([]topic.LessonRef, len(sums))
	for i, s := range sums {
		out[i] = topic.LessonRef{
			ID: s.ID, Title: s.Title, Level: string(s.Level), TopicID: s.TopicID,
			AnnotationStatus: string(s.AnnotationStatus), Draft: s.Draft, CreatedAt: s.CreatedAt,
		}
	}
	return out, nil
}
