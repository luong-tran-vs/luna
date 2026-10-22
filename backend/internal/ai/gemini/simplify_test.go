package gemini_test

import (
	"strings"
	"testing"

	"github.com/luongtran/luna/backend/internal/ai"
)

func TestSimplifyWords(t *testing.T) {
	t.Parallel()
	c, prompt := capturePrompt(t, `{"texts":[{"index":2,"text":"She is sad."}]}`)
	got, err := c.SimplifyWords(t.Context(), ai.SimplifyRequest{Level: "A1", Texts: []ai.SimplifyText{
		{Index: 2, Text: "She is anxious.", Words: []string{"anxious", "abandon"}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Index != 2 || got[0].Text != "She is sad." {
		t.Fatalf("texts = %+v", got)
	}
	if !strings.Contains(*prompt, "Text 2. Words to replace: anxious, abandon\nShe is anxious.") || !strings.Contains(*prompt, "CEFR level A1") {
		t.Fatalf("prompt = %s", *prompt)
	}
	bad, _ := capturePrompt(t, "not json")
	if _, err := bad.SimplifyWords(t.Context(), ai.SimplifyRequest{Level: "A1"}); err == nil {
		t.Fatal("want decode error")
	}
}
