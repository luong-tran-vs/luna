package grammar

import (
	"context"
	"fmt"
)

// PointSource is a grammar point with a published lesson and the parts of it the games build
// "Bắn lỗi" sentences from (F26): its exercises (practice and mastery) and its common mistakes, the
// lesson's and the hand-written ones.
type PointSource struct {
	Point     Point
	Exercises []Exercise
	Mistakes  []Mistake
}

// PublishedSources returns every point with a published lesson, in syllabus order.
func (s *Service) PublishedSources(ctx context.Context) ([]PointSource, error) {
	all, err := s.lessons.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("grammar: list lessons: %w", err)
	}
	published := map[string]bool{}
	var ids []string
	for _, l := range all {
		if l.Status == StatusPublished {
			published[l.PointID] = true
			ids = append(ids, l.PointID)
		}
	}
	if len(ids) == 0 {
		return nil, nil
	}
	contents, err := s.lessons.Contents(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("grammar: lesson contents: %w", err)
	}
	var out []PointSource
	for _, p := range s.syllabus.All() {
		c, ok := contents[p.ID]
		if !published[p.ID] || !ok {
			continue
		}
		src := PointSource{Point: p, Exercises: append(append([]Exercise(nil), c.Practice...), c.Mastery...)}
		src.Mistakes = append(src.Mistakes, c.Mistakes...)
		for _, m := range MistakesOf(p.ID) {
			src.Mistakes = append(src.Mistakes, Mistake{Wrong: m.Wrong, Right: m.Right, NoteVi: m.NoteVi})
		}
		out = append(out, src)
	}
	return out, nil
}
