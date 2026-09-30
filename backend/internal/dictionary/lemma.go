// Package dictionary reads the offline English–Vietnamese dictionary and guesses base forms.
package dictionary

import "strings"

// Candidates returns possible dictionary forms of word, most likely first: the word itself
// (lowercased), its irregular base form, then regular-rule guesses (-s, -es, -ies, -ed, -ing,
// doubled consonants, 's). Callers keep the first candidate the dictionary knows, so wrong
// guesses are harmless.
func Candidates(word string) []string {
	w := strings.ToLower(strings.TrimSpace(strings.ReplaceAll(word, "’", "'")))
	if w == "" {
		return nil
	}

	var out []string
	seen := map[string]bool{}
	add := func(c string) {
		if seen[c] || (len(c) < 2 && c != w) {
			return
		}
		seen[c] = true
		out = append(out, c)
	}

	add(w)
	if base, ok := irregular[w]; ok {
		add(base)
	}
	if base, ok := strings.CutSuffix(w, "'s"); ok {
		add(base)
		w = base
	}
	if len(w) <= 3 {
		return out // short words ("is", "bus", "was") are not stripped by rules
	}

	switch {
	case strings.HasSuffix(w, "ies"):
		add(w[:len(w)-3] + "y")
	case strings.HasSuffix(w, "ied"):
		add(w[:len(w)-3] + "y")
	}
	if base, ok := strings.CutSuffix(w, "es"); ok {
		add(base)
	}
	if base, ok := strings.CutSuffix(w, "s"); ok && !strings.HasSuffix(w, "ss") {
		add(base)
	}
	if base, ok := strings.CutSuffix(w, "ed"); ok {
		add(base)
		add(base + "e")
		add(undouble(base))
	}
	if base, ok := strings.CutSuffix(w, "ing"); ok {
		add(base)
		add(base + "e")
		add(undouble(base))
	}
	return out
}

// undouble turns "stopp" into "stop" and "runn" into "run" (doubled final consonant).
func undouble(s string) string {
	n := len(s)
	if n >= 3 && s[n-1] == s[n-2] && !strings.ContainsRune("aeiouls", rune(s[n-1])) {
		return s[:n-1]
	}
	return s
}
