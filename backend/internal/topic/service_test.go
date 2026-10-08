package topic

import (
	"errors"
	"slices"
	"strings"
	"testing"
)

func (e *testEnv) create(t *testing.T, name string) Summary {
	t.Helper()
	s, err := e.svc.Create(t.Context(), Input{Name: name})
	if err != nil {
		t.Fatalf("create %s: %v", name, err)
	}
	return s
}

func fieldErr(t *testing.T, err error, field string) string {
	t.Helper()
	var verr *ValidationError
	if !errors.As(err, &verr) || verr.Fields[field] == "" {
		t.Fatalf("err = %v, want field %s", err, field)
	}
	return verr.Fields[field]
}

// --- US1: catalogue ---

func TestCreateValidation(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		in    Input
		field string
	}{
		"empty name":       {Input{Name: "  "}, "name"},
		"long name":        {Input{Name: strings.Repeat("ă", 61)}, "name"},
		"long description": {Input{Name: "Gia đình", Description: strings.Repeat("a", 201)}, "description"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := newEnv().svc.Create(t.Context(), tc.in)
			fieldErr(t, err, tc.field)
		})
	}
}

func TestCreateTrimsAndRejectsDuplicates(t *testing.T) {
	t.Parallel()
	e := newEnv()
	s, err := e.svc.Create(t.Context(), Input{Name: "  Gia  đình ", Description: " Người thân "})
	if err != nil {
		t.Fatal(err)
	}
	if s.Name != "Gia đình" || s.Description != "Người thân" || !s.CreatedAt.Equal(testNow) || s.Roadmaps == nil || s.Levels == nil {
		t.Fatalf("topic = %+v", s)
	}

	_, err = e.svc.Create(t.Context(), Input{Name: "GIA ĐÌNH"})
	if msg := fieldErr(t, err, "name"); msg != "Chủ đề này đã có" {
		t.Fatalf("message = %q", msg)
	}
}

func TestListSortedWithCountsPerLevel(t *testing.T) {
	t.Parallel()
	e := newEnv()
	e.create(t, "Mua sắm")
	family := e.create(t, "Gia đình")
	e.create(t, "Công việc")
	for _, id := range []string{"l1", "l2", "l3"} {
		e.lessons.add(id, id, family.ID, "A1")
	}
	e.lessons.add("l4", "l4", family.ID, "B1")
	_ = e.repo.SetLessons(t.Context(), family.ID, "A1", []string{"l1", "l2", "l3"})

	list, err := e.svc.List(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	var order []string
	for _, s := range list {
		order = append(order, s.Name)
	}
	if strings.Join(order, ",") != "Công việc,Gia đình,Mua sắm" {
		t.Fatalf("order = %v", order)
	}
	f := list[1]
	want := []LevelSummary{
		{Level: "A1", LessonCount: 3, RoadmapCount: 3, Remaining: 3},
		{Level: "B1", LessonCount: 1, Warning: true},
	}
	if f.LessonCount != 4 || !slices.Equal(f.Levels, want) {
		t.Fatalf("family = %+v", f)
	}
	if len(list[0].Levels) != 0 || list[0].LessonCount != 0 {
		t.Fatalf("empty topic = %+v", list[0])
	}
}

func TestUpdate(t *testing.T) {
	t.Parallel()
	e := newEnv()
	family := e.create(t, "Gia đình")
	e.create(t, "Du lịch")

	got, err := e.svc.Update(t.Context(), family.ID, Input{Name: "Gia đình và bạn bè", Description: "x"})
	if err != nil || got.Name != "Gia đình và bạn bè" || got.Description != "x" {
		t.Fatalf("rename = %+v, %v", got, err)
	}
	_, err = e.svc.Update(t.Context(), family.ID, Input{Name: "du lịch"})
	fieldErr(t, err, "name")

	if _, err := e.svc.Update(t.Context(), "nope", Input{Name: "x"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown: %v", err)
	}
}

func TestDelete(t *testing.T) {
	t.Parallel()
	e := newEnv()
	family := e.create(t, "Gia đình")
	e.lessons.add("l1", "l1", family.ID, "A1")
	e.lessons.add("l2", "l2", family.ID, "A1")
	e.lessons.add("l3", "l3", family.ID, "B2")
	var inUse *InUseError
	if err := e.svc.Delete(t.Context(), family.ID); !errors.As(err, &inUse) || inUse.Count != 3 {
		t.Fatalf("delete in use: %v", err)
	}

	empty := e.create(t, "Mua sắm")
	if err := e.svc.Delete(t.Context(), empty.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := e.repo.Get(t.Context(), empty.ID); !errors.Is(err, ErrNotFound) {
		t.Fatal("topic still there")
	}
	if err := e.svc.Delete(t.Context(), empty.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("delete twice: %v", err)
	}
}

func TestPublicOnlyTopicsWithARoadmapAtTheLevel(t *testing.T) {
	t.Parallel()
	e := newEnv()
	family := e.create(t, "Gia đình")
	work := e.create(t, "Công việc")
	_ = e.repo.SetLessons(t.Context(), family.ID, "A1", []string{"l1", "l2"})
	_ = e.repo.SetLessons(t.Context(), family.ID, "B1", []string{"l3"})
	_ = e.repo.SetLessons(t.Context(), work.ID, "B1", []string{"l4"})
	for _, l := range []struct{ id, topic, level string }{
		{"l1", family.ID, "A1"}, {"l2", family.ID, "A1"}, {"l3", family.ID, "B1"}, {"l4", work.ID, "B1"},
	} {
		e.lessons.add(l.id, "Bài", l.topic, l.level)
	}

	list, err := e.svc.Public(t.Context(), "A1")
	if err != nil || len(list) != 1 || list[0] != (Public{ID: family.ID, Name: "Gia đình", Level: "A1", LessonCount: 2}) {
		t.Fatalf("public A1 = %+v, %v", list, err)
	}
	list, _ = e.svc.Public(t.Context(), "B1")
	if len(list) != 2 || list[0].Name != "Công việc" || list[1].LessonCount != 1 {
		t.Fatalf("public B1 = %+v", list)
	}
	if _, err := e.svc.Public(t.Context(), ""); err == nil {
		t.Fatal("missing level accepted")
	}
}

func TestLearnersSeeOnlyPublishedLessons(t *testing.T) {
	t.Parallel()
	e := newEnv()
	family := e.create(t, "Gia đình")
	draftOnly := e.create(t, "Công việc")
	_ = e.repo.SetLessons(t.Context(), family.ID, "A1", []string{"l1", "l2", "l3", "gone"})
	_ = e.repo.SetLessons(t.Context(), draftOnly.ID, "A1", []string{"l4"})
	e.lessons.add("l1", "Bài 1", family.ID, "A1")
	e.lessons.add("l2", "Bài 2", family.ID, "A1")
	e.lessons.add("l3", "Bài 3", family.ID, "A1")
	e.lessons.add("l4", "Bài 4", draftOnly.ID, "A1")
	e.lessons.setDraft("l2")
	e.lessons.setDraft("l4")

	list, err := e.svc.Public(t.Context(), "A1")
	if err != nil || len(list) != 1 || list[0].ID != family.ID || list[0].LessonCount != 2 {
		t.Fatalf("public = %+v, %v", list, err)
	}
	tp, _ := e.repo.Get(t.Context(), family.ID)
	ids, err := e.svc.LearnerRoadmap(t.Context(), tp, "A1")
	if err != nil || !slices.Equal(ids, []string{"l1", "l3"}) {
		t.Fatalf("learner roadmap = %v, %v", ids, err)
	}
	// The admin roadmap keeps the drafts.
	r, _ := e.svc.Roadmap(t.Context(), family.ID, "A1")
	if len(r.Lessons) != 3 || !r.Lessons[1].Draft {
		t.Fatalf("admin roadmap = %+v", r.Lessons)
	}
}

// --- US2: port used by lessons ---

func TestMoveLesson(t *testing.T) {
	t.Parallel()
	e := newEnv()
	a := e.create(t, "Gia đình")
	b := e.create(t, "Du lịch")
	_ = e.repo.SetLessons(t.Context(), a.ID, "A1", []string{"l1", "l2"})
	_ = e.repo.SetLessons(t.Context(), b.ID, "A1", []string{"l9"})

	if err := e.svc.MoveLesson(t.Context(), "l1", Place{a.ID, "A1"}, Place{b.ID, "A1"}); err != nil {
		t.Fatal(err)
	}
	ta, _ := e.repo.Get(t.Context(), a.ID)
	tb, _ := e.repo.Get(t.Context(), b.ID)
	if !slices.Equal(ta.Roadmap("A1"), []string{"l2"}) || !slices.Equal(tb.Roadmap("A1"), []string{"l9", "l1"}) {
		t.Fatalf("roadmaps = %v %v", ta.Roadmaps, tb.Roadmaps)
	}

	// Same topic, other level: the lesson moves to that level's roadmap.
	if err := e.svc.MoveLesson(t.Context(), "l2", Place{a.ID, "A1"}, Place{a.ID, "A2"}); err != nil {
		t.Fatal(err)
	}
	if ta, _ := e.repo.Get(t.Context(), a.ID); len(ta.Roadmap("A1")) != 0 || !slices.Equal(ta.Roadmap("A2"), []string{"l2"}) {
		t.Fatalf("level move = %v", ta.Roadmaps)
	}

	// A lesson that was not in the old roadmap does not join the new one.
	if err := e.svc.MoveLesson(t.Context(), "l7", Place{a.ID, "A1"}, Place{b.ID, "A1"}); err != nil {
		t.Fatal(err)
	}
	if tb, _ := e.repo.Get(t.Context(), b.ID); len(tb.Roadmap("A1")) != 2 {
		t.Fatalf("l7 added: %v", tb.Roadmaps)
	}
}

func TestAppendLesson(t *testing.T) {
	t.Parallel()
	e := newEnv()
	a := e.create(t, "Gia đình")
	_ = e.repo.SetLessons(t.Context(), a.ID, "A1", []string{"l1"})

	for _, id := range []string{"l2", "l2"} {
		if err := e.svc.AppendLesson(t.Context(), Place{a.ID, "A1"}, id); err != nil {
			t.Fatal(err)
		}
	}
	if ta, _ := e.repo.Get(t.Context(), a.ID); !slices.Equal(ta.Roadmap("A1"), []string{"l1", "l2"}) {
		t.Fatalf("roadmap = %v", ta.Roadmaps)
	}
	if err := e.svc.AppendLesson(t.Context(), Place{"nope", "A1"}, "l3"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown topic: %v", err)
	}
}

func TestPortLookups(t *testing.T) {
	t.Parallel()
	e := newEnv()
	a := e.create(t, "Gia đình")
	b := e.create(t, "Du lịch")
	_ = e.repo.SetLessons(t.Context(), a.ID, "A1", []string{"l1"})
	_ = e.repo.SetLessons(t.Context(), a.ID, "B1", []string{"l3"})
	_ = e.repo.SetLessons(t.Context(), b.ID, "A2", []string{"l2"})

	got, err := e.svc.Get(t.Context(), b.ID)
	if err != nil || got.Name != "Du lịch" || !slices.Equal(got.Roadmap("A2"), []string{"l2"}) {
		t.Fatalf("get = %+v, %v", got, err)
	}
	if _, err := e.svc.Get(t.Context(), "nope"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("get unknown: %v", err)
	}
	all, _ := e.svc.All(t.Context())
	if len(all) != 2 {
		t.Fatalf("all = %+v", all)
	}
	ids, _ := e.svc.RoadmapLessonIDs(t.Context())
	if len(ids) != 3 || !ids["l1"] || !ids["l2"] || !ids["l3"] {
		t.Fatalf("roadmap ids = %v", ids)
	}
}

// --- US3: roadmap per topic and level ---

func TestRoadmap(t *testing.T) {
	t.Parallel()
	e := newEnv()
	family := e.create(t, "Gia đình")
	work := e.create(t, "Công việc")
	for _, id := range []string{"a", "b", "c"} {
		e.lessons.add(id, "Lesson "+id, family.ID, "A1")
	}
	e.lessons.add("w", "Work lesson", work.ID, "A1")
	e.lessons.add("x", "Harder", family.ID, "A2")

	r, err := e.svc.Roadmap(t.Context(), family.ID, "A1")
	lv := r.Topic.Level("A1")
	if err != nil || len(r.Lessons) != 0 || lv.Remaining != 0 || !lv.Warning || r.Lessons == nil || r.Level != "A1" {
		t.Fatalf("empty = %+v, %v", r, err)
	}

	for n, ids := range [][]string{{"c"}, {"c", "a"}, {"c", "a", "b"}} {
		r, err = e.svc.SetRoadmap(t.Context(), family.ID, "A1", ids)
		if err != nil {
			t.Fatal(err)
		}
		lv := r.Topic.Level("A1")
		if lv.Remaining != n+1 || lv.Warning != (n+1 < MinRemaining) || r.Lessons[0].ID != "c" {
			t.Fatalf("%d lessons: %+v", n+1, r)
		}
	}
	if r.Lessons[2].ID != "b" || r.Lessons[2].Title != "Lesson b" || r.Topic.Level("A1").LessonCount != 3 {
		t.Fatalf("roadmap = %+v", r)
	}

	cases := map[string]struct {
		ids []string
		msg string
	}{
		"duplicate":     {[]string{"a", "a"}, "Lộ trình có bài bị trùng"},
		"unknown":       {[]string{"a", "zzz"}, "Lộ trình có bài không tồn tại"},
		"another topic": {[]string{"a", "w"}, "Chỉ thêm được bài của chủ đề này"},
		"another level": {[]string{"a", "x"}, "Chỉ thêm được bài A1 của chủ đề này"},
	}
	for name, tc := range cases {
		_, err := e.svc.SetRoadmap(t.Context(), family.ID, "A1", tc.ids)
		if msg := fieldErr(t, err, "lessonIds"); msg != tc.msg {
			t.Fatalf("%s: message = %q", name, msg)
		}
	}
	if _, err := e.svc.SetRoadmap(t.Context(), family.ID, "Z1", nil); err == nil {
		t.Fatal("bad level accepted")
	}

	// A lesson deleted behind the roadmap's back is skipped.
	delete(e.lessons.lessons, "a")
	if r, _ := e.svc.Roadmap(t.Context(), family.ID, "A1"); len(r.Lessons) != 2 {
		t.Fatalf("deleted lesson shown: %+v", r.Lessons)
	}
	if _, err := e.svc.Roadmap(t.Context(), "nope", "A1"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown topic: %v", err)
	}
}
