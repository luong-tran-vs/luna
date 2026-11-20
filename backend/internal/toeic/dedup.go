package toeic

import (
	"fmt"
	"strings"
	"unicode"
)

// dupThreshold is how alike two groups' words must be (shared words / all words) to count as the
// same group: 0.8 catches a sentence or a text written again with a word or a name changed, while
// two different texts on the same subject share far fewer words.
const dupThreshold = 0.8

// dupMinWords is the fewest words a group's material needs to be compared at all.
const dupMinWords = 3

// fingerprint is the words of what makes a group itself, per part: the sentence of Part 5, the spoken
// question of Part 2, the scene and the statements of Part 1, the script of Part 3/4 and the texts
// of Part 6/7. Blanks, punctuation, case and accents are left out.
type fingerprint struct {
	key   string
	words map[string]bool
}

func fingerprintOf(g Group) fingerprint {
	var parts []string
	switch g.Part {
	case 1:
		parts = append(parts, g.Scene)
		for _, q := range g.Questions {
			parts = append(parts, q.Options...)
		}
	case 2, 5:
		for _, q := range g.Questions {
			parts = append(parts, q.Prompt)
		}
		if g.Part == 2 && len(parts) == 0 && len(g.Script) > 0 {
			parts = append(parts, g.Script[0].Text)
		}
	case 3, 4:
		for _, t := range g.Script {
			parts = append(parts, t.Text)
		}
	default:
		for _, d := range g.Docs {
			parts = append(parts, d.Title)
			for _, b := range d.Body {
				parts = append(parts, b.Text)
				parts = append(parts, b.Items...)
				if b.Table != nil {
					for _, row := range b.Table.Rows {
						parts = append(parts, row...)
					}
				}
			}
		}
	}
	return fingerprint{key: g.Key, words: wordSet(strings.Join(parts, " "))}
}

// wordSet is the distinct words of a text, folded: "[[2]]" and "___" are not words.
func wordSet(s string) map[string]bool {
	out := map[string]bool{}
	for _, w := range strings.FieldsFunc(s, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '\'' }) {
		w = strings.Trim(strings.ToLower(w), "'")
		if w == "" || isBlankNumber(w) {
			continue
		}
		out[w] = true
	}
	return out
}

// isBlankNumber is the digit of a Part 6 blank "[[n]]" once its brackets are gone: one digit 1-4 is
// left out so two texts are not alike because of their blanks.
func isBlankNumber(w string) bool {
	return len(w) == 1 && w[0] >= '1' && w[0] <= '4'
}

// likeness is shared words over all words (Jaccard), from 0 to 1.
func likeness(a, b map[string]bool) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	if len(a) > len(b) {
		a, b = b, a
	}
	shared := 0
	for w := range a {
		if b[w] {
			shared++
		}
	}
	return float64(shared) / float64(len(a)+len(b)-shared)
}

// dupFinder knows the groups of a part and finds the one a new group repeats.
type dupFinder struct {
	known []fingerprint
}

func newDupFinder(groups []Group) *dupFinder {
	f := &dupFinder{}
	for _, g := range groups {
		f.add(g)
	}
	return f
}

func (f *dupFinder) add(g Group) {
	if fp := fingerprintOf(g); len(fp.words) >= dupMinWords {
		f.known = append(f.known, fp)
	}
}

// find is the known group most like g when it is at least dupThreshold alike: its key and how alike.
func (f *dupFinder) find(g Group) (string, float64, bool) {
	fp := fingerprintOf(g)
	if len(fp.words) < dupMinWords {
		return "", 0, false
	}
	best, bestKey := 0.0, ""
	for _, k := range f.known {
		if l := likeness(fp.words, k.words); l > best {
			best, bestKey = l, k.key
		}
	}
	return bestKey, best, best >= dupThreshold
}

// dupMessage says which group a dropped draft repeats.
func dupMessage(key string, alike float64) string {
	return fmt.Sprintf("Trùng với cụm %s (giống %d%%)", key, int(alike*100+0.5))
}
