package lesson

import (
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/luongtran/luna/backend/internal/ai"
)

// Limits of a practice (F17).
const (
	maxPracticeWords  = 30
	maxPracticeText   = 200
	maxGrammarTip     = 400
	minDialogueTurns  = 4
	maxDialogueTurns  = 10
	maxTranslations   = 5
	maxDistractors    = 4
	minAnswerTiles    = 2
	maxAnswerTiles    = 15
	maxBlanks         = 5
	maxExtraBankWords = 2
)

// Example is an English example sentence for one vocabulary word, by normalized lemma.
type Example struct {
	Lemma    string
	Sentence string
	// MeaningVi is the sentence in Vietnamese; empty for practice generated before it was asked for.
	MeaningVi string
}

// Turn is one line of the sample dialogue; Speaker is 0 or 1.
type Turn struct {
	Speaker   int
	Text      string
	MeaningVi string
}

// Dialogue is the sample conversation between two named speakers.
type Dialogue struct {
	Speakers []string
	Turns    []Turn
}

// Translation is a Vietnamese sentence the learner translates by ordering word tiles.
type Translation struct {
	Vi          string
	En          string
	Distractors []string
}

// Practice is the cleaned vocabulary practice of a lesson (F17). Any part may be empty.
type Practice struct {
	ObjectiveVi  string
	Examples     []Example
	Dialogue     *Dialogue
	GrammarTipVi string
	Translations []Translation
}

// practiceWords lists the lesson's annotated words for the AI, one per normalized lemma in the
// order they first appear, at most maxPracticeWords.
func practiceWords(l Lesson) []ai.PracticeWord {
	anns := slices.Clone(l.Annotations)
	slices.SortStableFunc(anns, func(a, b Annotation) int { return a.SentenceIndex - b.SentenceIndex })
	seen := map[string]bool{}
	var out []ai.PracticeWord
	for _, a := range anns {
		lemma := normalize(a.Lemma)
		if lemma == "" || seen[lemma] {
			continue
		}
		seen[lemma] = true
		out = append(out, ai.PracticeWord{Lemma: lemma, Text: a.Text, MeaningVi: a.MeaningVi})
		if len(out) == maxPracticeWords {
			break
		}
	}
	return out
}

// CleanPractice keeps the parts of an AI practice that pass the checks of research R4 and
// returns ErrNoValidPractice when no example, dialogue or translation is left.
func CleanPractice(p ai.Practice, words []ai.PracticeWord) (Practice, error) {
	out := Practice{
		ObjectiveVi:  limited(p.ObjectiveVi, maxPracticeText),
		GrammarTipVi: limited(p.GrammarTipVi, maxGrammarTip),
		Examples:     cleanExamples(p.Examples, words),
		Dialogue:     cleanDialogue(p.Dialogue),
		Translations: cleanTranslations(p.Translations, words),
	}
	if len(out.Examples) == 0 && out.Dialogue == nil && len(out.Translations) == 0 {
		return Practice{}, ErrNoValidPractice
	}
	return out, nil
}

func cleanExamples(items []ai.Example, words []ai.PracticeWord) []Example {
	byLemma := map[string]ai.PracticeWord{}
	for _, w := range words {
		byLemma[normalize(w.Lemma)] = w
	}
	var out []Example
	seen := map[string]bool{}
	for _, it := range items {
		lemma := normalize(it.Lemma)
		w, ok := byLemma[lemma]
		sentence := limited(it.Sentence, maxPracticeText)
		if !ok || seen[lemma] || sentence == "" || !containsWord(sentence, w) {
			continue
		}
		seen[lemma] = true
		out = append(out, Example{Lemma: lemma, Sentence: sentence, MeaningVi: limited(it.MeaningVi, maxPracticeText)})
	}
	return out
}

func cleanDialogue(d ai.Dialogue) *Dialogue {
	if len(d.Speakers) != 2 {
		return nil
	}
	out := &Dialogue{Speakers: []string{collapse(d.Speakers[0]), collapse(d.Speakers[1])}}
	if out.Speakers[0] == "" || out.Speakers[1] == "" {
		return nil
	}
	for _, t := range d.Turns {
		text, meaning := limited(t.Text, maxPracticeText), limited(t.MeaningVi, maxPracticeText)
		if (t.Speaker != 0 && t.Speaker != 1) || text == "" || meaning == "" {
			continue
		}
		out.Turns = append(out.Turns, Turn{Speaker: t.Speaker, Text: text, MeaningVi: meaning})
	}
	if len(out.Turns) < minDialogueTurns {
		return nil
	}
	out.Turns = out.Turns[:min(len(out.Turns), maxDialogueTurns)]
	return out
}

func cleanTranslations(items []ai.Translation, words []ai.PracticeWord) []Translation {
	var out []Translation
	for _, it := range items {
		vi, en := limited(it.Vi, maxPracticeText), limited(it.En, maxPracticeText)
		tiles := answerTiles(en)
		if vi == "" || len(tiles) < minAnswerTiles || len(tiles) > maxAnswerTiles || !containsAnyWord(en, words) {
			continue
		}
		out = append(out, Translation{Vi: vi, En: en, Distractors: cleanDistractors(it.Distractors, tiles)})
		if len(out) == maxTranslations {
			break
		}
	}
	return out
}

// cleanDistractors trims and dedupes distractors (ignoring case) and drops those equal to an
// answer tile, so every tile of a sentence is either needed or wrong.
func cleanDistractors(items, tiles []string) []string {
	taken := map[string]bool{}
	for _, t := range tiles {
		taken[strings.ToLower(t)] = true
		taken[strings.ToLower(strings.TrimFunc(t, isEdgePunct))] = true
	}
	var out []string
	for _, d := range items {
		d = collapse(d)
		key := strings.ToLower(d)
		if d == "" || taken[key] {
			continue
		}
		taken[key] = true
		out = append(out, d)
		if len(out) == maxDistractors {
			break
		}
	}
	return out
}

// answerTiles splits an English answer into the tiles the learner orders; punctuation stays
// attached to its word ("you," "Vietnam.").
func answerTiles(en string) []string { return strings.Fields(en) }

// Fill is the fill-in-the-blank exercise built from the dialogue (step 3).
type Fill struct {
	Turns  []FillTurn
	Blanks []Blank
	// WordBank holds every answer plus up to two other lesson words, in a fixed order; the
	// client shuffles it.
	WordBank []string
}

// FillTurn is one dialogue turn split into text parts and blanks.
type FillTurn struct {
	TurnIndex int
	Speaker   int
	MeaningVi string
	Parts     []Part
}

// Part is either plain text or a blank (index into Fill.Blanks).
type Part struct {
	Text  string
	Blank *int
}

// Blank is the word expected in a blank, written as in the dialogue.
type Blank struct {
	Answer string
}

// BuildFill blanks out lesson words in the dialogue: whole-word matches of a word's text or
// lemma, at most maxBlanks, taken round-robin over the turns and preferring words not blanked
// yet. It returns nil when the dialogue is missing or contains no lesson word.
func BuildFill(d *Dialogue, words []ai.PracticeWord) *Fill {
	if d == nil {
		return nil
	}
	cands := make([][]match, len(d.Turns))
	total := 0
	for i, t := range d.Turns {
		cands[i] = findMatches(t.Text, words)
		total += len(cands[i])
	}
	if total == 0 {
		return nil
	}

	picked := make([][]match, len(d.Turns))
	usedWord := map[int]bool{}
	for n := 0; n < maxBlanks; {
		progress := false
		for i := range cands {
			if n == maxBlanks || len(cands[i]) == 0 {
				continue
			}
			k := slices.IndexFunc(cands[i], func(m match) bool { return !usedWord[m.word] })
			k = max(k, 0) // every word is blanked already: take the first candidate
			m := cands[i][k]
			cands[i] = slices.Delete(cands[i], k, k+1)
			picked[i] = append(picked[i], m)
			usedWord[m.word] = true
			n++
			progress = true
		}
		if !progress {
			break
		}
	}

	f := &Fill{}
	for i, t := range d.Turns {
		ms := picked[i]
		slices.SortFunc(ms, func(a, b match) int { return a.start - b.start })
		ft := FillTurn{TurnIndex: i, Speaker: t.Speaker, MeaningVi: t.MeaningVi}
		pos := 0
		for _, m := range ms {
			if m.start > pos {
				ft.Parts = append(ft.Parts, Part{Text: t.Text[pos:m.start]})
			}
			idx := len(f.Blanks)
			ft.Parts = append(ft.Parts, Part{Blank: &idx})
			f.Blanks = append(f.Blanks, Blank{Answer: t.Text[m.start:m.end]})
			f.WordBank = append(f.WordBank, t.Text[m.start:m.end])
			pos = m.end
		}
		if pos < len(t.Text) {
			ft.Parts = append(ft.Parts, Part{Text: t.Text[pos:]})
		}
		f.Turns = append(f.Turns, ft)
	}
	extra := 0
	for i, w := range words {
		if extra == maxExtraBankWords {
			break
		}
		if !usedWord[i] {
			f.WordBank = append(f.WordBank, w.Text)
			extra++
		}
	}
	return f
}

// match is a whole-word occurrence of words[word] at text[start:end].
type match struct {
	start, end, word int
}

// findMatches returns the non-overlapping occurrences of the words in text, left to right;
// at one position the longest phrase wins.
func findMatches(text string, words []ai.PracticeWord) []match {
	toks := tokens(text)
	forms := wordForms(words)
	var out []match
	for i := 0; i < len(toks); {
		best, bestLen := -1, 0
		for _, f := range forms {
			if len(f.toks) > bestLen && hasPrefix(toks[i:], f.toks) {
				best, bestLen = f.word, len(f.toks)
			}
		}
		if best < 0 {
			i++
			continue
		}
		out = append(out, match{start: toks[i].start, end: toks[i+bestLen-1].end, word: best})
		i += bestLen
	}
	return out
}

// containsWord reports whether sentence contains the word's text or lemma as whole words,
// ignoring case.
func containsWord(sentence string, w ai.PracticeWord) bool {
	return len(findMatches(sentence, []ai.PracticeWord{w})) > 0
}

func containsAnyWord(sentence string, words []ai.PracticeWord) bool {
	return len(findMatches(sentence, words)) > 0
}

type form struct {
	word int
	toks []string
}

func wordForms(words []ai.PracticeWord) []form {
	var out []form
	for i, w := range words {
		for _, s := range []string{w.Text, w.Lemma} {
			var ts []string
			for _, t := range tokens(s) {
				ts = append(ts, t.norm)
			}
			if len(ts) > 0 {
				out = append(out, form{word: i, toks: ts})
			}
		}
	}
	return out
}

type token struct {
	start, end int
	norm       string
}

// tokens splits text into words: runs of letters, digits, apostrophes and hyphens, without
// apostrophes or hyphens at their edges. norm is lowercase with ’ written as '.
func tokens(text string) []token {
	var out []token
	start := -1
	flush := func(end int) {
		if start < 0 {
			return
		}
		s, e := start, end
		for s < e {
			r, size := utf8.DecodeRuneInString(text[s:e])
			if !isJoiner(r) {
				break
			}
			s += size
		}
		for e > s && isJoiner(lastRune(text[s:e])) {
			_, size := utf8.DecodeLastRuneInString(text[s:e])
			e -= size
		}
		if s < e {
			out = append(out, token{start: s, end: e, norm: normalize(text[s:e])})
		}
		start = -1
	}
	for i, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || isJoiner(r) {
			if start < 0 {
				start = i
			}
			continue
		}
		flush(i)
	}
	flush(len(text))
	return out
}

func isJoiner(r rune) bool { return r == '\'' || r == '’' || r == '-' }

func lastRune(s string) rune {
	r, _ := utf8.DecodeLastRuneInString(s)
	return r
}

func hasPrefix(toks []token, want []string) bool {
	if len(toks) < len(want) {
		return false
	}
	for i, w := range want {
		if toks[i].norm != w {
			return false
		}
	}
	return true
}

func isEdgePunct(r rune) bool { return unicode.IsPunct(r) }

// collapse trims s and collapses inner whitespace.
func collapse(s string) string { return strings.Join(strings.Fields(s), " ") }

// limited collapses s and returns "" when it is longer than limit runes.
func limited(s string, limit int) string {
	s = collapse(s)
	if utf8.RuneCountInString(s) > limit {
		return ""
	}
	return s
}
