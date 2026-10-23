package llm_test

import (
	"strings"
	"testing"

	"github.com/luongtran/luna/backend/internal/ai"
)

func TestCheckSpeaking(t *testing.T) {
	t.Parallel()
	c, prompt := capturePrompt(t, `{"answers":[{"index":0,"corrected":"I like pop music.","noteVi":""},`+
		`{"index":1,"corrected":"Her name is Lan.","noteVi":"Dùng her name is."}]}`)
	got, err := c.CheckSpeaking(t.Context(), ai.SpeakCheckRequest{Level: "A1", Answers: []ai.SpeakAnswer{
		{Question: "What music do you like?", Said: "I like pop music"},
		{Question: "Who is your friend?", Said: "she name Lan"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[1].Index != 1 || got[1].Corrected != "Her name is Lan." || got[1].NoteVi != "Dùng her name is." {
		t.Fatalf("feedback = %+v", got)
	}
	if !strings.Contains(*prompt, "Answer: she name Lan") {
		t.Fatalf("prompt = %s", *prompt)
	}

	bad, _ := capturePrompt(t, "not json")
	if _, err := bad.CheckSpeaking(t.Context(), ai.SpeakCheckRequest{Level: "A1", Answers: []ai.SpeakAnswer{{Said: "x"}}}); err == nil {
		t.Fatal("want decode error")
	}
}
