package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/luongtran/luna/backend/internal/topic"
	"github.com/luongtran/luna/backend/internal/topicimage"
)

// initTopicImages builds the topic pictures (added 2026-11-16), drawn from the topics' names and
// words with the same AI as the other pictures.
func (c *Container) initTopicImages() {
	c.topicImages = topicimage.NewService(topicimage.Deps{
		Store:            c.store.TopicImages(),
		Topics:           imageTopics{c.topic},
		Gemini:           c.imageAI,
		Codex:            c.codexImageAI,
		CodexConcurrency: c.cfg.CodexImageConcurrency,
		Log:              c.log,
		Now:              time.Now,
	})
}

// imageTopics adapts topic.Service to topicimage.Topics.
type imageTopics struct {
	svc *topic.Service
}

func (t imageTopics) Get(ctx context.Context, id string) (topicimage.Topic, error) {
	tp, err := t.svc.Get(ctx, id)
	if errors.Is(err, topic.ErrNotFound) {
		return topicimage.Topic{}, topicimage.ErrTopicNotFound
	}
	if err != nil {
		return topicimage.Topic{}, fmt.Errorf("topic: %w", err)
	}
	return topicimage.Topic{ID: tp.ID, Name: tp.Name, Description: tp.Description, Words: topic.Texts(tp.Words)}, nil
}
