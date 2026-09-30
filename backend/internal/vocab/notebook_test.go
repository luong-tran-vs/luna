package vocab

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// saved stores a card of user saved at t.
func (e *testEnv) saved(user, text, lessonID string, t time.Time) Card {
	source := SourceAI
	if lessonID == "" {
		source = SourceManual
	}
	return e.repo.put(Card{
		UserID: user, Text: text, Lemma: strings.ToLower(text), MeaningVi: "nghĩa", LessonID: lessonID,
		Source: source, CreatedAt: t, Schedule: Schedule{Due: FirstDue(t, hcm), State: StateNew},
	})
}

func TestListGroupsByLocalDay(t *testing.T) {
	t.Parallel()
	e := newEnv() // now: 2026-09-29 15:00 in Viet Nam, 08:00 UTC
	// 23:30 and 00:30 in Viet Nam are 16:30 and 17:30 UTC of the same UTC day.
	late := e.saved("u1", "late", "lesson1", time.Date(2026, 9, 27, 16, 30, 0, 0, time.UTC))
	early := e.saved("u1", "early", "lesson1", time.Date(2026, 9, 27, 17, 30, 0, 0, time.UTC))
	today := e.saved("u1", "today", "", time.Date(2026, 9, 29, 1, 0, 0, 0, time.UTC))

	page, err := e.svc.List(t.Context(), "u1", ListQuery{Page: 1})
	if err != nil {
		t.Fatal(err)
	}
	if page.Today != "2026-09-29" || page.Yesterday != "2026-09-28" || page.HasMore {
		t.Fatalf("page = %+v", page)
	}
	days := map[string]string{}
	var order []string
	for _, c := range page.Cards {
		days[c.ID] = c.Day
		order = append(order, c.ID)
	}
	if days[late.ID] != "2026-09-27" || days[early.ID] != "2026-09-28" || days[today.ID] != "2026-09-29" {
		t.Fatalf("days = %v", days)
	}
	if strings.Join(order, ",") != today.ID+","+early.ID+","+late.ID {
		t.Fatalf("order = %v", order)
	}

	// A learner in UTC sees the two late cards on the same day.
	e.saved("u2", "a", "lesson1", time.Date(2026, 9, 27, 16, 30, 0, 0, time.UTC))
	e.saved("u2", "b", "lesson1", time.Date(2026, 9, 27, 17, 30, 0, 0, time.UTC))
	utc, _ := e.svc.List(t.Context(), "u2", ListQuery{Page: 1})
	if len(utc.Cards) != 2 || utc.Cards[0].Day != "2026-09-27" || utc.Cards[1].Day != "2026-09-27" || utc.Today != "2026-09-29" {
		t.Fatalf("utc page = %+v", utc)
	}
}

func TestListLegacyCardShowsEffectiveSchedule(t *testing.T) {
	t.Parallel()
	e := newEnv()
	e.repo.put(Card{
		UserID: "u1", Text: "old", Lemma: "old", MeaningVi: "cũ", LessonID: "lesson1", Source: SourceAI,
		CreatedAt: time.Date(2026, 9, 20, 3, 0, 0, 0, time.UTC),
	})
	page, _ := e.svc.List(t.Context(), "u1", ListQuery{Page: 1})
	if s := page.Cards[0].Schedule; !s.Due.Equal(time.Date(2026, 9, 21, 0, 0, 0, 0, hcm)) || s.State != StateNew {
		t.Fatalf("schedule = %+v", s)
	}
}

func TestListFilters(t *testing.T) {
	t.Parallel()
	e := newEnv()
	base := time.Date(2026, 9, 29, 1, 0, 0, 0, time.UTC)
	e.saved("u1", "go", "lesson1", base)
	e.saved("u1", "Give up", "lesson1", base.Add(time.Minute))
	e.saved("u1", "a.b", "", base.Add(2*time.Minute))
	e.saved("u1", "axb", "", base.Add(3*time.Minute))
	e.saved("u2", "give", "lesson1", base)

	texts := func(q ListQuery) string {
		t.Helper()
		q.Page = 1
		page, err := e.svc.List(t.Context(), "u1", q)
		if err != nil {
			t.Fatal(err)
		}
		var out []string
		for _, c := range page.Cards {
			out = append(out, c.Text)
		}
		return strings.Join(out, ",")
	}
	if got := texts(ListQuery{Q: " GIV "}); got != "Give up" {
		t.Fatalf("search = %q", got)
	}
	if got := texts(ListQuery{Q: "a.b"}); got != "a.b" {
		t.Fatalf("regex characters = %q", got)
	}
	if got := texts(ListQuery{LessonID: "lesson1"}); got != "Give up,go" {
		t.Fatalf("lesson filter = %q", got)
	}
	if got := texts(ListQuery{LessonID: ManualLesson}); got != "axb,a.b" {
		t.Fatalf("manual filter = %q", got)
	}
	if got := texts(ListQuery{LessonID: ManualLesson, Q: "x"}); got != "axb" {
		t.Fatalf("search + filter = %q", got)
	}
}

func TestListPages(t *testing.T) {
	t.Parallel()
	e := newEnv()
	base := time.Date(2026, 9, 1, 1, 0, 0, 0, time.UTC)
	for i := range 31 {
		e.saved("u1", "w"+string(rune('a'+i%26))+string(rune('a'+i/26)), "lesson1", base.Add(time.Duration(i)*time.Hour))
	}
	first, _ := e.svc.List(t.Context(), "u1", ListQuery{Page: 1})
	second, _ := e.svc.List(t.Context(), "u1", ListQuery{Page: 2})
	if len(first.Cards) != pageSize || !first.HasMore || len(second.Cards) != 1 || second.HasMore {
		t.Fatalf("pages = %d/%v, %d/%v", len(first.Cards), first.HasMore, len(second.Cards), second.HasMore)
	}

	for name, q := range map[string]ListQuery{"page 0": {Page: 0}, "long q": {Page: 1, Q: strings.Repeat("á", 101)}} {
		var verr *ValidationError
		if _, err := e.svc.List(t.Context(), "u1", q); !errors.As(err, &verr) {
			t.Fatalf("%s: %v", name, err)
		}
	}
}

func TestLessons(t *testing.T) {
	t.Parallel()
	e := newEnv()
	now := e.clock.Now()
	e.saved("u1", "go", "lesson1", now)
	e.saved("u1", "run", "lesson1", now)
	e.saved("u1", "gone", "deleted-lesson", now)
	e.saved("u1", "hi", "", now)
	e.saved("u2", "x", "lesson1", now)

	lessons, manual, err := e.svc.Lessons(t.Context(), "u1")
	if err != nil {
		t.Fatal(err)
	}
	if len(lessons) != 1 || lessons[0] != (LessonCount{ID: "lesson1", Title: "A day at the park", Count: 2}) || manual != 1 {
		t.Fatalf("lessons = %+v, manual %d", lessons, manual)
	}
}

func TestUpdateKeepsSchedule(t *testing.T) {
	t.Parallel()
	e := newEnv()
	sched := Schedule{Due: time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC), Stability: 4, Difficulty: 5, Reps: 2, State: StateReview}
	c := e.dueCard("went", sched)
	_ = e.logs.Add(t.Context(), ReviewLog{UserID: "u1", CardID: c.ID, Rating: Good})

	meaning := "  đã đi (quá khứ của go) "
	got, err := e.svc.Update(t.Context(), "u1", c.ID, Details{MeaningVi: &meaning})
	if err != nil {
		t.Fatal(err)
	}
	if got.MeaningVi != "đã đi (quá khứ của go)" || got.Schedule != sched || got.Text != "went" {
		t.Fatalf("updated = %+v", got)
	}
	if n := len(e.logs.forCard(c.ID)); n != 1 {
		t.Fatalf("logs = %d", n)
	}

	cases := map[string]Details{
		"empty meaning": {MeaningVi: ptr(" ")},
		"long meaning":  {MeaningVi: ptr(strings.Repeat("ă", 201))},
		"long ipa":      {IPA: ptr(strings.Repeat("a", 101))},
		"long example":  {ContextSentence: ptr(strings.Repeat("a", 1001))},
	}
	for name, d := range cases {
		var verr *ValidationError
		if _, err := e.svc.Update(t.Context(), "u1", c.ID, d); !errors.As(err, &verr) {
			t.Fatalf("%s: %v", name, err)
		}
	}
	if _, err := e.svc.Update(t.Context(), "u2", c.ID, Details{MeaningVi: &meaning}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("other user: %v", err)
	}
}

func TestDeleteRemovesLogs(t *testing.T) {
	t.Parallel()
	e := newEnv()
	c := e.dueCard("went", Schedule{State: StateNew})
	other := e.dueCard("run", Schedule{State: StateNew})
	for _, id := range []string{c.ID, c.ID, other.ID} {
		_ = e.logs.Add(t.Context(), ReviewLog{UserID: "u1", CardID: id, Rating: Good})
	}

	if err := e.svc.Delete(t.Context(), "u2", c.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("other user: %v", err)
	}
	if err := e.svc.Delete(t.Context(), "u1", c.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := e.repo.Get(t.Context(), "u1", c.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("card still there: %v", err)
	}
	if len(e.logs.forCard(c.ID)) != 0 || len(e.logs.forCard(other.ID)) != 1 {
		t.Fatal("wrong logs deleted")
	}
	if err := e.svc.Delete(t.Context(), "u1", c.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("second delete: %v", err)
	}

	// Leftover logs of an already deleted card are still cleaned up.
	_ = e.logs.Add(t.Context(), ReviewLog{UserID: "u1", CardID: "gone", Rating: Good})
	if err := e.svc.Delete(t.Context(), "u1", "gone"); err != nil || len(e.logs.forCard("gone")) != 0 {
		t.Fatalf("leftover logs: %v", err)
	}

	// The word can be saved again as a new card.
	in := validInput()
	in.Text, in.Lemma = "went", "went"
	if _, err := e.svc.Save(t.Context(), "u1", in); err != nil {
		t.Fatalf("save again: %v", err)
	}
}

func ptr(s string) *string { return &s }
