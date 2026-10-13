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
//	!the Communist Party   an idiom; its meanings ("- …") are skipped until the next "*" or "@"
//
// Entries without any meaning (e.g. "@went" with only "* xem go") are left out.
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
			pos, inIdiom = "", false
		case cur == nil:
			continue
		case strings.HasPrefix(line, "*"):
			head, _, _ := strings.Cut(line[1:], "+")
			pos, inIdiom = hndPOS(head), false
		case strings.HasPrefix(line, "!"):
			inIdiom = true
		case strings.HasPrefix(line, "- "):
			text := strings.TrimSpace(line[2:])
			if !inIdiom && text != "" {
				cur.Meanings = append(cur.Meanings, meaning{POS: pos, Text: text, Source: sourceHND})
			}
		}
	}
	flush()
	return out, sc.Err()
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
