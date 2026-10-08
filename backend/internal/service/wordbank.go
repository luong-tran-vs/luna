package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/luongtran/luna/backend/internal/lesson"
	"github.com/luongtran/luna/backend/internal/topic"
	"github.com/luongtran/luna/backend/internal/wordbank"
)

// initWordBank builds the shared word bank (F24), which the reader and the reviews read from.
func (c *Container) initWordBank() {
	c.wordBank = wordbank.NewService(wordbank.Deps{
		Repo:       c.store.WordBank(),
		Dict:       c.dict,
		Topics:     topicWords{c.topic},
		ImageAI:    c.imageAI,
		MeaningAI:  c.meaningAI,
		FetchImage: lesson.NewImageFetcher(),
		Now:        time.Now,
		Log:        c.log,
	})
}

// topicWords adapts topic.Service to wordbank.TopicWords.
type topicWords struct {
	svc *topic.Service
}

func (t topicWords) All(ctx context.Context) ([]wordbank.TopicList, error) {
	topics, err := t.svc.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list topics: %w", err)
	}
	out := make([]wordbank.TopicList, len(topics))
	for i, tp := range topics {
		out[i] = wordbank.TopicList{TopicRef: wordbank.TopicRef{ID: tp.ID, Name: tp.Name}, Words: topic.Texts(tp.Words)}
	}
	return out, nil
}

func (t topicWords) AddWord(ctx context.Context, topicID, text string) error {
	err := t.svc.AddWord(ctx, topicID, text)
	var verr *topic.ValidationError
	switch {
	case errors.Is(err, topic.ErrWordInList):
		return wordbank.ErrInTopic
	case errors.Is(err, topic.ErrNotFound):
		return &wordbank.ValidationError{Fields: map[string]string{"topicId": "Chủ đề không tồn tại"}}
	case errors.As(err, &verr):
		return &wordbank.ValidationError{Fields: map[string]string{"lemma": "Chủ đề không nhận từ này: " + verr.Fields["word"]}}
	}
	return err
}

// lessonWordBank adapts wordbank.Service to lesson.WordBank.
type lessonWordBank struct {
	svc *wordbank.Service
}

func (b lessonWordBank) Words(ctx context.Context, lemmas []string) (map[string]lesson.BankWord, error) {
	words, err := b.svc.Words(ctx, lemmas)
	if err != nil {
		return nil, err
	}
	out := make(map[string]lesson.BankWord, len(words))
	for lemma, w := range words {
		out[lemma] = lesson.BankWord{IPA: w.IPA, HasImage: w.HasImage()}
	}
	return out, nil
}

func (b lessonWordBank) Image(ctx context.Context, lemma string) (lesson.WordImage, error) {
	img, err := b.svc.Image(ctx, lemma)
	if errors.Is(err, wordbank.ErrImageNotFound) {
		return lesson.WordImage{}, lesson.ErrImageNotFound
	}
	if err != nil {
		return lesson.WordImage{}, err
	}
	return lesson.WordImage{Lemma: img.Lemma, MIME: img.MIME, Data: img.Data, CreatedAt: img.CreatedAt}, nil
}
