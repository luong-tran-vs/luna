package topic

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/luongtran/luna/backend/internal/ai"
	"github.com/luongtran/luna/backend/internal/cefr"
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
func (s *Service) SetWords(ctx context.Context, id string, words []Word) ([]WordUse, error) {
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
	s.wordsAdded(ctx, Texts(clean))
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

// WordPlan proposes target words for count lessons of level with perLesson words each (see
// PlanWords).
func (s *Service) WordPlan(ctx context.Context, id, level string, count, perLesson int) (TargetPlan, error) {
	fields := map[string]string{}
	if !ValidLevel(level) {
		fields["level"] = "Vui lòng chọn trình độ từ A1 đến C2"
	}
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
	for _, u := range FitWords(uses, level) {
		if !u.Used {
			unused++
		}
	}
	shortage := max(0, min(count*perLesson-unused, MaxWords-len(uses)))
	return TargetPlan{Groups: PlanWords(uses, level, count, perLesson), Shortage: shortage}, nil
}

// WordSuggester asks an AI provider for new core words of a topic (implemented by ai.Provider).
type WordSuggester interface {
	SuggestWords(ctx context.Context, req ai.SuggestWordsRequest) ([]string, error)
}

// ErrNoSuggestion means the AI suggested no word that is valid and new to the topic.
var ErrNoSuggestion = errors.New("topic: AI suggested no usable word")

// SuggestWords asks the AI once for count new words of the topic at level and appends the usable
// ones (valid by CleanWords rules, not already in the list) with that level, up to MaxWords. It
// returns the added words and the new coverage of the whole list.
func (s *Service) SuggestWords(ctx context.Context, id, level string, count int) ([]string, []WordUse, error) {
	fields := map[string]string{}
	if count < 1 || count > MaxSuggestWords {
		fields["count"] = fmt.Sprintf("Số từ từ 1 đến %d", MaxSuggestWords)
	}
	if !ValidLevel(level) {
		fields["level"] = "Vui lòng chọn trình độ từ A1 đến C2"
	}
	if len(fields) > 0 {
		return nil, nil, &ValidationError{Fields: fields}
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
		Level: level, TopicName: t.Name, Existing: Texts(t.Words), Count: count,
	})
	if err != nil {
		return nil, nil, err
	}
	seen := make(map[string]bool, len(t.Words))
	for _, w := range t.Words {
		seen[strings.ToLower(w.Text)] = true
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
	words := slices.Clone(t.Words)
	for _, w := range added {
		words = append(words, Word{Text: w, Level: level})
	}
	updated, err := s.repo.SetWords(ctx, id, words)
	if err != nil {
		return nil, nil, fmt.Errorf("topic: set words: %w", err)
	}
	s.wordsAdded(ctx, added)
	uses, err := s.coverage(ctx, updated)
	if err != nil {
		return nil, nil, err
	}
	return added, uses, nil
}

// ErrWordInList means the topic's list already holds the word (ignoring case).
var ErrWordInList = errors.New("topic: word already in list")

// AddWord appends one word, for every level, to the end of the topic's list (from the word bank,
// F24). An invalid word or a full list gives a ValidationError keyed "word"; a word already in the
// list gives ErrWordInList.
func (s *Service) AddWord(ctx context.Context, id, text string) error {
	t, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	w, msg := cleanWord(text)
	if msg != "" {
		return &ValidationError{Fields: map[string]string{"word": msg}}
	}
	for _, have := range t.Words {
		if strings.EqualFold(have.Text, w) {
			return ErrWordInList
		}
	}
	if len(t.Words) >= MaxWords {
		return &ValidationError{Fields: map[string]string{"word": fmt.Sprintf("Chủ đề đã đủ %d từ", MaxWords)}}
	}
	if _, err := s.repo.SetWords(ctx, id, append(slices.Clone(t.Words), Word{Text: w})); err != nil {
		return fmt.Errorf("topic: set words: %w", err)
	}
	return nil
}

// LessonFill is how many words usable at a level a topic's list is filled up to from the words of
// its lessons of that level; a list that has them takes none (added 2026-11-16).
const LessonFill = 30

// AddLessonWords puts the words of the topic's lessons of level in its list (added 2026-11-16,
// tightened the same day: the AI annotates everyday words too). The lessons' target words, chosen
// for them, are all appended with level when new (ignoring case). The other annotated words only
// fill a list that has fewer than LessonFill words usable at level, up to LessonFill: the new ones
// the CEFR-J list does not rank below level, those of level first, then the ones it does not know,
// then those above. MaxWords bounds both; words the list does not accept are skipped and words
// already there keep their level. It returns the given words the list now holds, as written in it,
// and how many were added. It does not call onWords: the caller puts the words in the word bank
// with the meanings it has.
func (s *Service) AddLessonWords(ctx context.Context, id, level string, targets, others []string) ([]string, int, error) {
	if !ValidLevel(level) {
		return nil, 0, &ValidationError{Fields: map[string]string{"level": "Vui lòng chọn trình độ từ A1 đến C2"}}
	}
	t, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, 0, err
	}
	have := make(map[string]string, len(t.Words))
	fit := 0
	for _, w := range t.Words {
		have[strings.ToLower(w.Text)] = w.Text
		if w.FitsLevel(level) {
			fit++
		}
	}
	words := slices.Clone(t.Words)
	held, added := []string{}, 0
	seen := map[string]bool{}
	// take returns the cleaned new word, or "" when it is invalid, seen or already in the list.
	take := func(raw string) string {
		w, msg := cleanWord(raw)
		key := strings.ToLower(w)
		if msg != "" || seen[key] {
			return ""
		}
		seen[key] = true
		if text, ok := have[key]; ok {
			held = append(held, text)
			return ""
		}
		return w
	}
	push := func(w string) {
		words = append(words, Word{Text: w, Level: level})
		have[strings.ToLower(w)] = w
		held = append(held, w)
		fit++
		added++
	}
	for _, raw := range targets {
		if w := take(raw); w != "" && len(words) < MaxWords {
			push(w)
		}
	}
	type candidate struct {
		text string
		rank int // 0 at level, 1 unknown to CEFR-J, 2 above
	}
	var cands []candidate
	for _, raw := range others {
		w := take(raw)
		if w == "" {
			continue
		}
		c := candidate{text: w, rank: 1}
		if l, ok := cefr.Level(strings.ToLower(w)); ok {
			switch r := cefr.Rank(l); {
			case r < cefr.Rank(level):
				continue // an everyday word for this level
			case r == cefr.Rank(level):
				c.rank = 0
			default:
				c.rank = 2
			}
		}
		cands = append(cands, c)
	}
	slices.SortStableFunc(cands, func(a, b candidate) int { return a.rank - b.rank })
	for _, c := range cands {
		if fit >= LessonFill || len(words) >= MaxWords {
			break
		}
		push(c.text)
	}
	if added > 0 {
		if _, err := s.repo.SetWords(ctx, id, words); err != nil {
			return nil, 0, fmt.Errorf("topic: set words: %w", err)
		}
	}
	return held, added, nil
}
