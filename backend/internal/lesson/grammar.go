package lesson

import (
	"context"
	"fmt"

	"github.com/luongtran/luna/backend/internal/ai"
	"github.com/luongtran/luna/backend/internal/grammar"
)

// checkGrammarPoint accepts "" (no point) or the id of a syllabus point of the given level.
func checkGrammarPoint(id string, level Level) error {
	if id == "" {
		return nil
	}
	p, ok := grammar.Default().Get(id)
	switch {
	case !ok:
		return &ValidationError{Fields: map[string]string{"grammarPointId": "Điểm ngữ pháp không tồn tại"}}
	case p.Level != string(level):
		return &ValidationError{Fields: map[string]string{"grammarPointId": "Điểm ngữ pháp không thuộc trình độ của bài"}}
	}
	return nil
}

// GrammarTitle is the Vietnamese title of a syllabus point, "" when id is empty or unknown.
func GrammarTitle(id string) string {
	p, _ := grammar.Default().Get(id)
	return p.TitleVi
}

// grammarFocus tells the AI which syllabus point to teach; nil when id is empty or unknown.
func grammarFocus(id string) *ai.GrammarFocus {
	p, ok := grammar.Default().Get(id)
	if !ok {
		return nil
	}
	f := &ai.GrammarFocus{TitleVi: p.TitleVi, TitleEn: p.TitleEn, Pattern: p.Pattern, HintVi: p.HintVi}
	if len(p.Examples) > 0 {
		f.Example = p.Examples[0]
	}
	return f
}

// GrammarPoint is a syllabus point with the number of lessons that use it.
type GrammarPoint struct {
	grammar.Point
	LessonCount int
}

// GrammarPoints lists the syllabus points of a level, in teaching order, with how many lessons of
// topicID (every topic when empty) use each.
func (s *Service) GrammarPoints(ctx context.Context, level, topicID string) ([]GrammarPoint, error) {
	if !ValidLevel(level) {
		return nil, &ValidationError{Fields: map[string]string{"level": "Trình độ không hợp lệ"}}
	}
	counts, err := s.Lessons.CountByGrammarPoint(ctx, topicID)
	if err != nil {
		return nil, fmt.Errorf("lesson: count grammar points: %w", err)
	}
	pts := grammar.Default().ByLevel(level)
	out := make([]GrammarPoint, len(pts))
	for i, p := range pts {
		out[i] = GrammarPoint{Point: p, LessonCount: counts[p.ID]}
	}
	return out, nil
}
