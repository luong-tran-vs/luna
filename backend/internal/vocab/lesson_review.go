package vocab

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Limits of a lesson review result: questions in one round and the length of a question id.
const (
	maxRoundQuestions = 100
	maxQuestionID     = 100
)

// LessonResult is the score of the latest lesson review round of one lesson: how many questions
// were right out of how many, and the ids of the ones got wrong (the page's own ids, e.g. "v:desk").
type LessonResult struct {
	UserID     string
	LessonID   string
	Correct    int
	Total      int
	Wrong      []string
	ReviewedAt time.Time
}

// LessonStats is the review state of one lesson, for the review tab's list of lessons
// (design/on-tap-theo-bai.png): its saved cards and the latest lesson review round.
type LessonStats struct {
	LessonID string
	Cards    int
	// Due counts the cards due now.
	Due int
	// Remembered counts the cards in the review stage (passed their learning steps).
	Remembered int
	// LastReview is the latest review of any of the cards; zero when none was reviewed.
	LastReview time.Time
	// NextDue is the earliest due time after now; zero when every card is due now.
	NextDue time.Time
	// Result is the latest lesson review round; nil when the lesson was never reviewed so.
	Result *LessonResult
}

// LessonStats gives, for every lesson userID has cards from or reviewed, how many cards are due
// and remembered, when they were last and are next reviewed, and the latest lesson review round.
// Cards added by hand are left out.
func (s *Service) LessonStats(ctx context.Context, userID string) ([]LessonStats, error) {
	loc, err := s.location(ctx, userID)
	if err != nil {
		return nil, err
	}
	cards, err := s.repo.Schedules(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("vocab: schedules: %w", err)
	}
	results, err := s.results.List(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("vocab: lesson results: %w", err)
	}
	now := s.now()
	byLesson := map[string]*LessonStats{}
	var order []string
	stats := func(id string) *LessonStats {
		st, ok := byLesson[id]
		if !ok {
			st = &LessonStats{LessonID: id}
			byLesson[id] = st
			order = append(order, id)
		}
		return st
	}
	for _, c := range cards {
		if c.LessonID == "" {
			continue
		}
		st := stats(c.LessonID)
		sched := effective(c, loc)
		st.Cards++
		if sched.State == StateReview {
			st.Remembered++
		}
		if sched.LastReview.After(st.LastReview) {
			st.LastReview = sched.LastReview
		}
		if sched.Due.After(now) {
			if st.NextDue.IsZero() || sched.Due.Before(st.NextDue) {
				st.NextDue = sched.Due
			}
		} else {
			st.Due++
		}
	}
	for _, r := range results {
		stats(r.LessonID).Result = &r
	}
	out := make([]LessonStats, len(order))
	for i, id := range order {
		out[i] = *byLesson[id]
	}
	return out, nil
}

// SaveLessonResult keeps the score of a lesson review round, replacing the previous one.
func (s *Service) SaveLessonResult(ctx context.Context, userID, lessonID string, correct, total int, wrong []string) (LessonResult, error) {
	fields := map[string]string{}
	if total < 1 || total > maxRoundQuestions {
		fields["total"] = fmt.Sprintf("Cần từ 1 đến %d câu", maxRoundQuestions)
	}
	if correct < 0 || correct > total {
		fields["correct"] = "Số câu đúng không hợp lệ"
	}
	ids := make([]string, 0, len(wrong))
	seen := map[string]bool{}
	for _, w := range wrong {
		w = strings.TrimSpace(w)
		if w == "" || len(w) > maxQuestionID {
			fields["wrong"] = "Mã câu không hợp lệ"
			break
		}
		if !seen[w] {
			seen[w] = true
			ids = append(ids, w)
		}
	}
	if len(ids) > total-correct && total >= correct {
		fields["wrong"] = "Số câu sai không khớp"
	}
	if len(fields) > 0 {
		return LessonResult{}, &ValidationError{Fields: fields}
	}
	ok, err := s.lessonExists(ctx, lessonID)
	if err != nil {
		return LessonResult{}, fmt.Errorf("vocab: lesson exists: %w", err)
	}
	if !ok {
		return LessonResult{}, ErrLessonNotFound
	}
	r := LessonResult{UserID: userID, LessonID: lessonID, Correct: correct, Total: total, Wrong: ids, ReviewedAt: s.now().UTC()}
	if err := s.results.Save(ctx, r); err != nil {
		return LessonResult{}, fmt.Errorf("vocab: save lesson result: %w", err)
	}
	return r, nil
}
