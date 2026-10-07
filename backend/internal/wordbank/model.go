// Package wordbank is the shared word bank (F24): one entry per English base form with its
// Vietnamese meaning, IPA and picture, managed by admins and used by every lesson and topic.
package wordbank

import (
	"errors"
	"strings"
	"time"
)

const (
	// MaxLemma is the longest word or phrase, in characters.
	MaxLemma = 100
	// MaxMeaning is the longest Vietnamese meaning, in characters.
	MaxMeaning = 300
	// MaxIPA is the longest IPA, in characters.
	MaxIPA = 100
	// PageSize is how many words one page of the admin list holds.
	PageSize = 50
	// MaxSearch is the longest search text, in characters.
	MaxSearch = 100
)

// Word is one entry of the bank. Lemma is its key: lowercase, single spaces.
type Word struct {
	Lemma     string
	MeaningVi string
	IPA       string
	// ImageAt is when the picture was saved, zero when the word has none.
	ImageAt   time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

// HasImage says the word has a picture.
func (w Word) HasImage() bool { return !w.ImageAt.IsZero() }

// Image is the picture of a word: a JPEG of at most 360 px, like the lesson pictures (F23).
type Image struct {
	Lemma     string
	MIME      string
	Data      []byte
	CreatedAt time.Time
}

// Missing filters the list to the words lacking one thing; "" lists every word.
type Missing string

const (
	MissingNone    Missing = ""
	MissingImage   Missing = "image"
	MissingIPA     Missing = "ipa"
	MissingMeaning Missing = "meaning"
)

// ListQuery selects a page of words, by lemma order. Search matches part of the lemma or meaning.
type ListQuery struct {
	Search  string
	Missing Missing
	Limit   int
	Skip    int
}

var (
	// ErrNotFound means the bank has no such word.
	ErrNotFound = errors.New("wordbank: word not found")
	// ErrExists means the word is already in the bank.
	ErrExists = errors.New("wordbank: word already exists")
	// ErrImageNotFound means the word has no picture.
	ErrImageNotFound = errors.New("wordbank: image not found")
	// ErrImagesUnavailable means drawing pictures is not set up on this server.
	ErrImagesUnavailable = errors.New("wordbank: image generation is not available")
	// ErrDrawFailed means the AI did not give a usable picture.
	ErrDrawFailed = errors.New("wordbank: drawing the picture failed")
)

// ValidationError lists the invalid fields with a Vietnamese message each.
type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string { return "wordbank: invalid input" }

// Normalize is the key form of a word: lowercase, curly apostrophes made straight, single spaces
// (the same form as lesson lemmas).
func Normalize(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.ReplaceAll(s, "’", "'"))), " ")
}
