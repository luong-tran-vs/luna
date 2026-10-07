package topic

import (
	"cmp"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/luongtran/luna/backend/internal/wordmatch"
)

// Limits of a topic word list (F18).
const (
	MaxWords      = 300
	maxWordLength = 40
)

// wordPattern allows English letters, spaces, hyphens, apostrophes and "/", starting with a letter.
var wordPattern = regexp.MustCompile(`^[A-Za-z][A-Za-z '\-/]*$`)

const (
	msgWordChars = "Chỉ dùng chữ cái tiếng Anh, khoảng trắng, - ' /"
	msgWordDup   = "Từ bị trùng"
)

// cleanWord trims and collapses spaces in w, and returns the Vietnamese error for a bad word.
func cleanWord(w string) (string, string) {
	w = strings.Join(strings.Fields(w), " ")
	switch {
	case utf8.RuneCountInString(w) > maxWordLength:
		return w, fmt.Sprintf("Tối đa %d ký tự", maxWordLength)
	case !wordPattern.MatchString(w):
		return w, msgWordChars
	}
	return w, ""
}

// CleanWords normalizes an admin's word list: blank entries are dropped, spaces collapsed. Any
// invalid word or level, a duplicate (ignoring case) or more than MaxWords words gives a
// ValidationError keyed "words.{i}" (or "words.{i}.level") by the index in the input (or
// "words"), and no words.
func CleanWords(in []Word) ([]Word, error) {
	out := []Word{}
	fields := map[string]string{}
	seen := map[string]bool{}
	for i, raw := range in {
		if strings.TrimSpace(raw.Text) == "" {
			continue
		}
		w, msg := cleanWord(raw.Text)
		key := strings.ToLower(w)
		level := strings.ToUpper(strings.TrimSpace(raw.Level))
		switch {
		case msg != "":
			fields[fmt.Sprintf("words.%d", i)] = msg
		case seen[key]:
			fields[fmt.Sprintf("words.%d", i)] = msgWordDup
		case level != "" && !ValidLevel(level):
			fields[fmt.Sprintf("words.%d.level", i)] = "Trình độ không hợp lệ"
		default:
			seen[key] = true
			out = append(out, Word{Text: w, Level: level})
		}
	}
	if len(out) > MaxWords {
		fields["words"] = fmt.Sprintf("Tối đa %d từ", MaxWords)
	}
	if len(fields) > 0 {
		return nil, &ValidationError{Fields: fields}
	}
	return out, nil
}

// Coverage tells for each word whether the lessons contain it and in how many lessons (whole
// words or phrases, ignoring case and simple inflections, see package wordmatch).
func Coverage(words []Word, lessons []LessonText) []WordUse {
	texts := make([]*wordmatch.Text, len(lessons))
	for i, l := range lessons {
		texts[i] = wordmatch.New(l.Content, l.Lemmas)
	}
	out := make([]WordUse, len(words))
	for i, w := range words {
		n := 0
		for _, t := range texts {
			if t.Contains(w.Text) {
				n++
			}
		}
		out[i] = WordUse{Text: w.Text, Level: w.Level, Used: n > 0, LessonCount: n}
	}
	return out
}

// Limits of a target word plan.
const (
	MaxPlanLessons = 10
	MaxTargetWords = 15
	// MaxSuggestWords is the most words one AI suggestion may add.
	MaxSuggestWords = 50
)

// PlanWords splits the topic words that fit level into count groups of perLesson words for the
// lessons of one generation: unused words first, then words used in fewer lessons, words of exactly
// level before lower or unset ones, then list order. Groups do
// not share words while there are enough; otherwise the order starts over (least used first).
// A list shorter than perLesson gives every group the whole list.
func PlanWords(uses []WordUse, level string, count, perLesson int) [][]string {
	order := FitWords(uses, level)
	exact := func(u WordUse) int {
		if u.Level == level {
			return 0
		}
		return 1
	}
	slices.SortStableFunc(order, func(a, b WordUse) int {
		return cmp.Or(cmp.Compare(a.LessonCount, b.LessonCount), cmp.Compare(exact(a), exact(b)))
	})
	k := min(perLesson, len(order))
	out := make([][]string, count)
	next := 0
	for i := range out {
		group := make([]string, 0, k)
		for len(group) < k {
			w := order[next%len(order)].Text
			next++
			if !slices.Contains(group, w) {
				group = append(group, w)
			}
		}
		out[i] = group
	}
	return out
}

// FitWords returns the words that may be given to a lesson of level, in list order.
func FitWords(uses []WordUse, level string) []WordUse {
	out := []WordUse{}
	for _, u := range uses {
		if (Word{Text: u.Text, Level: u.Level}).FitsLevel(level) {
			out = append(out, u)
		}
	}
	return out
}
