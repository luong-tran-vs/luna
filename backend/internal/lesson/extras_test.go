package lesson

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/luongtran/luna/backend/internal/ai"
)

const extrasContent = "Tom lives in Hanoi with his family.  He goes to school by bus. On Sunday, they visit Grandma."

func okQuestion(prompt string) ai.Question {
	return ai.Question{
		Prompt: prompt, Options: []string{"Hanoi", "Hue", "Da Nang", "Can Tho"}, AnswerIndex: 0,
		ExplanationVi: "Bài viết: Tom sống ở Hà Nội.",
	}
}

func TestCleanExtrasQuestions(t *testing.T) {
	t.Parallel()

	bad := func(mutate func(*ai.Question)) ai.Question {
		q := okQuestion("Bad?")
		q.Options = append([]string(nil), q.Options...)
		mutate(&q)
		return q
	}
	in := ai.LessonExtras{Questions: []ai.Question{
		{
			Prompt: "  Where does Tom live? ", Options: []string{" Hanoi ", "Hue", "Da Nang", "Can Tho"}, AnswerIndex: 0,
			ExplanationVi: " Tom sống ở Hà Nội. ",
		},
		bad(func(q *ai.Question) { q.Options = q.Options[:3] }),
		bad(func(q *ai.Question) { q.Options = append(q.Options, "Vinh") }),
		bad(func(q *ai.Question) { q.Options[1] = "  hanoi" }),
		bad(func(q *ai.Question) { q.Options[2] = " " }),
		bad(func(q *ai.Question) { q.AnswerIndex = -1 }),
		bad(func(q *ai.Question) { q.AnswerIndex = 4 }),
		bad(func(q *ai.Question) { q.ExplanationVi = "" }),
		bad(func(q *ai.Question) { q.Prompt = " " }),
		okQuestion("where does  TOM live?"), // duplicate prompt
		okQuestion("How does he go to school?"),
		okQuestion("Who do they visit?"),
		okQuestion("When do they visit Grandma?"),
		okQuestion("Who lives with Tom?"),
		okQuestion("One too many?"),
	}}

	got := CleanExtras(in, extrasContent).Questions
	if len(got) != 5 {
		t.Fatalf("questions = %d, want 5: %+v", len(got), got)
	}
	first := got[0]
	if first.Prompt != "Where does Tom live?" || first.Options[0] != "Hanoi" || first.ExplanationVi != "Tom sống ở Hà Nội." {
		t.Errorf("first = %+v, want trimmed", first)
	}
	if got[4].Prompt != "Who lives with Tom?" {
		t.Errorf("last kept = %q", got[4].Prompt)
	}
}

func TestCleanExtrasGrammarNote(t *testing.T) {
	t.Parallel()

	note := func(examples ...string) *ai.GrammarNote {
		return &ai.GrammarNote{Title: " Thì hiện tại đơn ", BodyVi: " Dùng cho thói quen. ", Examples: examples}
	}
	tests := []struct {
		name string
		in   *ai.GrammarNote
		want []string // nil = no note
	}{
		{
			name: "examples found ignoring case, spaces and final punctuation",
			in:   note("tom lives in  Hanoi", "He goes to school by bus!", "They play football.", "On Sunday, they visit Grandma.", "x"),
			want: []string{"tom lives in  Hanoi", "He goes to school by bus!", "On Sunday, they visit Grandma."},
		},
		{name: "no example in the lesson", in: note("They play football."), want: nil},
		{name: "empty title", in: &ai.GrammarNote{BodyVi: "b", Examples: []string{"He goes to school"}}, want: nil},
		{name: "empty body", in: &ai.GrammarNote{Title: "t", Examples: []string{"He goes to school"}}, want: nil},
		{name: "missing", in: nil, want: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := CleanExtras(ai.LessonExtras{GrammarNote: tt.in}, extrasContent).GrammarNote
			if tt.want == nil {
				if got != nil {
					t.Fatalf("note = %+v, want nil", got)
				}
				return
			}
			if got == nil || got.Title != "Thì hiện tại đơn" || got.BodyVi != "Dùng cho thói quen." ||
				strings.Join(got.Examples, "|") != strings.Join(tt.want, "|") {
				t.Fatalf("note = %+v, want examples %v", got, tt.want)
			}
		})
	}
}

func TestCleanExtrasWritingPrompt(t *testing.T) {
	t.Parallel()
	got := CleanExtras(ai.LessonExtras{WritingPrompt: "  " + strings.Repeat("ý", 600) + " "}, extrasContent)
	if n := utf8.RuneCountInString(got.WritingPrompt); n != 500 {
		t.Errorf("writing prompt = %d runes, want 500", n)
	}
	if empty := CleanExtras(ai.LessonExtras{}, extrasContent); len(empty.Questions) != 0 || empty.GrammarNote != nil ||
		empty.WritingPrompt != "" {
		t.Errorf("empty input = %+v", empty)
	}
}
