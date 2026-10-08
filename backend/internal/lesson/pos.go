package lesson

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	"github.com/luongtran/luna/backend/internal/dictionary"
)

// Parts of speech of an annotation, as stored and sent to clients ("" when unknown).
const (
	POSNoun         = "noun"
	POSVerb         = "verb"
	POSAdjective    = "adjective"
	POSAdverb       = "adverb"
	POSPronoun      = "pronoun"
	POSPreposition  = "preposition"
	POSConjunction  = "conjunction"
	POSDeterminer   = "determiner"
	POSInterjection = "interjection"
	POSPhrasalVerb  = "phrasal verb"
	POSPhrase       = "phrase"
)

// POSValues lists the parts of speech the AI may give, in the order of its prompt.
var POSValues = []string{
	POSNoun, POSVerb, POSAdjective, POSAdverb, POSPronoun, POSPreposition, POSConjunction,
	POSDeterminer, POSInterjection, POSPhrasalVerb, POSPhrase,
}

// posAliases maps what the AI or the dictionary may write to a part of speech: the dictionary's
// codes (N, V, A, ADJ, ADV, R, P, PREP, C, I) and common English names.
var posAliases = map[string]string{
	"n": POSNoun, "noun": POSNoun,
	"v": POSVerb, "verb": POSVerb,
	"a": POSAdjective, "adj": POSAdjective, "adjective": POSAdjective,
	"adv": POSAdverb, "r": POSAdverb, "adverb": POSAdverb,
	"p": POSPronoun, "pron": POSPronoun, "pronoun": POSPronoun,
	"prep": POSPreposition, "preposition": POSPreposition,
	"c": POSConjunction, "conj": POSConjunction, "conjunction": POSConjunction,
	"det": POSDeterminer, "determiner": POSDeterminer, "article": POSDeterminer,
	"i": POSInterjection, "interj": POSInterjection, "interjection": POSInterjection, "exclamation": POSInterjection,
	"phrasal verb": POSPhrasalVerb, "phrasal-verb": POSPhrasalVerb,
	"phrase": POSPhrase, "idiom": POSPhrase, "expression": POSPhrase, "collocation": POSPhrase,
}

// NormalizePOS returns the part of speech s stands for, "" when it is none of them.
func NormalizePOS(s string) string {
	return posAliases[strings.ToLower(strings.TrimSpace(s))]
}

// dictionaryPOS guesses the part of speech of a lesson word whose annotation has none (lessons
// annotated before the AI was asked for it). It must not show a wrong one, so it answers "" unless
// one of these is clear:
//   - several words make a phrase;
//   - the word in the sentence is an -ed form of its base ("watched"), so a verb;
//   - the dictionary gives the word one part of speech only;
//   - the lesson's Vietnamese meaning matches the senses of one part of speech best ("watch",
//     "xem" matches the verb "xem, nhìn", not the noun "đồng hồ").
func dictionaryPOS(ctx context.Context, d dictionary.Resolver, lemma, text, meaningVi string) (string, error) {
	lemma = strings.ToLower(strings.TrimSpace(lemma))
	text = strings.ToLower(strings.TrimSpace(text))
	if strings.Contains(lemma, " ") {
		return POSPhrase, nil
	}
	if text != lemma && strings.HasSuffix(text, "ed") && !strings.HasSuffix(lemma, "ed") {
		return POSVerb, nil
	}
	if d == nil {
		return "", nil
	}
	e, ok, err := d.Resolve(ctx, lemma)
	if err != nil {
		return "", fmt.Errorf("dictionary: resolve %q: %w", lemma, err)
	}
	if !ok {
		return "", nil
	}
	senses := e.Senses
	if len(senses) == 0 {
		senses = e.Meanings
	}
	kinds := map[string]bool{}
	for _, s := range senses {
		if p := NormalizePOS(s.POS); p != "" {
			kinds[p] = true
		}
	}
	switch len(kinds) {
	case 0:
		return "", nil
	case 1:
		for p := range kinds {
			return p, nil
		}
	}
	return bestSensePOS(senses, meaningVi), nil
}

// meaningFiller are Vietnamese words that say nothing about which sense is meant.
var meaningFiller = map[string]bool{
	"đã": true, "đang": true, "sẽ": true, "sự": true, "việc": true, "cái": true, "những": true, "các": true,
	"một": true, "được": true, "bị": true, "của": true, "là": true, "và": true, "hoặc": true, "với": true,
}

// syllables are the lowercase Vietnamese syllables of s, without filler words.
func syllables(s string) map[string]bool {
	out := map[string]bool{}
	for _, w := range strings.FieldsFunc(strings.ToLower(s), func(r rune) bool { return !unicode.IsLetter(r) }) {
		if !meaningFiller[w] {
			out[w] = true
		}
	}
	return out
}

// bestSensePOS is the part of speech of the senses sharing most syllables with meaningVi, "" when
// none shares any or the best ones disagree.
func bestSensePOS(senses []dictionary.Meaning, meaningVi string) string {
	want := syllables(meaningVi)
	best, pos := 0, ""
	for _, s := range senses {
		p := NormalizePOS(s.POS)
		if p == "" {
			continue
		}
		n := 0
		for w := range syllables(s.Text) {
			if want[w] {
				n++
			}
		}
		switch {
		case n > best:
			best, pos = n, p
		case n == best && n > 0 && p != pos:
			pos = "" // a tie between two parts of speech: unclear
		}
	}
	return pos
}
