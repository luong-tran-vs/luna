package grammar

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/luongtran/luna/backend/internal/ai"
)

// checkTimeout bounds the AI call that solves the exercises of one lesson.
const checkTimeout = 5 * time.Minute

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

	aiCtx, cancel := context.WithTimeout(ctx, checkTimeout)
	defer cancel()
	byID, err := s.solve(aiCtx, p, all)
	if err != nil {
		if errors.Is(err, ai.ErrNotConfigured) || errors.Is(err, ai.ErrInvalidKey) || errors.Is(err, ai.ErrQuota) {
			return Lesson{}, err
		}
		return Lesson{}, fmt.Errorf("%w: %w", errAI, err)
	}

	// Fixed 2026-10-26: models sometimes leave an exercise out or write their notes into its answer.
	// Those exercises are asked once more on their own; a failed retry keeps the first answers.
	var retry []Exercise
	for _, e := range all {
		if sol, ok := byID[e.ID]; !ok || unusable(e, sol) {
			if ok {
				slog.WarnContext(ctx, "grammar check: unusable AI answer", slog.String("point", p.ID),
					slog.String("exercise", e.ID), slog.String("kind", string(e.Kind)), slog.Any("words", e.Words),
					slog.String("sentence", sol.Sentence), slog.String("answer", sol.Answer))
			}
			retry = append(retry, e)
		}
	}
	if len(retry) > 0 {
		again, err := s.solve(aiCtx, p, retry)
		if err != nil {
			slog.WarnContext(ctx, "grammar check: retry failed", slog.String("point", p.ID), slog.Any("error", err))
		}
		for _, e := range retry {
			if sol, ok := again[e.ID]; ok {
				byID[e.ID] = sol
			}
		}
	}

	checks := []Check{}
	for _, e := range all {
		sol, ok := byID[e.ID]
		if !ok {
			checks = append(checks, Check{ExerciseID: e.ID, Kind: CheckUnchecked, NoteVi: missingNote})
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

// solve has the AI answer exercises without their keys, by exercise id.
func (s *Service) solve(ctx context.Context, p Point, exercises []Exercise) (map[string]ai.Solution, error) {
	req := ai.SolveRequest{Level: p.Level, TitleEn: p.TitleEn, Pattern: p.Pattern}
	for _, e := range exercises {
		req.Exercises = append(req.Exercises, ai.SolveExercise{
			ID: e.ID, Kind: string(e.Kind), PromptVi: e.PromptVi, Text: e.Text, Options: e.Options, Words: e.Words,
		})
	}
	solutions, err := s.ai.SolveGrammarExercises(ctx, req)
	if err != nil {
		return nil, err
	}
	asked := make(map[string]Exercise, len(exercises))
	for _, e := range exercises {
		asked[e.ID] = e
	}
	byID := make(map[string]ai.Solution, len(solutions))
	for _, sol := range solutions {
		// Some models write the sentence of a reorder or a fix into answer instead.
		if sol.Sentence == "" {
			sol.Sentence = sol.Answer
		}
		byID[sol.ExerciseID] = keepLeadingAnswer(asked[sol.ExerciseID], sol)
	}
	return byID, nil
}

// compareSolution reports whether the AI's answer to e disagrees with the stored key.
func compareSolution(e Exercise, sol ai.Solution) (Check, bool) {
	if unusable(e, sol) {
		return Check{ExerciseID: e.ID, Kind: CheckUnchecked, NoteVi: garbledNote}, true
	}
	// The Vietnamese not matching the answer is checked first: the AI usually still finds the stored
	// answer then (the tiles lead to it), so nothing else would flag it.
	if sol.MeaningOff {
		note := strings.TrimSpace(sol.NoteVi)
		if note == "" {
			note = "AI thấy phần tiếng Việt không cùng nghĩa với đáp án tiếng Anh."
		}
		return Check{ExerciseID: e.ID, Kind: CheckMeaning, NoteVi: note}, true
	}
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
	case KindFix:
		got := foldSentence(sol.Sentence)
		for _, a := range append([]string{e.Sentence}, e.Answers...) {
			if got != "" && got == foldSentence(a) {
				return Check{}, false
			}
		}
		return mismatch(fmt.Sprintf("AI sửa thành \"%s\", khác các câu đã lưu.", strings.TrimSpace(sol.Sentence)))
	}
	return Check{}, false
}

// maxFixExtraWords is how many words an AI correction may have beyond the longest stored one.
const maxFixExtraWords = 3

// keepLeadingAnswer cuts what follows a stored answer the AI's sentence starts with, e.g. "We were at
// the beach last summer. Correct error was -> were." keeps the first sentence.
func keepLeadingAnswer(e Exercise, sol ai.Solution) ai.Solution {
	if e.Kind != KindReorder && e.Kind != KindFix {
		return sol
	}
	words := strings.Fields(sol.Sentence)
	bad := unusable(e, sol)
	for _, a := range append([]string{e.Sentence}, e.Answers...) {
		n := len(strings.Fields(a))
		if n == 0 || n >= len(words) || foldSentence(strings.Join(words[:n], " ")) != foldSentence(a) {
			continue
		}
		// What follows the end of a stored sentence (".", "?" or "!") is never part of the answer. A
		// longer answer without that, e.g. "See you on Monday morning.", is cut only when unusable,
		// since it may be a real other answer.
		if bad || strings.ContainsAny(words[n-1][len(words[n-1])-1:], ".?!") {
			sol.Sentence = strings.Join(words[:n], " ")
			return sol
		}
	}
	return sol
}

const (
	missingNote = "AI không trả lời câu này, hãy tự kiểm tra."
	garbledNote = "AI trả lời sai dạng hai lần nên chưa kiểm tra được câu này. Hãy tự kiểm tra hoặc bấm kiểm tra lại."
)

// unusable reports whether the AI's answer to e is no answer at all, e.g. the model's notes written
// into the sentence field. Quoting that text would only confuse the admin.
func unusable(e Exercise, sol ai.Solution) bool {
	switch e.Kind {
	case KindReorder:
		return !usesOnlyTiles(sol.Sentence, e.Words)
	case KindFix:
		// A fix changes a few words, so an answer much longer than every stored correction carries
		// something else.
		longest := 0
		for _, a := range append([]string{e.Sentence}, e.Answers...) {
			longest = max(longest, len(strings.Fields(a)))
		}
		return utf8.RuneCountInString(sol.Sentence) > maxText || len(strings.Fields(sol.Sentence)) > longest+maxFixExtraWords
	}
	return false
}

// usesOnlyTiles reports whether every word of sentence is one of the tiles, each used at most once.
func usesOnlyTiles(sentence string, tiles []string) bool {
	left := map[string]int{}
	for _, w := range tiles {
		left[tileKey(w)]++
	}
	words := strings.Fields(sentence)
	for _, w := range words {
		if left[tileKey(w)] == 0 {
			return false
		}
		left[tileKey(w)]--
	}
	return len(words) > 0
}

func foldAnswer(s string) string { return strings.ToLower(normalizeAnswer(s)) }

// foldSentence ignores case, spacing, curly apostrophes and a final full stop or question mark.
func foldSentence(s string) string {
	return strings.TrimRight(foldAnswer(s), ".?! ")
}
