package grammar

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// Limits of a grammar lesson (F20).
const (
	maxText        = 400
	maxExplanation = 1200

	minExplanation, maxExplanationParas = 1, 6
	minUsage, maxUsage                  = 1, 8
	minStructures, maxStructures        = 1, 6
	minExamples, maxExamples            = 3, 12
	maxMistakes                         = 6
	minPractice, maxPractice            = 6, 12
	minMastery, maxMastery              = 5, 10

	minSentenceWords, maxSentenceWords = 3, 14
	maxExtraWords                      = 2
	optionCount                        = 4

	blank = "___"
)

// ValidationError lists invalid input fields with Vietnamese messages, keyed like
// "content.practice[2].options".
type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string { return fmt.Sprintf("grammar: invalid input %v", e.Fields) }

// fieldErrors collects the messages of the first problem of each field.
type fieldErrors map[string]string

func (f fieldErrors) add(key, msg string) {
	if _, ok := f[key]; !ok {
		f[key] = msg
	}
}

// text checks one required string of at most limit characters.
func (f fieldErrors) text(key, s string, limit int, required bool) {
	switch {
	case required && strings.TrimSpace(s) == "":
		f.add(key, "Không được để trống")
	case utf8.RuneCountInString(s) > limit:
		f.add(key, fmt.Sprintf("Tối đa %d ký tự", limit))
	}
}

func (f fieldErrors) count(key string, n, lo, hi int, what string) {
	switch {
	case n < lo:
		f.add(key, fmt.Sprintf("Cần ít nhất %d %s", lo, what))
	case n > hi:
		f.add(key, fmt.Sprintf("Tối đa %d %s", hi, what))
	}
}

// Normalize trims every string of c and gives the exercises without a valid id one ("p1"... and "m1"...). It
// returns a copy; nil lists become empty ones.
func Normalize(c Content) Content {
	out := Content{
		Objective:   strings.TrimSpace(c.Objective),
		Explanation: trimAll(c.Explanation),
		Usage:       trimAll(c.Usage),
		Structures:  make([]Structure, len(c.Structures)),
		Examples:    make([]Example, len(c.Examples)),
		Mistakes:    make([]Mistake, len(c.Mistakes)),
		Practice:    normalizeExercises(c.Practice, "p"),
		Mastery:     normalizeExercises(c.Mastery, "m"),
	}
	for i, s := range c.Structures {
		out.Structures[i] = Structure{Label: strings.TrimSpace(s.Label), Pattern: strings.TrimSpace(s.Pattern), Example: strings.TrimSpace(s.Example)}
	}
	for i, e := range c.Examples {
		out.Examples[i] = Example{En: strings.TrimSpace(e.En), Vi: strings.TrimSpace(e.Vi)}
	}
	for i, m := range c.Mistakes {
		out.Mistakes[i] = Mistake{Wrong: strings.TrimSpace(m.Wrong), Right: strings.TrimSpace(m.Right), NoteVi: strings.TrimSpace(m.NoteVi)}
	}
	return out
}

func trimAll(in []string) []string {
	out := make([]string, len(in))
	for i, s := range in {
		out[i] = strings.TrimSpace(s)
	}
	return out
}

// keepID reports whether id is "<prefix><n>" with n a positive number written without leading zeros.
func keepID(id, prefix string) bool {
	rest, ok := strings.CutPrefix(id, prefix)
	if !ok || rest == "" || rest[0] == '0' {
		return false
	}
	for _, r := range rest {
		if r < '0' || r > '9' {
			return false
		}
	}
	return len(rest) <= 6
}

// assignIDs keeps the valid, unique ids of in and gives the other exercises the smallest unused
// "<prefix><n>", so removing one exercise never renames the others.
func assignIDs(in []Exercise, prefix string) []string {
	ids := make([]string, len(in))
	used := map[string]bool{}
	for i, e := range in {
		id := strings.TrimSpace(e.ID)
		if keepID(id, prefix) && !used[id] {
			ids[i], used[id] = id, true
		}
	}
	next := 1
	for i := range ids {
		if ids[i] != "" {
			continue
		}
		for used[fmt.Sprintf("%s%d", prefix, next)] {
			next++
		}
		ids[i] = fmt.Sprintf("%s%d", prefix, next)
		used[ids[i]] = true
	}
	return ids
}

func normalizeExercises(in []Exercise, idPrefix string) []Exercise {
	out := make([]Exercise, len(in))
	ids := assignIDs(in, idPrefix)
	for i, e := range in {
		e.ID = ids[i]
		e.Kind = Kind(strings.TrimSpace(string(e.Kind)))
		e.PromptVi = strings.TrimSpace(e.PromptVi)
		e.Text = strings.TrimSpace(e.Text)
		e.Sentence = strings.TrimSpace(e.Sentence)
		e.ExplanationVi = strings.TrimSpace(e.ExplanationVi)
		e.Options = trimAll(e.Options)
		e.Answers = trimAll(e.Answers)
		e.Words = trimAll(e.Words)
		// Only the fields of the kind are kept, so a stale field never reaches the learner.
		switch e.Kind {
		case KindChoice:
			e.Answers, e.Words, e.Sentence = nil, nil, ""
		case KindFill:
			e.Options, e.AnswerIndex, e.Words, e.Sentence = nil, 0, nil, ""
		case KindReorder:
			e.Options, e.AnswerIndex, e.Answers = nil, 0, nil
		}
		out[i] = e
	}
	return out
}

// Validate checks a normalized content against the limits of the lesson. Keys are prefixed with
// "content.". It returns nil when the content is valid.
func Validate(c Content) map[string]string {
	f := fieldErrors{}
	f.text("content.objective", c.Objective, maxText, true)

	f.count("content.explanation", len(c.Explanation), minExplanation, maxExplanationParas, "đoạn giải thích")
	for i, p := range c.Explanation {
		f.text(fmt.Sprintf("content.explanation[%d]", i), p, maxExplanation, true)
	}
	f.count("content.usage", len(c.Usage), minUsage, maxUsage, "ý về cách dùng")
	for i, u := range c.Usage {
		f.text(fmt.Sprintf("content.usage[%d]", i), u, maxText, true)
	}
	f.count("content.structures", len(c.Structures), minStructures, maxStructures, "cấu trúc")
	for i, s := range c.Structures {
		k := fmt.Sprintf("content.structures[%d]", i)
		f.text(k+".label", s.Label, maxText, true)
		f.text(k+".pattern", s.Pattern, maxText, true)
		f.text(k+".example", s.Example, maxText, true)
	}
	f.count("content.examples", len(c.Examples), minExamples, maxExamples, "ví dụ")
	for i, e := range c.Examples {
		k := fmt.Sprintf("content.examples[%d]", i)
		f.text(k+".en", e.En, maxText, true)
		f.text(k+".vi", e.Vi, maxText, true)
	}
	f.count("content.mistakes", len(c.Mistakes), 0, maxMistakes, "lỗi thường gặp")
	for i, m := range c.Mistakes {
		k := fmt.Sprintf("content.mistakes[%d]", i)
		f.text(k+".wrong", m.Wrong, maxText, true)
		f.text(k+".right", m.Right, maxText, true)
		f.text(k+".noteVi", m.NoteVi, maxText, true)
	}

	f.count("content.practice", len(c.Practice), minPractice, maxPractice, "bài luyện tập")
	for i, e := range c.Practice {
		for key, msg := range exerciseProblems(e) {
			f.add(fmt.Sprintf("content.practice[%d].%s", i, key), msg)
		}
	}
	f.count("content.mastery", len(c.Mastery), minMastery, maxMastery, "bài kiểm tra")
	for i, e := range c.Mastery {
		for key, msg := range exerciseProblems(e) {
			f.add(fmt.Sprintf("content.mastery[%d].%s", i, key), msg)
		}
	}
	if len(f) == 0 {
		return nil
	}
	return f
}

// exerciseProblems checks one normalized exercise; the keys are the field names ("options",
// "answerIndex", "text"...). It returns nil for a valid exercise.
func exerciseProblems(e Exercise) map[string]string {
	f := fieldErrors{}
	f.text("promptVi", e.PromptVi, maxText, false)
	f.text("explanationVi", e.ExplanationVi, maxText, true)
	f.text("text", e.Text, maxText, true)
	switch e.Kind {
	case KindChoice:
		if len(e.Options) != optionCount {
			f.add("options", "Cần đúng 4 đáp án")
		} else {
			seen := map[string]bool{}
			for _, o := range e.Options {
				f.text("options", o, maxText, true)
				key := strings.ToLower(o)
				if seen[key] {
					f.add("options", "Các đáp án phải khác nhau")
				}
				seen[key] = true
			}
		}
		if e.AnswerIndex < 0 || e.AnswerIndex >= optionCount {
			f.add("answerIndex", "Đáp án đúng phải từ 0 đến 3")
		}
	case KindFill:
		if strings.Count(e.Text, blank) != 1 {
			f.add("text", "Câu phải có đúng một chỗ trống ___")
		}
		if len(e.Answers) == 0 {
			f.add("answers", "Cần ít nhất một đáp án")
		}
		for _, a := range e.Answers {
			f.text("answers", a, maxText, true)
		}
	case KindReorder:
		checkReorder(f, e)
	default:
		f.add("kind", "Loại bài tập không hợp lệ")
	}
	if len(f) == 0 {
		return nil
	}
	return f
}

func checkReorder(f fieldErrors, e Exercise) {
	sentence := strings.Fields(e.Sentence)
	switch {
	case e.Sentence == "":
		f.add("sentence", "Không được để trống")
		return
	case utf8.RuneCountInString(e.Sentence) > maxText:
		f.add("sentence", fmt.Sprintf("Tối đa %d ký tự", maxText))
		return
	case len(sentence) < minSentenceWords || len(sentence) > maxSentenceWords:
		f.add("sentence", fmt.Sprintf("Câu cần từ %d đến %d từ", minSentenceWords, maxSentenceWords))
		return
	}
	left := map[string]int{}
	for _, w := range sentence {
		left[w]++
	}
	extra := 0
	for _, w := range e.Words {
		switch {
		case w == "" || strings.ContainsAny(w, " \t\n"):
			f.add("words", "Mỗi ô phải là một từ")
			return
		case left[w] > 0:
			left[w]--
		default:
			extra++
		}
	}
	missing := 0
	for _, n := range left {
		missing += n
	}
	switch {
	case missing > 0:
		f.add("words", "Các ô phải gồm đủ các từ của câu đúng")
	case extra > maxExtraWords:
		f.add("words", "Tối đa 2 từ gây nhiễu")
	}
}

// FilterGenerated cleans AI content: it normalizes the strings, drops the exercises that fail
// their checks (and the mastery exercises that repeat a practice one), and cuts the lists to
// their limits. The result may still be invalid (too few exercises); run Validate on it.
func FilterGenerated(c Content) Content {
	c = Normalize(c)
	c.Explanation = keepFirst(nonEmpty(c.Explanation), maxExplanationParas)
	c.Usage = keepFirst(nonEmpty(c.Usage), maxUsage)

	var structures []Structure
	for _, s := range c.Structures {
		if s.Label != "" && s.Pattern != "" && s.Example != "" {
			structures = append(structures, s)
		}
	}
	c.Structures = keepFirst(structures, maxStructures)
	var examples []Example
	for _, e := range c.Examples {
		if e.En != "" && e.Vi != "" {
			examples = append(examples, e)
		}
	}
	c.Examples = keepFirst(examples, maxExamples)
	var mistakes []Mistake
	for _, m := range c.Mistakes {
		if m.Wrong != "" && m.Right != "" && m.NoteVi != "" {
			mistakes = append(mistakes, m)
		}
	}
	c.Mistakes = keepFirst(mistakes, maxMistakes)

	seen := map[string]bool{}
	c.Practice = keepFirst(validExercises(c.Practice, seen, false), maxPractice)
	c.Mastery = keepFirst(validExercises(c.Mastery, seen, true), maxMastery)
	return Normalize(c)
}

// validExercises keeps the valid exercises. seen holds the texts already used; with skipSeen the
// exercises whose text is in it are dropped, otherwise their texts are added to it.
func validExercises(in []Exercise, seen map[string]bool, skipSeen bool) []Exercise {
	var out []Exercise
	for _, e := range in {
		if exerciseProblems(e) != nil {
			continue
		}
		key := strings.ToLower(e.Text)
		if skipSeen && seen[key] {
			continue
		}
		if !skipSeen {
			seen[key] = true
		}
		out = append(out, e)
	}
	return out
}

func nonEmpty(in []string) []string {
	var out []string
	for _, s := range in {
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

func keepFirst[T any](in []T, n int) []T {
	if len(in) > n {
		return in[:n]
	}
	return in
}
