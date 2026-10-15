package vocab

import (
	"context"
	"fmt"
	"time"
)

// ForecastDays is how many days, today included, the review forecast covers.
const ForecastDays = 7

// knownDays is the interval from which a card in review counts as known (Anki's "mature").
const knownDays = 21

// Forecast is the overview of the review tab: the learner's cards by stage and how many come
// due on each of the next days.
type Forecast struct {
	New, Learning, Known int
	// Days has ForecastDays entries from today; today counts the overdue cards too.
	Days []DayCount
}

// DayCount is the number of cards due on Day (YYYY-MM-DD in the learner's timezone).
type DayCount struct {
	Day   string
	Count int
}

// Forecast counts userID's cards by stage (new, learning, known) and by due day over the
// next ForecastDays days.
func (s *Service) Forecast(ctx context.Context, userID string) (Forecast, error) {
	loc, err := s.location(ctx, userID)
	if err != nil {
		return Forecast{}, err
	}
	cards, err := s.repo.Schedules(ctx, userID)
	if err != nil {
		return Forecast{}, fmt.Errorf("vocab: schedules: %w", err)
	}
	today := startOfDay(s.now(), loc)
	out := Forecast{Days: make([]DayCount, ForecastDays)}
	for i := range out.Days {
		out.Days[i].Day = dayOf(today.AddDate(0, 0, i), loc)
	}
	for _, c := range cards {
		sched := effective(c, loc)
		switch {
		case sched.State == StateNew:
			out.New++
		case sched.State == StateReview && sched.ScheduledDays >= knownDays:
			out.Known++
		default:
			out.Learning++
		}
		if i := daysFrom(today, sched.Due, loc); i < ForecastDays {
			out.Days[max(i, 0)].Count++
		}
	}
	return out, nil
}

// daysFrom is the number of local days from the day starting at today to t's day (negative before).
func daysFrom(today, t time.Time, loc *time.Location) int {
	day := startOfDay(t, loc)
	y, m, d := day.Date()
	ty, tm, td := today.Date()
	return int(time.Date(y, m, d, 0, 0, 0, 0, time.UTC).Sub(time.Date(ty, tm, td, 0, 0, 0, 0, time.UTC)).Hours() / 24)
}
