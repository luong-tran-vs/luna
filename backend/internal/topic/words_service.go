package topic

import (
	"context"
	"fmt"
)

// fillCoverage sets WordCount and UsedWordCount of topics with words, reading the lessons of
// all of them at once.
func (s *Service) fillCoverage(ctx context.Context, items []Summary) error {
	var ids []string
	for _, it := range items {
		if len(it.Words) > 0 {
			ids = append(ids, it.ID)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	texts, err := s.lessons.Texts(ctx, ids)
	if err != nil {
		return fmt.Errorf("topic: lesson texts: %w", err)
	}
	for i := range items {
		it := &items[i]
		if len(it.Words) == 0 {
			continue
		}
		it.WordCount = len(it.Words)
		for _, u := range Coverage(it.Words, texts[it.ID]) {
			if u.Used {
				it.UsedWordCount++
			}
		}
	}
	return nil
}

// Words returns the topic's words with their coverage in the topic's lessons (F18).
func (s *Service) Words(ctx context.Context, id string) ([]WordUse, error) {
	t, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.coverage(ctx, t)
}

// SetWords replaces the topic's words after CleanWords; the startup seed never overwrites them
// afterwards, even when the list is empty.
func (s *Service) SetWords(ctx context.Context, id string, words []string) ([]WordUse, error) {
	if _, err := s.repo.Get(ctx, id); err != nil {
		return nil, err
	}
	clean, err := CleanWords(words)
	if err != nil {
		return nil, err
	}
	t, err := s.repo.SetWords(ctx, id, clean)
	if err != nil {
		return nil, fmt.Errorf("topic: set words: %w", err)
	}
	return s.coverage(ctx, t)
}

func (s *Service) coverage(ctx context.Context, t Topic) ([]WordUse, error) {
	if len(t.Words) == 0 {
		return []WordUse{}, nil
	}
	texts, err := s.lessons.Texts(ctx, []string{t.ID})
	if err != nil {
		return nil, fmt.Errorf("topic: lesson texts: %w", err)
	}
	return Coverage(t.Words, texts[t.ID]), nil
}

// WordPlan proposes target words for count lessons of perLesson words each (see PlanWords).
func (s *Service) WordPlan(ctx context.Context, id string, count, perLesson int) ([][]string, error) {
	fields := map[string]string{}
	if count < 1 || count > MaxPlanLessons {
		fields["count"] = fmt.Sprintf("Số bài từ 1 đến %d", MaxPlanLessons)
	}
	if perLesson < 0 || perLesson > MaxTargetWords {
		fields["perLesson"] = fmt.Sprintf("Số từ mỗi bài từ 0 đến %d", MaxTargetWords)
	}
	if len(fields) > 0 {
		return nil, &ValidationError{Fields: fields}
	}
	uses, err := s.Words(ctx, id)
	if err != nil {
		return nil, err
	}
	return PlanWords(uses, count, perLesson), nil
}
