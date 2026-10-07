package mysql

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/luongtran/luna/backend/internal/vocab"
)

func cardsTestCard(user, lemma string, created time.Time) vocab.Card {
	return vocab.Card{
		UserID: user, Text: lemma, Lemma: lemma, IPA: "/x/", MeaningVi: "nghia " + lemma, ContextSentence: "ctx " + lemma,
		Source: vocab.SourceAI, CreatedAt: created,
	}
}

func TestCardsCreateFindGet(t *testing.T) {
	t.Parallel()
	r := NewCards(testDB(t))
	ctx := t.Context()
	user, lesson := newID(), newID()
	created := time.Date(2026, 1, 2, 3, 4, 5, 123000, time.UTC)
	in := cardsTestCard(user, "run", created)
	in.LessonID = lesson
	in.Schedule = vocab.Schedule{Due: created.Add(time.Hour), Stability: 1.5, Difficulty: 2.5, ElapsedDays: 1, ScheduledDays: 2, Reps: 3, Lapses: 1, State: vocab.StateReview, LastReview: created}
	got, err := r.Create(ctx, in)
	if err != nil || len(got.ID) != 24 {
		t.Fatalf("Create = %+v, %v", got, err)
	}
	if _, err := r.Create(ctx, cardsTestCard(user, "run", created)); !errors.Is(err, vocab.ErrExists) {
		t.Fatalf("duplicate err = %v", err)
	}
	if _, err := r.Create(ctx, cardsTestCard(newID(), "run", created)); err != nil {
		t.Fatalf("same lemma for another user: %v", err)
	}
	for name, find := range map[string]func() (vocab.Card, error){
		"lemma": func() (vocab.Card, error) { return r.FindByLemma(ctx, user, "run") },
		"get":   func() (vocab.Card, error) { return r.Get(ctx, user, got.ID) },
	} {
		c, err := find()
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if c.ID != got.ID || c.LessonID != lesson || c.Source != vocab.SourceAI || !c.CreatedAt.Equal(created) || c.IPA != "/x/" {
			t.Fatalf("%s = %+v", name, c)
		}
		s := c.Schedule
		if !s.Due.Equal(in.Schedule.Due) || !s.LastReview.Equal(created) || s.Stability != 1.5 || s.Difficulty != 2.5 ||
			s.ElapsedDays != 1 || s.ScheduledDays != 2 || s.Reps != 3 || s.Lapses != 1 || s.State != vocab.StateReview {
			t.Fatalf("%s schedule = %+v", name, s)
		}
	}
	if _, err := r.FindByLemma(ctx, user, "nope"); !errors.Is(err, vocab.ErrNotFound) {
		t.Fatalf("missing lemma err = %v", err)
	}
	if _, err := r.Get(ctx, newID(), got.ID); !errors.Is(err, vocab.ErrNotFound) {
		t.Fatalf("other user's card err = %v", err)
	}
	if _, err := r.Get(ctx, user, "not-an-id"); !errors.Is(err, vocab.ErrNotFound) {
		t.Fatalf("malformed id err = %v", err)
	}
	// A card without a schedule reads back a zero schedule.
	plain, _ := r.Create(ctx, cardsTestCard(user, "walk", created))
	back, _ := r.Get(ctx, user, plain.ID)
	if back.Schedule != (vocab.Schedule{}) || back.LessonID != "" {
		t.Fatalf("plain card = %+v", back)
	}
}

func TestCardsWordsCountDelete(t *testing.T) {
	t.Parallel()
	r := NewCards(testDB(t))
	ctx := t.Context()
	user := newID()
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	var ids []string
	for i, l := range []string{"b", "a", "c"} {
		c, err := r.Create(ctx, cardsTestCard(user, l, base.Add(time.Duration(i)*time.Minute)))
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, c.ID)
	}
	words, err := r.Words(ctx, user)
	if err != nil || len(words) != 3 || words[0].Lemma != "b" || words[2].Lemma != "c" {
		t.Fatalf("Words = %+v, %v", words, err)
	}
	if w, _ := r.Words(ctx, newID()); len(w) != 0 {
		t.Fatalf("other user's words = %v", w)
	}
	if n, _ := r.Count(ctx, user, nil); n != 3 {
		t.Fatalf("Count = %d", n)
	}
	since := base.Add(time.Minute) // "a" and "c" were created since
	if n, err := r.Count(ctx, user, &since); err != nil || n != 2 {
		t.Fatalf("Count since = %d, %v", n, err)
	}
	if ok, err := r.Delete(ctx, newID(), ids[0]); ok || err != nil {
		t.Fatalf("Delete other user = %v, %v", ok, err)
	}
	if ok, err := r.Delete(ctx, user, ids[0]); !ok || err != nil {
		t.Fatalf("Delete = %v, %v", ok, err)
	}
	if ok, err := r.Delete(ctx, user, ids[0]); ok || err != nil {
		t.Fatalf("Delete again = %v, %v", ok, err)
	}
	if ok, err := r.Delete(ctx, user, "bad"); ok || err != nil {
		t.Fatalf("Delete malformed = %v, %v", ok, err)
	}
	if n, _ := r.Count(ctx, user, nil); n != 2 {
		t.Fatalf("Count after delete = %d", n)
	}
}

func TestCardsListFiltersAndPages(t *testing.T) {
	t.Parallel()
	r := NewCards(testDB(t))
	ctx := t.Context()
	user, lesson := newID(), newID()
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mk := func(text, lemma, lessonID string, at time.Time) {
		c := cardsTestCard(user, lemma, at)
		c.Text, c.LessonID = text, lessonID
		if _, err := r.Create(ctx, c); err != nil {
			t.Fatal(err)
		}
	}
	mk("Apple", "apple", lesson, base)
	mk("100% sure", "sure", "", base.Add(time.Minute))
	mk("a_b", "ab", "", base.Add(time.Minute)) // same created_at as the previous: later insert first
	mk("a.b", "a.b", lesson, base.Add(2*time.Minute))
	mk("Cat's", "cat", lesson, base.Add(3*time.Minute))

	lemmas := func(q vocab.ListQuery, limit, skip int) (out []string, more bool) {
		cards, more, err := r.List(ctx, user, q, limit, skip)
		if err != nil {
			t.Fatal(err)
		}
		for _, c := range cards {
			out = append(out, c.Lemma)
		}
		return out, more
	}
	eq := func(got, want []string) bool { return fmt.Sprint(got) == fmt.Sprint(want) }

	if got, more := lemmas(vocab.ListQuery{}, 10, 0); !eq(got, []string{"cat", "a.b", "ab", "sure", "apple"}) || more {
		t.Fatalf("all = %v %v", got, more)
	}
	if got, more := lemmas(vocab.ListQuery{}, 2, 0); !eq(got, []string{"cat", "a.b"}) || !more {
		t.Fatalf("page 1 = %v %v", got, more)
	}
	if got, more := lemmas(vocab.ListQuery{}, 2, 4); !eq(got, []string{"apple"}) || more {
		t.Fatalf("last page = %v %v", got, more)
	}
	if got, more := lemmas(vocab.ListQuery{}, 2, 3); !eq(got, []string{"sure", "apple"}) || more {
		t.Fatalf("exact end = %v %v", got, more)
	}
	if got, _ := lemmas(vocab.ListQuery{Q: "APP"}, 10, 0); !eq(got, []string{"apple"}) {
		t.Fatalf("case-insensitive text = %v", got)
	}
	if got, _ := lemmas(vocab.ListQuery{Q: "CAT"}, 10, 0); !eq(got, []string{"cat"}) {
		t.Fatalf("match on text and lemma = %v", got)
	}
	if got, _ := lemmas(vocab.ListQuery{Q: "a.b"}, 10, 0); !eq(got, []string{"a.b"}) {
		t.Fatalf("dot is literal = %v", got)
	}
	if got, _ := lemmas(vocab.ListQuery{Q: "_"}, 10, 0); !eq(got, []string{"ab"}) {
		t.Fatalf("underscore is literal = %v", got)
	}
	if got, _ := lemmas(vocab.ListQuery{Q: "%"}, 10, 0); !eq(got, []string{"sure"}) {
		t.Fatalf("percent is literal = %v", got)
	}
	if got, _ := lemmas(vocab.ListQuery{Q: `\`}, 10, 0); len(got) != 0 {
		t.Fatalf("backslash = %v", got)
	}
	if got, _ := lemmas(vocab.ListQuery{Q: "!"}, 10, 0); len(got) != 0 {
		t.Fatalf("bang = %v", got)
	}
	if got, _ := lemmas(vocab.ListQuery{Q: "'"}, 10, 0); !eq(got, []string{"cat"}) {
		t.Fatalf("quote = %v", got)
	}
	if got, _ := lemmas(vocab.ListQuery{LessonID: lesson}, 10, 0); !eq(got, []string{"cat", "a.b", "apple"}) {
		t.Fatalf("lesson = %v", got)
	}
	if got, _ := lemmas(vocab.ListQuery{LessonID: vocab.ManualLesson}, 10, 0); !eq(got, []string{"ab", "sure"}) {
		t.Fatalf("manual = %v", got)
	}
	if got, _ := lemmas(vocab.ListQuery{LessonID: lesson, Q: "a"}, 10, 0); !eq(got, []string{"cat", "a.b", "apple"}) {
		t.Fatalf("lesson+q = %v", got)
	}
	if got, _ := lemmas(vocab.ListQuery{LessonID: "malformed"}, 10, 0); len(got) != 0 {
		t.Fatalf("malformed lesson = %v", got)
	}

	counts, err := r.LessonCounts(ctx, user)
	if err != nil || counts[lesson] != 3 || counts[""] != 2 || len(counts) != 2 {
		t.Fatalf("LessonCounts = %v, %v", counts, err)
	}
	if c, _ := r.LessonCounts(ctx, newID()); len(c) != 0 {
		t.Fatalf("other user's counts = %v", c)
	}
}

func TestCardsUpdateDetails(t *testing.T) {
	t.Parallel()
	r := NewCards(testDB(t))
	ctx := t.Context()
	user := newID()
	due := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	in := cardsTestCard(user, "go", due)
	in.Schedule = vocab.Schedule{Due: due, Reps: 2, Stability: 3}
	c, _ := r.Create(ctx, in)

	meaning, empty := "moi", ""
	got, err := r.UpdateDetails(ctx, user, c.ID, vocab.Details{MeaningVi: &meaning, ContextSentence: &empty})
	if err != nil || got.MeaningVi != "moi" || got.ContextSentence != "" || got.IPA != "/x/" {
		t.Fatalf("UpdateDetails = %+v, %v", got, err)
	}
	if got.Schedule.Reps != 2 || got.Schedule.Stability != 3 || !got.Schedule.Due.Equal(due) {
		t.Fatalf("schedule changed: %+v", got.Schedule)
	}
	same, err := r.UpdateDetails(ctx, user, c.ID, vocab.Details{MeaningVi: &meaning}) // no actual change
	if err != nil || same.MeaningVi != "moi" {
		t.Fatalf("same value = %+v, %v", same, err)
	}
	none, err := r.UpdateDetails(ctx, user, c.ID, vocab.Details{})
	if err != nil || none.MeaningVi != "moi" {
		t.Fatalf("no fields = %+v, %v", none, err)
	}
	if _, err := r.UpdateDetails(ctx, newID(), c.ID, vocab.Details{MeaningVi: &meaning}); !errors.Is(err, vocab.ErrNotFound) {
		t.Fatalf("other user err = %v", err)
	}
	if _, err := r.UpdateDetails(ctx, user, newID(), vocab.Details{}); !errors.Is(err, vocab.ErrNotFound) {
		t.Fatalf("missing, no fields err = %v", err)
	}
	if _, err := r.UpdateDetails(ctx, user, "bad", vocab.Details{IPA: &meaning}); !errors.Is(err, vocab.ErrNotFound) {
		t.Fatalf("malformed err = %v", err)
	}
}

func TestCardsUpdateScheduleChecksReps(t *testing.T) {
	t.Parallel()
	r := NewCards(testDB(t))
	ctx := t.Context()
	user := newID()
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	c, _ := r.Create(ctx, cardsTestCard(user, "x", now.Add(-48*time.Hour))) // legacy: no schedule

	s1 := vocab.Schedule{Due: now.Add(time.Hour), Stability: 1, Difficulty: 5, Reps: 1, State: vocab.StateLearning, LastReview: now}
	if ok, err := r.UpdateSchedule(ctx, user, c.ID, 1, s1); ok || err != nil {
		t.Fatalf("wrong reps = %v, %v", ok, err)
	}
	if got, _ := r.Get(ctx, user, c.ID); got.Schedule != (vocab.Schedule{}) {
		t.Fatalf("a refused write changed the card: %+v", got.Schedule)
	}
	if ok, err := r.UpdateSchedule(ctx, user, c.ID, 0, s1); !ok || err != nil {
		t.Fatalf("first write = %v, %v", ok, err)
	}
	if ok, _ := r.UpdateSchedule(ctx, user, c.ID, 0, s1); ok {
		t.Fatal("stale writer won")
	}
	s2 := s1
	s2.Reps, s2.Lapses, s2.State = 2, 1, vocab.StateRelearning
	if ok, _ := r.UpdateSchedule(ctx, user, c.ID, 1, s2); !ok {
		t.Fatal("second write refused")
	}
	got, _ := r.Get(ctx, user, c.ID)
	if got.Schedule.Reps != 2 || got.Schedule.Lapses != 1 || got.Schedule.State != vocab.StateRelearning ||
		!got.Schedule.Due.Equal(s1.Due) || !got.Schedule.LastReview.Equal(now) {
		t.Fatalf("schedule = %+v", got.Schedule)
	}
	// Rewriting equal data still counts as a match.
	if ok, _ := r.UpdateSchedule(ctx, user, c.ID, 2, s2); !ok {
		t.Fatal("identical rewrite refused")
	}
	if ok, _ := r.UpdateSchedule(ctx, newID(), c.ID, 2, s2); ok {
		t.Fatal("other user wrote")
	}
	if ok, err := r.UpdateSchedule(ctx, user, "bad", 0, s2); ok || err != nil {
		t.Fatalf("malformed id = %v, %v", ok, err)
	}
}

func TestCardsDueNextDueCountDue(t *testing.T) {
	t.Parallel()
	r := NewCards(testDB(t))
	ctx := t.Context()
	user := newID()
	start := time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC)
	now := start.Add(10 * time.Hour)
	mk := func(lemma string, created time.Time, due time.Time) string {
		c := cardsTestCard(user, lemma, created)
		c.Schedule.Due = due
		if !due.IsZero() {
			c.Schedule.Reps = 1
		}
		out, err := r.Create(ctx, c)
		if err != nil {
			t.Fatal(err)
		}
		return out.ID
	}
	mk("legacy-old", start.Add(-24*time.Hour), time.Time{})
	mk("legacy-today", start.Add(time.Hour), time.Time{})
	mk("overdue", start.Add(-72*time.Hour), now.Add(-5*time.Hour))
	mk("just-due", start.Add(-72*time.Hour), now)
	mk("older-overdue", start.Add(-72*time.Hour), now.Add(-30*time.Hour))
	mk("later", start.Add(-72*time.Hour), now.Add(3*time.Hour))
	mk("much-later", start.Add(-72*time.Hour), now.Add(48*time.Hour))
	other := cardsTestCard(newID(), "other", start.Add(-72*time.Hour))
	other.Schedule.Due = now.Add(-time.Hour)
	_, _ = r.Create(ctx, other)

	cards, total, err := r.Due(ctx, user, now, start, 3)
	if err != nil || total != 4 {
		t.Fatalf("Due total = %d, %v", total, err)
	}
	var got []string
	for _, c := range cards {
		got = append(got, c.Lemma)
	}
	if fmt.Sprint(got) != "[legacy-old older-overdue overdue]" {
		t.Fatalf("Due = %v", got)
	}
	if cards, total, _ := r.Due(ctx, user, now, start, 10); len(cards) != 4 || total != 4 {
		t.Fatalf("Due all = %d of %d", len(cards), total)
	}
	if cards, total, _ := r.Due(ctx, newID(), now, start, 10); len(cards) != 0 || total != 0 {
		t.Fatalf("Due other user = %d of %d", len(cards), total)
	}

	next, ok, err := r.NextDue(ctx, user, now, start)
	if err != nil || !ok || !next.Equal(now.Add(3*time.Hour)) {
		t.Fatalf("NextDue = %v %v %v", next, ok, err)
	}
	// Without the 'later' cards, the new card saved today is due tomorrow.
	user2 := newID()
	c := cardsTestCard(user2, "n", start.Add(time.Hour))
	_, _ = r.Create(ctx, c)
	next, ok, _ = r.NextDue(ctx, user2, now, start)
	if !ok || !next.Equal(start.AddDate(0, 0, 1)) {
		t.Fatalf("NextDue legacy = %v %v", next, ok)
	}
	late := cardsTestCard(user2, "late", start.Add(time.Hour))
	late.Schedule.Due = start.Add(30 * time.Hour)
	_, _ = r.Create(ctx, late)
	next, _, _ = r.NextDue(ctx, user2, now, start)
	if !next.Equal(start.AddDate(0, 0, 1)) {
		t.Fatalf("NextDue picks the earlier = %v", next)
	}
	if _, ok, _ := r.NextDue(ctx, newID(), now, start); ok {
		t.Fatal("NextDue for an empty notebook")
	}

	// CountDue: due < before (strict), legacy created before createdBefore.
	if n, err := r.CountDue(ctx, user, now, start); err != nil || n != 3 {
		t.Fatalf("CountDue = %d, %v", n, err)
	}
	if n, _ := r.CountDue(ctx, user, now.Add(time.Microsecond), start.AddDate(0, 0, 1)); n != 5 {
		t.Fatalf("CountDue later = %d", n)
	}
}
