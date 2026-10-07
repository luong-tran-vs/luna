package topic

import (
	"slices"
	"strings"
	"time"
)

// LegacyLesson is a lesson as F2 stored it: a free-text topic name, or already a topic id when
// an earlier migration run was interrupted.
type LegacyLesson struct {
	ID        string
	Level     string
	TopicName string
	TopicID   string
	CreatedAt time.Time
}

// LegacyTopic is a topic as F14 stored it before topics were shared by every level (2026-10-07):
// one level, one roadmap.
type LegacyTopic struct {
	ID          string
	Name        string
	Level       string
	Description string
	LessonIDs   []string
	Words       []Word
	WordsSeeded bool
	CreatedAt   time.Time
}

// Key identifies a topic by level and name key.
type Key struct {
	Level   string
	NameKey string
}

// PlannedTopic is a topic the migration makes sure exists, with its roadmap.
type PlannedTopic struct {
	Key
	// ID is set when the topic already exists.
	ID    string
	Name  string
	Level string
	// LessonIDs is the part of the old global roadmap that belongs to this topic, in order.
	LessonIDs []string
}

// Plan is the result of PlanMigration.
type Plan struct {
	Topics []PlannedTopic
	// Assign maps every lesson id to its topic.
	Assign map[string]Key
}

// PlanMigration turns F2 data into topics (F14): one topic per (level, topic name) pair, names
// compared case- and space-insensitively and spelled like the earliest lesson, lessons without a
// topic in "Chung" of their level, and the old global roadmap split per topic keeping the
// relative order. Existing topics with the same key are reused, so running it again after an
// interrupted migration gives the same result. It is pure: storage applies the plan.
func PlanMigration(lessons []LegacyLesson, roadmap []string, existing []LegacyTopic) Plan {
	byID := map[string]LegacyTopic{}
	byKey := map[Key]LegacyTopic{}
	for _, t := range existing {
		byID[t.ID] = t
		byKey[Key{t.Level, NameKey(t.Name)}] = t
	}

	sorted := slices.Clone(lessons)
	slices.SortStableFunc(sorted, func(a, b LegacyLesson) int { return a.CreatedAt.Compare(b.CreatedAt) })

	p := Plan{Assign: map[string]Key{}}
	index := map[Key]int{}
	for _, l := range sorted {
		name := strings.Join(strings.Fields(l.TopicName), " ")
		level := l.Level
		if t, ok := byID[l.TopicID]; ok {
			name, level = t.Name, t.Level
		}
		if name == "" {
			name = DefaultName
		}
		key := Key{level, NameKey(name)}
		if _, ok := index[key]; !ok {
			pt := PlannedTopic{Key: key, Name: name, Level: level, LessonIDs: []string{}}
			if t, ok := byKey[key]; ok {
				pt.ID, pt.Name = t.ID, t.Name
			}
			index[key] = len(p.Topics)
			p.Topics = append(p.Topics, pt)
		}
		p.Assign[l.ID] = key
	}

	seen := map[string]bool{}
	for _, lid := range roadmap {
		key, ok := p.Assign[lid]
		if !ok || seen[lid] { // deleted lesson or duplicate
			continue
		}
		seen[lid] = true
		pt := &p.Topics[index[key]]
		pt.LessonIDs = append(pt.LessonIDs, lid)
	}
	return p
}
