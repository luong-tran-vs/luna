package ai

import (
	"fmt"
	"strconv"
	"strings"
)

// Numbers in words (added 2026-11-16): a word picture must not show text, so "seven" came out as
// seven apples; a number word is drawn as its numeral instead.

var numberUnits = map[string]int{
	"zero": 0, "one": 1, "two": 2, "three": 3, "four": 4, "five": 5, "six": 6, "seven": 7, "eight": 8,
	"nine": 9, "ten": 10, "eleven": 11, "twelve": 12, "thirteen": 13, "fourteen": 14, "fifteen": 15,
	"sixteen": 16, "seventeen": 17, "eighteen": 18, "nineteen": 19, "twenty": 20, "thirty": 30,
	"forty": 40, "fifty": 50, "sixty": 60, "seventy": 70, "eighty": 80, "ninety": 90,
}

var numberScales = map[string]int{"hundred": 100, "thousand": 1000, "million": 1000000, "billion": 1000000000}

// ordinalWords maps the ordinals that are not a cardinal plus "th" to their cardinal. "second" is
// left out: as a word to learn it is mostly the unit of time.
var ordinalWords = map[string]string{
	"first": "one", "third": "three", "fifth": "five", "eighth": "eight", "ninth": "nine",
	"twelfth": "twelve", "twentieth": "twenty", "thirtieth": "thirty", "fortieth": "forty",
	"fiftieth": "fifty", "sixtieth": "sixty", "seventieth": "seventy", "eightieth": "eighty", "ninetieth": "ninety",
}

// NumberWord reads a number written as a word or in digits: "seven", "Twenty-one", "one hundred",
// "a thousand", "two hundred and five", "7", or an ordinal ("first", "twenty-third", "100th"). It
// returns the numeral to draw ("7", "1,000", "23rd") and false for any other word.
func NumberWord(word string) (string, bool) {
	w := strings.ToLower(strings.Join(strings.Fields(strings.ReplaceAll(word, "-", " ")), " "))
	if w == "" {
		return "", false
	}
	if n, err := strconv.Atoi(strings.ReplaceAll(w, ",", "")); err == nil && n >= 0 {
		return group(n), true
	}
	for _, suf := range []string{"st", "nd", "rd", "th"} {
		if n, err := strconv.Atoi(strings.TrimSuffix(w, suf)); err == nil && strings.HasSuffix(w, suf) && n >= 0 {
			return group(n) + ordinalSuffix(n), true
		}
	}
	parts := strings.Fields(w)
	last := parts[len(parts)-1]
	ordinal := false
	if c, ok := ordinalWords[last]; ok {
		parts[len(parts)-1], ordinal = c, true
	} else if last == "second" && len(parts) > 1 { // "twenty-second", never the unit of time
		parts[len(parts)-1], ordinal = "two", true
	} else if c, ok := strings.CutSuffix(last, "th"); ok && (numberUnits[c] > 0 || numberScales[c] > 0) {
		parts[len(parts)-1], ordinal = c, true
	}
	n, ok := cardinal(parts)
	if !ok {
		return "", false
	}
	if ordinal {
		if n == 0 {
			return "", false
		}
		return group(n) + ordinalSuffix(n), true
	}
	return group(n), true
}

// cardinal adds up number words; "a" or "one" may lead a scale ("a hundred") and "and" may join.
func cardinal(parts []string) (int, bool) {
	total, current, seen := 0, 0, false
	for i, p := range parts {
		switch {
		case p == "and" && seen && i < len(parts)-1:
			continue
		case p == "a" && i == 0 && len(parts) > 1 && numberScales[parts[1]] > 0:
			current = 1
		case numberScales[p] > 0:
			if current == 0 {
				current = 1
			}
			if numberScales[p] == 100 {
				current *= 100
			} else {
				total += current * numberScales[p]
				current = 0
			}
			seen = true
		default:
			v, ok := numberUnits[p]
			if !ok {
				return 0, false
			}
			current += v
			seen = true
		}
	}
	return total + current, seen
}

// group writes n with thousands separators: 1000 → "1,000".
func group(n int) string {
	s := strconv.Itoa(n)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}

func ordinalSuffix(n int) string {
	if n%100 >= 11 && n%100 <= 13 {
		return "th"
	}
	switch n % 10 {
	case 1:
		return "st"
	case 2:
		return "nd"
	case 3:
		return "rd"
	}
	return "th"
}

// numberPrompt describes the picture of a number word: its numeral as a real object, large and clear, which
// is the one text the picture shows.
func numberPrompt(word, numeral, style string) string {
	var b strings.Builder
	fmt.Fprintf(&b, photoOpening+"that helps a Vietnamese learner of English remember the English number word %q.\n", word)
	fmt.Fprintf(&b, "Show the numeral %q as a real physical object, large and centred, filling about two thirds of the frame: "+
		"thick, solid characters (painted wood or matte metal, like a large house number) standing on a plain surface, "+
		"easy to read at a glance, with exactly these characters and nothing else written. ", numeral)
	b.WriteString("Do not replace the numeral with a group of objects to count.\n")
	b.WriteString(photoDirection(style))
	b.WriteString("Composition: landscape 4:3. A plain, softly lit, light neutral studio backdrop with plenty of empty space, " +
		"a soft natural shadow under the numeral.\n")
	b.WriteString(photoAvoid)
	b.WriteString("Do not show any letters or words, no other numbers, no border, frame or watermark.")
	return b.String()
}
