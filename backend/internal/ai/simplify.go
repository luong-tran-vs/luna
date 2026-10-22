package ai

import "context"

// SimplifyText is one generated lesson whose words above the level should be replaced (added
// 2026-10-22).
type SimplifyText struct {
	Index int
	Text  string
	// Words are the words of Text above the level, as written in it.
	Words []string
}

// SimplifyRequest asks, in one request, to replace the listed words of each text with simpler
// words of the level, keeping everything else.
type SimplifyRequest struct {
	Level string
	Texts []SimplifyText
}

// SimplifiedText is a text with its hard words replaced.
type SimplifiedText struct {
	Index int    `json:"index"`
	Text  string `json:"text"`
}

// Simplifier replaces words above a lesson's level. The Gemini client implements it; without it
// generated lessons keep their words and the admin sees the warning.
type Simplifier interface {
	SimplifyWords(ctx context.Context, req SimplifyRequest) ([]SimplifiedText, error)
}
