package grammar

import "strings"

// maxVariants bounds how many forms one answer expands to.
const maxVariants = 32

// negatable lists the auxiliaries that form a "not" contraction: "is not" <-> "isn't".
var negatable = map[string]bool{
	"is": true, "are": true, "was": true, "were": true, "do": true, "does": true, "did": true,
	"have": true, "has": true, "had": true, "could": true, "would": true, "should": true, "must": true,
}

type pronounForm struct{ suffix, full string }

// pronounForms maps a pronoun to its contraction suffixes and the full words they stand for.
// "'s" and "'d" are ambiguous, so they list both readings.
var pronounForms = map[string][]pronounForm{
	"i":     {{"'m", "am"}, {"'ve", "have"}, {"'ll", "will"}, {"'d", "would"}, {"'d", "had"}},
	"you":   {{"'re", "are"}, {"'ve", "have"}, {"'ll", "will"}, {"'d", "would"}, {"'d", "had"}},
	"we":    {{"'re", "are"}, {"'ve", "have"}, {"'ll", "will"}, {"'d", "would"}, {"'d", "had"}},
	"they":  {{"'re", "are"}, {"'ve", "have"}, {"'ll", "will"}, {"'d", "would"}, {"'d", "had"}},
	"he":    {{"'s", "is"}, {"'s", "has"}, {"'ll", "will"}, {"'d", "would"}, {"'d", "had"}},
	"she":   {{"'s", "is"}, {"'s", "has"}, {"'ll", "will"}, {"'d", "would"}, {"'d", "had"}},
	"it":    {{"'s", "is"}, {"'s", "has"}, {"'ll", "will"}, {"'d", "would"}, {"'d", "had"}},
	"that":  {{"'s", "is"}, {"'s", "has"}, {"'ll", "will"}, {"'d", "would"}},
	"there": {{"'s", "is"}, {"'s", "has"}, {"'ll", "will"}, {"'d", "would"}},
}

// normalizeAnswer replaces curly apostrophes by "'" and squeezes spaces.
func normalizeAnswer(s string) string {
	s = strings.NewReplacer("’", "'", "‘", "'", "ʼ", "'").Replace(s)
	return strings.Join(strings.Fields(s), " ")
}

// ExpandAnswers returns the accepted answers of a fill exercise plus their contracted and full
// forms ("is not" <-> "isn't", "I am" <-> "I'm", "cannot" <-> "can't" ...). The given answers come
// first, in order (curly apostrophes replaced, spaces squeezed); variants follow; forms that differ
// only in case are dropped.
func ExpandAnswers(answers []string) []string {
	out := make([]string, 0, len(answers))
	seen := map[string]bool{}
	var originals []string
	for _, a := range answers {
		a = normalizeAnswer(a)
		if a == "" || seen[strings.ToLower(a)] {
			continue
		}
		seen[strings.ToLower(a)] = true
		out = append(out, a)
		originals = append(originals, a)
	}
	for _, a := range originals {
		variants := 0
		work := []string{a}
		for len(work) > 0 && variants < maxVariants {
			cur := work[0]
			work = work[1:]
			for _, v := range variantsOf(strings.Fields(cur)) {
				key := strings.ToLower(v)
				if seen[key] {
					continue
				}
				seen[key] = true
				out = append(out, v)
				work = append(work, v)
				variants++
			}
		}
	}
	return out
}

// variantsOf applies one contraction or expansion at each position of the words.
func variantsOf(w []string) []string {
	var out []string
	emit := func(i, n int, repl ...string) {
		words := make([]string, 0, len(w)+len(repl))
		words = append(words, w[:i]...)
		words = append(words, repl...)
		words = append(words, w[i+n:]...)
		out = append(out, strings.Join(words, " "))
	}
	for i, tok := range w {
		lower := strings.ToLower(tok)
		next := ""
		if i+1 < len(w) {
			next = strings.ToLower(w[i+1])
		}

		// pronoun + full word -> contraction
		if forms, ok := pronounForms[lower]; ok && next != "" {
			for _, f := range forms {
				if f.full == next {
					emit(i, 2, tok+f.suffix)
					break
				}
			}
		}
		// contraction -> pronoun + full word
		if p, suffix, ok := strings.Cut(lower, "'"); ok && p != "" {
			for _, f := range pronounForms[p] {
				if f.suffix == "'"+suffix {
					emit(i, 1, tok[:len(p)], f.full)
				}
			}
		}

		switch {
		case negatable[lower] && next == "not":
			emit(i, 2, tok+"n't")
		case lower == "will" && next == "not":
			emit(i, 2, "won't")
		case lower == "can" && next == "not":
			emit(i, 2, "cannot")
			emit(i, 2, "can't")
		case lower == "cannot":
			emit(i, 1, "can", "not")
			emit(i, 1, "can't")
		case lower == "can't":
			emit(i, 1, "can", "not")
			emit(i, 1, "cannot")
		case lower == "won't":
			emit(i, 1, "will", "not")
		case strings.HasSuffix(lower, "n't") && negatable[strings.TrimSuffix(lower, "n't")]:
			emit(i, 1, tok[:len(tok)-3], "not")
		}
	}
	return out
}
