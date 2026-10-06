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
	// HasImage says the word has a picture (F23).
	HasImage bool
}

// WithImages lets the reader serve word pictures (F23); without it no word has one.
func (r *Reader) WithImages(images ImageRepository) *Reader {
	r.images = images
	return r
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
	var pictured []string
	if r.images != nil {
		if pictured, err = r.images.Lemmas(ctx, id); err != nil {
			return nil, false, fmt.Errorf("lesson: image lemmas: %w", err)
		}
	}
	items = vocabularyWords(l)
	for i := range items {
		item := &items[i]
		e, ok, err := r.dict.Resolve(ctx, item.Lemma)
		if err != nil {
			return nil, false, fmt.Errorf("lesson: resolve %q: %w", item.Lemma, err)
		}
		if ok && e.Word == item.Lemma {
			item.IPA = e.IPA
		}
		item.HasImage = slices.Contains(pictured, item.Lemma)
	}
	return items, true, nil
}

// Image returns the picture of one word of a lesson, or ErrImageNotFound.
func (r *Reader) Image(ctx context.Context, id, lemma string) (WordImage, error) {
	if r.images == nil {
		return WordImage{}, ErrImageNotFound
	}
	return r.images.Get(ctx, id, normalize(lemma))
}
