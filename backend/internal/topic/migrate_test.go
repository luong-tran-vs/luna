package topic

import (
	"slices"
	"testing"
	"time"
)

var base = time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)

func legacy(id, level, name string, minute int) LegacyLesson {
	return LegacyLesson{ID: id, Level: level, TopicName: name, CreatedAt: base.Add(time.Duration(minute) * time.Minute)}
}

// planned finds a planned topic by level and name key.
func planned(t *testing.T, p Plan, level, name string) PlannedTopic {
	t.Helper()
	for _, pt := range p.Topics {
		if pt.Level == level && NameKey(pt.Name) == NameKey(name) {
			return pt
		}
	}
	t.Fatalf("no topic %s %s in %+v", level, name, p.Topics)
	return PlannedTopic{}
}

func TestPlanMigration(t *testing.T) {
	t.Parallel()
	lessons := []LegacyLesson{
		legacy("z", "A1", "gia  đình ", 3), // created later: its spelling loses
		legacy("x", "A1", "Gia đình", 1),
		legacy("y", "B1", "Công việc", 2),
		legacy("r", "B1", "", 0),
		legacy("w", "A2", "  ", 4),
	}
	roadmap := []string{"x", "r", "deleted", "y", "z"}

	p := PlanMigration(lessons, roadmap, nil)
	if len(p.Topics) != 4 {
		t.Fatalf("topics = %+v", p.Topics)
	}
	family := planned(t, p, "A1", "Gia đình")
	if family.Name != "Gia đình" || family.ID != "" || !slices.Equal(family.LessonIDs, []string{"x", "z"}) {
		t.Fatalf("family = %+v", family)
	}
	if work := planned(t, p, "B1", "Công việc"); !slices.Equal(work.LessonIDs, []string{"y"}) {
		t.Fatalf("work = %+v", work)
	}
	if general := planned(t, p, "B1", DefaultName); !slices.Equal(general.LessonIDs, []string{"r"}) {
		t.Fatalf("B1 general = %+v", general)
	}
	// A lesson outside the old roadmap gets a topic but joins no roadmap.
	if general := planned(t, p, "A2", DefaultName); len(general.LessonIDs) != 0 || general.LessonIDs == nil {
		t.Fatalf("A2 general = %+v", general)
	}

	// Every lesson is assigned exactly once, to its level.
	if len(p.Assign) != len(lessons) {
		t.Fatalf("assign = %v", p.Assign)
	}
	for _, l := range lessons {
		key, ok := p.Assign[l.ID]
		if !ok || key.Level != l.Level {
			t.Fatalf("lesson %s assigned to %+v", l.ID, key)
		}
	}
	if p.Assign["x"] != p.Assign["z"] || p.Assign["x"] == p.Assign["y"] {
		t.Fatalf("assign = %v", p.Assign)
	}
}

func TestPlanMigrationReusesExistingTopics(t *testing.T) {
	t.Parallel()
	existing := []LegacyTopic{
		{ID: "t1", Name: "Gia Đình", Level: "A1", LessonIDs: []string{"x"}},
		{ID: "t2", Name: "Du lịch", Level: "A2", LessonIDs: []string{}},
	}
	lessons := []LegacyLesson{
		{ID: "x", Level: "A1", TopicID: "t1", CreatedAt: base}, // migrated in an interrupted run
		legacy("z", "A1", "gia đình", 1),
		{ID: "q", Level: "A2", TopicID: "gone", TopicName: "Du lịch", CreatedAt: base}, // unknown id: falls back to the name
	}
	p := PlanMigration(lessons, []string{"z", "x"}, existing)

	if len(p.Topics) != 2 {
		t.Fatalf("topics = %+v", p.Topics)
	}
	family := planned(t, p, "A1", "gia đình")
	if family.ID != "t1" || family.Name != "Gia Đình" || !slices.Equal(family.LessonIDs, []string{"z", "x"}) {
		t.Fatalf("family = %+v", family)
	}
	if travel := planned(t, p, "A2", "Du lịch"); travel.ID != "t2" {
		t.Fatalf("travel = %+v", travel)
	}
	if p.Assign["x"] != p.Assign["z"] || p.Assign["q"].NameKey != "du lịch" {
		t.Fatalf("assign = %v", p.Assign)
	}
}

func TestPlanMigrationIsStable(t *testing.T) {
	t.Parallel()
	lessons := []LegacyLesson{legacy("a", "A1", "X", 0), legacy("b", "A1", "Y", 1), legacy("c", "C2", "", 2)}
	first := PlanMigration(lessons, []string{"c", "b", "a"}, nil)
	second := PlanMigration(lessons, []string{"c", "b", "a"}, nil)
	if len(first.Topics) != len(second.Topics) {
		t.Fatal("different topic counts")
	}
	for i := range first.Topics {
		if first.Topics[i].Name != second.Topics[i].Name || !slices.Equal(first.Topics[i].LessonIDs, second.Topics[i].LessonIDs) {
			t.Fatalf("run 1 %+v, run 2 %+v", first.Topics[i], second.Topics[i])
		}
	}
	if empty := PlanMigration(nil, nil, nil); len(empty.Topics) != 0 || len(empty.Assign) != 0 {
		t.Fatalf("empty = %+v", empty)
	}
}
