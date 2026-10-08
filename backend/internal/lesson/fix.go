package lesson

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/luongtran/luna/backend/internal/ai"
)

// ErrFixTarget means the lesson has no item at that area and index.
var ErrFixTarget = errors.New("lesson: no item to fix")

// FixSuggestion is the AI's corrected version of one flagged item (F22); only the fields of its
// area are set. The admin may edit it before applying it.
type FixSuggestion struct {
	Area      FlagArea
	Index     int
	Text      string
	MeaningVi string
	Question  *Question
	Vi        string
	En        string
	NoteVi    string
}

// FixInput is a correction the admin applies to one item: the fields of its area.
type FixInput struct {
	Area      FlagArea
	Index     int
	Text      string
	MeaningVi string
	Question  *Question
	Vi        string
	En        string
}

// SuggestFix asks the AI for a corrected version of the item at area and index, using the note of
// its flag (if any) as the problem to fix. Nothing is saved.
func (s *Service) SuggestFix(ctx context.Context, id string, area FlagArea, index int) (FixSuggestion, error) {
	l, err := s.Lessons.Get(ctx, id)
	if err != nil {
		return FixSuggestion{}, err
	}
	req, err := fixRequest(l, area, index)
	if err != nil {
		return FixSuggestion{}, err
	}
	aiCtx, cancel := context.WithTimeout(ctx, reviewTimeout)
	defer cancel()
	res, err := s.AI.SuggestFix(aiCtx, req)
	if err != nil {
		if errors.Is(err, ai.ErrNotConfigured) || errors.Is(err, ai.ErrInvalidKey) || errors.Is(err, ai.ErrQuota) {
			return FixSuggestion{}, err
		}
		return FixSuggestion{}, fmt.Errorf("%w: %w", errReviewAI, err)
	}
	out := FixSuggestion{Area: area, Index: index, NoteVi: strings.TrimSpace(res.NoteVi)}
	switch area {
	case AreaSentence:
		out.Text = strings.TrimSpace(res.Text)
	case AreaAnnotation:
		out.MeaningVi = strings.TrimSpace(res.MeaningVi)
	case AreaQuestion:
		q := Question{
			Prompt: strings.TrimSpace(res.Question.Prompt), AnswerIndex: res.Question.AnswerIndex,
			ExplanationVi: strings.TrimSpace(res.Question.ExplanationVi),
		}
		for _, o := range res.Question.Options {
			q.Options = append(q.Options, strings.TrimSpace(o))
		}
		out.Question = &q
	case AreaTranslation:
		out.Vi, out.En = strings.TrimSpace(res.Vi), strings.TrimSpace(res.En)
	}
	return out, nil
}

// fixRequest describes the item at area and index for the AI; ErrFixTarget when there is none.
func fixRequest(l Lesson, area FlagArea, index int) (ai.FixRequest, error) {
	req := ai.FixRequest{
		Level: string(l.Level), Title: l.Title, Sentences: sentenceTexts(l.Sentences), Area: string(area),
	}
	if l.Review != nil {
		if i := slices.IndexFunc(l.Review.Flags, func(f Flag) bool { return f.Area == area && f.Index == index }); i >= 0 {
			req.Problem = l.Review.Flags[i].NoteVi
		}
	}
	switch {
	case index < 0:
		return req, ErrFixTarget
	case area == AreaSentence && index < len(l.Sentences):
		req.Sentence = index
	case area == AreaAnnotation && index < len(l.Annotations):
		a := l.Annotations[index]
		req.Annotation = &ai.ReviewAnnotation{
			Index: index, Text: a.Text, Lemma: a.Lemma, MeaningVi: a.MeaningVi, SentenceIndex: a.SentenceIndex,
		}
	case area == AreaQuestion && index < len(l.Extras.Questions):
		q := l.Extras.Questions[index]
		req.Question = &ai.FixQuestion{
			Prompt: q.Prompt, Options: q.Options, AnswerIndex: q.AnswerIndex, ExplanationVi: q.ExplanationVi,
		}
	case area == AreaTranslation && l.Practice != nil && index < len(l.Practice.Translations):
		t := l.Practice.Translations[index]
		req.Translation = &ai.ReviewTranslation{Index: index, Vi: t.Vi, En: t.En}
	default:
		return req, ErrFixTarget
	}
	return req, nil
}

// ApplyFix saves the admin's correction of one item through the usual edit of its part, so the
// same checks apply and the other flags of that part are kept. A sentence is changed in the
// content: the lesson is annotated again and its check is dropped, like any content edit.
func (s *Service) ApplyFix(ctx context.Context, id string, in FixInput) (Lesson, error) {
	l, err := s.Lessons.Get(ctx, id)
	if err != nil {
		return Lesson{}, err
	}
	if _, err := fixRequest(l, in.Area, in.Index); err != nil {
		return Lesson{}, err
	}
	switch in.Area {
	case AreaSentence:
		content, ok := replaceSentence(l.Content, l.Sentences, in.Index, in.Text)
		if !ok {
			return Lesson{}, ErrFixTarget
		}
		return s.Update(ctx, id, Input{
			Title: l.Title, Content: content, TopicID: l.TopicID, Level: l.Level,
			Source: l.Source, License: l.License, KeepGrammarPoint: true,
		})
	case AreaAnnotation:
		items := make([]AnnotationInput, len(l.Annotations))
		for i, a := range l.Annotations {
			items[i] = AnnotationInput{Text: a.Text, Lemma: a.Lemma, MeaningVi: a.MeaningVi}
		}
		items[in.Index].MeaningVi = in.MeaningVi
		return s.UpdateAnnotations(ctx, id, items)
	case AreaQuestion:
		if in.Question == nil {
			return Lesson{}, &ValidationError{Fields: map[string]string{"question": "Thiếu câu hỏi"}}
		}
		qs := slices.Clone(l.Extras.Questions)
		qs[in.Index] = *in.Question
		return s.UpdateExtras(ctx, id, ExtrasInput{
			Questions: qs, GrammarNote: l.Extras.GrammarNote, WritingPrompt: l.Extras.WritingPrompt,
		})
	default: // AreaTranslation, checked by fixRequest
		items := make([]TranslationInput, len(l.Practice.Translations))
		for i, t := range l.Practice.Translations {
			items[i] = TranslationInput{Vi: t.Vi, En: t.En, Distractors: t.Distractors}
		}
		items[in.Index].Vi, items[in.Index].En = in.Vi, in.En
		return s.UpdateTranslations(ctx, id, items)
	}
}

// replaceSentence puts text in place of the sentence at index in content, keeping everything
// else (line breaks, paragraphs) as it is. Sentences are found in order, with any whitespace
// between their words; ok is false when they cannot be found.
func replaceSentence(content string, sentences []Sentence, index int, text string) (string, bool) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "", false
	}
	cursor := 0
	for i := 0; i <= index && i < len(sentences); i++ {
		words := strings.Fields(sentences[i].Text)
		if len(words) == 0 {
			return "", false
		}
		for w := range words {
			words[w] = regexp.QuoteMeta(words[w])
		}
		loc := regexp.MustCompile(strings.Join(words, `\s+`)).FindStringIndex(content[cursor:])
		if loc == nil {
			return "", false
		}
		start, end := cursor+loc[0], cursor+loc[1]
		if i == index {
			return content[:start] + text + content[end:], true
		}
		cursor = end
	}
	return "", false
}
