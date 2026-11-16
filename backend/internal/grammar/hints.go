package grammar

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// Hint pictures (added 2026-11-16): a choice exercise of a lesson's practice may have a picture of
// its sentence with the right answer filled in, shown above the question so the learner sees the
// situation. Every point may have them, drawable or not: the picture shows the situation, not the
// grammar. They are kept with the example pictures (same store, same batch), under a key of their own.

// hintPrefix marks the hint picture keys in the map of picture keys by exercise id.
const hintPrefix = "hint:"

// HintSentence is the sentence a hint picture shows: the stem of a choice exercise with its one
// blank filled with the right option; "" for an exercise that has none (not a choice, no single
// blank, no valid answer).
func HintSentence(e Exercise) string {
	if e.Kind != KindChoice || strings.Count(e.Text, "___") != 1 || e.AnswerIndex < 0 || e.AnswerIndex >= len(e.Options) {
		return ""
	}
	right := strings.TrimSpace(e.Options[e.AnswerIndex])
	if right == "" {
		return ""
	}
	return strings.Join(strings.Fields(strings.Replace(e.Text, "___", right, 1)), " ")
}

// HintKey is the key of the hint picture of e: it follows the sentence, so a changed stem or
// answer has no picture until drawn again; "" when e has no hint sentence.
func HintKey(e Exercise) string {
	s := HintSentence(e)
	if s == "" {
		return ""
	}
	return ExampleKey("hint: " + s)
}

// hasHintExercises reports whether the practice of c has a choice exercise a hint picture fits.
func hasHintExercises(c Content) bool {
	for _, e := range c.Practice {
		if HintKey(e) != "" {
			return true
		}
	}
	return false
}

// HintPicture is a practice choice exercise of a lesson and whether its hint picture is drawn.
type HintPicture struct {
	ExerciseID string
	Sentence   string
	Key        string
	Drawn      bool
}

// HintPictures lists the practice choice exercises of a point's lesson with whether each has its
// hint picture.
func (s *Service) HintPictures(ctx context.Context, pointID string) ([]HintPicture, error) {
	l, err := s.lessons.Get(ctx, pointID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("grammar: get lesson: %w", err)
	}
	have, err := s.imageKeys(ctx, pointID)
	if err != nil {
		return nil, err
	}
	out := []HintPicture{}
	for _, e := range l.Content.Practice {
		if k := HintKey(e); k != "" {
			out = append(out, HintPicture{ExerciseID: e.ID, Sentence: HintSentence(e), Key: k, Drawn: have[k]})
		}
	}
	return out, nil
}

// HintCount is how many practice choice exercises the lessons of some points have, and how many
// of them have their hint picture.
type HintCount struct {
	Total int
	Drawn int
}

// CountHintPictures counts the hint pictures of the lessons of the points; a point without a
// lesson counts nothing.
func (s *Service) CountHintPictures(ctx context.Context, pointIDs []string) (HintCount, error) {
	var c HintCount
	seen := map[string]bool{}
	for _, id := range pointIDs {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		list, err := s.HintPictures(ctx, id)
		if errors.Is(err, ErrNotFound) {
			continue
		}
		if err != nil {
			return HintCount{}, err
		}
		for _, h := range list {
			c.Total++
			if h.Drawn {
				c.Drawn++
			}
		}
	}
	return c, nil
}

// hintBatchItems lists the hint pictures a batch draws (ImageBatchRequest.Hints), in the order of
// the points, and how many lessons they come from.
func (s *Service) hintBatchItems(ctx context.Context, req ImageBatchRequest) ([]batchItem, int, error) {
	only := map[string]bool{}
	for _, k := range req.Keys {
		only[strings.TrimSpace(k)] = true
	}
	seen := map[string]bool{}
	var out []batchItem
	points := 0
	for _, id := range req.PointIDs {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		p, err := s.point(id)
		if err != nil {
			continue
		}
		list, err := s.HintPictures(ctx, id)
		if errors.Is(err, ErrNotFound) {
			continue
		}
		if err != nil {
			return nil, 0, err
		}
		n := len(out)
		drawn := map[string]bool{}
		for _, h := range list {
			if (len(only) > 0 && !only[h.Key]) || (h.Drawn && !req.Replace) || drawn[h.Key] {
				continue
			}
			drawn[h.Key] = true // the same sentence twice is drawn once
			out = append(out, batchItem{PointID: id, Key: h.Key, En: h.Sentence, Point: pointName(p)})
		}
		if len(out) > n {
			points++
		}
	}
	return out, points, nil
}
