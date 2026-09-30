package dictionary

import (
	"slices"
	"testing"
)

func TestCandidates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		word     string
		mustHave string
	}{
		{"went", "go"},
		{"gone", "go"},
		{"saw", "see"},
		{"studies", "study"},
		{"studied", "study"},
		{"watches", "watch"},
		{"parks", "park"},
		{"liked", "like"},
		{"stopped", "stop"},
		{"making", "make"},
		{"running", "run"},
		{"laughing", "laugh"},
		{"children", "child"},
		{"people", "person"},
		{"feet", "foot"},
		{"John's", "john"},
		{"WENT", "go"},
		{"It’s", "it"},
	}
	for _, tt := range tests {
		got := Candidates(tt.word)
		if !slices.Contains(got, tt.mustHave) {
			t.Errorf("Candidates(%q) = %q, missing %q", tt.word, got, tt.mustHave)
		}
	}
}

func TestCandidatesOrderAndUniqueness(t *testing.T) {
	t.Parallel()

	got := Candidates("Parks")
	if got[0] != "parks" {
		t.Fatalf("first candidate = %q, want the word itself lowercased", got[0])
	}
	seen := map[string]bool{}
	for _, c := range got {
		if seen[c] {
			t.Fatalf("duplicate candidate %q in %q", c, got)
		}
		seen[c] = true
		if c == "" {
			t.Fatalf("empty candidate in %q", got)
		}
	}
	// Irregular forms come before rule-based guesses.
	if w := Candidates("went"); w[1] != "go" {
		t.Fatalf("Candidates(went) = %q, want go right after the word", w)
	}
	// Short words are not stripped by rules ("is" must not give "i", "bus" must not give "bu").
	if got := Candidates("is"); !slices.Equal(got, []string{"is", "be"}) {
		t.Errorf("Candidates(is) = %q", got)
	}
	if got := Candidates("bus"); !slices.Equal(got, []string{"bus"}) {
		t.Errorf("Candidates(bus) = %q", got)
	}
}

func TestCandidatesEmpty(t *testing.T) {
	t.Parallel()
	if got := Candidates("  "); got != nil {
		t.Fatalf("Candidates(blank) = %q, want nil", got)
	}
}
