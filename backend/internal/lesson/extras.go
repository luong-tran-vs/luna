package lesson

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/luongtran/luna/backend/internal/ai"
)

// Limits of the comprehension questions, grammar note and writing prompt (F15).
const (
	maxQuestions      = 5
	questionOptions   = 4
	maxPrompt         = 300
	maxOption         = 150
	maxExplanation    = 500
	maxGrammarTitle   = 100
	maxGrammarBody    = 1500
	maxGrammarExample = 300
	maxExamples       = 3
	maxWritingPrompt  = 500
)

// CleanExtras keeps the usable parts of what the AI wrote with the annotations: valid
// questions (at most 5), a grammar note whose examples really appear in the lesson, and the
// writing prompt. A broken part is left empty; it never fails the annotation.
func CleanExtras(x ai.LessonExtras, content string) Extras {
	out := Extras{Questions: []Question{}, WritingPrompt: truncate(strings.TrimSpace(x.WritingPrompt), maxWritingPrompt)}

	seen := map[string]bool{}
	for _, aq := range x.Questions {
		q := Question{
			Prompt:        strings.TrimSpace(aq.Prompt),
			Options:       trimAll(aq.Options),
			AnswerIndex:   aq.AnswerIndex,
			ExplanationVi: strings.TrimSpace(aq.ExplanationVi),
		}
		if len(questionErrors(q)) > 0 || seen[textKey(q.Prompt)] {
			continue
		}
		seen[textKey(q.Prompt)] = true
		out.Questions = append(out.Questions, q)
		if len(out.Questions) == maxQuestions {
			break
		}
	}

	if n := x.GrammarNote; n != nil {
		note := GrammarNote{Title: strings.TrimSpace(n.Title), BodyVi: strings.TrimSpace(n.BodyVi)}
		for _, e := range trimAll(n.Examples) {
			if e != "" && utf8.RuneCountInString(e) <= maxGrammarExample && inContent(e, content) {
				note.Examples = append(note.Examples, e)
			}
			if len(note.Examples) == maxExamples {
				break
			}
		}
		if note.Title != "" && utf8.RuneCountInString(note.Title) <= maxGrammarTitle &&
			note.BodyVi != "" && utf8.RuneCountInString(note.BodyVi) <= maxGrammarBody && len(note.Examples) > 0 {
			out.GrammarNote = &note
		}
	}
	return out
}

// questionErrors checks one trimmed question; keys are relative to the question
// ("prompt", "options.2", "answerIndex", "explanationVi").
func questionErrors(q Question) map[string]string {
	fields := map[string]string{}
	switch n := utf8.RuneCountInString(q.Prompt); {
	case n == 0:
		fields["prompt"] = "Vui lòng nhập câu hỏi"
	case n > maxPrompt:
		fields["prompt"] = "Tối đa 300 ký tự"
	}
	if len(q.Options) != questionOptions {
		fields["options"] = "Cần đúng 4 lựa chọn"
	}
	seen := map[string]bool{}
	for j, o := range q.Options {
		key := "options." + strconv.Itoa(j)
		switch n := utf8.RuneCountInString(o); {
		case n == 0:
			fields[key] = "Vui lòng nhập lựa chọn"
		case n > maxOption:
			fields[key] = "Tối đa 150 ký tự"
		case seen[textKey(o)]:
			fields[key] = "Lựa chọn bị trùng"
		}
		seen[textKey(o)] = true
	}
	if q.AnswerIndex < 0 || q.AnswerIndex >= questionOptions {
		fields["answerIndex"] = "Chọn đáp án đúng"
	}
	switch n := utf8.RuneCountInString(q.ExplanationVi); {
	case n == 0:
		fields["explanationVi"] = "Vui lòng nhập giải thích"
	case n > maxExplanation:
		fields["explanationVi"] = "Tối đa 500 ký tự"
	}
	return fields
}

// inContent reports whether example appears in content, ignoring case, extra spaces and the
// example's final punctuation.
func inContent(example, content string) bool {
	needle := strings.TrimRightFunc(textKey(example), unicode.IsPunct)
	return needle != "" && strings.Contains(textKey(content), needle)
}

// textKey compares texts ignoring case and extra spaces.
func textKey(s string) string {
	return strings.ToLower(strings.Join(strings.Fields(s), " "))
}

func trimAll(items []string) []string {
	out := make([]string, len(items))
	for i, s := range items {
		out[i] = strings.TrimSpace(s)
	}
	return out
}

func truncate(s string, limit int) string {
	if utf8.RuneCountInString(s) <= limit {
		return s
	}
	return strings.TrimSpace(string([]rune(s)[:limit]))
}

// ExtrasInput is what an admin saves on the lesson page (F15).
type ExtrasInput struct {
	Questions     []Question
	GrammarNote   *GrammarNote
	WritingPrompt string
}

// ValidateExtras trims an admin's extras and checks them against the lesson content. Unlike
// CleanExtras it reports every problem by field ("questions.1.options.2",
// "grammarNote.examples.0", ...) instead of dropping parts.
func ValidateExtras(in ExtrasInput, content string) (Extras, error) {
	out := Extras{Questions: make([]Question, 0, len(in.Questions)), WritingPrompt: strings.TrimSpace(in.WritingPrompt)}
	fields := map[string]string{}

	if len(in.Questions) > maxQuestions {
		fields["questions"] = "Tối đa 5 câu hỏi"
	}
	seen := map[string]bool{}
	for i, q := range in.Questions {
		q = Question{
			Prompt: strings.TrimSpace(q.Prompt), Options: trimAll(q.Options),
			AnswerIndex: q.AnswerIndex, ExplanationVi: strings.TrimSpace(q.ExplanationVi),
		}
		prefix := "questions." + strconv.Itoa(i) + "."
		for k, msg := range questionErrors(q) {
			fields[prefix+k] = msg
		}
		if key := textKey(q.Prompt); key != "" && seen[key] {
			fields[prefix+"prompt"] = "Câu hỏi bị trùng"
		} else {
			seen[key] = true
		}
		out.Questions = append(out.Questions, q)
	}

	if n := in.GrammarNote; n != nil {
		note := GrammarNote{Title: strings.TrimSpace(n.Title), BodyVi: strings.TrimSpace(n.BodyVi), Examples: trimAll(n.Examples)}
		switch l := utf8.RuneCountInString(note.Title); {
		case l == 0:
			fields["grammarNote.title"] = "Vui lòng nhập tiêu đề"
		case l > maxGrammarTitle:
			fields["grammarNote.title"] = "Tối đa 100 ký tự"
		}
		switch l := utf8.RuneCountInString(note.BodyVi); {
		case l == 0:
			fields["grammarNote.bodyVi"] = "Vui lòng nhập giải thích"
		case l > maxGrammarBody:
			fields["grammarNote.bodyVi"] = "Tối đa 1500 ký tự"
		}
		if len(note.Examples) == 0 || len(note.Examples) > maxExamples {
			fields["grammarNote.examples"] = "Cần 1–3 ví dụ"
		}
		for k, e := range note.Examples {
			key := "grammarNote.examples." + strconv.Itoa(k)
			switch {
			case e == "":
				fields[key] = "Vui lòng nhập ví dụ"
			case utf8.RuneCountInString(e) > maxGrammarExample:
				fields[key] = "Tối đa 300 ký tự"
			case !inContent(e, content):
				fields[key] = "Ví dụ phải có trong bài"
			}
		}
		out.GrammarNote = &note
	}

	if utf8.RuneCountInString(out.WritingPrompt) > maxWritingPrompt {
		fields["writingPrompt"] = "Tối đa 500 ký tự"
	}
	if len(fields) > 0 {
		return Extras{}, &ValidationError{Fields: fields}
	}
	return out, nil
}

// sameQuestions reports whether two question sets are identical, so saving other extras keeps
// the learners' answers.
func sameQuestions(a, b []Question) bool {
	return slices.EqualFunc(a, b, func(x, y Question) bool {
		return x.Prompt == y.Prompt && slices.Equal(x.Options, y.Options) &&
			x.AnswerIndex == y.AnswerIndex && x.ExplanationVi == y.ExplanationVi
	})
}

// UpdateExtras saves an admin's questions, grammar note and writing prompt. Changing the
// questions starts a new question set, so earlier answers no longer apply.
func (s *Service) UpdateExtras(ctx context.Context, id string, in ExtrasInput) (Lesson, error) {
	l, err := s.Lessons.Get(ctx, id)
	if err != nil {
		return Lesson{}, err
	}
	if l.AnnotationStatus == StatusRunning {
		return Lesson{}, ErrAnnotationRunning
	}
	x, err := ValidateExtras(in, l.Content)
	if err != nil {
		return Lesson{}, err
	}
	if err := s.Lessons.ReplaceExtras(ctx, id, x, !sameQuestions(l.Extras.Questions, x.Questions)); err != nil {
		return Lesson{}, fmt.Errorf("lesson: replace extras: %w", err)
	}
	s.keepReview(ctx, l, AreaQuestion, questionKeys(l.Extras.Questions), questionKeys(x.Questions))
	return s.Lessons.Get(ctx, id)
}
