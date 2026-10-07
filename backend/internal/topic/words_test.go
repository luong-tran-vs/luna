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

	in := words("  Family ", "take   a shower", "", "o'clock", "T-shirt", "and/or")
	in[1].Level = " a2 "
	got, err := CleanWords(in)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	want := words("Family", "take a shower", "o'clock", "T-shirt", "and/or")
	want[1].Level = "A2"
	if !slices.Equal(got, want) {
		t.Fatalf("words = %q", got)
	}

	if got, err := CleanWords(nil); err != nil || len(got) != 0 || got == nil {
		t.Fatalf("empty = %#v, %v", got, err)
	}
}

func TestCleanWordsErrors(t *testing.T) {
	t.Parallel()

	in := words("Family", "family", "Intensive care unit (ICU)", strings.Repeat("a", 41), "bố", "-x", "Cousin")
	in[6].Level = "D1"
	_, err := CleanWords(in)
	var verr *ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("err = %v", err)
	}
	want := map[string]string{
		"words.1":       "Từ bị trùng",
		"words.2":       "Chỉ dùng chữ cái tiếng Anh, khoảng trắng, - ' /",
		"words.3":       "Tối đa 40 ký tự",
		"words.4":       "Chỉ dùng chữ cái tiếng Anh, khoảng trắng, - ' /",
		"words.5":       "Chỉ dùng chữ cái tiếng Anh, khoảng trắng, - ' /",
		"words.6.level": "Trình độ không hợp lệ",
	}
	if fmt.Sprint(verr.Fields) != fmt.Sprint(want) {
		t.Fatalf("fields = %v", verr.Fields)
	}

	many := make([]Word, 301)
	for i := range many {
		many[i] = Word{Text: fmt.Sprintf("word%c%c", 'a'+i/26, 'a'+i%26)}
	}
	if _, err := CleanWords(many); !errors.As(err, &verr) || verr.Fields["words"] != "Tối đa 300 từ" {
		t.Fatalf("301 words: %v", err)
	}
	if _, err := CleanWords(many[:300]); err != nil {
		t.Fatalf("300 words: %v", err)
	}
}

func TestCoverage(t *testing.T) {
	t.Parallel()

	ws := words("Family", "Grandmother", "take a shower", "go", "son", "Cousin")
	ws[1].Level = "A2"
	lessons := []LessonText{
		{Content: "My family is big. I love my grandmothers."},
		{Content: "Every FAMILY took a shower. He went home.", Lemmas: map[string]string{"went": "go"}},
		{Content: "It is the season of rain."},
	}
	got := Coverage(ws, lessons)
	want := []WordUse{
		{Text: "Family", Used: true, LessonCount: 2},
		{Text: "Grandmother", Level: "A2", Used: true, LessonCount: 1},
		{Text: "take a shower", Used: true, LessonCount: 1},
		{Text: "go", Used: true, LessonCount: 1},
		{Text: "son", Used: false, LessonCount: 0},
		{Text: "Cousin", Used: false, LessonCount: 0},
	}
	if !slices.Equal(got, want) {
		t.Fatalf("coverage =\n%+v\nwant\n%+v", got, want)
	}
	if got := Coverage(ws, nil); len(got) != len(ws) || got[0].Used {
		t.Fatalf("no lessons = %+v", got)
	}
}

func uses(spec ...string) []WordUse {
	out := make([]WordUse, len(spec))
	for i, s := range spec {
		// "word:n" means used in n lessons, "word@B1" a word of level B1 ("word@B1:n" both).
		text, n, _ := strings.Cut(s, ":")
		text, level, _ := strings.Cut(text, "@")
		c := 0
		if n != "" {
			c = int(n[0] - '0')
		}
		out[i] = WordUse{Text: text, Level: level, Used: c > 0, LessonCount: c}
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
		if got := PlanWords(ws, "A1", tt.count, tt.perLesson); fmt.Sprint(got) != fmt.Sprint(tt.want) {
			t.Errorf("%s: PlanWords = %v, want %v", tt.name, got, tt.want)
		}
	}
	if got := PlanWords(nil, "A1", 3, 8); len(got) != 3 || len(got[0]) != 0 || got[0] == nil {
		t.Errorf("no words = %#v", got)
	}
}

func TestPlanWordsByLevel(t *testing.T) {
	t.Parallel()

	// Only words of the level or lower (or without a level) are given; among the unused ones,
	// words of exactly the level come first.
	ws := uses("any", "low@A1", "mid@A2", "high@B2", "midused@A2:1")
	if got := PlanWords(ws, "A2", 1, 5); fmt.Sprint(got) != "[[mid any low midused]]" {
		t.Errorf("A2 = %v", got)
	}
	if got := PlanWords(ws, "A1", 1, 5); fmt.Sprint(got) != "[[low any]]" {
		t.Errorf("A1 = %v", got)
	}
	if got := FitWords(ws, "C1"); len(got) != 5 {
		t.Errorf("C1 fit = %v", got)
	}
}
