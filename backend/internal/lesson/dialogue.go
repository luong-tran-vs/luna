package lesson

import (
	"regexp"
	"strings"
)

// TextTurn is one line of a dialogue lesson: who speaks and what they say (without the name).
type TextTurn struct {
	Speaker string
	Text    string
	// Sentences are the indexes of the lesson sentences the turn is made of.
	Sentences []int
}

// speakerLine is "Name: what they say", the form of a dialogue lesson (ai.KindDialogue); a name is
// one to three capitalized words.
var speakerLine = regexp.MustCompile(`^(\p{Lu}[\p{L}'.-]*(?: \p{Lu}[\p{L}'.-]*){0,2}):\s+(\S.*)$`)

const (
	minTextTurns    = 2
	minTextSpeakers = 2
	maxTextSpeakers = 4
)

// DialogueTurns reads a dialogue lesson as turns: every non-empty line of content must be
// "Name: text", with two to four speakers, and the lines must split into exactly the lesson's
// sentences, in order (so each turn knows its sentences). Anything else is not a dialogue: nil.
func DialogueTurns(content string, sentences []Sentence) []TextTurn {
	var turns []TextTurn
	speakers := map[string]bool{}
	next := 0
	for line := range strings.SplitSeq(strings.ReplaceAll(content, "\r\n", "\n"), "\n") {
		line = strings.Join(strings.Fields(line), " ")
		if line == "" {
			continue
		}
		m := speakerLine.FindStringSubmatch(line)
		if m == nil {
			return nil
		}
		t := TextTurn{Speaker: m[1], Text: m[2]}
		for _, s := range SplitSentences(line) {
			if next >= len(sentences) || sentences[next].Text != s {
				return nil
			}
			t.Sentences = append(t.Sentences, sentences[next].Index)
			next++
		}
		speakers[t.Speaker] = true
		turns = append(turns, t)
	}
	if next != len(sentences) || len(turns) < minTextTurns ||
		len(speakers) < minTextSpeakers || len(speakers) > maxTextSpeakers {
		return nil
	}
	return turns
}
