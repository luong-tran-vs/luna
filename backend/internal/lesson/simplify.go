package lesson

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/luongtran/luna/backend/internal/ai"
	"github.com/luongtran/luna/backend/internal/cefr"
	"github.com/luongtran/luna/backend/internal/dictionary"
	"github.com/luongtran/luna/backend/internal/wordmatch"
)

// Words above the lesson's level (added 2026-10-22): the prompt asks for words of the level, and the
// drafts that still use harder ones (by the CEFR-J list) go back to the AI once, in one request for
// the whole batch, to replace only those words. A draft is kept as written when the AI's version is
// no better; the words left are in Draft.HardWords.

// hardWords returns the words of text above level, as written, leaving out the target words (the
// admin chose them for the lesson) in any of their forms.
func hardWords(text string, level Level, targets []string) []string {
	allowed := map[string]bool{}
	for _, t := range targets {
		for _, c := range dictionary.Candidates(t) {
			allowed[c] = true
		}
	}
	var out []string
	for _, w := range cefr.Above(text, string(level)) {
		target := false
		for _, c := range dictionary.Candidates(w.Text) {
			if allowed[c] {
				target = true
				break
			}
		}
		if !target {
			out = append(out, w.Text)
		}
	}
	return out
}

// simplifyDrafts replaces the hard words of the drafts with one AI request. Any failure keeps the
// drafts as they are: the words only cost a warning.
func (s *Service) simplifyDrafts(ctx context.Context, level Level, drafts []Draft, targets [][]string) {
	simplifier, ok := s.AI.(ai.Simplifier)
	targetsOf := func(d Draft) []string {
		if d.index < len(targets) {
			return targets[d.index]
		}
		return nil
	}
	req := ai.SimplifyRequest{Level: string(level)}
	for i := range drafts {
		drafts[i].HardWords = hardWords(drafts[i].Content, level, targetsOf(drafts[i]))
		if len(drafts[i].HardWords) > 0 {
			req.Texts = append(req.Texts, ai.SimplifyText{Index: i, Text: drafts[i].Content, Words: drafts[i].HardWords})
		}
	}
	if !ok || len(req.Texts) == 0 {
		return
	}
	res, err := simplifier.SimplifyWords(ctx, req)
	if err != nil {
		return
	}
	for _, r := range res {
		if r.Index < 0 || r.Index >= len(drafts) {
			continue
		}
		d := &drafts[r.Index]
		text := strings.TrimSpace(strings.ReplaceAll(r.Text, "\r\n", "\n"))
		n := len(strings.Fields(text))
		if text == "" || utf8.RuneCountInString(text) > maxContent || n*10 < d.Words*8 || n*10 > d.Words*12 {
			continue
		}
		left := hardWords(text, level, targetsOf(*d))
		if len(left) >= len(d.HardWords) || missingTargets(text, targetsOf(*d)) > missingTargets(d.Content, targetsOf(*d)) {
			continue
		}
		d.Content, d.Words, d.HardWords = text, n, left
	}
}

func missingTargets(text string, targets []string) int {
	n := 0
	m := wordmatch.New(text, nil)
	for _, w := range targets {
		if !m.Contains(w) {
			n++
		}
	}
	return n
}
