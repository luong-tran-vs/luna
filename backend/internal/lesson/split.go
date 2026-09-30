package lesson

import (
	"regexp"
	"strings"
	"unicode"
)

var paragraphBreak = regexp.MustCompile(`\n[ \t]*\n`)

// Abbreviations that are always followed by more of the same sentence.
var neverEndsSentence = map[string]bool{
	// titles before a name
	"mr": true, "mrs": true, "ms": true, "dr": true, "prof": true, "st": true, "sr": true, "jr": true,
	"mt": true, "capt": true, "gen": true, "rev": true,
	// abbreviations that introduce more text
	"e.g": true, "i.e": true, "vs": true, "cf": true, "approx": true, "fig": true,
}

// SplitSentences splits English text into trimmed sentences. Blank lines always end a
// sentence; inside a paragraph, line breaks and repeated spaces are collapsed.
func SplitSentences(text string) []string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	var out []string
	for _, p := range paragraphBreak.Split(text, -1) {
		p = strings.Join(strings.Fields(p), " ")
		if p != "" {
			out = append(out, splitParagraph([]rune(p))...)
		}
	}
	return out
}

// splitParagraph expects single spaces between words.
func splitParagraph(r []rune) []string {
	var out []string
	start := 0
	for i := 0; i < len(r); i++ {
		if !isTerminator(r[i]) {
			continue
		}
		end := i // last terminator of the run: "...", "?!"
		for end+1 < len(r) && isTerminator(r[end+1]) {
			end++
		}
		last := end // include closing quotes and brackets
		for last+1 < len(r) && isCloser(r[last+1]) {
			last++
		}

		atEnd := last+1 >= len(r)
		if !atEnd && (r[last+1] != ' ' || !endsSentence(r, i, end, last)) {
			i = last
			continue
		}
		out = append(out, strings.TrimSpace(string(r[start:last+1])))
		start = last + 1
		i = last
	}
	if rest := strings.TrimSpace(string(r[start:])); rest != "" {
		out = append(out, rest)
	}
	return out
}

// endsSentence decides whether the terminators r[first..end] (followed by closers up to
// last and then a space) end the sentence.
func endsSentence(r []rune, first, end, last int) bool {
	next := r[last+2] // first rune of the following word
	if !startsSentence(next) {
		return false
	}
	if first != end || r[first] != '.' {
		return true // "!", "?", "...", "…" before a sentence start
	}

	word := wordBefore(r, first)
	lower := strings.ToLower(word)
	switch {
	case neverEndsSentence[lower]:
		return false
	case lower == "no" && unicode.IsDigit(next):
		return false
	case isInitial(word):
		return false
	}
	return true
}

// wordBefore returns the non-space text right before position i, without leading
// opening quotes or brackets (for example "a.m" before the final dot of "a.m.").
func wordBefore(r []rune, i int) string {
	j := i
	for j > 0 && r[j-1] != ' ' {
		j--
	}
	return strings.TrimLeft(string(r[j:i]), `"'“‘(`)
}

func isInitial(word string) bool {
	rs := []rune(word)
	return len(rs) == 1 && unicode.IsUpper(rs[0])
}

func isTerminator(c rune) bool { return c == '.' || c == '!' || c == '?' || c == '…' }

func isCloser(c rune) bool { return strings.ContainsRune(`"'”’)]`, c) }

func startsSentence(c rune) bool {
	return unicode.IsUpper(c) || unicode.IsDigit(c) || strings.ContainsRune(`"'“‘([`, c)
}
