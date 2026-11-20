package toeic

import "math"

// scaleMark is a point of the estimated conversion table: questions right of 100 → section score.
type scaleMark struct{ right, score int }

// Estimated conversion tables (research R5); not the official ETS one, which is not published.
var scaleTables = map[Section][]scaleMark{
	Listening: {{0, 5}, {10, 20}, {25, 105}, {50, 245}, {65, 330}, {75, 385}, {85, 435}, {92, 470}, {96, 495}, {100, 495}},
	Reading:   {{0, 5}, {10, 15}, {25, 85}, {50, 215}, {65, 300}, {75, 360}, {85, 420}, {92, 460}, {97, 495}, {100, 495}},
}

// scaled converts correct answers of total questions in a section to an estimated 5–495 score: the
// count is brought to 100 questions, interpolated between the marks and rounded to a step of 5.
func scaled(sec Section, correct, total int) int {
	if total <= 0 {
		return 5
	}
	c100 := float64(correct) * 100 / float64(total)
	table := scaleTables[sec]
	score := float64(table[len(table)-1].score)
	for i := 1; i < len(table); i++ {
		lo, hi := table[i-1], table[i]
		if c100 <= float64(hi.right) {
			frac := (c100 - float64(lo.right)) / float64(hi.right-lo.right)
			score = float64(lo.score) + frac*float64(hi.score-lo.score)
			break
		}
	}
	out := int(math.Round(score/5)) * 5
	return min(max(out, 5), 495)
}

// sectionParts are the parts of each section.
var sectionParts = map[Section][]Part{Listening: {1, 2, 3, 4}, Reading: {5, 6, 7}}

// grade marks the counted questions of an attempt with its answers. The sections get an estimated
// score only for a whole test (Mode test or a random test) that has every part of the section; the
// 990 total needs both.
func grade(groups map[string]Group, a Attempt) Result {
	r := Result{ByQuestion: map[string]string{}, Version: map[string]int{}}
	type tally struct{ correct, total int }
	sec := map[Section]*tally{Listening: {}, Reading: {}}
	parts := map[Part]bool{}
	for _, id := range a.QuestionIDs {
		key, no, ok := SplitQuestionID(id)
		g, found := groups[key]
		if !ok || !found {
			continue
		}
		q, found := g.Question(no)
		if !found {
			continue
		}
		r.Version[key] = g.Version
		parts[g.Part] = true
		t := sec[SectionOf(g.Part)]
		t.total++
		choice, answered := a.Answers[id]
		switch {
		case !answered || choice < 0:
			r.ByQuestion[id] = OutcomeSkipped
			r.Skipped++
		case choice == q.Answer:
			r.ByQuestion[id] = OutcomeRight
			r.Correct++
			t.correct++
		default:
			r.ByQuestion[id] = OutcomeWrong
			r.Wrong++
		}
	}
	if a.Mode == ModeTest || a.Source.Kind == SourceRandom {
		for s, ps := range sectionParts {
			whole := true
			for _, p := range ps {
				whole = whole && parts[p]
			}
			if whole {
				t := sec[s]
				score := &SectionScore{Correct: t.correct, Total: t.total, Scaled: scaled(s, t.correct, t.total)}
				if s == Listening {
					r.Listening = score
				} else {
					r.Reading = score
				}
			}
		}
		if r.Listening != nil && r.Reading != nil {
			r.Total = r.Listening.Scaled + r.Reading.Scaled
		}
	}
	r.DurationMs = durationOf(a)
	return r
}

// fullParts reports whether parts has all seven.
func fullParts(parts []Part) bool {
	seen := map[Part]bool{}
	for _, p := range parts {
		seen[p] = true
	}
	return len(seen) == 7
}

// durationOf is the time spent on an attempt: what the clock counted, else what the limit lost.
func durationOf(a Attempt) int64 {
	if a.ElapsedMs > 0 {
		return a.ElapsedMs
	}
	if a.TimeLimitMs > 0 {
		return max(a.TimeLimitMs-a.RemainingMs, 0)
	}
	return 0
}
