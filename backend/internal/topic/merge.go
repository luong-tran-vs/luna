package topic

import (
	"cmp"
	"slices"
	"strings"
	"time"
)

// MergeSource is a stored topic as the shared-topics migration (2026-10-07) finds it: either a
// legacy topic of one level (Level set, LessonIDs its roadmap) or a topic already shared by every
// level (Level "", Roadmaps), which a run interrupted half way may leave next to legacy ones.
type MergeSource struct {
	ID          string
	Name        string
	Description string
	Level       string
	LessonIDs   []string
	Roadmaps    map[string][]string
	Words       []Word
	WordsSeeded bool
	CreatedAt   time.Time
}

func (m MergeSource) legacy() bool { return m.Level != "" }

// rank orders sources inside a group: shared ones first (they are what an earlier run kept), then
// by level, creation and id.
func (m MergeSource) rank() int {
	if !m.legacy() {
		return -1
	}
	return levelRank(m.Level)
}

// MergeGroup is one topic after the merge: Keep (with the id of a source) replaces the sources
// whose ids are in Remove.
type MergeGroup struct {
	Keep Topic
	// Remove lists the ids of the other sources of the same name; their lessons, goals and
	// progress move to Keep.ID.
	Remove []string
	// Changed is false when Keep is already stored as it is (a shared topic alone with its name).
	Changed bool
}

// PlanMerge groups topics by name (NameKey) into one topic shared by every level. The kept topic
// is a shared one when there is one, else the one of the lowest level; its name is kept, and its
// description unless empty (then the first non-empty one). Each legacy roadmap becomes the roadmap
// of its level, appended after what is there without duplicates. Words are merged keeping the first
// spelling and the lowest level ("" counts as lowest), a legacy topic's words without a level taking
// the topic's level. Running it on its own result changes nothing. It is pure: storage applies it.
func PlanMerge(sources []MergeSource) []MergeGroup {
	groups := map[string][]MergeSource{}
	var order []string
	for _, s := range sources {
		k := NameKey(s.Name)
		if _, ok := groups[k]; !ok {
			order = append(order, k)
		}
		groups[k] = append(groups[k], s)
	}
	out := make([]MergeGroup, 0, len(order))
	for _, k := range order {
		out = append(out, mergeGroup(groups[k]))
	}
	return out
}

func mergeGroup(src []MergeSource) MergeGroup {
	src = slices.Clone(src)
	slices.SortStableFunc(src, func(a, b MergeSource) int {
		return cmp.Or(cmp.Compare(a.rank(), b.rank()), a.CreatedAt.Compare(b.CreatedAt), strings.Compare(a.ID, b.ID))
	})
	first := src[0]
	g := MergeGroup{
		Keep: Topic{
			ID: first.ID, Name: first.Name, Description: first.Description, Roadmaps: map[string][]string{},
			CreatedAt: first.CreatedAt,
		},
		Changed: len(src) > 1 || first.legacy(),
	}
	var words []Word
	wordAt := map[string]int{}
	for _, s := range src {
		if s.ID != first.ID {
			g.Remove = append(g.Remove, s.ID)
		}
		if g.Keep.Description == "" {
			g.Keep.Description = s.Description
		}
		if s.CreatedAt.Before(g.Keep.CreatedAt) {
			g.Keep.CreatedAt = s.CreatedAt
		}
		g.Keep.WordsSeeded = g.Keep.WordsSeeded || s.WordsSeeded

		roadmaps := s.Roadmaps
		if s.legacy() {
			roadmaps = map[string][]string{s.Level: s.LessonIDs}
		}
		for _, level := range Levels {
			for _, lid := range roadmaps[level] {
				if !slices.Contains(g.Keep.Roadmaps[level], lid) {
					g.Keep.Roadmaps[level] = append(g.Keep.Roadmaps[level], lid)
				}
			}
		}

		for _, w := range s.Words {
			if w.Level == "" && s.legacy() {
				w.Level = s.Level
			}
			key := strings.ToLower(w.Text)
			i, ok := wordAt[key]
			switch {
			case !ok:
				wordAt[key] = len(words)
				words = append(words, w)
			case wordRank(w.Level) < wordRank(words[i].Level):
				words[i].Level = w.Level
			}
		}
	}
	g.Keep.Words = words
	return g
}

// wordRank orders word levels with "" (every level) lowest.
func wordRank(level string) int {
	if level == "" {
		return -1
	}
	return levelRank(level)
}
