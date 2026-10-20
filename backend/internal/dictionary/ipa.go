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

// weakForms are the unstressed forms of the small words, as they are said inside a phrase ("read
// a book" → "/ˈriːd ə ˈbʊk/", "dance to" → "/ˈdæns tə/"); dictionaries give their strong forms.
var weakForms = map[string]string{
	"a": "ə", "an": "ən", "the": "ðə", "to": "tə", "of": "əv", "for": "fər", "and": "ənd", "at": "ət",
	"from": "frəm",
}

// ipaCleaner writes the offline dictionary's transcriptions (Wiktionary, several conventions)
// with the symbols of USIPA: no non-syllabic mark ("eɪ̯"), "r" for the turned r, "aʊ" and "aɪ"
// for "ɑʊ" and "ɑɪ", "e" for "ɛ", "ər" and "ɜːr" for the r-coloured vowels, no syllable dots.
var ipaCleaner = strings.NewReplacer(
	"̯", "", "ɹ", "r", "ɑʊ", "aʊ", "ɑɪ", "aɪ", "ɛ", "e", "ɚ", "ər", "ɝ", "ɜːr", "(r)", "r", ".", "",
)

// IPA is the General American transcription of a word or phrase, "" when unknown (updated
// 2026-10-19: one standard, the CMU dictionary, before the offline dictionary). A word comes from
// USIPA, else from the offline dictionary in that exact form ("left" is not given the IPA of
// "leave") or as a regular "-s" form ("dishes" → "/ˈdɪʃɪz/"), cleaned up (cleanIPA). A phrase joins
// its words, the small ones weak and the others stressed ("good morning" → "/ˈɡʊd ˈmɔːrnɪŋ/"),
// only when every word is known.
func IPA(ctx context.Context, r Resolver, text string) (string, error) {
	words := strings.Fields(text)
	if len(words) < 2 {
		return wordIPA(ctx, r, text)
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
		core := strings.Trim(ipa, "/")
		if !strings.ContainsAny(core, "ˈˌ") {
			core = "ˈ" + core
		}
		parts = append(parts, core)
	}
	return "/" + strings.Join(parts, " ") + "/", nil
}

func wordIPA(ctx context.Context, r Resolver, w string) (string, error) {
	if ipa := USIPA(w); ipa != "" {
		return ipa, nil
	}
	e, ok, err := r.Resolve(ctx, w)
	if err != nil {
		return "", fmt.Errorf("dictionary: resolve %q: %w", w, err)
	}
	switch {
	case ok && e.Word == w && e.IPA != "":
		return cleanIPA(e.IPA), nil
	case ok && e.IPA != "" && (w == e.Word+"s" || w == e.Word+"es"):
		return withS(cleanIPA(e.IPA)), nil
	}
	return "", nil
}

// cleanIPA keeps the first of the variants an entry lists ("/hæv, həv, v/" → "/hæv/"), between
// slashes, with the symbols of USIPA.
func cleanIPA(ipa string) string {
	core := strings.Trim(strings.TrimSpace(ipa), "/[]")
	if i := strings.IndexAny(core, ",;"); i >= 0 {
		core = core[:i]
	}
	core = strings.TrimSpace(ipaCleaner.Replace(core))
	if core == "" {
		return ""
	}
	return "/" + core + "/"
}

// SameIPA reports whether two transcriptions write the same sounds, stress marks, slashes and
// spaces aside ("/ˈhoʊm/" and "/hoʊm/").
func SameIPA(a, b string) bool {
	strip := strings.NewReplacer("ˈ", "", "ˌ", "", "/", "", " ", "", "[", "", "]", "")
	return strip.Replace(a) == strip.Replace(b)
}

// suspiciousMarks are signs of a transcription copied from a source with another convention.
var suspiciousMarks = []string{",", ";", "ɹ", "ɑʊ", "ɑɪ", "̯", "ɚ", "ɝ", "(", "ɛ"}

// IPASuggestion is the standard transcription (IPA) to offer for a stored one that looks wrong or
// writes other sounds than the standard; "" when the stored one is fine or no standard is known.
func IPASuggestion(ctx context.Context, r Resolver, text, stored string) (string, error) {
	if strings.TrimSpace(stored) == "" {
		return "", nil
	}
	want, err := IPA(ctx, r, text)
	if err != nil || want == "" || want == stored {
		return "", err
	}
	if !OddIPA(stored) && SameIPA(stored, want) {
		return "", nil
	}
	return want, nil
}

// OddIPA reports whether a transcription looks copied from a source with another convention: not
// between slashes, several variants, or symbols USIPA does not use ("/hæv, həv, v/", "/tɹæk/").
func OddIPA(ipa string) bool {
	if !strings.HasPrefix(ipa, "/") || !strings.HasSuffix(ipa, "/") {
		return true
	}
	for _, m := range suspiciousMarks {
		if strings.Contains(ipa, m) {
			return true
		}
	}
	return false
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
