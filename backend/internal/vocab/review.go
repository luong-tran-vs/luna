package vocab

import (
	"context"
	"fmt"
	"time"
)

const maxDueLimit = 200

// ReviewInput is one answer of the review session. Reps is the number of reviews the client
// saw on the card; a different stored count means the answer was already recorded.
type ReviewInput struct {
	Rating int
	Mode   string
	Reps   uint64
	// Context is "daily" (review step of the day's lesson) or "free" (default).
	Context string
}

// Due returns up to limit cards due now, most overdue first, with the interval of each rating.
func (s *Service) Due(ctx context.Context, userID string, limit int) (DueList, error) {
	if limit < 1 || limit > maxDueLimit {
		return DueList{}, &ValidationError{Fields: map[string]string{"limit": "Số thẻ cần từ 1 đến 200"}}
	}
	loc, err := s.location(ctx, userID)
	if err != nil {
		return DueList{}, err
	}
	now := s.now()
	today := startOfDay(now, loc)
	cards, total, err := s.repo.Due(ctx, userID, now, today, limit)
	if err != nil {
		return DueList{}, fmt.Errorf("vocab: due: %w", err)
	}
	list := DueList{Cards: make([]DueCard, len(cards)), Total: total}
	for i, c := range cards {
		c.Schedule = effective(c, loc)
		list.Cards[i] = DueCard{Card: c, Intervals: intervalsFor(c.Schedule, now)}
	}
	if next, ok, err := s.repo.NextDue(ctx, userID, now, today); err != nil {
		return DueList{}, fmt.Errorf("vocab: next due: %w", err)
	} else if ok {
		list.NextDue = next
	}
	return list, nil
}

// Review records rating for a card and reschedules it with FSRS. *ConflictError when the card
// was reviewed since the client loaded it (in.Reps no longer matches).
func (s *Service) Review(ctx context.Context, userID, cardID string, in ReviewInput) (DueCard, error) {
	fields := map[string]string{}
	rating := Rating(in.Rating)
	if rating < Again || rating > Easy {
		fields["rating"] = "Đánh giá không hợp lệ"
	}
	mode := Mode(in.Mode)
	if mode != ModeFlip && mode != ModeListen {
		fields["mode"] = "Kiểu ôn không hợp lệ"
	}
	context := Context(in.Context)
	switch context {
	case "":
		context = ContextFree
	case ContextDaily, ContextFree:
	default:
		fields["context"] = "Ngữ cảnh ôn không hợp lệ"
	}
	if len(fields) > 0 {
		return DueCard{}, &ValidationError{Fields: fields}
	}

	loc, err := s.location(ctx, userID)
	if err != nil {
		return DueCard{}, err
	}
	c, err := s.repo.Get(ctx, userID, cardID)
	if err != nil {
		return DueCard{}, fmt.Errorf("vocab: get card: %w", err)
	}
	now := s.now()
	before := effective(c, loc)
	if before.Reps != in.Reps {
		return DueCard{}, s.conflict(c, before, now)
	}

	after := next(before, now, rating)
	ok, err := s.repo.UpdateSchedule(ctx, userID, cardID, in.Reps, after)
	if err != nil {
		return DueCard{}, fmt.Errorf("vocab: update schedule: %w", err)
	}
	if !ok { // reviewed concurrently
		current, err := s.repo.Get(ctx, userID, cardID)
		if err != nil {
			return DueCard{}, fmt.Errorf("vocab: get card: %w", err)
		}
		return DueCard{}, s.conflict(current, effective(current, loc), now)
	}
	log := ReviewLog{UserID: userID, CardID: cardID, Rating: rating, Mode: mode, Context: context, ReviewedAt: now, Before: before}
	if err := s.logs.Add(ctx, log); err != nil {
		return DueCard{}, fmt.Errorf("vocab: add review log: %w", err)
	}
	c.Schedule = after
	return DueCard{Card: c, Intervals: intervalsFor(after, now)}, nil
}

func (s *Service) conflict(c Card, sched Schedule, now time.Time) error {
	c.Schedule = sched
	return &ConflictError{Card: DueCard{Card: c, Intervals: intervalsFor(sched, now)}}
}

// ReviewedSince counts userID's reviews in context c since t (the daily limit of L).
func (s *Service) ReviewedSince(ctx context.Context, userID string, c Context, since time.Time) (int, error) {
	n, err := s.logs.CountSince(ctx, userID, c, since)
	if err != nil {
		return 0, fmt.Errorf("vocab: count reviews: %w", err)
	}
	return n, nil
}

// DueBefore counts the cards due before `before` (overdue cards included); cards without a
// schedule count when saved before createdBefore.
func (s *Service) DueBefore(ctx context.Context, userID string, before, createdBefore time.Time) (int, error) {
	n, err := s.repo.CountDue(ctx, userID, before, createdBefore)
	if err != nil {
		return 0, fmt.Errorf("vocab: count due: %w", err)
	}
	return n, nil
}
