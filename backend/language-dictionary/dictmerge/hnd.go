package main

import (
	"bufio"
	"io"
	"strings"
)

// hndEntry is one entry of the Hồ Ngọc Đức English–Vietnamese dictionary (Free Vietnamese
// Dictionary Project, anhviet109K.txt).
type hndEntry struct {
	Word     string
	IPA      string // "/ˈmjuːzɪk/" style, "" when missing or unusable
	Meanings []meaning
}

// parseHND reads the anhviet109K.txt format:
//
//	@music /'mju:zik/      an entry: the word and its pronunciation
//	*  danh từ             a part of speech for the meanings below
//	- nhạc, âm nhạc        a meaning
//	=to have an ear...+ …  an example (skipped)
//	!to give up            a phrase or idiom; the meanings ("- …") below it are its own, until the
//	- bỏ, từ bỏ            next "*", "@" or "!" (several "!" lines in a row share the meanings)
//
// Phrases become entries of their own ("give up"), after the words. Entries without any meaning
// (e.g. "@went" with only "* xem go") are left out.
func parseHND(r io.Reader) ([]hndEntry, error) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)

	var out []hndEntry
	var cur *hndEntry
	pos := ""
	inIdiom := false
	flush := func() {
		if cur != nil && len(cur.Meanings) > 0 {
			out = append(out, *cur)
		}
		cur = nil
	}

	// Phrases by text, in the order first met; the same phrase under several words is merged.
	phrases := map[string]*hndEntry{}
	var phraseOrder []string
	var group []string // the phrases of the "!" lines being read
	groupOpen := false // more "!" lines may still join the group (no meaning read yet)
	addPhraseMeaning := func(text string) {
		for _, p := range group {
			e, ok := phrases[p]
			if !ok {
				e = &hndEntry{Word: p}
				phrases[p] = e
				phraseOrder = append(phraseOrder, p)
			}
			e.Meanings = append(e.Meanings, meaning{Text: text, Source: sourceHND})
		}
	}
	first := true
	for sc.Scan() {
		line := sc.Text()
		if first {
			line = strings.TrimPrefix(line, "\ufeff")
			first = false
		}
		line = strings.TrimRight(line, " \t\r")
		switch {
		case strings.HasPrefix(line, "@"):
			flush()
			word, ipa := splitHeadword(line[1:])
			if word == "" {
				continue
			}
			cur = &hndEntry{Word: word, IPA: hndIPA(word, ipa)}
			pos, inIdiom, group = "", false, nil
		case cur == nil:
			continue
		case strings.HasPrefix(line, "*"):
			head, _, _ := strings.Cut(line[1:], "+")
			pos, inIdiom, group = hndPOS(head), false, nil
		case strings.HasPrefix(line, "!"):
			if !groupOpen {
				group = nil
			}
			if p := phraseOf(line[1:]); p != "" {
				group = append(group, p)
			}
			inIdiom, groupOpen = true, true
		case strings.HasPrefix(line, "- "):
			text := strings.TrimSpace(line[2:])
			switch {
			case text == "", isCrossReference(text):
			case inIdiom:
				groupOpen = false
				addPhraseMeaning(text)
			default:
				cur.Meanings = append(cur.Meanings, meaning{POS: pos, Text: text, Source: sourceHND})
			}
		}
	}
	flush()
	for _, p := range phraseOrder {
		out = append(out, *phrases[p])
	}
	return out, sc.Err()
}

// isCrossReference tells a "meaning" that only points to another entry, "(xem) by": it says
// nothing to a learner, who is better offered "Hỏi AI".
func isCrossReference(text string) bool {
	rest, ok := strings.CutPrefix(text, "(xem)")
	return ok && len(strings.Fields(rest)) <= 1
}

// maxPhraseWords is the longest phrase kept: a learner selects at most 6 words.
const maxPhraseWords = 6

// phraseOf turns the text of a "!" line into the phrase a learner would select: "to give up" →
// "give up", "good morning!" → "good morning", "to give up (resign) all hope" → "give up all hope".
// Single words, phrases longer than 6 words and patterns ("to cast [in] a bone between...",
// "and/or") give "".
func phraseOf(s string) string {
	s = parenthetical.ReplaceAllString(s, " ")
	if strings.ContainsAny(s, "[]/=+…") || strings.Contains(s, "...") {
		return ""
	}
	p := strings.Trim(normalizeWord(s), " !?.,;:")
	p = strings.TrimPrefix(p, "to ")
	if n := len(strings.Fields(p)); n < 2 || n > maxPhraseWords {
		return ""
	}
	return p
}

// splitHeadword splits "music /'mju:zik/" into the word and the text between the first two "/".
// "acid-proof /'æsid'pru:f/ (acid-resisting) /…/" keeps only the first pronunciation.
func splitHeadword(s string) (word, ipa string) {
	head, rest, found := strings.Cut(s, " /")
	word = normalizeWord(head)
	if !found {
		return word, ""
	}
	ipa, _, _ = strings.Cut(rest, "/")
	return word, strings.TrimSpace(ipa)
}

// hndIPA turns the dictionary's ASCII-flavoured transcription ("'mju:zik", ",ʌndə'stænd",
// "si:ɳ") into IPA between slashes ("/ˈmjuːzik/"). Unusable values give "": empty, the word
// itself (some entries repeat the word, "/problem/"), or a space inside a single word (lost
// characters).
func hndIPA(word, ipa string) string {
	ipa = strings.TrimSpace(ipa)
	if ipa == "" || strings.EqualFold(ipa, word) {
		return ""
	}
	if !strings.Contains(word, " ") && strings.Contains(ipa, " ") && !strings.Contains(ipa, ", ") {
		return ""
	}
	var b strings.Builder
	runes := []rune(ipa)
	for i, r := range runes {
		switch {
		case r == '\'':
			b.WriteRune('ˈ')
		case r == ',' && (i+1 == len(runes) || runes[i+1] != ' '):
			// A comma before a syllable is secondary stress; ", " separates two pronunciations.
			b.WriteRune('ˌ')
		case r == 'ɳ':
			b.WriteRune('ŋ')
		case r == ':':
			b.WriteRune('ː')
		default:
			b.WriteRune(r)
		}
	}
	return "/" + b.String() + "/"
}

// hndPOS maps the dictionary's Vietnamese part-of-speech line ("danh từ", "ngoại động từ",
// "tính từ & phó từ", …) to the codes of the merged dictionary (N, V, A, R, …), "" when unknown.
func hndPOS(head string) string {
	h := strings.ToLower(strings.TrimSpace(head))
	prefixes := []struct{ prefix, code string }{
		{"danh từ", "N"},
		{"ngoại động từ", "V"},
		{"nội động từ", "V"},
		{"động từ", "V"},
		{"tính từ", "A"},
		{"phó từ", "R"},
		{"đại từ", "P"},
		{"giới từ", "PREP"},
		{"liên từ", "C"},
		{"thán từ", "I"},
	}
	for _, p := range prefixes {
		if strings.HasPrefix(h, p.prefix) {
			return p.code
		}
	}
	return ""
}
