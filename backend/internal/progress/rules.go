package progress

import "time"

// Pure rules of the daily flow (L): no storage, no clock. The service reads the data, passes
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

// TodayLesson decides today's lesson: the lesson fixed for today once a step was done, otherwise
// the first lesson of the goal's roadmap not completed yet. Days off never add lessons.
func TodayLesson(goal *Goal, roadmap []string, completed map[string]bool, today *StudyDay) TodayState {
	if today != nil && today.LessonID != "" {
		if today.Completed {
			return TodayState{Kind: TodayDone, LessonID: today.LessonID, Started: true}
		}
		return TodayState{Kind: TodayStudying, LessonID: today.LessonID, Started: true}
	}
	if goal == nil {
		return TodayState{Kind: TodayNoGoal}
	}
	for _, id := range roadmap {
		if !completed[id] {
			return TodayState{Kind: TodayStudying, LessonID: id}
		}
	}
	return TodayState{Kind: TodayNoNewLesson}
}

// CanStartNewLesson reports whether a new lesson may still be studied today.
func CanStartNewLesson(today *StudyDay) bool {
	return today == nil || !today.Completed
}

// Streak counts consecutive days with a completed lesson, ending today when today is done and
// yesterday otherwise; a whole day missed resets it to 0.
func Streak(completedDays []string, today string) int {
	done := make(map[string]bool, len(completedDays))
	for _, d := range completedDays {
		done[d] = true
	}
	day := today
	if !done[day] {
		day = PrevDayKey(today)
	}
	n := 0
	for done[day] {
		n++
		day = PrevDayKey(day)
	}
	return n
}

// ReviewQuota is how many more cards the review step may show today.
func ReviewQuota(limit, reviewedToday int) int {
	return max(0, limit-reviewedToday)
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

// EffectiveDayKey is the study day: the local day, or the latest day already studied when a
// timezone change moved the local day back (F12). The study day never goes back, so a timezone
// change never gives a second lesson in the same real day. Keys are YYYY-MM-DD; "" means none.
func EffectiveDayKey(localKey, latestKey string) string {
	return max(localKey, latestKey)
}
