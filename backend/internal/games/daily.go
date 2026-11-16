package games

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"math/rand/v2"
	"slices"
	"time"
)

// "Thử thách hằng ngày" (P6, research R6): 10 short questions a day, the same all day for a learner
// (drawn from a seed of the learner and the day, so nothing is stored before the round), only the
// first round counted.

// Streak is the challenge streak: days in a row ending today (or yesterday when today is not played
// yet), and the longest run.
type Streak struct {
	Current int
	Longest int
}

// Compare is today's score against the learner's own: yesterday, the mean of the days played among
// the 7 before today, and the best before today.
type Compare struct {
	Yesterday *int
	Avg7      *float64
	Best      int
}

// DailyView is the daily challenge page.
type DailyView struct {
	Day       string
	Seed      int64
	Done      bool
	Streak    Streak
	Week      [7]bool // Monday to Sunday of this week
	Result    *DailyResult
	Compare   Compare
	Questions []DailyQuestion
}

// AlreadyDoneError carries the result already counted for the day.
type AlreadyDoneError struct{ Result DailyResult }

func (e *AlreadyDoneError) Error() string { return "games: daily challenge already done" }

// Is makes errors.Is(err, ErrAlreadyDone) true.
func (e *AlreadyDoneError) Is(target error) bool { return target == ErrAlreadyDone }

// today is the learner's date.
func (s *Service) today(ctx context.Context, userID string) (string, *time.Location, error) {
	loc, err := s.d.Timezones.Location(ctx, userID)
	if err != nil {
		return "", nil, fmt.Errorf("games: timezone: %w", err)
	}
	return dayKey(s.d.Now(), loc), loc, nil
}

// dailyHistory is the learner's date and every counted result, newest first.
func (s *Service) dailyHistory(ctx context.Context, userID string) (string, []DailyResult, error) {
	today, _, err := s.today(ctx, userID)
	if err != nil {
		return "", nil, err
	}
	results, err := s.d.Daily.List(ctx, userID)
	if err != nil {
		return "", nil, fmt.Errorf("games: daily results: %w", err)
	}
	slices.SortFunc(results, func(a, b DailyResult) int {
		switch {
		case a.Day > b.Day:
			return -1
		case a.Day < b.Day:
			return 1
		}
		return 0
	})
	return today, results, nil
}

func daysOf(results []DailyResult) []string {
	out := make([]string, len(results))
	for i, r := range results {
		out[i] = r.Day
	}
	return out
}

// addDays moves a YYYY-MM-DD date by n days.
func addDays(day string, n int) string {
	t, err := time.Parse(time.DateOnly, day)
	if err != nil {
		return ""
	}
	return t.AddDate(0, 0, n).Format(time.DateOnly)
}

// streakOf computes the streak from the days played.
func streakOf(days []string, today string) Streak {
	played := make(map[string]bool, len(days))
	for _, d := range days {
		played[d] = true
	}
	var s Streak
	start := today
	if !played[start] {
		start = addDays(today, -1)
	}
	for d := start; played[d]; d = addDays(d, -1) {
		s.Current++
	}
	sorted := slices.Clone(days)
	slices.Sort(sorted)
	sorted = slices.Compact(sorted)
	run := 0
	for i, d := range sorted {
		if i > 0 && addDays(sorted[i-1], 1) == d {
			run++
		} else {
			run = 1
		}
		s.Longest = max(s.Longest, run)
	}
	return s
}

// weekOf marks the days played this week, Monday first.
func weekOf(days []string, today string) [7]bool {
	var w [7]bool
	t, err := time.Parse(time.DateOnly, today)
	if err != nil {
		return w
	}
	monday := t.AddDate(0, 0, -((int(t.Weekday()) + 6) % 7))
	played := make(map[string]bool, len(days))
	for _, d := range days {
		played[d] = true
	}
	for i := range w {
		w[i] = played[monday.AddDate(0, 0, i).Format(time.DateOnly)]
	}
	return w
}

// compareOf compares with the days before today.
func compareOf(results []DailyResult, today string) Compare {
	var c Compare
	yesterday, weekAgo := addDays(today, -1), addDays(today, -7)
	sum, n := 0, 0
	for _, r := range results {
		if r.Day >= today {
			continue
		}
		if r.Day == yesterday {
			score := r.Score
			c.Yesterday = &score
		}
		if r.Day >= weekAgo {
			sum += r.Score
			n++
		}
		c.Best = max(c.Best, r.Score)
	}
	if n > 0 {
		avg := float64(sum) / float64(n)
		c.Avg7 = &avg
	}
	return c
}

// seedOf is the seed of a learner's day.
func seedOf(userID, day string) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(userID + "|" + day))
	return h.Sum64()
}

// Daily returns today's challenge (FR-027): the streak, the week, the comparisons, the result when
// it is done, and the 10 questions (still given once done, to play again for practice).
func (s *Service) Daily(ctx context.Context, userID string, isGuest bool) (DailyView, error) {
	today, results, err := s.dailyHistory(ctx, userID)
	if err != nil {
		return DailyView{}, err
	}
	days := daysOf(results)
	seed := seedOf(userID, today)
	v := DailyView{
		Day: today, Seed: int64(seed >> 33), Streak: streakOf(days, today), Week: weekOf(days, today),
		Compare: compareOf(results, today),
	}
	if len(results) > 0 && results[0].Day == today {
		r := results[0]
		v.Done, v.Result = true, &r
	}
	if v.Questions, err = s.dailyQuestions(ctx, userID, isGuest, rand.New(rand.NewPCG(seed, seed^0x5bd1e995))); err != nil {
		return DailyView{}, err
	}
	return v, nil
}

// dailyQuestions draws the day's 10 questions: 3 pictures, 3 verbs, 2 spellings, 2 grammar (guests:
// 4, 3, 3, no grammar). A kind without enough content hands its share to the next one; verbs always
// have enough (an empty base lets the page pick a common verb).
func (s *Service) dailyQuestions(ctx context.Context, userID string, isGuest bool, rnd *rand.Rand) ([]DailyQuestion, error) {
	want := map[string]int{KindPicture: 3, KindLetters: 2, KindGrammar: 2, KindVerb: 3}
	if isGuest {
		want = map[string]int{KindPicture: 4, KindLetters: 3, KindGrammar: 0, KindVerb: 3}
	}
	var out []DailyQuestion
	short := 0

	pictures, err := s.pictureQuestions(ctx, userID, want[KindPicture], rnd)
	if err != nil && !errors.Is(err, ErrNotEnough) {
		return nil, err
	}
	shown := map[string]bool{}
	for i := range pictures {
		if shown[pictures[i].Lemma] {
			continue
		}
		shown[pictures[i].Lemma] = true
		out = append(out, DailyQuestion{Kind: KindPicture, Picture: &pictures[i]})
	}
	short += want[KindPicture] - len(out)

	n := want[KindLetters] + short
	letters, err := s.letterWords(ctx, userID, n+len(shown), 1, rnd)
	if err != nil && !errors.Is(err, ErrNotEnough) {
		return nil, err
	}
	got := 0
	for i := range letters {
		if got == n {
			break
		}
		if shown[letters[i].Lemma] {
			continue
		}
		out = append(out, DailyQuestion{Kind: KindLetters, Letters: &letters[i]})
		got++
	}
	short = n - got

	if n = want[KindGrammar] + short; n > 0 && want[KindGrammar] > 0 {
		got = 0
		for try := n; try > 0 && got == 0; try-- {
			spots, err := s.spotQuestions(ctx, userID, try, try/3, rnd)
			if errors.Is(err, ErrNotEnough) {
				continue
			}
			if err != nil {
				return nil, err
			}
			for i := range spots {
				out = append(out, DailyQuestion{Kind: KindGrammar, Grammar: &spots[i]})
			}
			got = len(spots)
		}
		short = n - got
	}

	verbs, err := s.dailyVerbs(ctx, userID, want[KindVerb]+short, rnd)
	if err != nil {
		return nil, err
	}
	out = append(out, verbs...)
	shuffle(rnd, out)
	return out, nil
}

// dailyVerbs picks n learned verbs, due ones first; missing ones have an empty base.
func (s *Service) dailyVerbs(ctx context.Context, userID string, n int, rnd *rand.Rand) ([]DailyQuestion, error) {
	due, other, err := s.d.Verbs.LearnedBases(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("games: learned verbs: %w", err)
	}
	due, other = slices.Clone(due), slices.Clone(other)
	shuffle(rnd, due)
	shuffle(rnd, other)
	bases := append(due, other...)
	out := make([]DailyQuestion, n)
	for i := range out {
		v := &DailyVerb{}
		if i < len(bases) {
			v.Base = bases[i]
		}
		out[i] = DailyQuestion{Kind: KindVerb, Verb: v}
	}
	return out, nil
}

// DailyBody is a finished daily round.
type DailyBody struct {
	Day        string
	Score      int
	Answers    []DailyAnswer
	WrongWords []string
	WrongVerbs []VerbMiss
}

// DailyReply is the streak and comparisons after the round, and the cards brought forward.
type DailyReply struct {
	Streak      Streak
	Compare     Compare
	Rescheduled int
}

// SaveDaily counts the first round of a day (FR-028): day is today or yesterday (a round started
// before midnight), the result is stored, today becomes a study day, and the missed words and
// verbs come back to the review queue. A day already counted is an *AlreadyDoneError.
func (s *Service) SaveDaily(ctx context.Context, userID string, b DailyBody) (DailyReply, error) {
	today, _, err := s.today(ctx, userID)
	if err != nil {
		return DailyReply{}, err
	}
	if b.Day != today && b.Day != addDays(today, -1) {
		return DailyReply{}, invalid("day", "Ngày không hợp lệ")
	}
	if len(b.Answers) != DailyCount {
		return DailyReply{}, invalid("answers", fmt.Sprintf("Cần đúng %d câu trả lời", DailyCount))
	}
	for _, a := range b.Answers {
		if !validKind(a.Kind) {
			return DailyReply{}, invalid("answers", "Loại câu hỏi không hợp lệ")
		}
	}
	if err := checkScore(GameDaily, b.Score); err != nil {
		return DailyReply{}, err
	}
	if err := checkMisses(b.WrongWords, b.WrongVerbs); err != nil {
		return DailyReply{}, err
	}
	r := DailyResult{UserID: userID, Day: b.Day, Score: b.Score, Answers: b.Answers, FinishedAt: s.d.Now().UTC()}
	if err := s.d.Daily.Save(ctx, r); errors.Is(err, ErrExists) {
		saved, gerr := s.d.Daily.Get(ctx, userID, b.Day)
		if gerr != nil {
			return DailyReply{}, fmt.Errorf("games: daily result: %w", gerr)
		}
		return DailyReply{}, &AlreadyDoneError{Result: saved}
	} else if err != nil {
		return DailyReply{}, fmt.Errorf("games: save daily result: %w", err)
	}
	if s.d.StudyDays != nil {
		if err := s.d.StudyDays.MarkStudied(ctx, userID); err != nil {
			return DailyReply{}, fmt.Errorf("games: study day: %w", err)
		}
	}
	moved, err := s.bringBack(ctx, userID, b.WrongWords, b.WrongVerbs)
	if err != nil {
		return DailyReply{}, err
	}
	today, results, err := s.dailyHistory(ctx, userID)
	if err != nil {
		return DailyReply{}, err
	}
	return DailyReply{Streak: streakOf(daysOf(results), today), Compare: compareOf(results, today), Rescheduled: moved}, nil
}

// DailyPractice takes a round played again for practice: only the misses count.
func (s *Service) DailyPractice(ctx context.Context, userID string, words []string, verbs []VerbMiss) (int, error) {
	if err := checkMisses(words, verbs); err != nil {
		return 0, err
	}
	return s.bringBack(ctx, userID, words, verbs)
}
