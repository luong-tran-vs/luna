package progress

import "time"

// Pure rules of the study flow (L): no storage, no clock. The service reads the data, passes
// now and the learner's timezone, and applies these functions.

// DayKey is the learner's calendar day of now (YYYY-MM-DD); a day starts at 0:00 in loc.
func DayKey(now time.Time, loc *time.Location) string {
	return now.In(loc).Format(time.DateOnly)
}

func shiftDay(key string, days int) string {
	t, err := time.Parse(time.DateOnly, key)
	if err != nil {
		return key
	}
	return t.AddDate(0, 0, days).Format(time.DateOnly)
}

// NextDayKey is the day after key.
func NextDayKey(key string) string { return shiftDay(key, 1) }

// PrevDayKey is the day before key.
func PrevDayKey(key string) string { return shiftDay(key, -1) }

// EffectiveGoal is the learner's active goal, if any.
func EffectiveGoal(goals []Goal) (Goal, bool) {
	for _, g := range goals {
		if g.Status == GoalActive {
			return g, true
		}
	}
	return Goal{}, false
}

// CurrentLesson is the lesson the learner studies now: the first lesson of the goal's roadmap not
// completed yet. There is no daily limit: once a lesson is completed the next one opens.
func CurrentLesson(goal *Goal, roadmap []string, completed map[string]bool) StudyState {
	if goal == nil {
		return StudyState{Kind: StudyNoGoal}
	}
	for _, id := range roadmap {
		if !completed[id] {
			return StudyState{Kind: StudyStudying, LessonID: id}
		}
	}
	return StudyState{Kind: StudyNoNewLesson}
}

// FreezeEvery is how many study days must pass before another missed day is forgiven.
const FreezeEvery = 7

// Streak counts consecutive days with a completed lesson, ending today when today is done and
// yesterday otherwise. One missed day between two study days is forgiven (it is not counted), and
// again only after FreezeEvery study days since the last one forgiven; two missed days in a row
// reset it to 0. Nothing is stored: the rule reads the days alone.
func Streak(completedDays []string, today string) int {
	done := make(map[string]bool, len(completedDays))
	for _, d := range completedDays {
		done[d] = true
	}
	day := today
	if !done[day] {
		day = PrevDayKey(today)
	}
	n, sinceFreeze := 0, FreezeEvery
	for {
		switch {
		case done[day]:
			n++
			sinceFreeze++
			day = PrevDayKey(day)
		case sinceFreeze >= FreezeEvery && done[PrevDayKey(day)]:
			sinceFreeze = 0
			day = PrevDayKey(day)
		default:
			return n
		}
	}
}

// NextStep is the first step not done, or StepDone.
func NextStep(done map[Step]bool) Step {
	for _, s := range Steps {
		if !done[s] {
			return s
		}
	}
	return StepDone
}

// Period is the span the stats page covers.
type Period string

const (
	PeriodWeek  Period = "week"
	PeriodMonth Period = "month"
	PeriodAll   Period = "all"
)

// ValidPeriod reports whether p is one of the periods.
func ValidPeriod(p Period) bool { return p == PeriodWeek || p == PeriodMonth || p == PeriodAll }

// PeriodStart is when the calendar period holding now starts in loc: Monday 0:00 for a week, the
// 1st at 0:00 for a month. It is nil for PeriodAll (no lower bound).
func PeriodStart(p Period, now time.Time, loc *time.Location) *time.Time {
	t := now.In(loc)
	y, m, d := t.Date()
	var start time.Time
	switch p {
	case PeriodWeek:
		// Weekday counts from Sunday; shift it so Monday is 0.
		start = time.Date(y, m, d-(int(t.Weekday())+6)%7, 0, 0, 0, 0, loc)
	case PeriodMonth:
		start = time.Date(y, m, 1, 0, 0, 0, 0, loc)
	default:
		return nil
	}
	return &start
}

// GuestLesson is CurrentLesson for a guest, who may study only the first lesson of the roadmap:
// once it is completed the others are for members.
func GuestLesson(goal *Goal, roadmap []string, completed map[string]bool) StudyState {
	if len(roadmap) <= 1 {
		return CurrentLesson(goal, roadmap, completed)
	}
	st := CurrentLesson(goal, roadmap[:1], completed)
	if st.Kind == StudyNoNewLesson {
		st.Kind = StudyMembersOnly
	}
	return st
}
