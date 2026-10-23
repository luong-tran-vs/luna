package progress

import (
	"context"
	"maps"
	"testing"
)

// pathStates gives the state of every lesson of a milestone, by lesson id.
func pathStates(m Milestone) map[string]PathLessonState {
	out := map[string]PathLessonState{}
	for _, r := range m.Roadmaps {
		for _, l := range r.Lessons {
			out[l.ID] = l.State
		}
	}
	return out
}

func roadmapNames(m Milestone) []string {
	var out []string
	for _, r := range m.Roadmaps {
		out = append(out, r.TopicName)
	}
	return out
}

func TestPath(t *testing.T) {
	t.Parallel()
	e := newStudyEnv()

	// No goal: every roadmap shows, locked.
	v, err := e.svc.Path(t.Context(), "u1")
	if err != nil || v.TopicID != "" || len(v.Milestones) != 6 || len(v.Milestones[0].Roadmaps) != 2 {
		t.Fatalf("no goal = %+v, %v", v, err)
	}
	if got := pathStates(v.Milestones[0]); got["f1"] != PathLocked || got["s1"] != PathLocked {
		t.Fatalf("no goal states = %v", got)
	}

	e.setGoal(t, "family")
	e.studyLesson(t) // f1
	v, err = e.svc.Path(t.Context(), "u1")
	if err != nil {
		t.Fatal(err)
	}
	if v.TopicID != "family" || v.TopicName != "Gia đình" || v.Streak != 1 || len(v.Milestones) != 6 {
		t.Fatalf("path = %+v", v)
	}
	a1 := v.Milestones[0]
	if a1.Level != "A1" || !a1.Active || a1.Completed != 1 || a1.Total != 5 {
		t.Fatalf("A1 = %+v", a1)
	}
	// Every topic with lessons at A1, one roadmap after another; only the goal's is active.
	if names := roadmapNames(a1); len(names) != 2 || names[0] != "Gia đình" || names[1] != "Mua sắm" {
		t.Fatalf("A1 roadmaps = %v", names)
	}
	if !a1.Roadmaps[0].Active || a1.Roadmaps[1].Active || a1.Roadmaps[0].Completed != 1 ||
		a1.Roadmaps[0].Lessons[1].Title != "Family 2" || a1.Roadmaps[0].TopicDescription != "Các thành viên" {
		t.Fatalf("A1 roadmaps = %+v", a1.Roadmaps)
	}
	want := map[string]PathLessonState{
		"f1": PathDone, "f2": PathCurrent, "f3": PathLocked, "s1": PathLocked, "s2": PathLocked,
	}
	if got := pathStates(a1); !maps.Equal(got, want) {
		t.Fatalf("A1 states = %v", got)
	}
	if b1 := v.Milestones[2]; b1.Level != "B1" || b1.Active || len(b1.Roadmaps) != 1 || b1.Roadmaps[0].Lessons[0].State != PathLocked {
		t.Fatalf("B1 = %+v", b1)
	}
	if a2 := v.Milestones[1]; a2.Level != "A2" || len(a2.Roadmaps) != 0 || a2.Total != 0 {
		t.Fatalf("A2 = %+v", a2)
	}

	// A goal whose roadmap has no lesson yet still shows, after the others.
	e.setGoal(t, "empty")
	v, _ = e.svc.Path(t.Context(), "u1")
	a1 = v.Milestones[0]
	if names := roadmapNames(a1); len(names) != 3 || names[2] != "Trống" || !a1.Roadmaps[2].Active || a1.Roadmaps[0].Active {
		t.Fatalf("empty goal = %+v", a1.Roadmaps)
	}
	if got := pathStates(a1); got["f1"] != PathDone || got["f2"] != PathLocked {
		t.Fatalf("paused roadmap states = %v", got)
	}
}

func TestPathGuest(t *testing.T) {
	t.Parallel()
	e := newStudyEnv()
	e.members.setGuest("u1", true)
	e.setGoal(t, "family")
	e.studyLesson(t) // f1

	v, err := e.svc.Path(t.Context(), "u1")
	if err != nil {
		t.Fatal(err)
	}
	// The first lesson of another roadmap is open to guests once it is the goal.
	want := map[string]PathLessonState{
		"f1": PathDone, "f2": PathMembersOnly, "f3": PathMembersOnly, "s1": PathLocked, "s2": PathMembersOnly,
	}
	if got := pathStates(v.Milestones[0]); !maps.Equal(got, want) {
		t.Fatalf("guest states = %v", got)
	}
}

func TestPathCourse(t *testing.T) {
	t.Parallel()
	e := newStudyEnv()
	e.svc.d.Course = func(_ context.Context, level, _, name string) (CourseTopic, bool) {
		if level != "A1" || name != "Gia đình" {
			return CourseTopic{}, false
		}
		return CourseTopic{Number: 2, CanDoVi: "giới thiệu gia đình", GrammarVi: []string{"Have / has"},
			IsExtra: func(title string) bool { return title == "Family 3" }}, true
	}
	v, err := e.svc.Path(t.Context(), "u1")
	if err != nil {
		t.Fatal(err)
	}
	family, shopping := v.Milestones[0].Roadmaps[0], v.Milestones[0].Roadmaps[1]
	if c := family.Course; c == nil || c.Number != 2 || c.GrammarVi[0] != "Have / has" || shopping.Course != nil {
		t.Fatalf("courses = %+v, %+v", family.Course, shopping.Course)
	}
	if family.Lessons[0].Extra || !family.Lessons[2].Extra {
		t.Errorf("extra lessons = %+v", family.Lessons)
	}
}
