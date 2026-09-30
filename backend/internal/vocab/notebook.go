package vocab

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"
)

// pageSize is the number of cards per notebook page.
const pageSize = 30

// List returns one page of userID's notebook, newest first, each card with the day it was
// saved in the learner's timezone.
func (s *Service) List(ctx context.Context, userID string, q ListQuery) (Page, error) {
	q.Q = strings.TrimSpace(q.Q)
	q.LessonID = strings.TrimSpace(q.LessonID)
	fields := map[string]string{}
	if q.Page < 1 {
		fields["page"] = "Trang không hợp lệ"
	}
	if utf8.RuneCountInString(q.Q) > 100 {
		fields["q"] = "Từ cần tìm tối đa 100 ký tự"
	}
	if len(fields) > 0 {
		return Page{}, &ValidationError{Fields: fields}
	}

	loc, err := s.location(ctx, userID)
	if err != nil {
		return Page{}, err
	}
	cards, more, err := s.repo.List(ctx, userID, q, pageSize, (q.Page-1)*pageSize)
	if err != nil {
		return Page{}, fmt.Errorf("vocab: list: %w", err)
	}
	today := startOfDay(s.now(), loc)
	page := Page{
		Cards:     make([]DayCard, len(cards)),
		HasMore:   more,
		Today:     dayOf(today, loc),
		Yesterday: dayOf(today.AddDate(0, 0, -1), loc),
	}
	for i, c := range cards {
		c.Schedule = effective(c, loc)
		page.Cards[i] = DayCard{Card: c, Day: dayOf(c.CreatedAt, loc)}
	}
	return page, nil
}

// Lessons lists the lessons that still exist and have cards in userID's notebook, by title,
// and how many cards were added by hand.
func (s *Service) Lessons(ctx context.Context, userID string) ([]LessonCount, int, error) {
	counts, err := s.repo.LessonCounts(ctx, userID)
	if err != nil {
		return nil, 0, fmt.Errorf("vocab: lesson counts: %w", err)
	}
	ids := make([]string, 0, len(counts))
	for id := range counts {
		if id != "" {
			ids = append(ids, id)
		}
	}
	titles, err := s.titles.Titles(ctx, ids)
	if err != nil {
		return nil, 0, fmt.Errorf("vocab: lesson titles: %w", err)
	}
	lessons := make([]LessonCount, 0, len(titles))
	for id, title := range titles {
		lessons = append(lessons, LessonCount{ID: id, Title: title, Count: counts[id]})
	}
	slices.SortFunc(lessons, func(a, b LessonCount) int { return strings.Compare(a.Title, b.Title) })
	return lessons, counts[""], nil
}

// Update edits the meaning, IPA or example of a card. The word and the schedule never change.
func (s *Service) Update(ctx context.Context, userID, cardID string, d Details) (Card, error) {
	fields := map[string]string{}
	trim := func(p *string) {
		if p != nil {
			*p = strings.TrimSpace(*p)
		}
	}
	trim(d.MeaningVi)
	trim(d.IPA)
	trim(d.ContextSentence)
	if d.MeaningVi != nil {
		lengthBetween(fields, "meaningVi", *d.MeaningVi, 1, 200, "Nghĩa cần từ 1 đến 200 ký tự")
	}
	if d.IPA != nil {
		lengthBetween(fields, "ipa", *d.IPA, 0, 100, "Phiên âm tối đa 100 ký tự")
	}
	if d.ContextSentence != nil {
		lengthBetween(fields, "contextSentence", *d.ContextSentence, 0, 1000, "Câu ví dụ tối đa 1000 ký tự")
	}
	if len(fields) > 0 {
		return Card{}, &ValidationError{Fields: fields}
	}

	loc, err := s.location(ctx, userID)
	if err != nil {
		return Card{}, err
	}
	c, err := s.repo.UpdateDetails(ctx, userID, cardID, d)
	if err != nil {
		return Card{}, fmt.Errorf("vocab: update card: %w", err)
	}
	c.Schedule = effective(c, loc)
	return c, nil
}

// Delete removes a card and its review history. Calling it again cleans up history left by
// an interrupted delete; ErrNotFound when there was neither card nor history.
func (s *Service) Delete(ctx context.Context, userID, cardID string) error {
	deleted, err := s.repo.Delete(ctx, userID, cardID)
	if err != nil {
		return fmt.Errorf("vocab: delete card: %w", err)
	}
	logs, err := s.logs.DeleteByCard(ctx, userID, cardID)
	if err != nil {
		return fmt.Errorf("vocab: delete review logs: %w", err)
	}
	if !deleted && logs == 0 {
		return ErrNotFound
	}
	return nil
}

// Count is the number of cards in the user's notebook.
func (s *Service) Count(ctx context.Context, userID string) (int, error) {
	n, err := s.repo.Count(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("vocab: count cards: %w", err)
	}
	return n, nil
}
