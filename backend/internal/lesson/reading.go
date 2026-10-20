package lesson

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/sync/singleflight"

	"github.com/luongtran/luna/backend/internal/ai"
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
	// SourceSentence: no meaning of the selection itself, the translation of its whole sentence.
	SourceSentence = "sentence"
)

// ErrLookupNotFound means neither the lesson annotations, the dictionary nor a sentence
// translation give a meaning.
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
	// Quiz is nil when the lesson has no comprehension questions (F15).
	Quiz        *QuizView
	GrammarNote *GrammarNote
	// GrammarPointID is the syllabus point the lesson teaches, "" when none is assigned (F20).
	GrammarPointID string
	// Turns is the lesson as a conversation when it is a dialogue ("Name: text" lines), nil otherwise.
	Turns []TextTurn
	// SentenceTranslations is the Vietnamese translation of each sentence, by index; empty for a
	// lesson annotated before they were written.
	SentenceTranslations []string
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
	// Note explains an asked AI meaning in context (F9); empty otherwise.
	Note string
	// Context is the meaning in this lesson, shown first (added 2026-10-20); nil when unknown.
	Context *LookupContext
	// Others are a few dictionary meanings shown after Context, "Nghĩa khác".
	Others []LookupMeaning
}

// LookupContext is a word's meaning in the lesson, with the few words around it and their
// Vietnamese when known ("a great way to relax" = "một cách tuyệt vời để thư giãn").
type LookupContext struct {
	MeaningVi string
	Phrase    string
	PhraseVi  string
}

// maxOthers is how many dictionary meanings follow the meaning in context.
const maxOthers = 3

// Reader serves the Reading step. Lookups never call the AI provider: meanings come from stored
// annotations, stored AI explanations or the offline dictionary. Only Ask calls the AI (F9).
type Reader struct {
	lessons Repository
	dict    Dictionary
	topics  Topics
	answers AnswerRepository
	// asks and ai serve "Hỏi AI" (F9); group makes one AI call per key at a time.
	asks  AskRepository
	ai    ai.Provider
	group singleflight.Group
	// images serves the word pictures (F23); nil when there are none.
	images ImageRepository
	// bank is the shared word bank (F24): IPA and pictures for words the lesson has none of.
	bank WordBank
	// usage limits the AI asks of each learner per lesson (AskLimit); nil means no limit.
	usage AskUsageRepository
	// glosses give the meaning in context of every word of a lesson; nil when there are none.
	glosses GlossRepository
}

// WithGlosses lets lookups give the meaning in context of words the annotations do not cover.
func (r *Reader) WithGlosses(g GlossRepository) *Reader {
	r.glosses = g
	return r
}

// NewReader returns a Reader.
func NewReader(lessons Repository, dict Dictionary, topics Topics, answers AnswerRepository, asks AskRepository,
	provider ai.Provider,
) *Reader {
	return &Reader{lessons: lessons, dict: dict, topics: topics, answers: answers, asks: asks, ai: provider}
}

// View returns the lesson for reading, with base forms for highlighting saved words, the
// grammar note and the user's comprehension quiz (answers only for answered questions).
func (r *Reader) View(ctx context.Context, userID, id string) (ReadingView, error) {
	l, err := r.lessons.Get(ctx, id)
	if err != nil {
		return ReadingView{}, err
	}
	quiz, err := r.quiz(ctx, userID, l)
	if err != nil {
		return ReadingView{}, err
	}
	v := ReadingView{
		ID: l.ID, Title: l.Title, Level: l.Level,
		Sentences:  l.Sentences,
		Paragraphs: paragraphs(l.Content, len(l.Sentences)),
		Lemmas:     map[string]string{},
		Phrases:    []Phrase{},
		Quiz:       quiz, GrammarNote: l.Extras.GrammarNote, GrammarPointID: l.GrammarPointID,
		Turns: DialogueTurns(l.Content, l.Sentences),
		// One per sentence, or none (CleanTranslations).
		SentenceTranslations: l.Extras.SentenceTranslations,
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
// first, preferring that sentence, then an explanation asked of the AI for that sentence (F9),
// then the offline dictionary, and last the Vietnamese translation of the whole sentence (a phrase
// such as "like to run" is in no dictionary).
func (r *Reader) Lookup(ctx context.Context, lessonID, q string, sentence int) (LookupResult, error) {
	l, err := r.lessons.Get(ctx, lessonID)
	if err != nil {
		return LookupResult{}, err
	}
	text := normalize(q)

	glosses := r.lessonGlosses(ctx, l)
	if a, ok := matchAnnotation(l.Annotations, text, sentence); ok {
		res := LookupResult{
			Source: SourceAI, Text: text, Lemma: normalize(a.Lemma),
			Meanings: []LookupMeaning{{Text: a.MeaningVi}},
		}
		res.IPA = r.ipaOf(ctx, res.Lemma)
		res.Context = &LookupContext{MeaningVi: a.MeaningVi}
		if g, ok := matchGloss(glosses, text, sentence); ok {
			res.Context.Phrase, res.Context.PhraseVi = g.Phrase, g.PhraseVi
		}
		res.Others = r.others(ctx, res.Lemma, a.MeaningVi)
		return res, nil
	}
	// The meaning in context written for every word of the lesson (added 2026-10-20).
	if g, ok := matchGloss(glosses, text, sentence); ok {
		res := LookupResult{
			Source: SourceAI, Text: text, Lemma: g.Lemma, Meanings: []LookupMeaning{{Text: g.MeaningVi}},
			Context: &LookupContext{MeaningVi: g.MeaningVi, Phrase: g.Phrase, PhraseVi: g.PhraseVi},
		}
		res.IPA = r.ipaOf(ctx, res.Lemma)
		res.Others = r.others(ctx, res.Lemma, g.MeaningVi)
		return res, nil
	}

	// F9: an explanation someone already asked the AI for, in this sentence of this revision.
	if sentence >= 0 {
		key := AskKey{LessonID: l.ID, Revision: l.Revision, SentenceIndex: sentence, Text: text}
		if stored, ok, err := r.asks.Get(ctx, key); err != nil {
			return LookupResult{}, fmt.Errorf("lesson: stored explanation: %w", err)
		} else if ok {
			return r.askResult(ctx, stored), nil
		}
	}

	e, ok, err := r.dict.Resolve(ctx, text)
	if err != nil {
		return LookupResult{}, fmt.Errorf("lesson: dictionary: %w", err)
	}
	if !ok {
		if tr := l.Extras.SentenceTranslations; sentence >= 0 && sentence < len(tr) {
			return LookupResult{Source: SourceSentence, Text: text, Lemma: text, Meanings: []LookupMeaning{{Text: tr[sentence]}}}, nil
		}
		return LookupResult{}, ErrLookupNotFound
	}
	res := LookupResult{Source: SourceDictionary, Text: text, Lemma: e.Word, IPA: r.ipaOf(ctx, e.Word)}
	for _, m := range e.Meanings {
		res.Meanings = append(res.Meanings, LookupMeaning(m))
	}
	return res, nil
}

// lessonGlosses returns the lesson's glosses when they were written for its current revision.
// Best effort: a failed read gives none.
func (r *Reader) lessonGlosses(ctx context.Context, l Lesson) []Gloss {
	if r.glosses == nil {
		return nil
	}
	rev, gs, ok, err := r.glosses.Get(ctx, l.ID)
	if err != nil || !ok || rev != l.Revision {
		return nil
	}
	return gs
}

// matchGloss finds the gloss of text the way matchAnnotation finds an annotation.
func matchGloss(gs []Gloss, text string, sentence int) (Gloss, bool) {
	if len(gs) == 0 {
		return Gloss{}, false
	}
	anns := make([]Annotation, len(gs))
	for i, g := range gs {
		anns[i] = Annotation{Text: g.Text, Lemma: g.Lemma, MeaningVi: g.MeaningVi, SentenceIndex: g.SentenceIndex}
	}
	a, ok := matchAnnotation(anns, text, sentence)
	if !ok {
		return Gloss{}, false
	}
	for _, g := range gs {
		if g.Text == a.Text && g.Lemma == a.Lemma && g.SentenceIndex == a.SentenceIndex {
			return g, true
		}
	}
	return Gloss{}, false
}

// others are up to maxOthers dictionary meanings of lemma besides the one in context. Best
// effort: a failed lookup gives none.
func (r *Reader) others(ctx context.Context, lemma, inContext string) []LookupMeaning {
	e, ok, err := r.dict.Resolve(ctx, lemma)
	if err != nil || !ok {
		return nil
	}
	same := strings.ToLower(strings.TrimSpace(inContext))
	var out []LookupMeaning
	for _, m := range e.Meanings {
		text := strings.TrimSpace(m.Text)
		if text == "" || strings.ToLower(strings.TrimRight(text, ".")) == same {
			continue
		}
		out = append(out, LookupMeaning(m))
		if len(out) == maxOthers {
			break
		}
	}
	return out
}

// ipaOf is the transcription shown for lemma: the word bank's, else the standard one
// (dictionary.IPA). Best effort: a failed lookup shows none.
func (r *Reader) ipaOf(ctx context.Context, lemma string) string {
	if r.bank != nil {
		if words, err := r.bank.Words(ctx, []string{lemma}); err == nil && words[lemma].IPA != "" {
			return words[lemma].IPA
		}
	}
	ipa, err := dictionary.IPA(ctx, r.dict, lemma)
	if err != nil {
		return ""
	}
	return ipa
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
