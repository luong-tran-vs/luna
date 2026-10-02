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

// NextStep is the first step not done, or StepDone.
func NextStep(done map[Step]bool) Step {
	for _, s := range Steps {
		if !done[s] {
			return s
		}
	}
	return StepDone
}
