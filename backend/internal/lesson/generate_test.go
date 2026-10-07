package lesson

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/luongtran/luna/backend/internal/ai"
)

// text returns content of exactly n words.
func text(n int) string {
	return strings.TrimSpace(strings.Repeat("word ", n-1) + "end.")
}

func validGenerate() GenerateInput {
	return GenerateInput{Level: "A1", Count: 3, Words: 100, Kind: "reading"}
}

func TestValidateGenerate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(*GenerateInput)
		field  string
		msg    string
	}{
		{name: "count 0", mutate: func(in *GenerateInput) { in.Count = 0 }, field: "count", msg: "Số bài từ 1 đến 10"},
		{name: "count 11", mutate: func(in *GenerateInput) { in.Count = 11 }, field: "count", msg: "Số bài từ 1 đến 10"},
		{name: "words 49", mutate: func(in *GenerateInput) { in.Words = 49 }, field: "words", msg: "Độ dài từ 50 đến 800 từ"},
		{name: "words 801", mutate: func(in *GenerateInput) { in.Words = 801 }, field: "words", msg: "Độ dài từ 50 đến 800 từ"},
		{name: "kind", mutate: func(in *GenerateInput) { in.Kind = "poem" }, field: "kind", msg: "Dạng bài không hợp lệ"},
		{name: "level", mutate: func(in *GenerateInput) { in.Level = "" }, field: "level", msg: "Vui lòng chọn trình độ"},
		{name: "idea", mutate: func(in *GenerateInput) { in.Idea = strings.Repeat("ý", 501) }, field: "idea", msg: "Ý chính tối đa 500 ký tự"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			in := validGenerate()
			tt.mutate(&in)
			_, err := ValidateGenerate(in)
			var verr *ValidationError
			if !errors.As(err, &verr) || verr.Fields[tt.field] != tt.msg {
				t.Fatalf("error = %v, want %s: %q", err, tt.field, tt.msg)
			}
		})
	}

	for _, in := range []GenerateInput{
		{Level: "A1", Count: 1, Words: 50, Kind: "reading"},
		{Level: "c2", Count: 10, Words: 800, Kind: "dialogue", Idea: "  " + strings.Repeat("ý", 500) + "  "},
	} {
		got, err := ValidateGenerate(in)
		if err != nil {
			t.Fatalf("ValidateGenerate(%+v) = %v", in, err)
		}
		if strings.HasPrefix(got.Idea, " ") {
			t.Errorf("idea not trimmed: %q", got.Idea)
		}
	}
}

func TestGeneratePassesTopicAndExistingTitles(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.create(t, func(in *Input) { in.Title = "My Brother Tom"; in.TopicID, in.Level = "topic-a1", "A1" })
	e.create(t, func(in *Input) { in.Title = "Office Day"; in.TopicID = "topic-b1" })
	e.ai.drafts = []ai.LessonDraft{{Title: "Sunday Lunch", Content: text(100)}}

	in := validGenerate()
	in.Kind = "dialogue"
	in.Idea = "  a birthday  "
	if _, err := e.svc.Generate(t.Context(), "topic-a1", in); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	req, calls := e.ai.lastRequest()
	if calls != 1 {
		t.Fatalf("AI calls = %d, want 1", calls)
	}
	if req.Level != "A1" || req.TopicName != "Family" || req.Count != 3 || req.Words != 100 ||
		req.Kind != ai.KindDialogue || req.Idea != "a birthday" {
		t.Errorf("request = %+v", req)
	}
	if len(req.ExistingTitles) != 1 || req.ExistingTitles[0] != "My Brother Tom" {
		t.Errorf("existing titles = %v, want only the topic's lessons", req.ExistingTitles)
	}
}

func TestGenerateFiltersDrafts(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.create(t, func(in *Input) { in.Title = "My Brother Tom"; in.TopicID, in.Level = "topic-a1", "A1" })
	e.ai.drafts = []ai.LessonDraft{
		{Title: "  Sunday Lunch ", Content: " " + strings.ReplaceAll(text(100), "word word", "word\r\nword") + "\n"},
		{Title: "", Content: text(100)},                       // empty title
		{Title: "Empty", Content: "   "},                      // empty content
		{Title: "sunday   lunch!", Content: text(100)},        // duplicate in the batch
		{Title: "MY BROTHER TOM.", Content: text(100)},        // existing lesson
		{Title: "Too Short", Content: text(79)},               // kept: off length only warns
		{Title: "Too Long", Content: text(121)},               // kept
		{Title: strings.Repeat("T", 201), Content: text(100)}, // title too long
		{Title: "Edge Low", Content: text(80)},                // beyond count, cut
		{Title: "Edge High", Content: text(120)},              // beyond count, cut
		{Title: "Extra", Content: text(100)},                  // beyond count, cut
	}

	got, err := e.svc.Generate(t.Context(), "topic-a1", validGenerate())
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if got.Requested != 3 || got.Dropped != 5 || got.DropReasons != (DropReasons{DuplicateTitle: 2, Empty: 2, TooLong: 1}) {
		t.Errorf("requested/dropped = %d/%d %+v, want 3/5", got.Requested, got.Dropped, got.DropReasons)
	}
	titles := make([]string, len(got.Drafts))
	for i, d := range got.Drafts {
		titles[i] = d.Title
	}
	if strings.Join(titles, "|") != "Sunday Lunch|Too Short|Too Long" {
		t.Fatalf("titles = %v", titles)
	}
	first := got.Drafts[0]
	if strings.Contains(first.Content, "\r") || strings.HasPrefix(first.Content, " ") || strings.HasSuffix(first.Content, "\n") {
		t.Errorf("content not normalised: %q", first.Content)
	}
	if first.Words != 100 || got.Drafts[1].Words != 79 || got.Drafts[2].Words != 121 {
		t.Errorf("words = %d %d %d", first.Words, got.Drafts[1].Words, got.Drafts[2].Words)
	}
	if lessons, _ := e.lessons.List(t.Context(), Filter{TopicID: "topic-a1"}); len(lessons) != 1 {
		t.Errorf("generate stored lessons: %d in topic, want 1", len(lessons))
	}
}

func TestGenerateDropsContentOverLimit(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	long := strings.Repeat("abcdefghij ", 700) // 700 words, 7700 chars
	e.ai.drafts = []ai.LessonDraft{{Title: "Long", Content: long + strings.Repeat("x", 3000)}}
	in := validGenerate()
	in.Words = 700
	if _, err := e.svc.Generate(t.Context(), "topic-a1", in); !errors.Is(err, ErrUnusableDraft) {
		t.Fatalf("error = %v, want ErrUnusableDraft", err)
	}
}

func TestGenerateNothingUsable(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.ai.drafts = []ai.LessonDraft{{Title: "", Content: ""}, {Title: "  ", Content: text(10)}}
	if _, err := e.svc.Generate(t.Context(), "topic-a1", validGenerate()); !errors.Is(err, ErrUnusableDraft) {
		t.Fatalf("error = %v, want ErrUnusableDraft", err)
	}

	e.ai.drafts = nil
	if _, err := e.svc.Generate(t.Context(), "topic-a1", validGenerate()); !errors.Is(err, ErrUnusableDraft) {
		t.Fatalf("empty answer: error = %v, want ErrUnusableDraft", err)
	}
}

func TestGenerateErrors(t *testing.T) {
	t.Parallel()

	t.Run("invalid input makes no AI call", func(t *testing.T) {
		t.Parallel()
		e := newEnv(t)
		in := validGenerate()
		in.Count = 11
		var verr *ValidationError
		if _, err := e.svc.Generate(t.Context(), "topic-a1", in); !errors.As(err, &verr) {
			t.Fatalf("error = %v, want ValidationError", err)
		}
		if _, calls := e.ai.lastRequest(); calls != 0 {
			t.Errorf("AI calls = %d, want 0", calls)
		}
	})

	t.Run("unknown topic", func(t *testing.T) {
		t.Parallel()
		e := newEnv(t)
		if _, err := e.svc.Generate(t.Context(), "nope", validGenerate()); !errors.Is(err, ErrTopicNotFound) {
			t.Fatalf("error = %v, want ErrTopicNotFound", err)
		}
	})

	for _, want := range []error{ai.ErrQuota, ai.ErrNotConfigured, ai.ErrInvalidKey} {
		t.Run(want.Error(), func(t *testing.T) {
			t.Parallel()
			e := newEnv(t)
			e.ai.genErr = want
			if _, err := e.svc.Generate(t.Context(), "topic-a1", validGenerate()); !errors.Is(err, want) {
				t.Fatalf("error = %v, want %v", err, want)
			}
		})
	}

	t.Run("timeout", func(t *testing.T) {
		t.Parallel()
		e := newEnv(t)
		e.svc.GenerateTimeout = 20 * time.Millisecond
		e.ai.genBlock = true
		start := time.Now()
		_, err := e.svc.Generate(t.Context(), "topic-a1", validGenerate())
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("error = %v, want DeadlineExceeded", err)
		}
		if time.Since(start) > 2*time.Second {
			t.Errorf("timeout took %v", time.Since(start))
		}
	})
}
