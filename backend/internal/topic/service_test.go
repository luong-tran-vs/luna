package topic

import (
	"errors"
	"slices"
	"strings"
	"testing"
)

func (e *testEnv) create(t *testing.T, name, level string) Summary {
	t.Helper()
	s, err := e.svc.Create(t.Context(), Input{Name: name, Level: level})
	if err != nil {
		t.Fatalf("create %s %s: %v", level, name, err)
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
		"empty name":       {Input{Name: "  ", Level: "A1"}, "name"},
		"long name":        {Input{Name: strings.Repeat("ă", 61), Level: "A1"}, "name"},
		"bad level":        {Input{Name: "Gia đình", Level: "D1"}, "level"},
		"long description": {Input{Name: "Gia đình", Level: "A1", Description: strings.Repeat("a", 201)}, "description"},
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
	s, err := e.svc.Create(t.Context(), Input{Name: "  Gia  đình ", Level: "a1", Description: " Người thân "})
	if err != nil {
		t.Fatal(err)
	}
	if s.Name != "Gia đình" || s.Level != "A1" || s.Description != "Người thân" || !s.CreatedAt.Equal(testNow) || s.LessonIDs == nil {
		t.Fatalf("topic = %+v", s)
	}

	_, err = e.svc.Create(t.Context(), Input{Name: "GIA ĐÌNH", Level: "A1"})
	if msg := fieldErr(t, err, "name"); msg != "Chủ đề này đã có ở trình độ A1" {
		t.Fatalf("message = %q", msg)
	}
	if _, err := e.svc.Create(t.Context(), Input{Name: "Gia đình", Level: "A2"}); err != nil {
		t.Fatalf("same name in A2: %v", err)
	}
}

func TestListSortedWithCountsAndWarnings(t *testing.T) {
	t.Parallel()
	e := newEnv()
	work := e.create(t, "Công việc", "B1")
	shop := e.create(t, "Mua sắm", "A1")
	family := e.create(t, "Gia đình", "A1")
	for _, id := range []string{"l1", "l2", "l3"} {
		e.lessons.add(id, id, family.ID)
	}
	_ = e.repo.SetLessons(t.Context(), family.ID, []string{"l1", "l2", "l3"})

	list, err := e.svc.List(t.Context(), "")
	if err != nil {
		t.Fatal(err)
	}
	var order []string
	for _, s := range list {
		order = append(order, s.Level+" "+s.Name)
	}
	if strings.Join(order, ",") != "A1 Gia đình,A1 Mua sắm,B1 Công việc" {
		t.Fatalf("order = %v", order)
	}
	if list[0].LessonCount != 3 || list[0].Remaining != 3 || list[0].Warning {
		t.Fatalf("family = %+v", list[0])
	}
	if list[1].ID != shop.ID || list[1].Remaining != 0 || !list[1].Warning {
		t.Fatalf("shopping = %+v", list[1])
	}

	b1, _ := e.svc.List(t.Context(), "B1")
	if len(b1) != 1 || b1[0].ID != work.ID {
		t.Fatalf("B1 = %+v", b1)
	}
	if _, err := e.svc.List(t.Context(), "Z9"); err == nil {
		t.Fatal("bad level accepted")
	}
}

func TestUpdate(t *testing.T) {
	t.Parallel()
	e := newEnv()
	family := e.create(t, "Gia đình", "A1")
	e.create(t, "Gia đình", "A2")
	e.lessons.add("l1", "Lesson", family.ID)

	got, err := e.svc.Update(t.Context(), family.ID, Input{Name: "Gia đình và bạn bè", Level: "A1", Description: "x"})
	if err != nil || got.Name != "Gia đình và bạn bè" || got.Description != "x" || len(e.lessons.levelSets) != 0 {
		t.Fatalf("rename = %+v, %v, %v", got, err, e.lessons.levelSets)
	}

	// Moving to a level that already has the name is rejected.
	e.create(t, "Du lịch", "B1")
	_, err = e.svc.Update(t.Context(), family.ID, Input{Name: "du lịch", Level: "B1"})
	fieldErr(t, err, "name")

	// Changing the level carries the lessons along.
	got, err = e.svc.Update(t.Context(), family.ID, Input{Name: "Gia đình và bạn bè", Level: "A2"})
	if err != nil || got.Level != "A2" || !slices.Equal(e.lessons.levelSets, []string{family.ID + "=A2"}) {
		t.Fatalf("level change = %+v, %v, %v", got, err, e.lessons.levelSets)
	}
	if e.lessons.lessons["l1"].Level != "A2" {
		t.Fatal("lesson level not updated")
	}

	if _, err := e.svc.Update(t.Context(), "nope", Input{Name: "x", Level: "A1"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown: %v", err)
	}
}

func TestDelete(t *testing.T) {
	t.Parallel()
	e := newEnv()
	family := e.create(t, "Gia đình", "A1")
	for _, id := range []string{"l1", "l2", "l3"} {
		e.lessons.add(id, id, family.ID)
	}
	var inUse *InUseError
	if err := e.svc.Delete(t.Context(), family.ID); !errors.As(err, &inUse) || inUse.Count != 3 {
		t.Fatalf("delete in use: %v", err)
	}

	empty := e.create(t, "Mua sắm", "A1")
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

func TestPublic(t *testing.T) {
	t.Parallel()
	e := newEnv()
	family := e.create(t, "Gia đình", "A1")
	e.create(t, "Công việc", "B1")
	_ = e.repo.SetLessons(t.Context(), family.ID, []string{"l1", "l2"})

	list, err := e.svc.Public(t.Context(), "A1")
	if err != nil || len(list) != 1 || list[0] != (Public{ID: family.ID, Name: "Gia đình", Level: "A1", LessonCount: 2}) {
		t.Fatalf("public = %+v, %v", list, err)
	}
}

// --- US2: port used by lessons ---

func TestMoveLesson(t *testing.T) {
	t.Parallel()
	e := newEnv()
	a := e.create(t, "Gia đình", "A1")
	b := e.create(t, "Du lịch", "A2")
	_ = e.repo.SetLessons(t.Context(), a.ID, []string{"l1", "l2"})
	_ = e.repo.SetLessons(t.Context(), b.ID, []string{"l9"})

	if err := e.svc.MoveLesson(t.Context(), "l1", a.ID, b.ID); err != nil {
		t.Fatal(err)
	}
	ta, _ := e.repo.Get(t.Context(), a.ID)
	tb, _ := e.repo.Get(t.Context(), b.ID)
	if !slices.Equal(ta.LessonIDs, []string{"l2"}) || !slices.Equal(tb.LessonIDs, []string{"l9", "l1"}) {
		t.Fatalf("roadmaps = %v %v", ta.LessonIDs, tb.LessonIDs)
	}

	// A lesson that was not in the old roadmap does not join the new one.
	if err := e.svc.MoveLesson(t.Context(), "l7", a.ID, b.ID); err != nil {
		t.Fatal(err)
	}
	if tb, _ := e.repo.Get(t.Context(), b.ID); len(tb.LessonIDs) != 2 {
		t.Fatalf("l7 added: %v", tb.LessonIDs)
	}
}

func TestAppendLesson(t *testing.T) {
	t.Parallel()
	e := newEnv()
	a := e.create(t, "Gia đình", "A1")
	_ = e.repo.SetLessons(t.Context(), a.ID, []string{"l1"})

	for _, id := range []string{"l2", "l2"} {
		if err := e.svc.AppendLesson(t.Context(), a.ID, id); err != nil {
			t.Fatal(err)
		}
	}
	if ta, _ := e.repo.Get(t.Context(), a.ID); !slices.Equal(ta.LessonIDs, []string{"l1", "l2"}) {
		t.Fatalf("roadmap = %v", ta.LessonIDs)
	}
	if err := e.svc.AppendLesson(t.Context(), "nope", "l3"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown topic: %v", err)
	}
}

func TestPortLookups(t *testing.T) {
	t.Parallel()
	e := newEnv()
	a := e.create(t, "Gia đình", "A1")
	b := e.create(t, "Du lịch", "A2")
	_ = e.repo.SetLessons(t.Context(), a.ID, []string{"l1"})
	_ = e.repo.SetLessons(t.Context(), b.ID, []string{"l2"})

	got, err := e.svc.Get(t.Context(), b.ID)
	if err != nil || got.Name != "Du lịch" || got.Level != "A2" {
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
	if len(ids) != 2 || !ids["l1"] || !ids["l2"] {
		t.Fatalf("roadmap ids = %v", ids)
	}
}

// --- US3: roadmap per topic ---

func TestRoadmap(t *testing.T) {
	t.Parallel()
	e := newEnv()
	family := e.create(t, "Gia đình", "A1")
	work := e.create(t, "Công việc", "B1")
	for _, id := range []string{"a", "b", "c"} {
		e.lessons.add(id, "Lesson "+id, family.ID)
	}
	e.lessons.add("w", "Work lesson", work.ID)

	r, err := e.svc.Roadmap(t.Context(), family.ID)
	if err != nil || len(r.Lessons) != 0 || r.Topic.Remaining != 0 || !r.Topic.Warning || r.Lessons == nil {
		t.Fatalf("empty = %+v, %v", r, err)
	}

	for n, ids := range [][]string{{"c"}, {"c", "a"}, {"c", "a", "b"}} {
		r, err = e.svc.SetRoadmap(t.Context(), family.ID, ids)
		if err != nil {
			t.Fatal(err)
		}
		if r.Topic.Remaining != n+1 || r.Topic.Warning != (n+1 < MinRemaining) || r.Lessons[0].ID != "c" {
			t.Fatalf("%d lessons: %+v", n+1, r)
		}
	}
	if r.Lessons[2].ID != "b" || r.Lessons[2].Title != "Lesson b" || r.Topic.LessonCount != 3 {
		t.Fatalf("roadmap = %+v", r)
	}

	cases := map[string][]string{
		"duplicate":     {"a", "a"},
		"unknown":       {"a", "zzz"},
		"another topic": {"a", "w"},
	}
	for name, ids := range cases {
		_, err := e.svc.SetRoadmap(t.Context(), family.ID, ids)
		msg := fieldErr(t, err, "lessonIds")
		if name == "another topic" && msg != "Chỉ thêm được bài của chủ đề này" {
			t.Fatalf("message = %q", msg)
		}
	}

	// A lesson deleted behind the roadmap's back is skipped.
	delete(e.lessons.lessons, "a")
	if r, _ := e.svc.Roadmap(t.Context(), family.ID); len(r.Lessons) != 2 {
		t.Fatalf("deleted lesson shown: %+v", r.Lessons)
	}
	if _, err := e.svc.Roadmap(t.Context(), "nope"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown topic: %v", err)
	}
}
