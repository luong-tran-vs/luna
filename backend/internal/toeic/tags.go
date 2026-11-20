package toeic

import "slices"

// Tag is a question type ("#dạng câu"): what the result page groups by and what the learner can
// practise on its own.
type Tag struct {
	Code    string
	LabelVi string
	Parts   []Part
	// GrammarPoints are the grammar points of the syllabus that teach it (the first is linked).
	GrammarPoints []string
}

// Tags is the fixed list of question types (research R6).
var Tags = []Tag{
	{Code: "photo-people", LabelVi: "Ảnh có người", Parts: []Part{1}},
	{Code: "photo-objects", LabelVi: "Ảnh đồ vật", Parts: []Part{1}},
	{Code: "photo-scene", LabelVi: "Ảnh khung cảnh", Parts: []Part{1}},

	{Code: "q-wh", LabelVi: "Câu hỏi WH", Parts: []Part{2}, GrammarPoints: []string{"a1-question-words"}},
	{Code: "q-yesno", LabelVi: "Câu hỏi Yes/No", Parts: []Part{2}},
	{Code: "q-choice", LabelVi: "Câu hỏi lựa chọn", Parts: []Part{2}},
	{Code: "q-request", LabelVi: "Lời đề nghị, yêu cầu", Parts: []Part{2}, GrammarPoints: []string{"a2-can-could"}},
	{Code: "q-statement", LabelVi: "Câu trần thuật", Parts: []Part{2}},
	{Code: "indirect-answer", LabelVi: "Trả lời gián tiếp", Parts: []Part{2}},

	{Code: "main-idea", LabelVi: "Ý chính", Parts: []Part{3, 4, 7}},
	{Code: "detail", LabelVi: "Chi tiết", Parts: []Part{3, 4, 7}},
	{Code: "inference", LabelVi: "Suy luận", Parts: []Part{3, 4, 7}},
	{Code: "speaker-intent", LabelVi: "Ý định người nói", Parts: []Part{3, 4, 7}},
	{Code: "next-action", LabelVi: "Việc sẽ làm tiếp", Parts: []Part{3, 4}},
	{Code: "graphic", LabelVi: "Câu hỏi biểu đồ", Parts: []Part{3, 4}},
	{Code: "who-where", LabelVi: "Ai, ở đâu", Parts: []Part{3, 4}},

	{Code: "word-form", LabelVi: "Từ loại", Parts: []Part{5, 6}, GrammarPoints: []string{"a2-adverbs-manner", "a2-adjectives"}},
	{Code: "verb-tense", LabelVi: "Thì của động từ", Parts: []Part{5, 6},
		GrammarPoints: []string{"b1-present-perfect", "b1-present-perfect-vs-past", "b2-future-forms", "b2-narrative-tenses"}},
	{Code: "verb-form", LabelVi: "Dạng động từ", Parts: []Part{5, 6},
		GrammarPoints: []string{"b1-gerund-infinitive", "b1-passive", "b2-participle-clauses"}},
	{Code: "preposition", LabelVi: "Giới từ", Parts: []Part{5, 6}, GrammarPoints: []string{"a1-prepositions-time", "a1-prepositions-place"}},
	{Code: "conjunction", LabelVi: "Liên từ", Parts: []Part{5, 6}, GrammarPoints: []string{"b2-linking-words", "a1-conjunctions"}},
	{Code: "pronoun", LabelVi: "Đại từ", Parts: []Part{5, 6},
		GrammarPoints: []string{"b1-reflexive-pronouns", "a2-possessive-pronouns", "a1-pronouns"}},
	{Code: "relative", LabelVi: "Mệnh đề quan hệ", Parts: []Part{5, 6}, GrammarPoints: []string{"b1-relative-clauses", "b2-non-defining-relative"}},
	{Code: "comparison", LabelVi: "So sánh", Parts: []Part{5, 6}, GrammarPoints: []string{"a2-comparatives", "a2-superlatives"}},
	{Code: "vocabulary", LabelVi: "Từ vựng", Parts: []Part{5, 6}},
	{Code: "collocation", LabelVi: "Cụm từ cố định", Parts: []Part{5, 6}, GrammarPoints: []string{"c2-idioms-collocations"}},
	{Code: "sentence-insert", LabelVi: "Chèn câu", Parts: []Part{6, 7}},

	{Code: "not-true", LabelVi: "Điều không đúng", Parts: []Part{7}},
	{Code: "vocabulary-in-context", LabelVi: "Nghĩa của từ trong bài", Parts: []Part{7}},
	{Code: "cross-reference", LabelVi: "Đối chiếu nhiều văn bản", Parts: []Part{7}},
}

// TagByCode returns the tag of a code.
func TagByCode(code string) (Tag, bool) {
	for _, t := range Tags {
		if t.Code == code {
			return t, true
		}
	}
	return Tag{}, false
}

// TagAllowed reports whether the tag exists and may be used in part p.
func TagAllowed(code string, p Part) bool {
	t, ok := TagByCode(code)
	return ok && slices.Contains(t.Parts, p)
}

// grammarPointOf returns the grammar point linked to a tag, "" when none.
func grammarPointOf(code string) string {
	if t, ok := TagByCode(code); ok && len(t.GrammarPoints) > 0 {
		return t.GrammarPoints[0]
	}
	return ""
}

// labelOf returns the Vietnamese label of a tag, the code itself when unknown.
func labelOf(code string) string {
	if t, ok := TagByCode(code); ok {
		return t.LabelVi
	}
	return code
}
