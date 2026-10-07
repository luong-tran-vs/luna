package lesson

import (
	"errors"
	"strings"
	"testing"
)

func validInput() Input {
	return Input{
		Title:   "A day at the park",
		Content: "We went to the park. It was sunny.",
		TopicID: "topic-b1",
		Level:   "B1",
		Source:  "Tự viết",
		License: "CC BY 4.0",
	}
}

func TestValidateInput(t *testing.T) {
	t.Parallel()

	in := validInput()
	in.Title = "  A day at the park  "
	got, sentences, err := ValidateInput(in)
	if err != nil {
		t.Fatalf("ValidateInput: %v", err)
	}
	if got.Title != "A day at the park" {
		t.Errorf("title not trimmed: %q", got.Title)
	}
	if len(sentences) != 2 {
		t.Errorf("sentences = %q, want 2", sentences)
	}
}

func TestValidateInputErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(*Input)
		fields []string
	}{
		{name: "empty title", mutate: func(in *Input) { in.Title = "  " }, fields: []string{"title"}},
		{name: "long title", mutate: func(in *Input) { in.Title = strings.Repeat("a", 201) }, fields: []string{"title"}},
		{name: "empty content", mutate: func(in *Input) { in.Content = "\n \n" }, fields: []string{"content"}},
		{name: "long content", mutate: func(in *Input) { in.Content = strings.Repeat("ă", 10001) }, fields: []string{"content"}},
		{name: "too many sentences", mutate: func(in *Input) { in.Content = strings.Repeat("Go. ", 201) }, fields: []string{"content"}},
		{name: "no topic", mutate: func(in *Input) { in.TopicID = "  " }, fields: []string{"topicId"}},
		{name: "no level", mutate: func(in *Input) { in.Level = "" }, fields: []string{"level"}},
		{name: "bad level", mutate: func(in *Input) { in.Level = "D1" }, fields: []string{"level"}},
		{name: "empty source", mutate: func(in *Input) { in.Source = "" }, fields: []string{"source"}},
		{name: "long license", mutate: func(in *Input) { in.License = strings.Repeat("l", 101) }, fields: []string{"license"}},
		{
			name:   "all together",
			mutate: func(in *Input) { *in = Input{} },
			fields: []string{"title", "content", "topicId", "level", "source", "license"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			in := validInput()
			tt.mutate(&in)

			_, _, err := ValidateInput(in)
			var verr *ValidationError
			if !errors.As(err, &verr) {
				t.Fatalf("err = %v, want ValidationError", err)
			}
			if len(verr.Fields) != len(tt.fields) {
				t.Fatalf("fields = %v, want %v", verr.Fields, tt.fields)
			}
			for _, f := range tt.fields {
				if verr.Fields[f] == "" {
					t.Errorf("missing message for %q in %v", f, verr.Fields)
				}
			}
		})
	}
}
