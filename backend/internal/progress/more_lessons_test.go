package progress

import (
	"errors"
	"slices"
	"testing"
)

func TestVisibleLessons(t *testing.T) {
	t.Parallel()
	ids := []string{"l1", "l2", "l3", "l4", "l5", "l6", "l7", "l8", "l9"}
	for _, c := range []struct {
		name      string
		more      int
		completed map[string]bool
		want      int
		held      int
	}{
		{"first five", 0, nil, 5, 3},
		{"one opened", 1, nil, 6, 2},
		{"all opened", 3, nil, 8, 0},
		{"never more than three", 7, nil, 8, 0},
		{"a lesson completed further on stays", 0, map[string]bool{"l7": true}, 7, 1},
	} {
		got := VisibleLessons(ids, c.more, c.completed)
		if len(got) != c.want || HeldLessons(ids, len(got)) != c.held {
			t.Errorf("%s: visible %v, held %d", c.name, got, HeldLessons(ids, len(got)))
		}
	}
	if got := VisibleLessons(ids[:3], 0, nil); len(got) != 3 || HeldLessons(ids[:3], 3) != 0 {
		t.Errorf("short roadmap: %v", got)
	}
}

// eightLessons gives u1's env a topic "travel" (A1) of eight lessons t1..t8.
func eightLessons(e *studyEnv) []string {
	ids := []string{"t1", "t2", "t3", "t4", "t5", "t6", "t7", "t8"}
	e.roadmaps.set(TopicInfo{ID: "travel", Name: "Du lịch", Level: "A1", LessonIDs: ids})
	for _, id := range ids {
		e.lessons.set(id, 1, 3)
	}
	return ids
}

func TestStudyMoreOpensHeldBackLessonsOneAtATime(t *testing.T) {
	t.Parallel()
	e := newStudyEnv()
	ids := eightLessons(e)

	g := e.setGoal(t, "travel")
	if g.TotalLessons != ShownLessons {
		t.Fatalf("goal = %+v", g)
	}
	if _, _, err := e.svc.StudyMore(t.Context(), "u1", "travel", "A1"); !errors.Is(err, ErrNoMoreLessons) {
		t.Fatalf("before the shown lessons are done: %v", err)
	}
	for range ShownLessons {
		e.studyLesson(t)
	}
	mine, _ := e.svc.MyLessons(t.Context(), "u1")
	if mine.Current != nil || len(mine.Upcoming) != 0 {
		t.Fatalf("after five lessons = %+v", mine)
	}
	path, _ := e.svc.Path(t.Context(), "u1")
	r := roadmapOf(path, "travel")
	if len(r.Lessons) != ShownLessons || r.MoreLessons != 3 {
		t.Fatalf("path roadmap = %+v", r)
	}

	// Away on another topic, then back: "Học bài mới" opens lesson 6 and makes travel the goal.
	e.setGoal(t, "family")
	for i, want := range ids[ShownLessons:] {
		g, next, err := e.svc.StudyMore(t.Context(), "u1", "travel", "A1")
		if err != nil || next.ID != want || g.TopicID != "travel" || g.TotalLessons != ShownLessons+i+1 {
			t.Fatalf("more %d = %+v %+v %v", i, g, next, err)
		}
		if cur := e.current(t); cur != want {
			t.Fatalf("current = %s, want %s", cur, want)
		}
		// Only one at a time: the lesson opened must be done first.
		if _, _, err := e.svc.StudyMore(t.Context(), "u1", "travel", "A1"); !errors.Is(err, ErrNoMoreLessons) {
			t.Fatalf("again before %s is done: %v", want, err)
		}
		e.studyLesson(t)
	}
	if _, _, err := e.svc.StudyMore(t.Context(), "u1", "travel", "A1"); !errors.Is(err, ErrNoMoreLessons) {
		t.Fatalf("after the three: %v", err)
	}
	path, _ = e.svc.Path(t.Context(), "u1")
	r = roadmapOf(path, "travel")
	if len(r.Lessons) != 8 || r.MoreLessons != 0 || r.Completed != 8 {
		t.Fatalf("path at the end = %+v", r)
	}
	sum, err := e.svc.TopicSummary(t.Context(), "u1", "travel", "A1")
	if err != nil || len(sum.Lessons) != 8 || sum.Completed != 8 {
		t.Fatalf("summary = %+v %v", sum, err)
	}
}

func TestTopicSummaryLeavesHeldBackLessonsOut(t *testing.T) {
	t.Parallel()
	e := newStudyEnv()
	eightLessons(e)
	sum, err := e.svc.TopicSummary(t.Context(), "u1", "travel", "A1")
	if err != nil || len(sum.Lessons) != ShownLessons ||
		!slices.Equal([]string{sum.Lessons[0].ID, sum.Lessons[4].ID}, []string{"t1", "t5"}) {
		t.Fatalf("summary = %+v %v", sum, err)
	}
}

func roadmapOf(p PathView, topicID string) PathRoadmap {
	for _, m := range p.Milestones {
		for _, r := range m.Roadmaps {
			if r.TopicID == topicID {
				return r
			}
		}
	}
	return PathRoadmap{}
}
