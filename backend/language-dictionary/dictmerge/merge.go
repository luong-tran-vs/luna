package main

import (
	"regexp"
	"strings"
)

// Sources of a meaning, stored in the sources table of the merged dictionary.
const (
	sourceMinhqnd = 1
	sourceHND     = 2
)

var sourceNames = map[int]string{
	sourceMinhqnd: "minhqnd/dictionary v2.0.0 (Wiktionary)",
	sourceHND:     "Hồ Ngọc Đức, Free Vietnamese Dictionary Project (anhviet109K)",
}

type meaning struct {
	POS    string
	Text   string
	Source int
}

type pronunciation struct {
	IPA    string
	Region string
}

// entry is one English word of the merged dictionary.
type entry struct {
	Word     string
	Meanings []meaning
	IPAs     []pronunciation
	// seen holds the normalized glosses of Meanings, to skip a meaning already given.
	seen map[string]bool
}

// stats counts what the merge did, printed at the end.
type stats struct {
	MinhqndWords, HNDOnlyWords, HNDFilledWords int
	MinhqndMeanings, HNDAdded, HNDSkipped      int
	HNDIPAs                                    int
}

// dictionary is the merged dictionary: entries by word, in the order they were first added.
type dictionary struct {
	order   []string
	entries map[string]*entry
	// empty holds the first source's words with no Vietnamese meaning ("music"), to count the
	// ones the second source fills.
	empty map[string]bool
	stats stats
}

func newDictionary() *dictionary {
	return &dictionary{entries: map[string]*entry{}, empty: map[string]bool{}}
}

func (d *dictionary) get(word string) (*entry, bool) {
	e, ok := d.entries[word]
	if !ok {
		e = &entry{Word: word, seen: map[string]bool{}}
		d.entries[word] = e
		d.order = append(d.order, word)
	}
	return e, ok
}

// addMinhqnd adds a word of the first source: all its meanings and pronunciations, as they are.
func (d *dictionary) addMinhqnd(word string, meanings []meaning, ipas []pronunciation) {
	if len(meanings) == 0 {
		if _, ok := d.entries[word]; !ok {
			d.empty[word] = true
		}
		return
	}
	delete(d.empty, word)
	e, existed := d.get(word)
	if !existed {
		d.stats.MinhqndWords++
	}
	for _, m := range meanings {
		e.Meanings = append(e.Meanings, m)
		for _, g := range glosses(m.Text) {
			e.seen[g] = true
		}
	}
	d.stats.MinhqndMeanings += len(meanings)
	for _, p := range ipas {
		if !hasIPA(e.IPAs, p) {
			e.IPAs = append(e.IPAs, p)
		}
	}
}

// addHND adds a word of the second source after the first one: a meaning is skipped when every
// idea in it is already given ("bài hát, khúc hát" is added for its "khúc hát"); its pronunciation
// is used only when the word has none.
func (d *dictionary) addHND(h hndEntry) {
	e, _ := d.get(h.Word)
	hadMeanings := len(e.Meanings) > 0
	added := 0
	for _, m := range h.Meanings {
		gs := glosses(m.Text)
		isNew := false
		for _, g := range gs {
			if !e.seen[g] {
				isNew = true
				break
			}
		}
		if !isNew {
			d.stats.HNDSkipped++
			continue
		}
		e.Meanings = append(e.Meanings, m)
		for _, g := range gs {
			e.seen[g] = true
		}
		added++
	}
	d.stats.HNDAdded += added
	if !hadMeanings && added > 0 {
		if d.empty[h.Word] {
			d.stats.HNDFilledWords++
		} else {
			d.stats.HNDOnlyWords++
		}
	}
	if len(e.IPAs) == 0 && h.IPA != "" && len(e.Meanings) > 0 {
		e.IPAs = append(e.IPAs, pronunciation{IPA: h.IPA})
		d.stats.HNDIPAs++
	}
}

func hasIPA(list []pronunciation, p pronunciation) bool {
	for _, q := range list {
		if q == p {
			return true
		}
	}
	return false
}

var (
	parenthetical = regexp.MustCompile(`\([^)]*\)`)
	glossTrim     = " \t.:!?…\"'“”"
)

// glosses splits a meaning into its ideas, normalized for comparison: "(âm nhạc) Bài hát, khúc
// hát." gives ["bài hát", "khúc hát"]. Remarks in brackets are left out; a meaning that is only a
// remark is compared as a whole.
func glosses(text string) []string {
	var out []string
	for _, part := range strings.FieldsFunc(parenthetical.ReplaceAllString(text, " "), func(r rune) bool {
		return r == ',' || r == ';'
	}) {
		if g := normalizeGloss(part); g != "" {
			out = append(out, g)
		}
	}
	if len(out) == 0 {
		if g := normalizeGloss(text); g != "" {
			out = append(out, g)
		}
	}
	return out
}

func normalizeGloss(s string) string {
	return strings.Trim(strings.Join(strings.Fields(strings.ToLower(s)), " "), glossTrim)
}

// normalizeWord is how words are stored and looked up: lowercase, single spaces.
func normalizeWord(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(s)), " ")
}
