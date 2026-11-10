package lesson

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/luongtran/luna/backend/internal/dictionary"
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
	// HasImage says the word has a picture (F23), its own or the word bank's (F24).
	HasImage bool
	// POS is the part of speech: the annotation's (as used in the lesson), else the dictionary's;
	// "" when unknown.
	POS string
}

// BankWord is what the shared word bank knows of a word (F24).
type BankWord struct {
	IPA      string
	HasImage bool
}

// WordBank is the shared word bank (F24), implemented over wordbank.Service.
type WordBank interface {
	// Words returns the entries of the bank among lemmas, by lemma.
	Words(ctx context.Context, lemmas []string) (map[string]BankWord, error)
	// Image returns ErrImageNotFound when the word has no picture in the bank.
	Image(ctx context.Context, lemma string) (WordImage, error)
}

// WithImages lets the reader serve word pictures (F23); without it no word has one.
func (r *Reader) WithImages(images ImageRepository) *Reader {
	r.images = images
	return r
}

// WithWordBank gives the words of a lesson the IPA and picture of the word bank (F24) when the
// lesson has none of its own.
func (r *Reader) WithWordBank(bank WordBank) *Reader {
	r.bank = bank
	return r
}

// Vocabulary lists the lesson's annotations, one item per base form in the order they first
// appear. IPA comes from the word bank, else the offline dictionary; a word without a picture of
// its own in the lesson shows the bank's. It never calls the AI provider. available is false until
// the annotations are done.
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
	bank := map[string]BankWord{}
	if r.bank != nil {
		lemmas := make([]string, len(items))
		for i, it := range items {
			lemmas[i] = it.Lemma
		}
		if bank, err = r.bank.Words(ctx, lemmas); err != nil {
			return nil, false, fmt.Errorf("lesson: word bank: %w", err)
		}
	}
	for i := range items {
		item := &items[i]
		b := bank[item.Lemma]
		item.IPA = b.IPA
		if item.IPA == "" {
			if item.IPA, err = dictionary.IPA(ctx, r.dict, item.Lemma); err != nil {
				return nil, false, fmt.Errorf("lesson: %w", err)
			}
		}
		item.HasImage = b.HasImage || slices.Contains(pictured, item.Lemma)
		if item.POS == "" {
			if item.POS, err = dictionaryPOS(ctx, r.dict, item.Lemma, item.Text, item.MeaningVi); err != nil {
				return nil, false, fmt.Errorf("lesson: %w", err)
			}
		}
	}
	return items, true, nil
}

// POS returns the words of a lesson with their part of speech only, as Vocabulary gives it, in
// one database read: no IPA, pictures or word bank, which the part of speech does not need.
func (r *Reader) POS(ctx context.Context, id string) ([]VocabItem, error) {
	l, err := r.lessons.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if l.AnnotationStatus != StatusDone || len(l.Annotations) == 0 {
		return []VocabItem{}, nil
	}
	items := vocabularyWords(l)
	for i := range items {
		item := &items[i]
		if item.POS == "" {
			if item.POS, err = dictionaryPOS(ctx, r.dict, item.Lemma, item.Text, item.MeaningVi); err != nil {
				return nil, fmt.Errorf("lesson: %w", err)
			}
		}
	}
	return items, nil
}

// Image returns the picture of one word of a lesson, the lesson's own or else the word bank's, or
// ErrImageNotFound.
func (r *Reader) Image(ctx context.Context, id, lemma string) (WordImage, error) {
	lemma = normalize(lemma)
	if r.images != nil {
		img, err := r.images.Get(ctx, id, lemma)
		if !errors.Is(err, ErrImageNotFound) {
			return img, err
		}
	}
	if r.bank == nil {
		return WordImage{}, ErrImageNotFound
	}
	return r.bank.Image(ctx, lemma)
}

// HasScene says the lesson has its scene picture (added 2026-11-05), served like a word picture
// under SceneKey.
func (r *Reader) HasScene(ctx context.Context, id string) (bool, error) {
	if r.images == nil {
		return false, nil
	}
	keys, err := r.images.Lemmas(ctx, id)
	if err != nil {
		return false, fmt.Errorf("lesson: image lemmas: %w", err)
	}
	_, scene := pictureKeys(keys)
	return scene, nil
}
