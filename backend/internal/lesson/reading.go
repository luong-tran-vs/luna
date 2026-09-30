package lesson

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/luongtran/luna/backend/internal/dictionary"
)

// Dictionary resolves a word or phrase to the dictionary entry of its base form
// (Entry.Word is the lemma). Implemented by internal/dictionary.
type Dictionary interface {
	Resolve(ctx context.Context, word string) (dictionary.Entry, bool, error)
}

// Lookup sources shown to the learner.
const (
	SourceAI         = "ai"
	SourceDictionary = "dictionary"
)

// ErrLookupNotFound means neither the lesson annotations nor the dictionary know the text.
var ErrLookupNotFound = errors.New("lesson: no meaning found")

// Phrase is a multi-word annotation, used to highlight saved phrases.
type Phrase struct {
	Text  string
	Lemma string
}

// ReadingView is a lesson as a learner reads it: no annotations or background-work details.
type ReadingView struct {
	ID         string
	Title      string
	Level      Level
	Topic      string
	Sentences  []Sentence
	Paragraphs [][]int
	// Lemmas maps each distinct lowercase word of the lesson to its base form, when known.
	Lemmas  map[string]string
	Phrases []Phrase
}

// LookupMeaning is one meaning; POS is the dictionary part-of-speech code (empty for AI).
type LookupMeaning struct {
	POS  string
	Text string
}

// LookupResult is what the reading popup shows.
type LookupResult struct {
	Source   string
	Text     string
	Lemma    string
	IPA      string
	Meanings []LookupMeaning
}

// Reader serves the Reading step. It never calls the AI provider: meanings come from stored
// annotations or the offline dictionary.
type Reader struct {
	lessons Repository
	dict    Dictionary
	topics  Topics
}

// NewReader returns a Reader.
func NewReader(lessons Repository, dict Dictionary, topics Topics) *Reader {
	return &Reader{lessons: lessons, dict: dict, topics: topics}
}

// View returns the lesson for reading, with base forms for highlighting saved words.
func (r *Reader) View(ctx context.Context, id string) (ReadingView, error) {
	l, err := r.lessons.Get(ctx, id)
	if err != nil {
		return ReadingView{}, err
	}
	v := ReadingView{
		ID: l.ID, Title: l.Title, Level: l.Level,
		Sentences:  l.Sentences,
		Paragraphs: paragraphs(l.Content, len(l.Sentences)),
		Lemmas:     map[string]string{},
		Phrases:    []Phrase{},
	}
	switch t, err := r.topics.Get(ctx, l.TopicID); {
	case err == nil:
		v.Topic = t.Name
	case !errors.Is(err, ErrTopicNotFound):
		return ReadingView{}, fmt.Errorf("lesson: topic: %w", err)
	}

	annotated := map[string]string{}
	seenPhrase := map[string]bool{}
	for _, a := range l.Annotations {
		text := normalize(a.Text)
		if strings.Contains(text, " ") {
			if !seenPhrase[text] {
				seenPhrase[text] = true
				v.Phrases = append(v.Phrases, Phrase{Text: text, Lemma: normalize(a.Lemma)})
			}
			continue
		}
		annotated[text] = normalize(a.Lemma)
	}

	for _, s := range l.Sentences {
		for _, w := range Words(s.Text) {
			key := strings.ToLower(w)
			if _, done := v.Lemmas[key]; done {
				continue
			}
			if lemma, ok := annotated[key]; ok {
				v.Lemmas[key] = lemma
				continue
			}
			e, ok, err := r.dict.Resolve(ctx, key)
			if err != nil {
				return ReadingView{}, fmt.Errorf("lesson: resolve %q: %w", key, err)
			}
			if ok {
				v.Lemmas[key] = e.Word
			}
		}
	}
	return v, nil
}

// Lookup finds the meaning of q (a word or phrase from sentence): the lesson's annotations
// first, preferring that sentence, then the offline dictionary.
func (r *Reader) Lookup(ctx context.Context, lessonID, q string, sentence int) (LookupResult, error) {
	l, err := r.lessons.Get(ctx, lessonID)
	if err != nil {
		return LookupResult{}, err
	}
	text := normalize(q)

	if a, ok := matchAnnotation(l.Annotations, text, sentence); ok {
		res := LookupResult{
			Source: SourceAI, Text: text, Lemma: normalize(a.Lemma),
			Meanings: []LookupMeaning{{Text: a.MeaningVi}},
		}
		if e, ok, err := r.dict.Resolve(ctx, res.Lemma); err == nil && ok {
			res.IPA = e.IPA
		}
		return res, nil
	}

	e, ok, err := r.dict.Resolve(ctx, text)
	if err != nil {
		return LookupResult{}, fmt.Errorf("lesson: dictionary: %w", err)
	}
	if !ok {
		return LookupResult{}, ErrLookupNotFound
	}
	res := LookupResult{Source: SourceDictionary, Text: text, Lemma: e.Word, IPA: e.IPA}
	for _, m := range e.Meanings {
		res.Meanings = append(res.Meanings, LookupMeaning(m))
	}
	return res, nil
}

// matchAnnotation finds an annotation for text by its text, its lemma, or (for single words)
// one of the word's base-form candidates, preferring annotations of the given sentence.
func matchAnnotation(anns []Annotation, text string, sentence int) (Annotation, bool) {
	keys := []string{text}
	if !strings.Contains(text, " ") {
		keys = dictionary.Candidates(text)
	}
	matches := func(a Annotation) bool {
		at, al := normalize(a.Text), normalize(a.Lemma)
		for _, k := range keys {
			if k == at || k == al {
				return true
			}
		}
		return false
	}

	var fallback *Annotation
	for i := range anns {
		if !matches(anns[i]) {
			continue
		}
		if anns[i].SentenceIndex == sentence {
			return anns[i], true
		}
		if fallback == nil {
			fallback = &anns[i]
		}
	}
	if fallback != nil {
		return *fallback, true
	}
	return Annotation{}, false
}

// paragraphs groups sentence indexes by the blank-line paragraphs of content. If re-splitting
// does not give n sentences, everything is one paragraph.
func paragraphs(content string, n int) [][]int {
	var out [][]int
	next := 0
	for _, p := range paragraphBreak.Split(strings.ReplaceAll(content, "\r\n", "\n"), -1) {
		count := len(SplitSentences(p))
		if count == 0 {
			continue
		}
		group := make([]int, count)
		for i := range group {
			group[i] = next
			next++
		}
		out = append(out, group)
	}
	if next != n {
		all := make([]int, n)
		for i := range all {
			all[i] = i
		}
		return [][]int{all}
	}
	return out
}

// normalize lowercases text, unifies apostrophes and collapses spaces.
func normalize(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.ReplaceAll(s, "’", "'"))), " ")
}
