package lesson

import (
	"fmt"
	"slices"
	"strings"

	"github.com/luongtran/luna/backend/internal/ai"
	"github.com/luongtran/luna/backend/internal/grammar"
)

// importRules are the rules of each part of an import, in words, shown when the checks dropped some
// of it (the AI's rules: see CleanAnnotations, CleanExtras, CleanPractice, CleanSteps, CleanGlosses).
var importRules = map[string]string{
	"annotate.annotations": "Mỗi chú thích cần text, lemma, meaningVi; text phải có trong câu. Bài có từ mục tiêu " +
		"thì chỉ giữ chú thích của các từ mục tiêu.",
	"annotate.questions": "Tối đa 5 câu; mỗi câu cần prompt, đúng 4 lựa chọn không trùng (tfn để options trống), " +
		"answerIndex hợp lệ và explanationVi; không trùng prompt.",
	"annotate.grammarNote":           "Cần title, bodyVi và ít nhất 1 ví dụ chép đúng từ bài.",
	"annotate.writingPrompt":         "Không được để trống.",
	"annotate.sentenceTranslations":  "Cần đúng một bản dịch cho mỗi câu của bài (xem số câu của báo cáo).",
	"practice.objectiveVi":           "Không được để trống, tối đa 200 ký tự.",
	"practice.examples":              "Mỗi từ đã chú thích (theo lemma) một câu, tối đa 200 ký tự, chứa đúng từ đó.",
	"practice.dialogue.turns":        "Đúng 2 speakers; mỗi lượt cần speaker 0 hoặc 1, text và meaningVi; ít nhất 4 lượt hợp lệ.",
	"practice.listeningQuestions":    "Cần hội thoại hợp lệ; tối đa 4 câu; gist, detail có 3–4 lựa chọn, tfn để options trống; cần prompt, explanationVi và answerIndex hợp lệ.",
	"practice.readingGuideVi":        "Không được để trống, tối đa 200 ký tự.",
	"practice.grammarStep":           "Cần titleVi và ít nhất 2 bài tập hợp lệ.",
	"practice.grammarStep.exercises": "Tối đa 4 bài: fill (text có ___, answers), choice (text có ___, đúng 4 options, answerIndex), fix (text sai, sentence đúng), reorder (text là nghĩa tiếng Việt, sentence, words).",
	"practice.translations":          "Tối đa 5 câu; vi không trống; en có 2–15 từ và chứa ít nhất một từ đã chú thích.",
	"practice.readAloud":             "Tối đa 2 câu, mỗi câu tối đa 16 từ, chép đúng từ bài hoặc hội thoại.",
	"practice.speakQuestions":        "Tối đa 3 câu, mỗi câu cần question.",
	"practice.writingHelp":           "Cần grammarLabelVi kèm anyOf, hoặc outlineVi, hoặc modelAnswer.",
	"glosses":                        "sentenceIndex hợp lệ, text có trong câu đó, meaningVi tối đa 80 ký tự, mỗi lemma một lần trong một câu.",
}

// explainImport adds to each part that lost something its rule and, for the parts that most often
// go wrong, the problem of each dropped item.
func explainImport(parts []ImportPart, c ImportContent, sentences []string, content string, focus []string, only bool) {
	for i := range parts {
		p := &parts[i]
		if p.Kept >= p.Given {
			continue
		}
		p.RuleVi = importRules[p.Key]
		switch p.Key {
		case "annotate.annotations":
			p.Reasons = annotationReasons(c.Annotate.Annotations, sentences, focus, only)
		case "annotate.questions":
			p.Reasons = questionReasons(c.Annotate.Questions, content)
		case "practice.grammarStep.exercises":
			if c.Practice.GrammarStep != nil {
				p.Reasons = grammar.ExplainAIExercises(c.Practice.GrammarStep.Exercises)
			}
		}
	}
}

func annotationReasons(items []ai.Annotation, sentences, focus []string, only bool) []string {
	var out []string
	for i, it := range items {
		text, lemma := strings.TrimSpace(it.Text), strings.TrimSpace(it.Lemma)
		switch _, found := findSentence(text, it.SentenceIndex, sentences); {
		case text == "" || lemma == "" || strings.TrimSpace(it.MeaningVi) == "":
			out = append(out, fmt.Sprintf("annotations[%d]: thiếu text, lemma hoặc meaningVi", i))
		case !found:
			out = append(out, fmt.Sprintf("annotations[%d] %q: không có trong câu nào của bài", i, text))
		case only && !slices.ContainsFunc(focus, func(w string) bool {
			return annotated([]Annotation{{Text: text, Lemma: lemma}}, w)
		}):
			out = append(out, fmt.Sprintf("annotations[%d] %q: không phải từ mục tiêu", i, text))
		}
	}
	return out
}

func questionReasons(items []ai.Question, content string) []string {
	var out []string
	seen := map[string]bool{}
	kept := 0
	for i, aq := range items {
		q := cleanQuestion(Question{
			Kind: aq.Kind, Prompt: aq.Prompt, PromptVi: aq.PromptVi, Quote: aq.Quote, Options: aq.Options,
			AnswerIndex: aq.AnswerIndex, ExplanationVi: aq.ExplanationVi,
		}, content)
		problems := questionErrors(q)
		keys := make([]string, 0, len(problems))
		for k := range problems {
			keys = append(keys, k)
		}
		slices.Sort(keys)
		for _, k := range keys {
			out = append(out, fmt.Sprintf("questions[%d]: %s: %s", i, k, problems[k]))
		}
		switch {
		case len(problems) > 0:
		case seen[textKey(q.Prompt)]:
			out = append(out, fmt.Sprintf("questions[%d]: trùng câu hỏi", i))
		case kept == maxQuestions:
			out = append(out, fmt.Sprintf("questions[%d]: quá %d câu", i, maxQuestions))
		default:
			seen[textKey(q.Prompt)] = true
			kept++
		}
	}
	return out
}
