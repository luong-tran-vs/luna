package games

import (
	"context"
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
	"time"
)

// PoolWord is a word the games may ask: from the learner's cards first, then the words of the
// lessons studied, then the lesson to study now (research R2).
type PoolWord struct {
	Lemma     string
	MeaningVi string
	LessonID  string
	ImageURL  string
	Due       bool
	// Tier is where the word comes from: 0 cards, 1 studied lessons, 2 the lesson to study now.
	Tier int
}

// normalize lowers and collapses spaces.
func normalize(s string) string { return strings.Join(strings.Fields(strings.ToLower(s)), " ") }

// wordPool returns the learner's words in the order the games ask them: cards due now (shuffled),
// the other cards (shuffled), then the words of the studied lessons and of the lesson to study now,
// read only while fewer than need words pass keep. Every word has its picture and meaning (card's
// first, word bank's next).
func (s *Service) wordPool(ctx context.Context, userID string, need int, keep func(PoolWord) bool, rnd *rand.Rand) ([]PoolWord, error) {
	now := s.d.Now()
	seen := map[string]bool{}
	var out []PoolWord
	kept := 0
	add := func(words []PoolWord) error {
		var fresh []PoolWord
		for _, w := range words {
			w.Lemma = normalize(w.Lemma)
			if w.Lemma == "" || seen[w.Lemma] {
				continue
			}
			seen[w.Lemma] = true
			fresh = append(fresh, w)
		}
		if err := s.decorate(ctx, fresh); err != nil {
			return err
		}
		for _, w := range fresh {
			if keep == nil || keep(w) {
				kept++
			}
		}
		out = append(out, fresh...)
		return nil
	}

	cards, err := s.d.Cards.Cards(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("games: cards: %w", err)
	}
	var due, other []PoolWord
	for _, c := range cards {
		w := PoolWord{Lemma: c.Lemma, MeaningVi: strings.TrimSpace(c.MeaningVi), LessonID: c.LessonID, Due: !c.Due.After(now)}
		if w.Due {
			due = append(due, w)
		} else {
			other = append(other, w)
		}
	}
	shuffle(rnd, due)
	shuffle(rnd, other)
	if err := add(append(due, other...)); err != nil {
		return nil, err
	}
	if kept >= need {
		return out, nil
	}

	studied, err := s.d.Study.StudiedLessons(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("games: studied lessons: %w", err)
	}
	for _, id := range studied {
		words, err := s.lessonPool(ctx, id, 1)
		if err != nil {
			return nil, err
		}
		if err := add(words); err != nil {
			return nil, err
		}
		if kept >= need {
			return out, nil
		}
	}
	current, err := s.d.Study.CurrentLesson(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("games: current lesson: %w", err)
	}
	if current != "" && !slices.Contains(studied, current) {
		words, err := s.lessonPool(ctx, current, 2)
		if err != nil {
			return nil, err
		}
		if err := add(words); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// lessonPool returns the words of a lesson as pool words of tier.
func (s *Service) lessonPool(ctx context.Context, lessonID string, tier int) ([]PoolWord, error) {
	words, err := s.d.Lessons.LessonWords(ctx, lessonID)
	if err != nil {
		return nil, fmt.Errorf("games: lesson words: %w", err)
	}
	out := make([]PoolWord, 0, len(words))
	for _, w := range words {
		out = append(out, PoolWord{Lemma: w.Lemma, MeaningVi: strings.TrimSpace(w.MeaningVi), LessonID: lessonID, Tier: tier})
	}
	return out, nil
}

// decorate fills the picture of each word and the meaning of those without one.
func (s *Service) decorate(ctx context.Context, words []PoolWord) error {
	if len(words) == 0 {
		return nil
	}
	lemmas := make([]string, len(words))
	var missing []string
	for i, w := range words {
		lemmas[i] = w.Lemma
		if w.MeaningVi == "" {
			missing = append(missing, w.Lemma)
		}
	}
	urls, err := s.d.Pictures.ImageURLs(ctx, lemmas)
	if err != nil {
		return fmt.Errorf("games: pictures: %w", err)
	}
	meanings := map[string]string{}
	if len(missing) > 0 && s.d.Meanings != nil {
		if meanings, err = s.d.Meanings.MeaningsOf(ctx, missing); err != nil {
			return fmt.Errorf("games: meanings: %w", err)
		}
	}
	for i := range words {
		words[i].ImageURL = urls[words[i].Lemma]
		if words[i].MeaningVi == "" {
			words[i].MeaningVi = strings.TrimSpace(meanings[words[i].Lemma])
		}
	}
	return nil
}

// spellable reports whether a lemma can be spelt letter by letter: one word of 3 to 8 letters a–z.
func spellable(lemma string) bool {
	if n := len(lemma); n < 3 || n > 8 {
		return false
	}
	for _, r := range lemma {
		if r < 'a' || r > 'z' {
			return false
		}
	}
	return true
}

// sameForm reports whether two words are one word in two forms: the same, or one is the other with
// -s, -es, -ed or -ing, or -d after an e (apple / apples, play / played, like / liked).
func sameForm(a, b string) bool {
	a, b = normalize(a), normalize(b)
	if a == b {
		return true
	}
	if len(a) > len(b) {
		a, b = b, a
	}
	if !strings.HasPrefix(b, a) {
		return false
	}
	switch b[len(a):] {
	case "s", "es", "ed", "ing":
		return true
	case "d": // like / liked, not car / card
		return strings.HasSuffix(a, "e")
	}
	return false
}

// sameMeaning reports whether two Vietnamese meanings are the same once normalized; empty ones never are.
func sameMeaning(a, b string) bool {
	a, b = normalize(a), normalize(b)
	return a != "" && a == b
}

func shuffle[T any](rnd *rand.Rand, list []T) {
	rnd.Shuffle(len(list), func(i, j int) { list[i], list[j] = list[j], list[i] })
}

// dayKey is the date of t in loc, YYYY-MM-DD.
func dayKey(t time.Time, loc *time.Location) string { return t.In(loc).Format(time.DateOnly) }
