// Package curriculum holds the fixed course of a level (added 2026-10-22, A1 first,
// design/de-xuat-giao-trinh-a1.png): its topics in order, each with its grammar points and lesson
// slots, 3 main lessons then 2–3 extra ones. A slot is filled by the lesson of the topic's roadmap
// with the slot's title. The admin applies the order to the topics, generates the empty slots and
// sees checks on drafts; learners see the topic's number, grammar and which lessons are extra.
package curriculum

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/luongtran/luna/backend/internal/grammar"
)

// Kind tells a main lesson, needed to move on to the next topic, from an extra one.
type Kind string

const (
	KindMain  Kind = "main"
	KindExtra Kind = "extra"
)

// MainPerTopic is how many main lessons every topic has.
const MainPerTopic = 3

// Slot is one lesson of the course.
type Slot struct {
	Key   string `json:"key"`
	Kind  Kind   `json:"kind"`
	Title string `json:"title"`
	// Grammar lists the slot's syllabus points, the first being the one the lesson is built on.
	Grammar []string `json:"grammar"`
	// Idea is the story given to the AI (in English, it goes into the prompt).
	Idea string `json:"idea"`
}

// Topic is one topic of the course, named as in the topic catalogue.
type Topic struct {
	// Number is the topic's place in the course, from 1.
	Number int    `json:"-"`
	Name   string `json:"name"`
	// CanDoVi ends "Sau chủ đề này bạn có thể …".
	CanDoVi string   `json:"canDoVi"`
	Grammar []string `json:"grammar"`
	Slots   []Slot   `json:"slots"`
}

// Plan is the course of one level.
type Plan struct {
	Level  string  `json:"level"`
	Topics []Topic `json:"topics"`
}

//go:embed a1.json
var a1JSON []byte

var plans = mustLoad(a1JSON)

func mustLoad(data ...[]byte) []Plan {
	out := make([]Plan, 0, len(data))
	for _, d := range data {
		var p Plan
		if err := json.Unmarshal(d, &p); err != nil {
			panic(fmt.Sprintf("curriculum: %v", err))
		}
		for i := range p.Topics {
			p.Topics[i].Number = i + 1
		}
		if err := p.Validate(grammar.Default()); err != nil {
			panic(err)
		}
		out = append(out, p)
	}
	return out
}

// ByLevel returns the course of level; ok is false when the level has none.
func ByLevel(level string) (Plan, bool) {
	for _, p := range plans {
		if p.Level == level {
			return p, true
		}
	}
	return Plan{}, false
}

// FindSlot returns the slot of key with its plan and topic.
func FindSlot(key string) (Plan, Topic, Slot, bool) {
	for _, p := range plans {
		for _, t := range p.Topics {
			for _, s := range t.Slots {
				if s.Key == key {
					return p, t, s, true
				}
			}
		}
	}
	return Plan{}, Topic{}, Slot{}, false
}

// Validate checks that every topic has its main lessons first, unique keys and titles, and points
// of the plan's level.
func (p Plan) Validate(syl *grammar.Syllabus) error {
	keys := map[string]bool{}
	names := map[string]bool{}
	checkPoints := func(where string, ids []string) error {
		if len(ids) == 0 {
			return fmt.Errorf("curriculum %s: %s has no grammar point", p.Level, where)
		}
		for _, id := range ids {
			pt, ok := syl.Get(id)
			if !ok || pt.Level != p.Level {
				return fmt.Errorf("curriculum %s: %s: %q is not a %s point", p.Level, where, id, p.Level)
			}
		}
		return nil
	}
	for _, t := range p.Topics {
		if names[NameKey(t.Name)] {
			return fmt.Errorf("curriculum %s: topic %q twice", p.Level, t.Name)
		}
		names[NameKey(t.Name)] = true
		if err := checkPoints(t.Name, t.Grammar); err != nil {
			return err
		}
		titles := map[string]bool{}
		for i, s := range t.Slots {
			if keys[s.Key] || s.Key == "" {
				return fmt.Errorf("curriculum %s: slot key %q empty or twice", p.Level, s.Key)
			}
			keys[s.Key] = true
			if titles[TitleKey(s.Title)] || strings.TrimSpace(s.Title) == "" || strings.TrimSpace(s.Idea) == "" {
				return fmt.Errorf("curriculum %s: slot %s: empty or repeated title, or no idea", p.Level, s.Key)
			}
			titles[TitleKey(s.Title)] = true
			if want := KindMain; i >= MainPerTopic {
				want = KindExtra
				if s.Kind != want {
					return fmt.Errorf("curriculum %s: slot %s should be %s", p.Level, s.Key, want)
				}
			} else if s.Kind != want {
				return fmt.Errorf("curriculum %s: slot %s should be %s", p.Level, s.Key, want)
			}
			if err := checkPoints(s.Key, s.Grammar); err != nil {
				return err
			}
		}
	}
	return nil
}

// NameKey compares topic names ignoring case and spaces.
func NameKey(s string) string {
	return strings.ToLower(strings.Join(strings.Fields(s), " "))
}

// TitleKey compares lesson titles ignoring case, spaces and the kind of apostrophe.
func TitleKey(s string) string {
	return NameKey(strings.ReplaceAll(s, "’", "'"))
}

// PointTopic is the first topic of a level's course that teaches a grammar point.
type PointTopic struct {
	Number int
	Name   string
}

// PointTopics maps each grammar point a level's course teaches to its first topic, and lists the
// points in course order; ok is false when the level has no course.
func PointTopics(level string) (map[string]PointTopic, []string, bool) {
	p, ok := ByLevel(level)
	if !ok {
		return nil, nil, false
	}
	byPoint := map[string]PointTopic{}
	var order []string
	for _, t := range p.Topics {
		for _, id := range t.Grammar {
			if _, seen := byPoint[id]; !seen {
				byPoint[id] = PointTopic{Number: t.Number, Name: t.Name}
				order = append(order, id)
			}
		}
	}
	return byPoint, order, true
}
