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
		grammarLessonCounter{c.lessons}, time.Now)
}

// grammarLessonCounter adapts the lesson repository to grammar.LessonCounter, so the grammar
// package does not import the lesson package.
type grammarLessonCounter struct {
	repo storage.LessonStore
}

func (g grammarLessonCounter) CountByGrammarPoint(ctx context.Context, topicID string) (map[string]int, error) {
	return g.repo.CountByGrammarPoint(ctx, topicID)
}
