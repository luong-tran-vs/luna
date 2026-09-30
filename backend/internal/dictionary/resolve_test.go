package dictionary

import (
	"context"
	"testing"
)

// mapLookuper is an in-memory Lookuper keyed by lowercase word.
type mapLookuper map[string][]string

func (m mapLookuper) Lookup(_ context.Context, word string) (Entry, bool, error) {
	defs, ok := m[word]
	if !ok {
		return Entry{}, false, nil
	}
	e := Entry{Word: word}
	for _, d := range defs {
		e.Meanings = append(e.Meanings, Meaning{Text: d})
	}
	return e, true, nil
}

var fakeDict = mapLookuper{
	"go":      {"Đi, đi đến."},
	"went":    {"động từ quá khứ của go."},
	"study":   {"Học."},
	"studies": {"Động từ chia ở ngôi thứ ba số ít của study"},
	"see":     {"Thấy."},
	"saw":     {"Cái cưa.", "Quá khứ của see"},
	"good":    {"Tốt."},
	"better":  {"cấp so sánh của good", "Hơn, tốt hơn."},
	"take":    {"Cầm, lấy."},
	"taken":   {"Dạng phân từ quá khứ của take."},
	"child":   {"Đứa trẻ."},
	"stop":    {"Dừng."},
	"laugh":   {"Cười."},
	"orphan":  {"Số nhiều của ghostword"}, // points to a word the dictionary does not have
}

func TestResolve(t *testing.T) {
	t.Parallel()

	tests := []struct {
		word, lemma, first string
	}{
		{"went", "go", "Đi, đi đến."},                  // inflection note → base entry
		{"Studies", "study", "Học."},                   // note, case-insensitive
		{"taken", "take", "Cầm, lấy."},                 // "Dạng phân từ quá khứ của take."
		{"better", "good", "Tốt."},                     // comparative note
		{"saw", "saw", "Cái cưa."},                     // own meaning first: keep it
		{"stopped", "stop", "Dừng."},                   // not in dictionary → rule candidate
		{"laughing", "laugh", "Cười."},                 // -ing rule
		{"children", "child", "Đứa trẻ."},              // irregular table
		{"orphan", "orphan", "Số nhiều của ghostword"}, // base unknown → keep own entry
	}
	for _, tt := range tests {
		e, ok, err := Resolve(t.Context(), fakeDict, tt.word)
		if err != nil || !ok {
			t.Errorf("Resolve(%q) = %v, %v", tt.word, ok, err)
			continue
		}
		if e.Word != tt.lemma || e.Meanings[0].Text != tt.first {
			t.Errorf("Resolve(%q) = %s %q, want %s %q", tt.word, e.Word, e.Meanings[0].Text, tt.lemma, tt.first)
		}
	}

	if _, ok, _ := Resolve(t.Context(), fakeDict, "banana"); ok {
		t.Error("Resolve(banana) found something")
	}
}

func TestInflectionBase(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"động từ quá khứ của go.":                    "go",
		"Quá khứ và phân từ quá khứ của stop":        "stop",
		"Số nhiều của child":                         "child",
		"Động từ chia ở ngôi thứ ba số ít của study": "study",
		"cấp so sánh của well":                       "well",
		"Dạng hiện tại phân từ của look forward to.": "look forward to",
		"Cái cưa.":             "",
		"Của cải, tài sản.":    "",
		"Người của công chúng": "",
	}
	for def, want := range cases {
		if got := inflectionBase(def); got != want {
			t.Errorf("inflectionBase(%q) = %q, want %q", def, got, want)
		}
	}
}
