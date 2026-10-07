package vocab

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

// Input is a card as submitted by the reading popup.
type Input struct {
	Text            string
	Lemma           string
	IPA             string
	MeaningVi       string
	ContextSentence string
	LessonID        string
	Source          string
}

// Deps are the collaborators of Service.
type Deps struct {
	Repo Repository
	Logs ReviewLogRepository
	// LessonExists checks the source lesson of a card.
	LessonExists func(ctx context.Context, id string) (bool, error)
	Timezones    Timezones
	Vocabulary   LessonVocabulary
	Titles       LessonTitles
	// Pronunciations fills the missing IPA of due cards; nil leaves them as saved.
	Pronunciations Pronunciations
	Now            func() time.Time
}

// Service manages notebook cards and their reviews. The user is always the session user.
type Service struct {
	repo           Repository
	logs           ReviewLogRepository
	lessonExists   func(ctx context.Context, id string) (bool, error)
	timezones      Timezones
	vocabulary     LessonVocabulary
	titles         LessonTitles
	pronunciations Pronunciations
	now            func() time.Time
}

// NewService returns a Service.
func NewService(d Deps) *Service {
	return &Service{
		repo: d.Repo, logs: d.Logs, lessonExists: d.LessonExists, timezones: d.Timezones,
		vocabulary: d.Vocabulary, titles: d.Titles, pronunciations: d.Pronunciations, now: d.Now,
	}
}

func (s *Service) location(ctx context.Context, userID string) (*time.Location, error) {
	loc, err := s.timezones.Location(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("vocab: timezone: %w", err)
	}
	return loc, nil
}

// Save adds a card to userID's notebook, first due the next day in the learner's timezone.
// A card with the same lemma (case- and space-insensitive) is never duplicated:
// *ExistsError carries the saved one. Only manual cards may have no lesson.
func (s *Service) Save(ctx context.Context, userID string, in Input) (Card, error) {
	loc, err := s.location(ctx, userID)
	if err != nil {
		return Card{}, err
	}
	c := Card{
		UserID:          userID,
		Text:            collapse(in.Text),
		Lemma:           strings.ToLower(collapse(in.Lemma)),
		IPA:             strings.TrimSpace(in.IPA),
		MeaningVi:       strings.TrimSpace(in.MeaningVi),
		ContextSentence: strings.TrimSpace(in.ContextSentence),
		LessonID:        strings.TrimSpace(in.LessonID),
		Source:          Source(in.Source),
		CreatedAt:       s.now(),
	}
	c.Schedule = Schedule{Due: FirstDue(c.CreatedAt, loc), State: StateNew}
	if err := s.validate(ctx, c); err != nil {
		return Card{}, err
	}

	saved, err := s.repo.Create(ctx, c)
	if errors.Is(err, ErrExists) {
		existing, ferr := s.repo.FindByLemma(ctx, userID, c.Lemma)
		if ferr != nil {
			return Card{}, fmt.Errorf("vocab: find existing: %w", ferr)
		}
		return Card{}, &ExistsError{Card: existing}
	}
	if err != nil {
		return Card{}, fmt.Errorf("vocab: create: %w", err)
	}
	return saved, nil
}

// Words lists userID's saved words.
func (s *Service) Words(ctx context.Context, userID string) ([]WordRef, error) {
	words, err := s.repo.Words(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("vocab: words: %w", err)
	}
	return words, nil
}

func (s *Service) validate(ctx context.Context, c Card) error {
	fields := map[string]string{}
	lengthBetween(fields, "text", c.Text, 1, 100, "Từ cần lưu không hợp lệ")
	lengthBetween(fields, "lemma", c.Lemma, 1, 100, "Dạng gốc không hợp lệ")
	lengthBetween(fields, "meaningVi", c.MeaningVi, 1, 200, "Nghĩa cần từ 1 đến 200 ký tự")
	lengthBetween(fields, "contextSentence", c.ContextSentence, 0, 1000, "Câu ngữ cảnh tối đa 1000 ký tự")
	if utf8.RuneCountInString(c.IPA) > 100 {
		fields["ipa"] = "Phiên âm tối đa 100 ký tự"
	}
	switch c.Source {
	case SourceAI, SourceDictionary, SourceManual:
	default:
		fields["source"] = "Nguồn nghĩa không hợp lệ"
	}
	switch {
	case c.LessonID == "" && c.Source != SourceManual:
		fields["lessonId"] = "Thiếu bài học của thẻ"
	case c.LessonID != "":
		if ok, err := s.lessonExists(ctx, c.LessonID); err != nil {
			return fmt.Errorf("vocab: check lesson: %w", err)
		} else if !ok {
			fields["lessonId"] = "Bài học không tồn tại"
		}
	}
	if len(fields) > 0 {
		return &ValidationError{Fields: fields}
	}
	return nil
}

func lengthBetween(fields map[string]string, key, v string, minLen, maxLen int, msg string) {
	if n := utf8.RuneCountInString(v); n < minLen || n > maxLen {
		fields[key] = msg
	}
}

func collapse(s string) string { return strings.Join(strings.Fields(s), " ") }
