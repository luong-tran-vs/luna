package mysql

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/luongtran/luna/backend/internal/progress"
)

func TestGoalsActivate(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	repo := NewGoals(testDB(t))
	u, a, b := newID(), newID(), newID()
	t0 := time.Date(2026, 9, 1, 8, 0, 0, 0, time.UTC)

	if got, err := repo.List(ctx, u); err != nil || len(got) != 0 {
		t.Fatalf("empty List = %v, %v", got, err)
	}
	if _, err := repo.Activate(ctx, u, "not-an-id", "A1", "2026-09-01", t0); !errors.Is(err, progress.ErrTopicNotFound) {
		t.Fatalf("bad topic id err = %v", err)
	}

	ga, err := repo.Activate(ctx, u, a, "A1", "2026-09-01", t0)
	if err != nil {
		t.Fatal(err)
	}
	if ga.ID == "" || ga.UserID != u || ga.TopicID != a || ga.Level != "A1" || ga.Status != progress.GoalActive ||
		ga.EffectiveFrom != "2026-09-01" || !ga.StartedAt.Equal(t0) {
		t.Fatalf("goal = %+v", ga)
	}
	t1 := t0.Add(time.Hour)
	gb, err := repo.Activate(ctx, u, b, "B1", "2026-09-02", t1)
	if err != nil {
		t.Fatal(err)
	}
	if gb.Status != progress.GoalActive || gb.ID == ga.ID {
		t.Fatalf("second goal = %+v", gb)
	}
	// Another user's goal is untouched.
	other, _ := repo.Activate(ctx, newID(), a, "A1", "2026-09-02", t1)

	list, err := repo.List(ctx, u)
	if err != nil || len(list) != 2 {
		t.Fatalf("List = %v, %v", list, err)
	}
	if list[0].TopicID != a || list[0].Status != progress.GoalPaused || list[1].TopicID != b || list[1].Status != progress.GoalActive {
		t.Fatalf("List = %+v", list)
	}

	// Re-activating the first goal pauses the second and keeps the first's id and start time.
	t2 := t1.Add(time.Hour)
	again, err := repo.Activate(ctx, u, a, "A2", "2026-09-03", t2)
	if err != nil {
		t.Fatal(err)
	}
	if again.ID != ga.ID || !again.StartedAt.Equal(t0) || again.Level != "A2" || again.EffectiveFrom != "2026-09-03" || again.Status != progress.GoalActive {
		t.Fatalf("re-activated = %+v", again)
	}
	list, _ = repo.List(ctx, u)
	if list[0].Status != progress.GoalActive || list[1].Status != progress.GoalPaused {
		t.Fatalf("List = %+v", list)
	}
	// Activating the active goal again changes nothing else.
	if _, err := repo.Activate(ctx, u, a, "A2", "2026-09-03", t2); err != nil {
		t.Fatal(err)
	}
	if o, _ := repo.List(ctx, other.UserID); len(o) != 1 || o[0].Status != progress.GoalActive {
		t.Fatalf("other user's goals = %+v", o)
	}
}

func TestLessonProgressLifecycle(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	repo := NewLessonProgress(testDB(t))
	u, l1, l2, l3, topic := newID(), newID(), newID(), newID(), newID()
	started := time.Date(2026, 9, 30, 1, 2, 3, 0, time.UTC)

	if _, ok, err := repo.Get(ctx, u, l1); ok || err != nil {
		t.Fatalf("Get missing = %v, %v", ok, err)
	}
	// SetPosition on a lesson never started changes nothing.
	if err := repo.SetPosition(ctx, u, l1, progress.StepListen, 2); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := repo.Get(ctx, u, l1); ok {
		t.Fatal("SetPosition created a row")
	}

	p := progress.LessonProgress{
		UserID: u, LessonID: l1, TopicID: topic, DayKey: "2026-09-30",
		Done: map[progress.Step]bool{progress.StepRead: true}, CurrentStep: progress.StepListen, SentenceIndex: 1, StartedAt: started,
	}
	if err := repo.Upsert(ctx, p); err != nil {
		t.Fatal(err)
	}
	got, ok, err := repo.Get(ctx, u, l1)
	if err != nil || !ok {
		t.Fatalf("Get = %v, %v", ok, err)
	}
	if got.TopicID != topic || got.DayKey != "2026-09-30" || !got.Done[progress.StepRead] || got.Done[progress.StepListen] ||
		got.CurrentStep != progress.StepListen || got.SentenceIndex != 1 || !got.StartedAt.Equal(started) || !got.CompletedAt.IsZero() {
		t.Fatalf("Get = %+v", got)
	}

	if err := repo.SetPosition(ctx, u, l1, progress.StepWrite, 5); err != nil {
		t.Fatal(err)
	}
	got, _, _ = repo.Get(ctx, u, l1)
	if got.CurrentStep != progress.StepWrite || got.SentenceIndex != 5 || !got.Done[progress.StepRead] {
		t.Fatalf("after SetPosition = %+v", got)
	}

	// Complete l1 early and l2 late; a lesson without topic reads back with an empty topic.
	done := map[progress.Step]bool{progress.StepRead: true, progress.StepListen: true, progress.StepWrite: true}
	c1 := started.Add(time.Hour)
	c2 := started.Add(2 * time.Hour)
	p.Done, p.CurrentStep, p.CompletedAt = done, progress.StepDone, c1
	if err := repo.Upsert(ctx, p); err != nil {
		t.Fatal(err)
	}
	if err := repo.Upsert(ctx, progress.LessonProgress{UserID: u, LessonID: l2, DayKey: "2026-09-30", Done: map[progress.Step]bool{progress.StepRead: true, progress.StepListen: true},
		CurrentStep: progress.StepWrite, StartedAt: started, CompletedAt: c2}); err != nil {
		t.Fatal(err)
	}
	if err := repo.Upsert(ctx, progress.LessonProgress{UserID: u, LessonID: l3, Done: map[progress.Step]bool{progress.StepRead: true}, CurrentStep: progress.StepListen, StartedAt: started}); err != nil {
		t.Fatal(err)
	}
	if err := repo.Upsert(ctx, progress.LessonProgress{UserID: newID(), LessonID: l1, Done: done, StartedAt: started, CompletedAt: c1}); err != nil {
		t.Fatal(err)
	}

	list, err := repo.Completed(ctx, u)
	if err != nil || len(list) != 2 || list[0].LessonID != l2 || list[1].LessonID != l1 || !list[0].CompletedAt.Equal(c2) || list[0].TopicID != "" {
		t.Fatalf("Completed = %+v, %v", list, err)
	}

	// An upsert with a zero CompletedAt keeps the stored completion, like Mongo's $set.
	p.CompletedAt = time.Time{}
	if err := repo.Upsert(ctx, p); err != nil {
		t.Fatal(err)
	}
	got, _, _ = repo.Get(ctx, u, l1)
	if !got.CompletedAt.Equal(c1) {
		t.Fatalf("CompletedAt = %v, want kept %v", got.CompletedAt, c1)
	}

	// Step times read back, and a missing one keeps the stored time.
	if got.DoneAt != nil {
		t.Fatalf("DoneAt without times = %v", got.DoneAt)
	}
	p.DoneAt = map[progress.Step]time.Time{progress.StepRead: started, progress.StepWrite: c1}
	if err := repo.Upsert(ctx, p); err != nil {
		t.Fatal(err)
	}
	p.DoneAt = map[progress.Step]time.Time{progress.StepListen: c2}
	if err := repo.Upsert(ctx, p); err != nil {
		t.Fatal(err)
	}
	got, _, _ = repo.Get(ctx, u, l1)
	if len(got.DoneAt) != 3 || !got.DoneAt[progress.StepRead].Equal(started) || !got.DoneAt[progress.StepListen].Equal(c2) ||
		!got.DoneAt[progress.StepWrite].Equal(c1) {
		t.Fatalf("DoneAt = %v", got.DoneAt)
	}
}

func TestLessonProgressStepCountsSince(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	repo := NewLessonProgress(testDB(t))
	u := newID()
	since := time.Date(2026, 9, 28, 0, 0, 0, 0, time.UTC)
	before, after := since.Add(-time.Hour), since.Add(time.Hour)
	all := map[progress.Step]bool{progress.StepRead: true, progress.StepListen: true, progress.StepWrite: true}
	for _, p := range []progress.LessonProgress{
		// Done before since.
		{UserID: u, LessonID: newID(), Done: all, StartedAt: before, CompletedAt: before,
			DoneAt: map[progress.Step]time.Time{progress.StepRead: before, progress.StepListen: before, progress.StepWrite: before}},
		// Read before since, the rest after.
		{UserID: u, LessonID: newID(), Done: all, StartedAt: before, CompletedAt: after,
			DoneAt: map[progress.Step]time.Time{progress.StepRead: before, progress.StepListen: after, progress.StepWrite: after}},
		// Read exactly at since.
		{UserID: u, LessonID: newID(), Done: map[progress.Step]bool{progress.StepRead: true}, StartedAt: since,
			DoneAt: map[progress.Step]time.Time{progress.StepRead: since}},
		// Saved before the step times existed: counts only without since.
		{UserID: u, LessonID: newID(), Done: map[progress.Step]bool{progress.StepRead: true}, StartedAt: after},
		{UserID: newID(), LessonID: newID(), Done: all, StartedAt: after, CompletedAt: after,
			DoneAt: map[progress.Step]time.Time{progress.StepRead: after, progress.StepListen: after, progress.StepWrite: after}},
	} {
		if err := repo.Upsert(ctx, p); err != nil {
			t.Fatal(err)
		}
	}
	got, err := repo.StepCounts(ctx, u, nil, nil)
	if err != nil || got != (progress.StepCounts{Read: 4, Listen: 2, Write: 2, Completed: 2}) {
		t.Fatalf("all = %+v, %v", got, err)
	}
	got, err = repo.StepCounts(ctx, u, nil, &since)
	if err != nil || got != (progress.StepCounts{Read: 1, Listen: 1, Write: 1, Completed: 1}) {
		t.Fatalf("since = %+v, %v", got, err)
	}
}

func TestLessonProgressStepCounts(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	repo := NewLessonProgress(testDB(t))
	u := newID()
	l := []string{newID(), newID(), newID()}
	started := time.Date(2026, 9, 30, 1, 0, 0, 0, time.UTC)

	if c, err := repo.StepCounts(ctx, u, nil, nil); err != nil || c != (progress.StepCounts{}) {
		t.Fatalf("empty StepCounts = %+v, %v", c, err)
	}
	put := func(user, lesson string, steps []progress.Step, completed bool) {
		t.Helper()
		p := progress.LessonProgress{UserID: user, LessonID: lesson, Done: map[progress.Step]bool{}, StartedAt: started}
		for _, s := range steps {
			p.Done[s] = true
		}
		if completed {
			p.CompletedAt = started.Add(time.Hour)
		}
		if err := repo.Upsert(ctx, p); err != nil {
			t.Fatal(err)
		}
	}
	put(u, l[0], progress.Steps, true)
	put(u, l[1], []progress.Step{progress.StepRead, progress.StepListen}, false)
	put(u, l[2], []progress.Step{progress.StepRead}, false)
	put(newID(), l[0], progress.Steps, true)

	all, err := repo.StepCounts(ctx, u, nil, nil)
	if err != nil || all != (progress.StepCounts{Read: 3, Listen: 2, Write: 1, Completed: 1}) {
		t.Fatalf("all = %+v, %v", all, err)
	}
	some, err := repo.StepCounts(ctx, u, []string{l[1], l[2], newID()}, nil)
	if err != nil || some != (progress.StepCounts{Read: 2, Listen: 1}) {
		t.Fatalf("some = %+v, %v", some, err)
	}
	none, err := repo.StepCounts(ctx, u, []string{}, nil)
	if err != nil || none != (progress.StepCounts{}) {
		t.Fatalf("empty list = %+v, %v", none, err)
	}
}

func TestStudyDays(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	repo := NewStudyDays(testDB(t))
	u := newID()

	if keys, err := repo.CompletedKeys(ctx, u); err != nil || len(keys) != 0 {
		t.Fatalf("empty keys = %v, %v", keys, err)
	}
	for _, k := range []string{"2026-09-30", "2026-09-29", "2026-09-30"} {
		if err := repo.MarkCompleted(ctx, u, k); err != nil {
			t.Fatalf("MarkCompleted %s: %v", k, err)
		}
	}
	if err := repo.MarkCompleted(ctx, newID(), "2026-01-01"); err != nil {
		t.Fatal(err)
	}
	keys, err := repo.CompletedKeys(ctx, u)
	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(keys)
	if !slices.Equal(keys, []string{"2026-09-29", "2026-09-30"}) {
		t.Fatalf("keys = %v", keys)
	}
}
