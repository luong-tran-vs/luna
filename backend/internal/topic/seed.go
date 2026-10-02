package topic

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
)

// seedJSON is the starting vocabulary of 42 topics (F18): English words only, taken as a
// reference from Langmaster's public "3000 common words by topic" list.
//
//go:embed seed/topic-words.json
var seedJSON []byte

// Seed is the starting vocabulary, by topic name (no level).
type Seed struct {
	Topics []SeedTopic `json:"topics"`
}

// SeedTopic is the word list of one topic name.
type SeedTopic struct {
	Name  string   `json:"name"`
	Words []string `json:"words"`
}

// SeedTarget is a topic the seed has not considered yet.
type SeedTarget struct {
	ID   string
	Name string
}

// SeedWrite is the word list to store on a topic, and whether its name matched the seed.
type SeedWrite struct {
	ID      string
	Words   []string
	Matched bool
}

// LoadSeed reads the embedded starting vocabulary.
func LoadSeed() (Seed, error) {
	var s Seed
	if err := json.Unmarshal(seedJSON, &s); err != nil {
		return Seed{}, fmt.Errorf("topic: read word seed: %w", err)
	}
	return s, nil
}

// toneOld maps the old tone-mark placement ("khoẻ", "thuý") to the new one ("khỏe", "thúy").
var toneOld = strings.NewReplacer(
	"oà", "òa", "oá", "óa", "oả", "ỏa", "oã", "õa", "oạ", "ọa",
	"oè", "òe", "oé", "óe", "oẻ", "ỏe", "oẽ", "õe", "oẹ", "ọe",
	"uỳ", "ùy", "uý", "úy", "uỷ", "ủy", "uỹ", "ũy", "uỵ", "ụy",
)

// SeedKey is the form used to match topic names with the seed: lowercase, single-spaced, and
// tone marks placed the new way, so "Sức khoẻ" matches "Sức khỏe".
func SeedKey(name string) string {
	return toneOld.Replace(NameKey(name))
}

// PlanSeed gives every target its seed words when its name matches a seed topic, or an empty
// list otherwise; either way the target is then seeded and never considered again. Words that
// fail CleanWords rules are skipped.
func PlanSeed(targets []SeedTarget, seed Seed) []SeedWrite {
	byKey := map[string][]string{}
	for _, st := range seed.Topics {
		byKey[SeedKey(st.Name)] = validSeedWords(st.Words)
	}
	out := make([]SeedWrite, len(targets))
	for i, t := range targets {
		words, ok := byKey[SeedKey(t.Name)]
		if !ok {
			words = []string{}
		}
		out[i] = SeedWrite{ID: t.ID, Words: words, Matched: ok}
	}
	return out
}

func validSeedWords(in []string) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, raw := range in {
		w, msg := cleanWord(raw)
		if msg != "" || seen[strings.ToLower(w)] || len(out) == MaxWords {
			continue
		}
		seen[strings.ToLower(w)] = true
		out = append(out, w)
	}
	return out
}
