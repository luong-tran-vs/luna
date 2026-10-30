package grammar

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
)

var (
	// patternArrow comes before the sample answers of a question pattern: "Am + I …? -> Yes, I am."
	patternArrow = regexp.MustCompile(`\s*(?:->|→|=>)\s*`)
	// patternLine is "/" or "hoặc" after a whole sentence, starting another line of the pattern.
	patternLine = regexp.MustCompile(`([.?!…])\s+(?:/|hoặc)\s+`)
)

// PatternPieces cuts a structure's pattern as the study page colours it: the answers after "->"
// left out, then line after line ("Am + I …? / Is + he …?" is two lines), each line at "+". "S + am /
// is / are + V-ing" has three pieces; the "{n:words}" marks of the examples count them from 1.
func PatternPieces(pattern string) []string {
	ask := patternArrow.Split(pattern, 2)[0]
	var out []string
	for _, line := range strings.Split(patternLine.ReplaceAllString(ask, "$1\n"), "\n") {
		for _, form := range patternForms(line) {
			for _, p := range strings.Split(form, "+") {
				if p = strings.TrimSpace(p); p != "" {
					out = append(out, p)
				}
			}
		}
	}
	return out
}

var formSlash = regexp.MustCompile(`\s+/\s+`)

// patternForms cuts a line at "/" when every form between has its "+" ("I + am / he, she, it + is"
// is two lines), and keeps it whole for short choices inside a piece ("S + am / is / are").
func patternForms(line string) []string {
	forms := formSlash.Split(line, -1)
	if len(forms) < 2 {
		return forms
	}
	for _, f := range forms {
		if !strings.Contains(f, "+") {
			return []string{line}
		}
	}
	return forms
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
