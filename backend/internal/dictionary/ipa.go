package dictionary

import (
	"context"
	"fmt"
	"strings"
)

// Resolver finds the entry of a word's base form (SQLite.Resolve, None.Resolve).
type Resolver interface {
	Resolve(ctx context.Context, word string) (Entry, bool, error)
}

// IPA is the transcription of a word or phrase, "" when unknown. A word must be in the dictionary
// in that exact form ("left" is not given the IPA of "leave"). A phrase joins the transcriptions of
// its words ("good morning" → "/ˈɡʊd ˈmɔr.nɪŋ/"), only when every word is known.
func IPA(ctx context.Context, r Resolver, text string) (string, error) {
	words := strings.Fields(text)
	if len(words) < 2 {
		return wordIPA(ctx, r, text)
	}
	parts := make([]string, 0, len(words))
	for _, w := range words {
		ipa, err := wordIPA(ctx, r, w)
		if err != nil || ipa == "" {
			return "", err
		}
		parts = append(parts, strings.Trim(ipa, "/[] "))
	}
	return "/" + strings.Join(parts, " ") + "/", nil
}

func wordIPA(ctx context.Context, r Resolver, w string) (string, error) {
	e, ok, err := r.Resolve(ctx, w)
	if err != nil {
		return "", fmt.Errorf("dictionary: resolve %q: %w", w, err)
	}
	if !ok || e.Word != w {
		return "", nil
	}
	return e.IPA, nil
}

// MeaningVi is the Vietnamese meaning of a word in that exact form: its first meanings joined with
// "; ", "" when unknown. A phrase is looked up as a whole.
func MeaningVi(ctx context.Context, r Resolver, text string) (string, error) {
	e, ok, err := r.Resolve(ctx, text)
	if err != nil {
		return "", fmt.Errorf("dictionary: resolve %q: %w", text, err)
	}
	if !ok || e.Word != text {
		return "", nil
	}
	parts := make([]string, 0, 2)
	for _, m := range e.Meanings {
		if t := strings.TrimSpace(m.Text); t != "" && len(parts) < 2 {
			parts = append(parts, t)
		}
	}
	return strings.Join(parts, "; "), nil
}
