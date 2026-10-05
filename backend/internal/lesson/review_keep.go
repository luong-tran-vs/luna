package lesson

import (
	"context"
	"log/slog"
	"slices"
	"strconv"
	"strings"
)

// F22b: editing one part of a lesson keeps the flags of everything the edit did not touch.

// keySep joins the fields of an item into one comparison key.
const keySep = "\x1f"

func questionKeys(qs []Question) []string {
	out := make([]string, len(qs))
	for i, q := range qs {
		out[i] = strings.Join(append(append([]string{q.Prompt}, q.Options...),
			strconv.Itoa(q.AnswerIndex), q.ExplanationVi), keySep)
	}
	return out
}

func annotationKeys(as []Annotation) []string {
	out := make([]string, len(as))
	for i, a := range as {
		out[i] = strings.Join([]string{a.Text, a.Lemma, a.MeaningVi}, keySep)
	}
	return out
}

func translationKeys(ts []Translation) []string {
	out := make([]string, len(ts))
	for i, t := range ts {
		out[i] = strings.Join(append([]string{t.Vi, t.En}, t.Distractors...), keySep)
	}
	return out
}

// remapFlags returns flags with those of area moved to the places their items now have. oldKeys and
// newKeys hold one comparison key per item before and after the edit. A flag survives when its item
// is still in newKeys unchanged (equal keys are paired in order); an item that was edited or removed
// loses its flag. Flags of other areas are kept as they are. The result is sorted by area, then index.
func remapFlags(area FlagArea, oldKeys, newKeys []string, flags []Flag) []Flag {
	// newAt[key] lists the new places of key; each old item takes the next one.
	newAt := map[string][]int{}
	for j, k := range newKeys {
		newAt[k] = append(newAt[k], j)
	}
	moved := make([]int, len(oldKeys))
	for i, k := range oldKeys {
		moved[i] = -1
		if at := newAt[k]; len(at) > 0 {
			moved[i], newAt[k] = at[0], at[1:]
		}
	}

	out := []Flag{}
	for _, f := range flags {
		if f.Area != area {
			out = append(out, f)
			continue
		}
		if f.Index < 0 || f.Index >= len(moved) || moved[f.Index] < 0 {
			continue
		}
		f.Index = moved[f.Index]
		out = append(out, f)
	}
	slices.SortStableFunc(out, func(a, b Flag) int {
		if c := slices.Index(areaOrder, a.Area) - slices.Index(areaOrder, b.Area); c != 0 {
			return c
		}
		return a.Index - b.Index
	})
	return out
}

// keepReview writes the review of l again after an edit of one area dropped it. The edit changed
// the content, so an earlier verification no longer holds; the check time stays. A failure is only
// logged: the edit itself has been saved.
func (s *Service) keepReview(ctx context.Context, l Lesson, area FlagArea, oldKeys, newKeys []string) {
	if l.Review == nil {
		return
	}
	r := &Review{CheckedAt: l.Review.CheckedAt, Flags: remapFlags(area, oldKeys, newKeys, l.Review.Flags)}
	ok, err := s.Lessons.SaveReview(ctx, l.ID, l.Revision, r)
	switch {
	case err != nil:
		s.Log.WarnContext(ctx, "lesson: keep review after edit", slog.String("lesson", l.ID), slog.Any("error", err))
	case !ok:
		s.Log.WarnContext(ctx, "lesson: keep review after edit: lesson changed meanwhile", slog.String("lesson", l.ID))
	}
}
