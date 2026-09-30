package lesson

import "regexp"

// wordPattern matches a word: letters, with apostrophes or hyphens allowed only between letters
// (don't, well-known). The frontend tokenizer uses the same rule.
var wordPattern = regexp.MustCompile(`\p{L}+(?:['’\-]\p{L}+)*`)

// Words returns the words of text in order, as the reader shows them.
func Words(text string) []string {
	return wordPattern.FindAllString(text, -1)
}
