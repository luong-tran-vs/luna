package games

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestStreakWeekCompare(t *testing.T) {
	t.Parallel()
	today := "2026-11-13" // a Friday
	if s := streakOf([]string{"2026-11-13", "2026-11-12", "2026-11-11", "2026-11-09"}, today); s != (Streak{Current: 3, Longest: 3}) {
		t.Fatalf("to today = %+v", s)
	}
	// Today not played yet: the streak up to yesterday still counts.
	if s := streakOf([]string{"2026-11-12", "2026-11-11"}, today); s.Current != 2 {
		t.Fatalf("to yesterday = %+v", s)
	}
	// A day missed: broken.
	if s := streakOf([]string{"2026-11-11", "2026-11-10", "2026-11-09", "2026-11-08"}, today); s != (Streak{Current: 0, Longest: 4}) {
		t.Fatalf("broken = %+v", s)
	}
	week := weekOf([]string{"2026-11-09", "2026-11-11", "2026-11-13", "2026-11-08"}, today)
	if week != [7]bool{true, false, true, false, true, false, false} {
		t.Fatalf("week = %v", week)
	}
	results := []DailyResult{{Day: "2026-11-13", Score: 999}, {Day: "2026-11-12", Score: 880}, {Day: "2026-11-10", Score: 800}, {Day: "2026-11-01", Score: 990}}
	c := compareOf(results, today)
	if c.Yesterday == nil || *c.Yesterday != 880 || c.Avg7 == nil || *c.Avg7 != 840 || c.Best != 990 {
		t.Fatalf("compare = %+v", c)
	}
	if c := compareOf(nil, today); c.Yesterday != nil || c.Avg7 != nil || c.Best != 0 {
		t.Fatalf("empty compare = %+v", c)
	}
}

func dailyEnv(t *testing.T) *env {
	t.Helper()
	e := newEnv(t)
	e.grammar.sources = []PointSource{presentSimple(), pastSimple()}
	e.verbs.due = []string{"go", "eat"}
	e.verbs.other = []string{"see", "run", "take"}
	return e
}

func kinds(qs []DailyQuestion) map[string]int {
	out := map[string]int{}
	for _, q := range qs {
		out[q.Kind]++
	}
	return out
}

func TestDailyQuestions(t *testing.T) {
	t.Parallel()
	e := dailyEnv(t)
	v, err := e.svc.Daily(t.Context(), "u1", false)
	if err != nil {
		t.Fatal(err)
	}
	if v.Day != "2026-11-13" || v.Done || len(v.Questions) != DailyCount {
		t.Fatalf("view = %+v", v)
	}
	if k := kinds(v.Questions); k[KindPicture] != 3 || k[KindVerb] != 3 || k[KindLetters] != 2 || k[KindGrammar] != 2 {
		t.Fatalf("kinds = %v", k)
	}
	// The same day gives the same questions; another day other ones.
	again, _ := e.svc.Daily(t.Context(), "u1", false)
	if !reflect.DeepEqual(again.Questions, v.Questions) || again.Seed != v.Seed {
		t.Fatal("not the same set within the day")
	}
	e.now = e.now.Add(24 * time.Hour)
	next, _ := e.svc.Daily(t.Context(), "u1", false)
	if next.Day != "2026-11-14" || reflect.DeepEqual(next.Questions, v.Questions) {
		t.Fatal("same set on the next day")
	}
	// Due verbs first.
	for _, q := range v.Questions {
		if q.Kind == KindVerb && q.Verb.Base != "go" && q.Verb.Base != "eat" && q.Verb.Base != "see" && q.Verb.Base != "run" && q.Verb.Base != "take" {
			t.Fatalf("verb = %+v", q.Verb)
		}
	}

	// Guests: no grammar, 4 / 3 / 3.
	e.cards.cards["u2"] = e.cards.cards["u1"]
	g, err := e.svc.Daily(t.Context(), "u2", true)
	if err != nil {
		t.Fatal(err)
	}
	if k := kinds(g.Questions); k[KindPicture] != 4 || k[KindVerb] != 3 || k[KindLetters] != 3 || k[KindGrammar] != 0 {
		t.Fatalf("guest kinds = %v", k)
	}
}

func TestDailyQuestionsShortContent(t *testing.T) {
	t.Parallel()
	e := dailyEnv(t)
	e.grammar.sources = nil
	e.verbs.due, e.verbs.other = nil, nil
	// A new learner with no picture: everything goes to the spelling, then the verbs.
	e.cards.cards["u9"] = []Card{{Lemma: "cat", MeaningVi: "mèo"}, {Lemma: "dog", MeaningVi: "chó"}}
	v, err := e.svc.Daily(t.Context(), "u9", false)
	if err != nil {
		t.Fatal(err)
	}
	if k := kinds(v.Questions); len(v.Questions) != 10 || k[KindLetters] != 2 || k[KindVerb] != 8 {
		t.Fatalf("kinds = %v", k)
	}
	for _, q := range v.Questions {
		if q.Kind == KindVerb && q.Verb.Base != "" {
			t.Fatalf("no verb learned, yet %+v", q.Verb)
		}
	}
}

func answers(right int) []DailyAnswer {
	out := make([]DailyAnswer, DailyCount)
	for i := range out {
		out[i] = DailyAnswer{Kind: KindPicture, Right: i < right}
	}
	return out
}

func TestSaveDaily(t *testing.T) {
	t.Parallel()
	e := dailyEnv(t)
	ctx := t.Context()
	_ = e.daily.Save(ctx, DailyResult{UserID: "u1", Day: "2026-11-12", Score: 880})
	var verr *ValidationError
	for name, b := range map[string]DailyBody{
		"day":      {Day: "2026-11-10", Answers: answers(9)},
		"answers":  {Day: "2026-11-13", Answers: answers(9)[:9]},
		"answers ": {Day: "2026-11-13", Answers: append(answers(9)[:9], DailyAnswer{Kind: "memory"})},
		"score":    {Day: "2026-11-13", Answers: answers(9), Score: 3001},
	} {
		if _, err := e.svc.SaveDaily(ctx, "u1", b); !errors.As(err, &verr) || verr.Fields[strings.TrimSpace(name)] == "" {
			t.Fatalf("%s: %v", name, err)
		}
	}

	reply, err := e.svc.SaveDaily(ctx, "u1", DailyBody{Day: "2026-11-13", Score: 920, Answers: answers(9), WrongWords: []string{"have"}})
	if err != nil {
		t.Fatal(err)
	}
	if reply.Streak.Current != 2 || reply.Rescheduled != 1 || reply.Compare.Yesterday == nil || *reply.Compare.Yesterday != 880 || e.days.n != 1 {
		t.Fatalf("reply = %+v, study days %d", reply, e.days.n)
	}
	_, err = e.svc.SaveDaily(ctx, "u1", DailyBody{Day: "2026-11-13", Score: 3000, Answers: answers(10)})
	var done *AlreadyDoneError
	if !errors.As(err, &done) || !errors.Is(err, ErrAlreadyDone) || done.Result.Score != 920 {
		t.Fatalf("second = %v", err)
	}
	v, _ := e.svc.Daily(ctx, "u1", false)
	if !v.Done || v.Result == nil || v.Result.Score != 920 || len(v.Questions) != DailyCount || v.Streak.Current != 2 {
		t.Fatalf("view = %+v", v)
	}
	list, _ := e.svc.List(ctx, "u1", false)
	if !list.Daily.Done || list.Daily.Streak != 2 {
		t.Fatalf("list daily = %+v", list.Daily)
	}

	// A round started before midnight counts for that day.
	e.now = e.now.Add(24 * time.Hour)
	if _, err := e.svc.SaveDaily(ctx, "u3", DailyBody{Day: "2026-11-13", Answers: answers(5)}); err != nil {
		t.Fatalf("yesterday: %v", err)
	}
	// Practice: only the misses.
	if n, err := e.svc.DailyPractice(ctx, "u1", []string{"apple", "pear"}, nil); err != nil || n != 2 {
		t.Fatalf("practice = %d, %v", n, err)
	}
}
