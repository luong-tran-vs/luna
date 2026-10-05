package grammar

import (
	"slices"
	"testing"
)

func TestExpandAnswers(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   []string
		want []string // all present, in this relative order
	}{
		{"plain word stays", []string{"is"}, []string{"is"}},
		{"is not", []string{"is not"}, []string{"is not", "isn't"}},
		{"isn't", []string{"isn't"}, []string{"isn't", "is not"}},
		{"are not", []string{"are not"}, []string{"are not", "aren't"}},
		{"was not", []string{"was not"}, []string{"was not", "wasn't"}},
		{"were not", []string{"were not"}, []string{"were not", "weren't"}},
		{"do not", []string{"do not"}, []string{"do not", "don't"}},
		{"does not", []string{"does not"}, []string{"does not", "doesn't"}},
		{"did not", []string{"did not"}, []string{"did not", "didn't"}},
		{"have not", []string{"have not"}, []string{"have not", "haven't"}},
		{"has not", []string{"has not"}, []string{"has not", "hasn't"}},
		{"had not", []string{"had not"}, []string{"had not", "hadn't"}},
		{"could not", []string{"could not"}, []string{"could not", "couldn't"}},
		{"would not", []string{"would not"}, []string{"would not", "wouldn't"}},
		{"should not", []string{"should not"}, []string{"should not", "shouldn't"}},
		{"must not", []string{"must not"}, []string{"must not", "mustn't"}},
		{"mustn't back", []string{"mustn't"}, []string{"mustn't", "must not"}},
		{"cannot", []string{"cannot"}, []string{"cannot", "can not", "can't"}},
		{"can not", []string{"can not"}, []string{"can not", "cannot", "can't"}},
		{"can't", []string{"can't"}, []string{"can't", "can not", "cannot"}},
		{"will not", []string{"will not"}, []string{"will not", "won't"}},
		{"won't", []string{"won't"}, []string{"won't", "will not"}},
		{"I am", []string{"I am"}, []string{"I am", "I'm"}},
		{"I'm", []string{"I'm"}, []string{"I'm", "I am"}},
		{"it is", []string{"it is"}, []string{"it is", "it's"}},
		{"it's both readings", []string{"it's"}, []string{"it's", "it is", "it has"}},
		{"they are", []string{"they are"}, []string{"they are", "they're"}},
		{"you're", []string{"you're"}, []string{"you're", "you are"}},
		{"we have", []string{"we have"}, []string{"we have", "we've"}},
		{"I will", []string{"I will"}, []string{"I will", "I'll"}},
		{"she'd", []string{"she'd"}, []string{"she'd", "she would", "she had"}},
		{"she would", []string{"she would"}, []string{"she would", "she'd"}},
		{"curly apostrophe", []string{"isn’t"}, []string{"isn't", "is not"}},
		{"pronoun with not", []string{"he is not"}, []string{"he is not", "he's not", "he isn't"}},
		{"he's not reaches isn't", []string{"he's not"}, []string{"he's not", "he is not", "he has not", "he isn't", "he hasn't"}},
		{"case kept", []string{"It is"}, []string{"It is", "It's"}},
		{"extra spaces", []string{"  is   not "}, []string{"is not", "isn't"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ExpandAnswers(tt.in)
			last := -1
			for _, w := range tt.want {
				i := slices.Index(got, w)
				if i < 0 {
					t.Fatalf("ExpandAnswers(%q) = %q, missing %q", tt.in, got, w)
				}
				if i < last {
					t.Errorf("ExpandAnswers(%q) = %q, %q out of order", tt.in, got, w)
				}
				last = i
			}
		})
	}
}

func TestExpandAnswersKeepsOrderAndDropsDuplicates(t *testing.T) {
	t.Parallel()
	got := ExpandAnswers([]string{"is not", "ISN'T", "was", "", "  ", "is not"})
	if len(got) < 3 || got[0] != "is not" || got[1] != "ISN'T" || got[2] != "was" {
		t.Fatalf("got %q", got)
	}
	seen := map[string]bool{}
	for _, g := range got {
		if seen[g] {
			t.Errorf("duplicate %q in %q", g, got)
		}
		seen[g] = true
	}
}

func TestExpandAnswersLeavesUnrelatedAndEmptyAlone(t *testing.T) {
	t.Parallel()
	if got := ExpandAnswers([]string{"went", "go to"}); !slices.Equal(got, []string{"went", "go to"}) {
		t.Errorf("got %q", got)
	}
	if got := ExpandAnswers(nil); len(got) != 0 {
		t.Errorf("got %q", got)
	}
	if got := ExpandAnswers([]string{"I is"}); !slices.Equal(got, []string{"I is"}) {
		t.Errorf("nonsense pair expanded: %q", got)
	}
}
