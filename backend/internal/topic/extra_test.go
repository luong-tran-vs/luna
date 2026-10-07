package topic

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

func TestExtraTopicsAreValid(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 10, 7, 0, 0, 0, 0, time.UTC)
	topics, err := LoadExtraTopics(now)
	if err != nil {
		t.Fatal(err)
	}
	if len(topics) != 24 {
		t.Fatalf("topics = %d, want 24", len(topics))
	}

	// Every word of the file is valid and given once per topic: none is dropped on load.
	var f extraFile
	if err := json.Unmarshal(extraJSON, &f); err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for i, tp := range topics {
		raw := 0
		for level, words := range f.Topics[i].Words {
			if !ValidLevel(level) {
				t.Errorf("%s: level %q", tp.Name, level)
			}
			raw += len(words)
		}
		if len(tp.Words) != raw {
			t.Errorf("%s: %d of %d words kept", tp.Name, len(tp.Words), raw)
		}
		if n := utf8.RuneCountInString(tp.Name); n == 0 || n > maxName || utf8.RuneCountInString(tp.Description) > maxDescription {
			t.Errorf("%s: name or description too long", tp.Name)
		}
		if names[SeedKey(tp.Name)] {
			t.Errorf("%s: twice", tp.Name)
		}
		names[SeedKey(tp.Name)] = true
		if len(tp.Words) < 15 || !tp.WordsSeeded || tp.Roadmaps == nil || !tp.CreatedAt.Equal(now) {
			t.Errorf("%s = %+v", tp.Name, tp)
		}
	}

	// None repeats a topic of the starting vocabulary.
	seed, err := LoadSeed()
	if err != nil {
		t.Fatal(err)
	}
	for _, st := range seed.Topics {
		if names[SeedKey(st.Name)] {
			t.Errorf("%s is already a seed topic", st.Name)
		}
	}
}

func TestExtraTopicWordsKeepTheLowestLevel(t *testing.T) {
	t.Parallel()
	topics, _ := LoadExtraTopics(time.Time{})
	for _, tp := range topics {
		if !strings.HasPrefix(tp.Name, "Chào hỏi") {
			continue
		}
		if tp.Words[0] != (Word{Text: "hello", Level: "A1"}) || tp.Words[len(tp.Words)-1].Level != "A2" {
			t.Fatalf("words = %+v", tp.Words)
		}
		return
	}
	t.Fatal("greetings topic missing")
}

func TestPlanExtraTopicsSkipsTakenNames(t *testing.T) {
	t.Parallel()
	extra := []Topic{{Name: "Âm nhạc"}, {Name: "Sức khỏe"}, {Name: "Lịch sử"}}
	got := PlanExtraTopics([]string{"âm  NHẠC", "Sức khoẻ"}, extra)
	if len(got) != 1 || got[0].Name != "Lịch sử" {
		t.Fatalf("planned = %+v", got)
	}
}
