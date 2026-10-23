package service

import (
	"context"
	"time"

	"github.com/luongtran/luna/backend/internal/grammar"
	"github.com/luongtran/luna/backend/internal/storage"
)

// initGrammar builds the grammar lessons service (F20).
func (c *Container) initGrammar() {
	c.grammar = grammar.NewService(c.store.GrammarLessons(), c.store.GrammarProgress(), c.store.GrammarReports(), c.ai,
		grammarLessonCounter{c.lessons}, c.studyDays, time.Now)
	c.grammar.SetCourse(c.grammarCourse)
}

// grammarCourse adapts the approved course of a level to the grammar list (A1 since 2026-10-22,
// every level from the database since 2026-10-23).
func (c *Container) grammarCourse(ctx context.Context, level string) (map[string]grammar.CourseTopic, []string, bool) {
	topics, order, ok := c.curriculum.PointTopics(ctx, level)
	if !ok {
		return nil, nil, false
	}
	out := make(map[string]grammar.CourseTopic, len(topics))
	for id, t := range topics {
		out[id] = grammar.CourseTopic{Number: t.Number, Name: t.Name}
	}
	return out, order, true
}

// grammarLessonCounter adapts the lesson repository to grammar.LessonCounter, so the grammar
// package does not import the lesson package.
type grammarLessonCounter struct {
	repo storage.LessonStore
}

func (g grammarLessonCounter) CountByGrammarPoint(ctx context.Context, topicID string) (map[string]int, error) {
	return g.repo.CountByGrammarPoint(ctx, topicID)
}
