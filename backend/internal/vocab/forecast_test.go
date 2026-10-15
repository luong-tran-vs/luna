package vocab

import (
	"testing"
	"time"
)

func TestForecast(t *testing.T) {
	t.Parallel()
	e := newEnv()
	now := e.clock.Now() // 15:00 on 2026-09-29 in Ho Chi Minh City
	e.dueCard("overdue", Schedule{Due: now.AddDate(0, 0, -5), State: StateLearning, Reps: 1})
	e.dueCard("tonight", Schedule{Due: now.Add(6 * time.Hour), State: StateReview, ScheduledDays: 3, Reps: 2})
	e.dueCard("known", Schedule{Due: now.AddDate(0, 0, 2), State: StateReview, ScheduledDays: 30, Reps: 5})
	e.dueCard("later", Schedule{Due: now.AddDate(0, 0, 40), State: StateReview, ScheduledDays: 40, Reps: 6})
	e.repo.put(Card{UserID: "u1", Text: "fresh", Lemma: "fresh", MeaningVi: "mới", Source: SourceManual, CreatedAt: now})
	e.repo.put(Card{UserID: "u2", Text: "other", Lemma: "other", MeaningVi: "khác", Source: SourceManual, CreatedAt: now})

	f, err := e.svc.Forecast(t.Context(), "u1")
	if err != nil {
		t.Fatal(err)
	}
	if f.New != 1 || f.Learning != 2 || f.Known != 2 {
		t.Fatalf("stages = %d/%d/%d, want 1/2/2", f.New, f.Learning, f.Known)
	}
	want := []DayCount{{"2026-09-29", 2}, {"2026-09-30", 1}, {"2026-10-01", 1}, {"2026-10-02", 0}, {"2026-10-03", 0}, {"2026-10-04", 0}, {"2026-10-05", 0}}
	if len(f.Days) != len(want) {
		t.Fatalf("days = %+v", f.Days)
	}
	for i := range want {
		if f.Days[i] != want[i] {
			t.Fatalf("day %d = %+v, want %+v", i, f.Days[i], want[i])
		}
	}
}
