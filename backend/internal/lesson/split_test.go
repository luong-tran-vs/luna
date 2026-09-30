package lesson

import (
	"slices"
	"testing"
)

func TestSplitSentences(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want []string
	}{
		{
			name: "spec example",
			in:   "Mr. Smith arrived at 9.30 a.m. He was late! Why?",
			want: []string{"Mr. Smith arrived at 9.30 a.m.", "He was late!", "Why?"},
		},
		{
			name: "titles never split",
			in:   "Mrs. Brown met Dr. Lee and Prof. Kim. They talked.",
			want: []string{"Mrs. Brown met Dr. Lee and Prof. Kim.", "They talked."},
		},
		{
			name: "initials",
			in:   "J. K. Rowling wrote it. It sold well.",
			want: []string{"J. K. Rowling wrote it.", "It sold well."},
		},
		{
			name: "decimals and money",
			in:   "It costs $3.50. Cheap! Pi is about 3.14 today.",
			want: []string{"It costs $3.50.", "Cheap!", "Pi is about 3.14 today."},
		},
		{
			name: "abbreviation before lowercase continues",
			in:   "U.S. troops left. They went home.",
			want: []string{"U.S. troops left.", "They went home."},
		},
		{
			name: "e.g. and vs. never split",
			in:   "Eat fruit, e.g. Apples or pears. It is cats vs. Dogs today.",
			want: []string{"Eat fruit, e.g. Apples or pears.", "It is cats vs. Dogs today."},
		},
		{
			name: "No. before a number",
			in:   "Room No. 5 is free. Go now.",
			want: []string{"Room No. 5 is free.", "Go now."},
		},
		{
			name: "ellipsis",
			in:   "Wait... what? Wait... Really? Hmm… Yes.",
			want: []string{"Wait... what?", "Wait...", "Really?", "Hmm…", "Yes."},
		},
		{
			name: "quoted speech",
			in:   `"Stop!" she said. He stopped. "Why?" He asked.`,
			want: []string{`"Stop!" she said.`, "He stopped.", `"Why?"`, "He asked."},
		},
		{
			name: "closing quote and parenthesis",
			in:   "He said “go home.” Then he left. (It was late.) We slept.",
			want: []string{"He said “go home.”", "Then he left.", "(It was late.)", "We slept."},
		},
		{
			name: "blank line forces a split",
			in:   "Title without dot\n\nFirst line\ncontinues here. Next one.",
			want: []string{"Title without dot", "First line continues here.", "Next one."},
		},
		{
			name: "windows newlines and extra spaces",
			in:   "  One.\r\n\r\n   Two   words.  ",
			want: []string{"One.", "Two words."},
		},
		{
			name: "no final punctuation",
			in:   "Hello there. How are you",
			want: []string{"Hello there.", "How are you"},
		},
		{
			name: "empty",
			in:   " \n\n\t ",
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := SplitSentences(tt.in); !slices.Equal(got, tt.want) {
				t.Fatalf("SplitSentences(%q)\n got  %q\n want %q", tt.in, got, tt.want)
			}
		})
	}
}
