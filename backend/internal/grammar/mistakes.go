package grammar

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

// L1Mistake is a mistake Vietnamese learners often make because of how Vietnamese works (no verb
// "be" before an adjective, no articles, no plural or tense endings, "rất thích" said word for
// word). The list is written by hand, not by the AI (added 2026-10-20), and each mistake belongs
// to the syllabus points it is about.
type L1Mistake struct {
	Points []string `json:"points"`
	Wrong  string   `json:"wrong"`
	Right  string   `json:"right"`
	NoteVi string   `json:"noteVi"`
}

//go:embed l1_mistakes.json
var embeddedMistakes []byte

// ParseMistakes reads and checks a list of mistakes against the syllabus: every mistake needs a
// wrong and a right sentence that differ, a note, and points that exist.
func ParseMistakes(data []byte, s *Syllabus) ([]L1Mistake, error) {
	var ms []L1Mistake
	if err := json.Unmarshal(data, &ms); err != nil {
		return nil, fmt.Errorf("grammar: parse mistakes: %w", err)
	}
	for i, m := range ms {
		switch {
		case strings.TrimSpace(m.Wrong) == "" || strings.TrimSpace(m.Right) == "" || strings.TrimSpace(m.NoteVi) == "":
			return nil, fmt.Errorf("grammar: mistake %d is missing a sentence or note", i)
		case m.Wrong == m.Right:
			return nil, fmt.Errorf("grammar: mistake %d has the same wrong and right sentence", i)
		case len(m.Points) == 0:
			return nil, fmt.Errorf("grammar: mistake %d has no point", i)
		}
		for _, id := range m.Points {
			if _, ok := s.Get(id); !ok {
				return nil, fmt.Errorf("grammar: mistake %d has unknown point %q", i, id)
			}
		}
	}
	return ms, nil
}

var (
	mistakesOnce sync.Once
	mistakes     []L1Mistake
	mistakesErr  error
)

// MistakesOf returns the hand-written mistakes of a point from the embedded list. A broken
// embedded file is a build mistake that the tests catch, so it panics like Default.
func MistakesOf(pointID string) []L1Mistake {
	mistakesOnce.Do(func() { mistakes, mistakesErr = ParseMistakes(embeddedMistakes, Default()) })
	if mistakesErr != nil {
		panic(mistakesErr)
	}
	var out []L1Mistake
	for _, m := range mistakes {
		for _, id := range m.Points {
			if id == pointID {
				out = append(out, m)
				break
			}
		}
	}
	return out
}
