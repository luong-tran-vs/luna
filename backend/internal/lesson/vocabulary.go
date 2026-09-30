package lesson

import (
	"context"
	"fmt"
	"slices"
)

// VocabItem is one annotated word or phrase of a lesson, by base form, as listed in the
// Vocabulary section of the Reading step.
type VocabItem struct {
	Lemma         string
	Text          string
	MeaningVi     string
	IPA           string
	SentenceIndex int
	Sentence      string
}

// Vocabulary lists the lesson's annotations, one item per base form in the order they first
// appear, with IPA from the offline dictionary. It never calls the AI provider. available is
// false until the annotations are done.
func (r *Reader) Vocabulary(ctx context.Context, id string) (items []VocabItem, available bool, err error) {
	l, err := r.lessons.Get(ctx, id)
	if err != nil {
		return nil, false, err
	}
	if l.AnnotationStatus != StatusDone || len(l.Annotations) == 0 {
		return []VocabItem{}, false, nil
	}

	anns := slices.Clone(l.Annotations)
	slices.SortStableFunc(anns, func(a, b Annotation) int { return a.SentenceIndex - b.SentenceIndex })
	seen := map[string]bool{}
	items = []VocabItem{}
	for _, a := range anns {
		lemma := normalize(a.Lemma)
		if lemma == "" || seen[lemma] {
			continue
		}
		seen[lemma] = true
		item := VocabItem{Lemma: lemma, Text: a.Text, MeaningVi: a.MeaningVi, SentenceIndex: a.SentenceIndex}
		if a.SentenceIndex >= 0 && a.SentenceIndex < len(l.Sentences) {
			item.Sentence = l.Sentences[a.SentenceIndex].Text
		}
		e, ok, err := r.dict.Resolve(ctx, lemma)
		if err != nil {
			return nil, false, fmt.Errorf("lesson: resolve %q: %w", lemma, err)
		}
		if ok && e.Word == lemma {
			item.IPA = e.IPA
		}
		items = append(items, item)
	}
	return items, true, nil
}
