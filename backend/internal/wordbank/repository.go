package wordbank

import "context"

// Repository stores the words of the bank and their pictures. Implementations live in
// internal/storage.
type Repository interface {
	// List returns a page of the words matching q, by lemma, and how many match in all.
	List(ctx context.Context, q ListQuery) ([]Word, int, error)
	// Get returns ErrNotFound when the bank has no such word.
	Get(ctx context.Context, lemma string) (Word, error)
	// Find returns the words of the bank among lemmas, in no particular order.
	Find(ctx context.Context, lemmas []string) ([]Word, error)
	// Create returns ErrExists when the lemma is taken.
	Create(ctx context.Context, w Word) error
	// InsertMissing adds the words whose lemma is not in the bank yet and returns how many it added.
	InsertMissing(ctx context.Context, ws []Word) (int, error)
	// Update sets the meaning, IPA and update time of a word; ErrNotFound when it does not exist.
	Update(ctx context.Context, w Word) error
	// Delete removes a word and its picture; ErrNotFound when it does not exist.
	Delete(ctx context.Context, lemma string) error
	// Image returns ErrImageNotFound when the word has no picture.
	Image(ctx context.Context, lemma string) (Image, error)
	// SaveImage replaces the picture of a word; ErrNotFound when the word does not exist.
	SaveImage(ctx context.Context, img Image) error
	// DeleteImage removes the picture of a word; a word without one is not an error.
	DeleteImage(ctx context.Context, lemma string) error
}
