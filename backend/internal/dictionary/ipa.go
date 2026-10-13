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

// weakForms are the unstressed forms of the articles, as they are said inside a phrase ("read a
// book" → "/ˈriːd ə ˈbʊk/"); the dictionary gives their stressed forms ("/eɪ/").
var weakForms = map[string]string{"a": "ə", "an": "ən", "the": "ðə"}

// missingIPA covers common words the dictionary has no transcription for.
var missingIPA = map[string]string{"have": "/ˈhæv/", "has": "/ˈhæz/", "had": "/ˈhæd/"}

// ipaCleaner removes marks the dictionary uses inconsistently: the non-syllabic mark ("eɪ̯") and
// the turned r ("ɹ"), written "r" as in learners' dictionaries.
var ipaCleaner = strings.NewReplacer("̯", "", "ɹ", "r")

// IPA is the transcription of a word or phrase, "" when unknown. A word must be in the dictionary
// in that exact form ("left" is not given the IPA of "leave"), except a regular plural or third
// person "-s" form ("dishes" → "/ˈdɪʃɪz/"). A phrase joins the transcriptions of its words
// ("good morning" → "/ˈɡʊd ˈmɔr.nɪŋ/"), only when every word is known.
func IPA(ctx context.Context, r Resolver, text string) (string, error) {
	words := strings.Fields(text)
	if len(words) < 2 {
		ipa, err := wordIPA(ctx, r, text)
		return ipaCleaner.Replace(ipa), err
	}
	parts := make([]string, 0, len(words))
	for _, w := range words {
		if weak, ok := weakForms[strings.ToLower(w)]; ok {
			parts = append(parts, weak)
			continue
		}
		ipa, err := wordIPA(ctx, r, w)
		if err != nil || ipa == "" {
			return "", err
		}
		parts = append(parts, strings.Trim(ipa, "/[] "))
	}
	return ipaCleaner.Replace("/" + strings.Join(parts, " ") + "/"), nil
}

func wordIPA(ctx context.Context, r Resolver, w string) (string, error) {
	e, ok, err := r.Resolve(ctx, w)
	if err != nil {
		return "", fmt.Errorf("dictionary: resolve %q: %w", w, err)
	}
	switch {
	case ok && e.Word == w && e.IPA != "":
		return e.IPA, nil
	case ok && e.IPA != "" && (w == e.Word+"s" || w == e.Word+"es"):
		return withS(e.IPA), nil
	}
	return missingIPA[strings.ToLower(w)], nil
}

// withS adds the "-s" ending to a transcription: /ɪz/ after a hissing sound, /s/ after another
// voiceless sound, /z/ otherwise.
func withS(ipa string) string {
	core := strings.Trim(ipa, "/[] ")
	switch {
	case hasAnySuffix(core, "s", "z", "ʃ", "ʒ", "tʃ", "dʒ"):
		core += "ɪz"
	case hasAnySuffix(core, "p", "t", "k", "f", "θ"):
		core += "s"
	default:
		core += "z"
	}
	return "/" + core + "/"
}

func hasAnySuffix(s string, suffixes ...string) bool {
	for _, suf := range suffixes {
		if strings.HasSuffix(s, suf) {
			return true
		}
	}
	return false
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
