package vocab

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"
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
	cache, kinds := sentenceCache{}, posCache{}
	for i, c := range cards {
		c.IPA = s.fillIPA(ctx, userID, c)
		c.POS = s.posOf(ctx, kinds, c)
		c.ContextSentenceVi = s.fillSentenceVi(ctx, userID, cache, c)
		c.Schedule = effective(c, loc)
		page.Cards[i] = DayCard{Card: c, Day: dayOf(c.CreatedAt, loc)}
	}
	s.fillDayPictures(ctx, page.Cards)
	return page, nil
}

// fillDayPictures gives the notebook's cards the word bank picture of their word (added
// 2026-11-16), as the review does; without them the cards are shown as before.
func (s *Service) fillDayPictures(ctx context.Context, cards []DayCard) {
	if s.pictures == nil || len(cards) == 0 {
		return
	}
	lemmas := make([]string, len(cards))
	for i, c := range cards {
		lemmas[i] = pictureLemma(c.Card)
	}
	urls, err := s.pictures.ImageURLs(ctx, lemmas)
	if err != nil {
		return
	}
	for i := range cards {
		cards[i].ImageURL = urls[lemmas[i]]
	}
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
	trim(d.ContextSentenceVi)
	if d.MeaningVi != nil {
		lengthBetween(fields, "meaningVi", *d.MeaningVi, 1, 200, "Nghĩa cần từ 1 đến 200 ký tự")
	}
	if d.IPA != nil {
		lengthBetween(fields, "ipa", *d.IPA, 0, 100, "Phiên âm tối đa 100 ký tự")
	}
	if d.ContextSentence != nil {
		lengthBetween(fields, "contextSentence", *d.ContextSentence, 0, 1000, "Câu ví dụ tối đa 1000 ký tự")
	}
	if d.ContextSentenceVi != nil {
		lengthBetween(fields, "contextSentenceVi", *d.ContextSentenceVi, 0, 1000, "Nghĩa câu ví dụ tối đa 1000 ký tự")
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

// Count is the number of cards in the user's notebook created at or after since (nil = all).
func (s *Service) Count(ctx context.Context, userID string, since *time.Time) (int, error) {
	n, err := s.repo.Count(ctx, userID, since)
	if err != nil {
		return 0, fmt.Errorf("vocab: count cards: %w", err)
	}
	return n, nil
}

// PictureWord is a word of the notebook with its word bank picture.
type PictureWord struct {
	Text      string
	MeaningVi string
	ImageURL  string
}

// lessonPicturesScan bounds the cards read from each lesson for LessonPictures.
const lessonPicturesScan = 100

// LessonPictures lists, at most limit, the words userID saved from lessonIDs whose word has a word
// bank picture, one per word, lesson by lesson (added 2026-11-16, the topic summary's "Từ đã học").
func (s *Service) LessonPictures(ctx context.Context, userID string, lessonIDs []string, limit int) ([]PictureWord, error) {
	out := []PictureWord{}
	if s.pictures == nil || limit <= 0 {
		return out, nil
	}
	var cards []Card
	for _, id := range lessonIDs {
		cs, _, err := s.repo.List(ctx, userID, ListQuery{LessonID: id}, lessonPicturesScan, 0)
		if err != nil {
			return nil, fmt.Errorf("vocab: list lesson cards: %w", err)
		}
		cards = append(cards, cs...)
	}
	lemmas := make([]string, len(cards))
	for i, c := range cards {
		lemmas[i] = pictureLemma(c)
	}
	urls, err := s.pictures.ImageURLs(ctx, lemmas)
	if err != nil {
		return nil, fmt.Errorf("vocab: pictures: %w", err)
	}
	seen := map[string]bool{}
	for i, c := range cards {
		u := urls[lemmas[i]]
		if u == "" || seen[lemmas[i]] {
			continue
		}
		seen[lemmas[i]] = true
		out = append(out, PictureWord{Text: c.Text, MeaningVi: c.MeaningVi, ImageURL: u})
		if len(out) == limit {
			break
		}
	}
	return out, nil
}
