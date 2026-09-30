package lesson

import (
	"errors"
	"strconv"
	"strings"

	"github.com/luongtran/luna/backend/internal/ai"
)

// ErrNoValidAnnotations means the AI answer contained nothing usable.
var ErrNoValidAnnotations = errors.New("lesson: AI returned no valid annotations")

// CleanAnnotations keeps AI annotations whose fields are present and whose text really
// appears in the lesson, fixes their sentence index and drops duplicates.
func CleanAnnotations(items []ai.Annotation, sentences []string) ([]Annotation, error) {
	out := make([]Annotation, 0, len(items))
	seen := map[string]bool{}
	for _, it := range items {
		a := Annotation{
			Text:      strings.TrimSpace(it.Text),
			Lemma:     strings.TrimSpace(it.Lemma),
			MeaningVi: strings.TrimSpace(it.MeaningVi),
		}
		if a.Text == "" || a.Lemma == "" || a.MeaningVi == "" {
			continue
		}
		idx, ok := findSentence(a.Text, it.SentenceIndex, sentences)
		if !ok {
			continue
		}
		a.SentenceIndex = idx
		key := strings.ToLower(a.Text) + "\x00" + strconv.Itoa(idx)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, a)
	}
	if len(out) == 0 {
		return nil, ErrNoValidAnnotations
	}
	return out, nil
}

// findSentence returns the index of the sentence containing text (case-insensitive),
// preferring hint; ok is false when no sentence contains it.
func findSentence(text string, hint int, sentences []string) (int, bool) {
	needle := strings.ToLower(strings.TrimSpace(text))
	if needle == "" {
		return 0, false
	}
	if hint >= 0 && hint < len(sentences) && strings.Contains(strings.ToLower(sentences[hint]), needle) {
		return hint, true
	}
	for i, s := range sentences {
		if strings.Contains(strings.ToLower(s), needle) {
			return i, true
		}
	}
	return 0, false
}
