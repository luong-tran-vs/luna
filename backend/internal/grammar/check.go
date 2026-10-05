package grammar

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/luongtran/luna/backend/internal/ai"
)

// checkTimeout bounds the AI call that solves the exercises of one lesson.
const checkTimeout = 120 * time.Second

// FlagsError means the lesson still has exercises flagged by the AI check and the admin did not
// acknowledge them when publishing.
type FlagsError struct{ Count int }

func (e *FlagsError) Error() string { return fmt.Sprintf("grammar: %d flagged exercises", e.Count) }

// expandFillAnswers returns c with the answers of every fill exercise expanded (contractions and
// full forms). The stored content is not touched.
func expandFillAnswers(c Content) Content {
	expand := func(in []Exercise) []Exercise {
		out := make([]Exercise, len(in))
		copy(out, in)
		for i := range out {
			if out[i].Kind == KindFill {
				out[i].Answers = ExpandAnswers(out[i].Answers)
			}
		}
		return out
	}
	c.Practice, c.Mastery = expand(c.Practice), expand(c.Mastery)
	return c
}

// Check has a second AI solve the exercises of a lesson without their keys and flags every one
// whose answer differs from the stored key, or that the AI finds ambiguous. The result replaces the
// lesson's flags. An AI failure returns an error and leaves the lesson unchanged.
func (s *Service) Check(ctx context.Context, pointID string) (Lesson, error) {
	p, err := s.point(pointID)
	if err != nil {
		return Lesson{}, err
	}
	l, err := s.lessons.Get(ctx, pointID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return Lesson{}, err
		}
		return Lesson{}, fmt.Errorf("grammar: get lesson: %w", err)
	}
	return s.runCheck(ctx, p, l)
}

func (s *Service) runCheck(ctx context.Context, p Point, l Lesson) (Lesson, error) {
	all := make([]Exercise, 0, len(l.Content.Practice)+len(l.Content.Mastery))
	all = append(all, l.Content.Practice...)
	all = append(all, l.Content.Mastery...)

	req := ai.SolveRequest{Level: p.Level, TitleEn: p.TitleEn, Pattern: p.Pattern}
	for _, e := range all {
		req.Exercises = append(req.Exercises, ai.SolveExercise{
			ID: e.ID, Kind: string(e.Kind), PromptVi: e.PromptVi, Text: e.Text, Options: e.Options, Words: e.Words,
		})
	}
	aiCtx, cancel := context.WithTimeout(ctx, checkTimeout)
	defer cancel()
	solutions, err := s.ai.SolveGrammarExercises(aiCtx, req)
	if err != nil {
		if errors.Is(err, ai.ErrNotConfigured) || errors.Is(err, ai.ErrInvalidKey) || errors.Is(err, ai.ErrQuota) {
			return Lesson{}, err
		}
		return Lesson{}, fmt.Errorf("%w: %w", errAI, err)
	}
	byID := make(map[string]ai.Solution, len(solutions))
	for _, sol := range solutions {
		byID[sol.ExerciseID] = sol
	}

	checks := []Check{}
	for _, e := range all {
		sol, ok := byID[e.ID]
		if !ok {
			checks = append(checks, Check{ExerciseID: e.ID, Kind: CheckUnchecked, NoteVi: "AI không trả lời câu này, hãy tự kiểm tra."})
			continue
		}
		if c, flagged := compareSolution(e, sol); flagged {
			checks = append(checks, c)
		}
	}
	l.Checks, l.CheckedAt, l.VerifiedAt = checks, s.now(), time.Time{}
	if err := s.lessons.Save(ctx, l); err != nil {
		return Lesson{}, fmt.Errorf("grammar: save lesson: %w", err)
	}
	return l, nil
}

// compareSolution reports whether the AI's answer to e disagrees with the stored key.
func compareSolution(e Exercise, sol ai.Solution) (Check, bool) {
	if sol.Ambiguous {
		note := strings.TrimSpace(sol.NoteVi)
		if note == "" {
			note = "AI thấy câu này mơ hồ: có thể có nhiều hơn một đáp án đúng, hoặc không có đáp án nào."
		}
		return Check{ExerciseID: e.ID, Kind: CheckAmbiguous, NoteVi: note}, true
	}
	mismatch := func(note string) (Check, bool) {
		return Check{ExerciseID: e.ID, Kind: CheckMismatch, NoteVi: note}, true
	}
	switch e.Kind {
	case KindChoice:
		if sol.ChoiceIndex == nil || *sol.ChoiceIndex != e.AnswerIndex {
			got := "không chọn được đáp án nào"
			if sol.ChoiceIndex != nil && *sol.ChoiceIndex >= 0 && *sol.ChoiceIndex < len(e.Options) {
				got = fmt.Sprintf("chọn \"%s\"", e.Options[*sol.ChoiceIndex])
			}
			return mismatch(fmt.Sprintf("AI %s, khác đáp án đã lưu.", got))
		}
	case KindFill:
		ans := foldAnswer(sol.Answer)
		for _, a := range ExpandAnswers(e.Answers) {
			if foldAnswer(a) == ans && ans != "" {
				return Check{}, false
			}
		}
		return mismatch(fmt.Sprintf("AI điền \"%s\", không nằm trong các đáp án đã lưu.", strings.TrimSpace(sol.Answer)))
	case KindReorder:
		if foldSentence(sol.Sentence) != foldSentence(e.Sentence) {
			return mismatch(fmt.Sprintf("AI ghép thành \"%s\", khác câu đã lưu.", strings.TrimSpace(sol.Sentence)))
		}
	}
	return Check{}, false
}

func foldAnswer(s string) string { return strings.ToLower(normalizeAnswer(s)) }

// foldSentence ignores case, spacing, curly apostrophes and a final full stop or question mark.
func foldSentence(s string) string {
	return strings.TrimRight(foldAnswer(s), ".?! ")
}
