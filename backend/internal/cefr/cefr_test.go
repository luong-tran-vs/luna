package cefr

import (
	"reflect"
	"testing"
)

func TestLevel(t *testing.T) {
	for word, want := range map[string]string{"hello": "A1", "plays": "A1", "track": "A2", "Festival": "A1", "abandon": "B1"} {
		if got, ok := Level(word); !ok || got != want {
			t.Errorf("Level(%q) = %q, %v; want %q", word, got, ok, want)
		}
	}
	if _, ok := Level("Minh"); ok {
		t.Error("a name has no level")
	}
}

func TestAbove(t *testing.T) {
	text := "Abandoned cats live here. They don't abandon Anna. She isn't sad, she is anxious."
	got := Above(text, "A1")
	want := []Word{{Text: "Abandoned", Level: "B2"}, {Text: "abandon", Level: "B1"}, {Text: "anxious", Level: "A2"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Above = %v, want %v", got, want)
	}
	if got := Above(text, "B2"); got != nil {
		t.Errorf("nothing is above B2 in the list, got %v", got)
	}
}
