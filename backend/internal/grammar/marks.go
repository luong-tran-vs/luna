package grammar

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
)

var (
	// patternArrow comes before the sample answers of a question pattern: "Am + I …? -> Yes, I am.".
	// After anything but a question it is part of the pattern: "y → ies", "go → went".
	patternArrow = regexp.MustCompile(`\s*(?:->|→|=>)\s*`)
	questionEnd  = regexp.MustCompile(`\?\s*$`)
	// patternLine is "/" or "hoặc" after a whole sentence, starting another line of the pattern.
	patternLine   = regexp.MustCompile(`([.?!])\s+(?:/|hoặc)\s+`)
	lineSemicolon = regexp.MustCompile(`\s*;\s+`)
	formSlash     = regexp.MustCompile(`\s+/\s+`)
	plusSign      = regexp.MustCompile(`\+`)
	bracketNote   = regexp.MustCompile(`\([^()]*\)`)
	subjectVi     = regexp.MustCompile(`(?i)chủ ngữ`)
	wordSep       = regexp.MustCompile(`[\s,/()\[\]?.!…]+`)
	hasLetter     = regexp.MustCompile(`\pL`)
)

// PatternPieces cuts a structure's pattern as the study page colours it (patternRows in
// grammar-logic.ts; keep the two alike): the answers after "->" of a question left out, then line
// after line (see patternLines), each line at "+" outside brackets. "S + am / is / are + V-ing" has
// three pieces; the "{n:words}" marks of the examples count them from 1.
func PatternPieces(pattern string) []string {
	ask := pattern
	if cut := patternArrow.Split(pattern, 2); len(cut) == 2 && questionEnd.MatchString(cut[0]) {
		ask = cut[0]
	}
	var out []string
	for _, line := range patternLines(ask) {
		out = append(out, linePieces(line)...)
	}
	return out
}

// splitOutside cuts text at sep outside brackets: "động từ (+ tân ngữ) + trạng từ" cuts once at "+".
func splitOutside(text string, sep *regexp.Regexp) []string {
	var out []string
	from, depth, at := 0, 0, 0
	for _, m := range sep.FindAllStringIndex(text, -1) {
		for ; at < m[0]; at++ {
			switch text[at] {
			case '(', '[':
				depth++
			case ')', ']':
				depth--
			}
		}
		if depth == 0 {
			out = append(out, text[from:m[0]])
			from = m[1]
		}
	}
	return append(out, text[from:])
}

// linePieces is one line of a pattern cut at "+" outside brackets.
func linePieces(line string) []string {
	var out []string
	for _, p := range splitOutside(line, plusSign) {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// patternLines are the lines of a pattern. Besides a "/" after a whole sentence, a ";" between forms
// ("I + was; you + were") starts a line, and so does a "/" between forms that each have their "+" and
// begin alike: with a subject, with a helping verb or with the same word ("I + am / he, she, it + is").
// Choices inside a piece stay: "S + am / is / are", "động từ + into / out of + nơi chốn".
func patternLines(text string) []string {
	var out []string
	for _, sentence := range strings.Split(patternLine.ReplaceAllString(text, "$1\n"), "\n") {
		parts := splitOutside(sentence, lineSemicolon)
		for _, p := range parts {
			if !hasLetter.MatchString(p) {
				parts = []string{sentence}
				break
			}
		}
		for _, line := range parts {
			for _, f := range patternForms(line) {
				if f = strings.TrimSpace(f); f != "" {
					out = append(out, f)
				}
			}
		}
	}
	return out
}

// patternForms cuts a line at " / " when the forms between are alike (see patternLines).
func patternForms(line string) []string {
	cut := splitOutside(line, formSlash)
	if len(cut) < 2 {
		return []string{line}
	}
	firsts := make([]string, len(cut))
	for i, f := range cut {
		pieces := linePieces(f)
		if len(pieces) < 2 {
			return []string{line}
		}
		firsts[i] = pieces[0]
	}
	role, word := pieceRole(firsts[0]), firstWord(firsts[0])
	sameRole, sameWord := role != "", true
	for _, p := range firsts[1:] {
		sameRole = sameRole && pieceRole(p) == role
		sameWord = sameWord && firstWord(p) == word
	}
	if !sameRole && !sameWord {
		return []string{line}
	}
	return cut
}

func firstWord(piece string) string {
	if f := strings.Fields(piece); len(f) > 0 {
		return strings.ToLower(f[0])
	}
	return ""
}

// subjectWords and helperWords are SUBJECT_WORDS and HELPER_WORDS of grammar-logic.ts.
var (
	subjectWords = wordSet("i you he she it we they subject sb this these those")
	helperWords  = wordSet("am is are was were be been being do does did have has had will would can could shall should may might must isn't aren't wasn't weren't don't doesn't didn't won't can't couldn't wouldn't shouldn't mustn't needn't haven't hasn't hadn't")
)

func wordSet(words string) map[string]bool {
	set := map[string]bool{}
	for _, w := range strings.Fields(words) {
		set[w] = true
	}
	return set
}

// pieceWords are the words that tell what a piece is: without a note in brackets, a leading "Yes,"
// or "No," and punctuation; "S" and "chủ ngữ" give subject.
func pieceWords(piece string) []string {
	text := bracketNote.ReplaceAllString(piece, " ")
	if !hasLetter.MatchString(text) {
		text = piece
	}
	text = strings.ReplaceAll(subjectVi.ReplaceAllString(text, "subject"), "’", "'")
	var words []string
	for _, w := range wordSep.Split(text, -1) {
		switch {
		case w == "":
		case w == "S":
			words = append(words, "subject")
		default:
			words = append(words, strings.ToLower(w))
		}
	}
	if len(words) > 1 && (words[0] == "yes" || words[0] == "no") {
		return words[1:]
	}
	return words
}

// pieceRole is "subject", "helper" (be or a helping verb) or "" for anything else.
func pieceRole(piece string) string {
	words := pieceWords(piece)
	if len(words) == 0 {
		return ""
	}
	subject := true
	for _, w := range words {
		subject = subject && subjectWords[w]
	}
	switch {
	case subject:
		return "subject"
	case helperWords[words[0]]:
		return "helper"
	}
	return ""
}

var errBadMark = errors.New("bad mark")

// unmark returns the sentence of a marked example without its marks ("{2:is}" gives "is") and the
// highest piece number used; an error for an unclosed or empty mark or a number below 1.
func unmark(marked string) (plain string, maxPiece int, err error) {
	var b strings.Builder
	rest := marked
	for {
		open := strings.IndexAny(rest, "{}")
		if open < 0 {
			b.WriteString(rest)
			return b.String(), maxPiece, nil
		}
		if rest[open] == '}' {
			return "", 0, errBadMark
		}
		b.WriteString(rest[:open])
		rest = rest[open+1:]
		end := strings.IndexAny(rest, "{}")
		if end < 0 || rest[end] == '{' {
			return "", 0, errBadMark
		}
		num, words, ok := strings.Cut(rest[:end], ":")
		n, convErr := strconv.Atoi(strings.TrimSpace(num))
		if !ok || convErr != nil || n < 1 || strings.TrimSpace(words) == "" {
			return "", 0, errBadMark
		}
		maxPiece = max(maxPiece, n)
		b.WriteString(words)
		rest = rest[end+1:]
	}
}

// exampleProblem says what is wrong with the structure and marks of an example, "" when nothing.
// The key is "structure" or "marked".
func exampleProblem(e Example, structures []Structure) (key, msg string) {
	if e.Structure == 0 {
		if e.Marked != "" {
			return "marked", "Câu có đánh dấu phải ghi số cấu trúc"
		}
		return "", ""
	}
	if e.Structure < 1 || e.Structure > len(structures) {
		return "structure", "Không có cấu trúc số " + strconv.Itoa(e.Structure)
	}
	if e.Marked == "" {
		return "", ""
	}
	plain, maxPiece, err := unmark(e.Marked)
	switch {
	case err != nil:
		return "marked", "Đánh dấu sai cú pháp, phải là {số:từ}"
	case plain != e.En:
		return "marked", "Bỏ dấu đi phải ra đúng câu tiếng Anh"
	case maxPiece > len(PatternPieces(structures[e.Structure-1].Pattern)):
		return "marked", "Số đánh dấu lớn hơn số phần của công thức"
	}
	return "", ""
}
