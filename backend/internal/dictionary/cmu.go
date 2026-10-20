package dictionary

import (
	"bufio"
	"bytes"
	"compress/gzip"
	_ "embed"
	"strings"
	"sync"
)

// cmuData is the CMU Pronouncing Dictionary (cmusphinx/cmudict at commit 74790861, BSD licence in
// data/LICENSE-cmudict.txt): General American transcriptions in ARPAbet, one word per line,
// variants as "word(2)".
//
//go:embed data/cmudict.dict.gz
var cmuData []byte

var (
	cmuOnce sync.Once
	// cmuWords holds each word's variants in file order.
	cmuWords map[string][][]string
)

// preferredVariant names the variant to use for a word whose first one is not its base form:
// "read" is first the past /red/.
var preferredVariant = map[string]int{"read": 1}

func loadCMU() {
	cmuWords = make(map[string][][]string, 130000)
	zr, err := gzip.NewReader(bytes.NewReader(cmuData))
	if err != nil {
		return // the embedded file is checked by the tests; without it every word falls back
	}
	sc := bufio.NewScanner(zr)
	for sc.Scan() {
		line := sc.Text()
		if i := strings.IndexByte(line, '#'); i >= 0 {
			line = line[:i]
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		word := fields[0]
		if i := strings.IndexByte(word, '('); i > 0 {
			word = word[:i]
		}
		cmuWords[word] = append(cmuWords[word], fields[1:])
	}
}

// USIPA is the General American transcription of one word, as learners' dictionaries write it
// ("sound" → "/saʊnd/", "water" → "/ˈwɔːtər/"), "" when the CMU dictionary does not have it. A word
// of one syllable has no stress mark.
func USIPA(word string) string {
	cmuOnce.Do(loadCMU)
	variants := cmuWords[strings.ToLower(strings.TrimSpace(word))]
	if len(variants) == 0 {
		return ""
	}
	v := variants[0]
	if i, ok := preferredVariant[strings.ToLower(word)]; ok && i < len(variants) {
		v = variants[i]
	}
	return "/" + arpabetToIPA(v) + "/"
}

// arpabetVowels maps the ARPAbet vowels by stress (0 unstressed, 1 primary, 2 secondary).
var arpabetVowels = map[string][3]string{
	"AA": {"ɑː", "ɑː", "ɑː"},
	"AE": {"æ", "æ", "æ"},
	"AH": {"ə", "ʌ", "ʌ"},
	"AO": {"ɔː", "ɔː", "ɔː"},
	"AW": {"aʊ", "aʊ", "aʊ"},
	"AY": {"aɪ", "aɪ", "aɪ"},
	"EH": {"e", "e", "e"},
	"ER": {"ər", "ɜːr", "ɜːr"},
	"EY": {"eɪ", "eɪ", "eɪ"},
	"IH": {"ɪ", "ɪ", "ɪ"},
	"IY": {"i", "iː", "iː"},
	"OW": {"oʊ", "oʊ", "oʊ"},
	"OY": {"ɔɪ", "ɔɪ", "ɔɪ"},
	"UH": {"ʊ", "ʊ", "ʊ"},
	"UW": {"u", "uː", "uː"},
}

var arpabetConsonants = map[string]string{
	"B": "b", "CH": "tʃ", "D": "d", "DH": "ð", "F": "f", "G": "ɡ", "HH": "h", "JH": "dʒ", "K": "k",
	"L": "l", "M": "m", "N": "n", "NG": "ŋ", "P": "p", "R": "r", "S": "s", "SH": "ʃ", "T": "t",
	"TH": "θ", "V": "v", "W": "w", "Y": "j", "Z": "z", "ZH": "ʒ",
}

// clusterOnsets are the consonant clusters that may begin an English syllable, so the stress
// mark goes before them ("ex-plain" → /ɪkˈspleɪn/). Any single consonant but NG may begin one.
var clusterOnsets = map[string]bool{
	"P R": true, "P L": true, "B R": true, "B L": true, "T R": true, "D R": true, "K R": true,
	"K L": true, "G R": true, "G L": true, "F R": true, "F L": true, "TH R": true, "SH R": true,
	"S P": true, "S T": true, "S K": true, "S M": true, "S N": true, "S L": true, "S W": true,
	"K W": true, "T W": true, "D W": true, "G W": true, "TH W": true,
	"P Y": true, "B Y": true, "K Y": true, "M Y": true, "F Y": true, "V Y": true, "HH Y": true,
	"S P R": true, "S P L": true, "S T R": true, "S K R": true, "S K W": true, "S K Y": true,
}

func isOnset(phones []string) bool {
	if len(phones) == 1 {
		return phones[0] != "NG"
	}
	return clusterOnsets[strings.Join(phones, " ")]
}

// vowelOf splits an ARPAbet vowel into its base and stress ("AW1" → "AW", 1); ok is false for a
// consonant.
func vowelOf(p string) (base string, stress int, ok bool) {
	if n := len(p); n > 1 && p[n-1] >= '0' && p[n-1] <= '2' {
		return p[:n-1], int(p[n-1] - '0'), true
	}
	return "", 0, false
}

// arpabetToIPA writes the phones in IPA, with a stress mark before the onset of each stressed
// syllable when the word has more than one. Of several primary stresses ("TV"), the last one
// stays primary and the others become secondary (/ˌtiːˈviː/).
func arpabetToIPA(phones []string) string {
	syllables, lastPrimary := 0, -1
	for i, p := range phones {
		if _, stress, ok := vowelOf(p); ok {
			syllables++
			if stress == 1 {
				lastPrimary = i
			}
		}
	}
	marks := make([]string, len(phones))
	if syllables > 1 {
		for i, p := range phones {
			_, stress, ok := vowelOf(p)
			if !ok || stress == 0 {
				continue
			}
			start := i
			for k := i - 1; k >= 0; k-- {
				if _, _, vowel := vowelOf(phones[k]); vowel || !isOnset(phones[k:i]) {
					break
				}
				start = k
			}
			marks[start] = "ˌ"
			if stress == 1 && i == lastPrimary {
				marks[start] = "ˈ"
			}
		}
	}
	var b strings.Builder
	for i, p := range phones {
		b.WriteString(marks[i])
		if base, stress, ok := vowelOf(p); ok {
			b.WriteString(arpabetVowels[base][stress])
		} else {
			b.WriteString(arpabetConsonants[p])
		}
	}
	return b.String()
}
