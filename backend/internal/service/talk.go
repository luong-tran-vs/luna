package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/luongtran/luna/backend/internal/grammar"
	"github.com/luongtran/luna/backend/internal/lesson"
	"github.com/luongtran/luna/backend/internal/progress"
	"github.com/luongtran/luna/backend/internal/talk"
	"github.com/luongtran/luna/backend/internal/topic"
)

// initTalk builds the conversation practice with the AI (added 2026-10-20). It reads which lessons
// the learner studied, so it comes after initProgress.
func (c *Container) initTalk() {
	c.talk = talk.NewService(talk.Deps{
		Repo:    c.store.TalkSessions(),
		Studies: talkStudies{study: c.study, lessons: c.lessons, topics: c.topic},
		AI:      c.talkAI,
		Now:     time.Now,
		Log:     c.log,
	})
}

// talkStudies adapts the study service, the lessons and the topics to talk.Studies.
type talkStudies struct {
	study   *progress.StudyService
	lessons lesson.Repository
	topics  *topic.Service
}

func (t talkStudies) Studied(ctx context.Context, userID string) ([]talk.StudiedLesson, error) {
	ids, err := t.study.StudiedLessons(ctx, userID)
	if err != nil || len(ids) == 0 {
		return nil, err
	}
	summaries, err := t.lessons.Summaries(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("lesson summaries: %w", err)
	}
	byID := make(map[string]lesson.Summary, len(summaries))
	for _, s := range summaries {
		byID[s.ID] = s
	}
	names := map[string]string{}
	out := make([]talk.StudiedLesson, 0, len(ids))
	for _, id := range ids {
		s, ok := byID[id]
		if !ok || s.TopicID == "" {
			continue // deleted, or in no topic
		}
		name, known := names[s.TopicID]
		if !known {
			tp, err := t.topics.Get(ctx, s.TopicID)
			switch {
			case err == nil:
				name = tp.Name
			case !errors.Is(err, topic.ErrNotFound):
				return nil, fmt.Errorf("get topic: %w", err)
			}
			names[s.TopicID] = name
		}
		if name == "" {
			continue // topic deleted
		}
		out = append(out, talk.StudiedLesson{ID: id, Title: s.Title, Level: string(s.Level), TopicID: s.TopicID, TopicName: name})
	}
	return out, nil
}

func (t talkStudies) Material(ctx context.Context, ids []string) (talk.Material, error) {
	var m talk.Material
	for _, id := range ids {
		l, err := t.lessons.Get(ctx, id)
		if errors.Is(err, lesson.ErrNotFound) {
			continue
		}
		if err != nil {
			return talk.Material{}, fmt.Errorf("get lesson: %w", err)
		}
		m.Words = append(m.Words, l.TargetWords...)
		for _, a := range l.Annotations {
			m.Words = append(m.Words, a.Lemma)
		}
		if p, ok := grammar.Default().Get(l.GrammarPointID); ok {
			m.Grammar = append(m.Grammar, fmt.Sprintf("%s (%s)", p.TitleEn, p.Pattern))
		}
	}
	return m, nil
}
