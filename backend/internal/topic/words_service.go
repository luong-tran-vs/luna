package topic

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/luongtran/luna/backend/internal/ai"
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

// TargetPlan is a target word split for one generation.
type TargetPlan struct {
	Groups [][]string
	// Shortage is how many more unused words the topic needs so that no two lessons share a
	// word and every word is new; SuggestWords can add them.
	Shortage int
}

// WordPlan proposes target words for count lessons of perLesson words each (see PlanWords).
func (s *Service) WordPlan(ctx context.Context, id string, count, perLesson int) (TargetPlan, error) {
	fields := map[string]string{}
	if count < 1 || count > MaxPlanLessons {
		fields["count"] = fmt.Sprintf("Số bài từ 1 đến %d", MaxPlanLessons)
	}
	if perLesson < 0 || perLesson > MaxTargetWords {
		fields["perLesson"] = fmt.Sprintf("Số từ mỗi bài từ 0 đến %d", MaxTargetWords)
	}
	if len(fields) > 0 {
		return TargetPlan{}, &ValidationError{Fields: fields}
	}
	uses, err := s.Words(ctx, id)
	if err != nil {
		return TargetPlan{}, err
	}
	unused := 0
	for _, u := range uses {
		if !u.Used {
			unused++
		}
	}
	shortage := max(0, min(count*perLesson-unused, MaxWords-len(uses)))
	return TargetPlan{Groups: PlanWords(uses, count, perLesson), Shortage: shortage}, nil
}

// WordSuggester asks an AI provider for new core words of a topic (implemented by ai.Provider).
type WordSuggester interface {
	SuggestWords(ctx context.Context, req ai.SuggestWordsRequest) ([]string, error)
}

// ErrNoSuggestion means the AI suggested no word that is valid and new to the topic.
var ErrNoSuggestion = errors.New("topic: AI suggested no usable word")

// SuggestWords asks the AI once for count new words of the topic and appends the usable ones
// (valid by CleanWords rules, not already in the list) up to MaxWords. It returns the added
// words and the new coverage of the whole list.
func (s *Service) SuggestWords(ctx context.Context, id string, count int) ([]string, []WordUse, error) {
	if count < 1 || count > MaxSuggestWords {
		return nil, nil, &ValidationError{Fields: map[string]string{
			"count": fmt.Sprintf("Số từ từ 1 đến %d", MaxSuggestWords),
		}}
	}
	t, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	room := MaxWords - len(t.Words)
	if room <= 0 {
		return nil, nil, &ValidationError{Fields: map[string]string{"count": fmt.Sprintf("Chủ đề đã đủ %d từ", MaxWords)}}
	}
	count = min(count, room)
	suggested, err := s.ai.SuggestWords(ctx, ai.SuggestWordsRequest{
		Level: t.Level, TopicName: t.Name, Existing: t.Words, Count: count,
	})
	if err != nil {
		return nil, nil, err
	}
	seen := make(map[string]bool, len(t.Words))
	for _, w := range t.Words {
		seen[strings.ToLower(w)] = true
	}
	added := []string{}
	for _, raw := range suggested {
		w, msg := cleanWord(raw)
		if msg != "" || seen[strings.ToLower(w)] || len(added) == count {
			continue
		}
		seen[strings.ToLower(w)] = true
		added = append(added, w)
	}
	if len(added) == 0 {
		return nil, nil, ErrNoSuggestion
	}
	updated, err := s.repo.SetWords(ctx, id, append(slices.Clone(t.Words), added...))
	if err != nil {
		return nil, nil, fmt.Errorf("topic: set words: %w", err)
	}
	uses, err := s.coverage(ctx, updated)
	if err != nil {
		return nil, nil, err
	}
	return added, uses, nil
}
