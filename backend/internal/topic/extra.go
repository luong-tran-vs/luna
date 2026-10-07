package topic

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// extraJSON lists the topics added on 2026-10-07 (greetings, time and dates, the house, …): each
// with a description and English words by level, written for the app.
//
//go:embed seed/extra-topics.json
var extraJSON []byte

// ExtraTopicsID names the one-time creation of the extra topics in storage markers.
const ExtraTopicsID = "f14-extra-topics-2026-10-07"

type extraFile struct {
	Topics []struct {
		Name        string              `json:"name"`
		Description string              `json:"description"`
		Words       map[string][]string `json:"words"`
	} `json:"topics"`
}

// LoadExtraTopics reads the embedded extra topics as topics to create: words of each level in
// list order, A1 → C2, cleaned by the CleanWords rules (invalid ones and repeats dropped, the
// first and lowest level kept), marked seeded so the startup seed leaves them alone.
func LoadExtraTopics(now time.Time) ([]Topic, error) {
	var f extraFile
	if err := json.Unmarshal(extraJSON, &f); err != nil {
		return nil, fmt.Errorf("topic: read extra topics: %w", err)
	}
	out := make([]Topic, 0, len(f.Topics))
	for _, et := range f.Topics {
		t := Topic{
			Name: et.Name, Description: et.Description, Roadmaps: map[string][]string{}, Words: []Word{},
			WordsSeeded: true, CreatedAt: now, UpdatedAt: now,
		}
		seen := map[string]bool{}
		for _, level := range Levels {
			for _, raw := range et.Words[level] {
				w, msg := cleanWord(raw)
				if msg != "" || seen[strings.ToLower(w)] || len(t.Words) == MaxWords {
					continue
				}
				seen[strings.ToLower(w)] = true
				t.Words = append(t.Words, Word{Text: w, Level: level})
			}
		}
		out = append(out, t)
	}
	return out, nil
}

// PlanExtraTopics keeps the extra topics whose name no existing topic has (compared like the
// seed: case, spaces and tone-mark placement ignored).
func PlanExtraTopics(existing []string, extra []Topic) []Topic {
	taken := make(map[string]bool, len(existing))
	for _, name := range existing {
		taken[SeedKey(name)] = true
	}
	var out []Topic
	for _, t := range extra {
		if !taken[SeedKey(t.Name)] {
			taken[SeedKey(t.Name)] = true
			out = append(out, t)
		}
	}
	return out
}
