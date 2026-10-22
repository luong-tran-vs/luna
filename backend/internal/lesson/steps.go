package lesson

import (
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/luongtran/luna/backend/internal/ai"
	"github.com/luongtran/luna/backend/internal/grammar"
)

// The six-step lesson (added 2026-10-22, design/de-xuat-quy-trinh-bai-hoc.png): Từ vựng → Nghe →
// Đọc → Ngữ pháp → Nói → Viết. The content of the new steps comes with the vocabulary practice,
// in the same AI request, and is kept in Practice.Steps.

// Limits of the steps' content.
const (
	listenQuestions     = 4
	minListenOptions    = 3
	maxListenOptions    = 4
	maxReadingGuide     = 200
	minNoticing         = 2
	maxNoticing         = 4
	maxGrammarOptions   = 4
	maxFormRows         = 5
	maxStepMistakes     = 3
	stepExercises       = 4
	minStepExercises    = 2
	readAloudCount      = 2
	maxReadAloudWords   = 16
	speakQuestionsCount = 3
	maxAnyOf            = 10
	maxAnyOfWord        = 30
	outlineHints        = 3
	maxModelAnswer      = 800
	dictationCount      = 3
	maxDictationWords   = 8
	maxFallbackWords    = 10
)

// ListenTFNOptions are the options of a true / false / not said question of the Listening step,
// in answer order.
var ListenTFNOptions = []string{"Đúng", "Sai", "Không nói đến"}

// Steps is the content of the Listening, Grammar and Speaking steps and the help of the Reading
// and Writing steps. Any part may be empty: the page then shows the step without it.
type Steps struct {
	ListeningQuestions []ListenQuestion
	ReadingGuideVi     string
	Grammar            *GrammarStep
	ReadAloud          []ReadAloud
	SpeakQuestions     []SpeakQuestion
	Writing            *WritingHelp
	// GrammarPointID is the syllabus point the AI picked for a lesson that had none; "" otherwise.
	GrammarPointID string
}

// ListenQuestion is a question on the dialogue; see ai.ListenQuestion.
type ListenQuestion struct {
	Kind          string
	Prompt        string
	PromptVi      string
	Options       []string
	AnswerIndex   int
	ExplanationVi string
	TurnIndex     int
}

// Noticing is a sentence of the lesson with the words that show the grammar point.
type Noticing struct {
	Sentence   string
	Highlights []string
}

// FormRow is one line of the rule table.
type FormRow struct {
	Subject, Form, Negative string
}

// MistakePair is a common mistake and its correction.
type MistakePair struct {
	Wrong, Right string
}

// GrammarStep is the grammar step of a lesson: noticing, a question, the rule and exercises.
type GrammarStep struct {
	TitleVi       string
	Noticing      []Noticing
	QuestionVi    string
	Options       []string
	AnswerIndex   int
	ExplanationVi string
	WhenVi        string
	Table         []FormRow
	Mistakes      []MistakePair
	NoteVi        string
	Exercises     []grammar.Exercise
}

// ReadAloud is a sentence to repeat with a pronunciation tip.
type ReadAloud struct {
	Text, TipVi string
}

// SpeakQuestion is a question the learner answers aloud, with an answer frame.
type SpeakQuestion struct {
	Question, QuestionVi, Frame string
}

// WritingHelp is the grammar requirement, outline and model answer of the Writing step.
type WritingHelp struct {
	GrammarLabelVi string
	AnyOf          []string
	OutlineVi      []string
	ModelAnswer    string
}

// CleanSteps keeps the parts of the AI's steps that pass their checks; l is the lesson the practice
// belongs to and d its cleaned dialogue (the listening questions are on it).
func CleanSteps(p ai.Practice, l Lesson, d *Dialogue) *Steps {
	texts := []string{l.Content}
	if d != nil {
		for _, t := range d.Turns {
			texts = append(texts, t.Text)
		}
	}
	s := &Steps{
		ListeningQuestions: cleanListenQuestions(p.ListeningQuestions, d),
		ReadingGuideVi:     limited(p.ReadingGuideVi, maxReadingGuide),
		Grammar:            cleanGrammarStep(p.GrammarStep, texts),
		ReadAloud:          cleanReadAloud(p.ReadAloud, texts),
		SpeakQuestions:     cleanSpeakQuestions(p.SpeakQuestions),
		Writing:            cleanWritingHelp(p.WritingHelp),
	}
	if id := strings.TrimSpace(p.GrammarPointID); l.GrammarPointID == "" && id != "" && checkGrammarPoint(id, l.Level) == nil {
		s.GrammarPointID = id
	}
	return s
}

func cleanListenQuestions(in []ai.ListenQuestion, d *Dialogue) []ListenQuestion {
	if d == nil {
		return nil
	}
	var out []ListenQuestion
	seen := map[string]bool{}
	for _, q := range in {
		c := ListenQuestion{
			Kind: strings.TrimSpace(q.Kind), Prompt: limited(q.Prompt, maxPrompt), PromptVi: limited(q.PromptVi, maxPrompt),
			AnswerIndex: q.AnswerIndex, ExplanationVi: limited(q.ExplanationVi, maxExplanation), TurnIndex: q.TurnIndex,
		}
		switch c.Kind {
		case KindTFN:
			c.Options = slices.Clone(ListenTFNOptions)
		case KindGist, KindDetail:
			c.Options = cleanOptions(q.Options)
			if len(c.Options) < minListenOptions || len(c.Options) > maxListenOptions {
				continue
			}
		default:
			continue
		}
		if c.Prompt == "" || c.ExplanationVi == "" || c.AnswerIndex < 0 || c.AnswerIndex >= len(c.Options) || seen[textKey(c.Prompt)] {
			continue
		}
		if c.TurnIndex < -1 || c.TurnIndex >= len(d.Turns) {
			c.TurnIndex = -1
		}
		seen[textKey(c.Prompt)] = true
		out = append(out, c)
		if len(out) == listenQuestions {
			break
		}
	}
	return out
}

// cleanOptions trims the options and returns nil when one is empty, too long or repeated.
func cleanOptions(in []string) []string {
	out := make([]string, 0, len(in))
	seen := map[string]bool{}
	for _, o := range in {
		o = collapse(o)
		if o == "" || utf8.RuneCountInString(o) > maxOption || seen[textKey(o)] {
			return nil
		}
		seen[textKey(o)] = true
		out = append(out, o)
	}
	return out
}

// inTexts reports whether sentence appears in one of texts, ignoring case, spaces and its final
// punctuation.
func inTexts(sentence string, texts []string) bool {
	return slices.ContainsFunc(texts, func(t string) bool { return inContent(sentence, t) })
}

func cleanGrammarStep(g *ai.GrammarStep, texts []string) *GrammarStep {
	if g == nil {
		return nil
	}
	out := &GrammarStep{
		TitleVi: limited(g.TitleVi, maxGrammarTitle), QuestionVi: limited(g.QuestionVi, maxPrompt),
		Options: cleanOptions(g.Options), AnswerIndex: g.AnswerIndex, ExplanationVi: limited(g.ExplanationVi, maxExplanation),
		WhenVi: limited(g.WhenVi, maxExplanation), NoteVi: limited(g.NoteVi, maxExplanation),
		Exercises: grammar.CleanAIExercises(g.Exercises, "g", stepExercises),
	}
	if out.TitleVi == "" || len(out.Exercises) < minStepExercises {
		return nil
	}
	for _, n := range g.Noticing {
		sentence := limited(n.Sentence, maxPracticeText)
		if sentence == "" || !inTexts(sentence, texts) {
			continue
		}
		var hs []string
		for _, h := range n.Highlights {
			h = collapse(h)
			if h != "" && strings.Contains(strings.ToLower(sentence), strings.ToLower(h)) &&
				!slices.ContainsFunc(hs, func(x string) bool { return strings.EqualFold(x, h) }) {
				hs = append(hs, h)
			}
		}
		if len(hs) > 0 {
			out.Noticing = append(out.Noticing, Noticing{Sentence: sentence, Highlights: hs})
		}
		if len(out.Noticing) == maxNoticing {
			break
		}
	}
	if len(out.Noticing) < minNoticing {
		out.Noticing = nil
	}
	// The question needs the sentences to look at.
	if out.Noticing == nil || out.QuestionVi == "" || out.ExplanationVi == "" || len(out.Options) < 2 ||
		len(out.Options) > maxGrammarOptions || out.AnswerIndex < 0 || out.AnswerIndex >= len(out.Options) {
		out.QuestionVi, out.Options, out.AnswerIndex, out.ExplanationVi = "", nil, 0, ""
	}
	for _, r := range g.Table {
		row := FormRow{Subject: limited(r.Subject, maxOption), Form: limited(r.Form, maxOption), Negative: limited(r.Negative, maxOption)}
		if row.Subject != "" && row.Form != "" {
			out.Table = append(out.Table, row)
		}
		if len(out.Table) == maxFormRows {
			break
		}
	}
	for _, m := range g.Mistakes {
		pair := MistakePair{Wrong: limited(m.Wrong, maxPracticeText), Right: limited(m.Right, maxPracticeText)}
		if pair.Wrong != "" && pair.Right != "" && textKey(pair.Wrong) != textKey(pair.Right) {
			out.Mistakes = append(out.Mistakes, pair)
		}
		if len(out.Mistakes) == maxStepMistakes {
			break
		}
	}
	return out
}

func cleanReadAloud(in []ai.ReadAloud, texts []string) []ReadAloud {
	var out []ReadAloud
	for _, r := range in {
		text := limited(r.Text, maxPracticeText)
		if text == "" || len(strings.Fields(text)) > maxReadAloudWords || !inTexts(text, texts) {
			continue
		}
		out = append(out, ReadAloud{Text: text, TipVi: limited(r.TipVi, maxExplanation)})
		if len(out) == readAloudCount {
			break
		}
	}
	return out
}

func cleanSpeakQuestions(in []ai.SpeakQuestion) []SpeakQuestion {
	var out []SpeakQuestion
	for _, q := range in {
		c := SpeakQuestion{Question: limited(q.Question, maxPrompt), QuestionVi: limited(q.QuestionVi, maxPrompt), Frame: limited(q.Frame, maxPrompt)}
		if c.Question == "" {
			continue
		}
		out = append(out, c)
		if len(out) == speakQuestionsCount {
			break
		}
	}
	return out
}

func cleanWritingHelp(w *ai.WritingHelp) *WritingHelp {
	if w == nil {
		return nil
	}
	out := &WritingHelp{GrammarLabelVi: limited(w.GrammarLabelVi, maxGrammarTitle), ModelAnswer: limited(w.ModelAnswer, maxModelAnswer)}
	for _, a := range w.AnyOf {
		a = strings.ToLower(collapse(a))
		if a != "" && utf8.RuneCountInString(a) <= maxAnyOfWord && !slices.Contains(out.AnyOf, a) {
			out.AnyOf = append(out.AnyOf, a)
		}
		if len(out.AnyOf) == maxAnyOf {
			break
		}
	}
	if out.GrammarLabelVi == "" || len(out.AnyOf) == 0 {
		out.GrammarLabelVi, out.AnyOf = "", nil
	}
	for _, h := range w.OutlineVi {
		if h = limited(h, maxPracticeText); h != "" {
			out.OutlineVi = append(out.OutlineVi, h)
		}
		if len(out.OutlineVi) == outlineHints {
			break
		}
	}
	if out.GrammarLabelVi == "" && len(out.OutlineVi) == 0 && out.ModelAnswer == "" {
		return nil
	}
	return out
}

// DictationItem is a sentence of the Listening step to write down.
type DictationItem struct {
	Text string
	// Lemma is the lesson word the sentence was written for; "" for a lesson sentence.
	Lemma string
}

// Dictation lists the sentences of the Listening step: the shortest example sentences (one per
// word, at most maxDictationWords words), else the shortest lesson sentences. Lessons without a
// practice keep every sentence, as before the six steps.
func Dictation(l Lesson) []DictationItem {
	if l.Practice == nil || l.Practice.Steps == nil {
		out := make([]DictationItem, len(l.Sentences))
		for i, s := range l.Sentences {
			out[i] = DictationItem{Text: s.Text}
		}
		return out
	}
	var out []DictationItem
	pick := func(items []DictationItem, limit int) {
		items = slices.Clone(items)
		slices.SortStableFunc(items, func(a, b DictationItem) int {
			return len(strings.Fields(a.Text)) - len(strings.Fields(b.Text))
		})
		for _, it := range items {
			n := len(strings.Fields(it.Text))
			if len(out) == dictationCount || n > limit {
				return
			}
			if n >= 3 && !slices.ContainsFunc(out, func(o DictationItem) bool { return sentenceKey(o.Text) == sentenceKey(it.Text) }) {
				out = append(out, it)
			}
		}
	}
	var examples []DictationItem
	for _, e := range l.Practice.Examples {
		examples = append(examples, DictationItem{Text: e.Sentence, Lemma: e.Lemma})
	}
	pick(examples, maxDictationWords)
	var sentences []DictationItem
	for _, s := range l.Sentences {
		sentences = append(sentences, DictationItem{Text: s.Text})
	}
	pick(sentences, maxFallbackWords)
	return out
}
