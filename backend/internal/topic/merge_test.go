package topic

import (
	"slices"
	"testing"
	"time"
)

func TestPlanMergeJoinsTopicsOfTheSameName(t *testing.T) {
	t.Parallel()
	at := func(h int) time.Time { return time.Date(2026, 9, 30, h, 0, 0, 0, time.UTC) }
	sources := []MergeSource{
		{ID: "b", Name: "gia đình", Level: "A2", LessonIDs: []string{"z"}, CreatedAt: at(1),
			Words: []Word{{Text: "cousin"}, {Text: "Family"}}, WordsSeeded: true},
		{ID: "a", Name: "Gia đình", Level: "A1", Description: "", LessonIDs: []string{"x", "y"}, CreatedAt: at(2),
			Words: []Word{{Text: "family"}, {Text: "mother"}}},
		{ID: "c", Name: "Gia  Đình", Level: "B1", Description: "Người thân", CreatedAt: at(0)},
		{ID: "w", Name: "Công việc", Level: "B1", LessonIDs: []string{"w1"}, CreatedAt: at(3)},
	}
	groups := PlanMerge(sources)
	if len(groups) != 2 {
		t.Fatalf("groups = %+v", groups)
	}

	// The A1 topic is kept; the others of the same name are merged into it.
	g := groups[0]
	if g.Keep.ID != "a" || g.Keep.Name != "Gia đình" || !slices.Equal(g.Remove, []string{"b", "c"}) || !g.Changed {
		t.Fatalf("family = %+v", g)
	}
	if g.Keep.Description != "Người thân" || !g.Keep.CreatedAt.Equal(at(0)) || !g.Keep.WordsSeeded {
		t.Fatalf("family fields = %+v", g.Keep)
	}
	if !slices.Equal(g.Keep.Roadmap("A1"), []string{"x", "y"}) || !slices.Equal(g.Keep.Roadmap("A2"), []string{"z"}) ||
		len(g.Keep.Roadmap("B1")) != 0 {
		t.Fatalf("roadmaps = %v", g.Keep.Roadmaps)
	}
	// Words take their old topic's level; a word in two topics keeps the first spelling and the lower level.
	want := []Word{{Text: "family", Level: "A1"}, {Text: "mother", Level: "A1"}, {Text: "cousin", Level: "A2"}}
	if !slices.Equal(g.Keep.Words, want) {
		t.Fatalf("words = %+v", g.Keep.Words)
	}

	// A topic alone with its name is turned into a shared one.
	if w := groups[1]; w.Keep.ID != "w" || len(w.Remove) != 0 || !w.Changed || !slices.Equal(w.Keep.Roadmap("B1"), []string{"w1"}) {
		t.Fatalf("work = %+v", w)
	}
}

func TestPlanMergeFinishesAnInterruptedRun(t *testing.T) {
	t.Parallel()
	// The kept topic was written, but the A2 one was not deleted yet.
	sources := []MergeSource{
		{ID: "b", Name: "Gia đình", Level: "A2", LessonIDs: []string{"z", "q"}, Words: []Word{{Text: "aunt"}}},
		{ID: "a", Name: "Gia đình", Roadmaps: map[string][]string{"A1": {"x"}, "A2": {"z"}},
			Words: []Word{{Text: "Aunt", Level: "A2"}, {Text: "baby"}}},
	}
	g := PlanMerge(sources)[0]
	if g.Keep.ID != "a" || !slices.Equal(g.Remove, []string{"b"}) {
		t.Fatalf("group = %+v", g)
	}
	if !slices.Equal(g.Keep.Roadmap("A2"), []string{"z", "q"}) || !slices.Equal(g.Keep.Roadmap("A1"), []string{"x"}) {
		t.Fatalf("roadmaps = %v", g.Keep.Roadmaps)
	}
	if !slices.Equal(g.Keep.Words, []Word{{Text: "Aunt", Level: "A2"}, {Text: "baby"}}) {
		t.Fatalf("words = %+v", g.Keep.Words)
	}

	// Planning the merged result again changes nothing.
	again := PlanMerge([]MergeSource{{ID: g.Keep.ID, Name: g.Keep.Name, Roadmaps: g.Keep.Roadmaps, Words: g.Keep.Words}})
	if len(again) != 1 || again[0].Changed || len(again[0].Remove) != 0 {
		t.Fatalf("again = %+v", again)
	}
}

func TestWordFitsLevel(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		word, level string
		want        bool
	}{{"", "A1", true}, {"A1", "A1", true}, {"A1", "B2", true}, {"B1", "A2", false}} {
		if got := (Word{Text: "x", Level: tc.word}).FitsLevel(tc.level); got != tc.want {
			t.Errorf("word %q at %s = %v", tc.word, tc.level, got)
		}
	}
}
