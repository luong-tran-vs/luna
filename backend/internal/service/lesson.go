package service

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/luongtran/luna/backend/internal/lesson"
	"github.com/luongtran/luna/backend/internal/topic"
)

// initLesson builds the lesson service: lessons and their background annotation and practice
// jobs (F2, F15, F17). Its jobs wake the worker, which initWorker creates later.
func (c *Container) initLesson() {
	c.lessonTopics = lessonTopicsPort{c.topic}
	c.lesson = lesson.NewService(lesson.Deps{
		Lessons: c.lessons,
		Topics:  c.lessonTopics,
		Jobs:    c.jobs,
		AI:      c.ai,
		Dict:    c.dict,
		Notify:  func() { c.worker.Notify() },
		Now:     time.Now,
		Log:     c.log,
		// Word pictures (F23).
		ImageStore: c.store.WordImages(),
		ImageAI:    c.imageAI,
	})
}

// initReader builds the learner-facing lesson reader: text, vocabulary lookups, comprehension
// quiz and "Hỏi AI" (F3, F15).
func (c *Container) initReader() {
	c.reader = lesson.NewReader(c.lessons, c.dict, c.lessonTopics, c.store.ReadingAnswers(), c.store.AILookups(), c.ai).
		WithImages(c.store.WordImages())
}

// lessonTopicsPort adapts topic.Service to lesson.Topics.
type lessonTopicsPort struct {
	svc *topic.Service
}

func toTopicRef(t topic.Topic) lesson.TopicRef {
	return lesson.TopicRef{ID: t.ID, Name: t.Name, Level: lesson.Level(t.Level), Words: t.Words}
}

func (p lessonTopicsPort) Get(ctx context.Context, id string) (lesson.TopicRef, error) {
	t, err := p.svc.Get(ctx, id)
	if errors.Is(err, topic.ErrNotFound) {
		return lesson.TopicRef{}, lesson.ErrTopicNotFound
	}
	if err != nil {
		return lesson.TopicRef{}, fmt.Errorf("get topic: %w", err)
	}
	return toTopicRef(t), nil
}

func (p lessonTopicsPort) Names(ctx context.Context) (map[string]lesson.TopicRef, error) {
	all, err := p.svc.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list topics: %w", err)
	}
	out := make(map[string]lesson.TopicRef, len(all))
	for _, t := range all {
		out[t.ID] = toTopicRef(t)
	}
	return out, nil
}

func (p lessonTopicsPort) RoadmapLessonIDs(ctx context.Context) (map[string]bool, error) {
	return p.svc.RoadmapLessonIDs(ctx)
}

func (p lessonTopicsPort) AppendLesson(ctx context.Context, topicID, lessonID string) error {
	err := p.svc.AppendLesson(ctx, topicID, lessonID)
	if errors.Is(err, topic.ErrNotFound) {
		return lesson.ErrTopicNotFound
	}
	return err
}

func (p lessonTopicsPort) MoveLesson(ctx context.Context, lessonID, from, to string) error {
	return p.svc.MoveLesson(ctx, lessonID, from, to)
}

func (p lessonTopicsPort) Position(ctx context.Context, topicID, lessonID string) (int, error) {
	t, err := p.svc.Get(ctx, topicID)
	if errors.Is(err, topic.ErrNotFound) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("get topic: %w", err)
	}
	return slices.Index(t.LessonIDs, lessonID) + 1, nil
}
