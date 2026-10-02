package topic

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
)

func TestCleanWords(t *testing.T) {
	t.Parallel()

	got, err := CleanWords([]string{"  Family ", "take   a shower", "", "o'clock", "T-shirt", "and/or"})
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if want := []string{"Family", "take a shower", "o'clock", "T-shirt", "and/or"}; !slices.Equal(got, want) {
		t.Fatalf("words = %q", got)
	}

	if got, err := CleanWords(nil); err != nil || len(got) != 0 || got == nil {
		t.Fatalf("empty = %#v, %v", got, err)
	}
}

func TestCleanWordsErrors(t *testing.T) {
	t.Parallel()

	_, err := CleanWords([]string{"Family", "family", "Intensive care unit (ICU)", strings.Repeat("a", 41), "bố", "-x", "Cousin"})
	var verr *ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("err = %v", err)
	}
	want := map[string]string{
		"words.1": "Từ bị trùng",
		"words.2": "Chỉ dùng chữ cái tiếng Anh, khoảng trắng, - ' /",
		"words.3": "Tối đa 40 ký tự",
		"words.4": "Chỉ dùng chữ cái tiếng Anh, khoảng trắng, - ' /",
		"words.5": "Chỉ dùng chữ cái tiếng Anh, khoảng trắng, - ' /",
	}
	if fmt.Sprint(verr.Fields) != fmt.Sprint(want) {
		t.Fatalf("fields = %v", verr.Fields)
	}

	many := make([]string, 101)
	for i := range many {
		many[i] = fmt.Sprintf("word%c%c", 'a'+i/26, 'a'+i%26)
	}
	if _, err := CleanWords(many); !errors.As(err, &verr) || verr.Fields["words"] != "Tối đa 100 từ" {
		t.Fatalf("101 words: %v", err)
	}
	if _, err := CleanWords(many[:100]); err != nil {
		t.Fatalf("100 words: %v", err)
	}
}

func TestCoverage(t *testing.T) {
	t.Parallel()

	words := []string{"Family", "Grandmother", "take a shower", "go", "son", "Cousin"}
	lessons := []LessonText{
		{Content: "My family is big. I love my grandmothers."},
		{Content: "Every FAMILY took a shower. He went home.", Lemmas: map[string]string{"went": "go"}},
		{Content: "It is the season of rain."},
	}
	got := Coverage(words, lessons)
	want := []WordUse{
		{Text: "Family", Used: true, LessonCount: 2},
		{Text: "Grandmother", Used: true, LessonCount: 1},
		{Text: "take a shower", Used: true, LessonCount: 1},
		{Text: "go", Used: true, LessonCount: 1},
		{Text: "son", Used: false, LessonCount: 0},
		{Text: "Cousin", Used: false, LessonCount: 0},
	}
	if !slices.Equal(got, want) {
		t.Fatalf("coverage =\n%+v\nwant\n%+v", got, want)
	}
	if got := Coverage(words, nil); len(got) != len(words) || got[0].Used {
		t.Fatalf("no lessons = %+v", got)
	}
}

func uses(spec ...string) []WordUse {
	out := make([]WordUse, len(spec))
	for i, s := range spec {
		// "word:n" means used in n lessons.
		text, n, _ := strings.Cut(s, ":")
		c := 0
		if n != "" {
			c = int(n[0] - '0')
		}
		out[i] = WordUse{Text: text, Used: c > 0, LessonCount: c}
	}
	return out
}

func TestPlanWords(t *testing.T) {
	t.Parallel()

	ws := uses("a:2", "b", "c:1", "d", "e", "f:1", "g")
	tests := []struct {
		name             string
		count, perLesson int
		want             [][]string
	}{
		{"unused first, no overlap", 2, 2, [][]string{{"b", "d"}, {"e", "g"}}},
		{"then least used", 2, 3, [][]string{{"b", "d", "e"}, {"g", "c", "f"}}},
		{"reuse when short", 3, 3, [][]string{{"b", "d", "e"}, {"g", "c", "f"}, {"a", "b", "d"}}},
		{"fewer words than per lesson", 2, 9, [][]string{{"b", "d", "e", "g", "c", "f", "a"}, {"b", "d", "e", "g", "c", "f", "a"}}},
		{"zero per lesson", 2, 0, [][]string{{}, {}}},
	}
	for _, tt := range tests {
		if got := PlanWords(ws, tt.count, tt.perLesson); fmt.Sprint(got) != fmt.Sprint(tt.want) {
			t.Errorf("%s: PlanWords = %v, want %v", tt.name, got, tt.want)
		}
	}
	if got := PlanWords(nil, 3, 8); len(got) != 3 || len(got[0]) != 0 || got[0] == nil {
		t.Errorf("no words = %#v", got)
	}
}
