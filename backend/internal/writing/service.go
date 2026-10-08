package writing

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/luongtran/luna/backend/internal/job"
)

// Deps are the collaborators of Service.
type Deps struct {
	Repo    Repository
	Lessons Lessons
	Steps   Steps
	Jobs    Jobs
	AI      Grader
	// Notify wakes the job worker after a job is queued; may be nil.
	Notify func()
	Now    func() time.Time
	Log    *slog.Logger
}

// Service implements the Write step and the learner's writings.
type Service struct {
	d Deps
}

// NewService returns a Service. Notify and Log default to no-ops.
func NewService(d Deps) *Service {
	if d.Notify == nil {
		d.Notify = func() {}
	}
	if d.Log == nil {
		d.Log = slog.New(slog.DiscardHandler)
	}
	return &Service{d: d}
}

func countWords(s string) int { return len(strings.Fields(s)) }

// checkLength trims a writing and checks its length (5–400 words).
func checkLength(text string) (string, error) {
	text = strings.TrimSpace(text)
	switch n := countWords(text); {
	case n < MinWords:
		return "", &ValidationError{Fields: map[string]string{"text": "Bài viết cần ít nhất 5 từ"}}
	case n > MaxWords || utf8.RuneCountInString(text) > maxChars:
		return "", &ValidationError{Fields: map[string]string{"text": "Bài viết tối đa 400 từ"}}
	}
	return text, nil
}

// Get returns the Write step of a lesson: the prompt, the level, whether the learner can write
// now, and their writing (draft or submitted) if any.
func (s *Service) Get(ctx context.Context, userID, lessonID string) (LessonView, error) {
	info, err := s.d.Lessons.Info(ctx, lessonID)
	if err != nil {
		return LessonView{}, err
	}
	v := LessonView{Prompt: promptOf(info), Level: info.Level}
	w, ok, err := s.d.Repo.GetByLesson(ctx, userID, lessonID)
	if err != nil {
		return LessonView{}, fmt.Errorf("writing: get: %w", err)
	}
	if ok {
		v.Writing = &w
		if w.Status == StatusSubmitted {
			v.Prompt = w.Prompt
		}
	}
	if !ok || w.Status != StatusSubmitted {
		if v.CanWrite, err = s.d.Steps.CanWrite(ctx, userID, lessonID); err != nil {
			return LessonView{}, fmt.Errorf("writing: can write: %w", err)
		}
	}
	return v, nil
}

func promptOf(info LessonInfo) string {
	if p := strings.TrimSpace(info.WritingPrompt); p != "" {
		return p
	}
	return DefaultPrompt
}

// canWrite returns ErrLocked unless the lesson is today's lesson at the Write step.
// A submitted writing reports ErrSubmitted first, even once the lesson is over.
func (s *Service) canWrite(ctx context.Context, userID, lessonID string) error {
	if done, err := s.Submitted(ctx, userID, lessonID); err != nil {
		return err
	} else if done {
		return ErrSubmitted
	}
	ok, err := s.d.Steps.CanWrite(ctx, userID, lessonID)
	if err != nil {
		return fmt.Errorf("writing: can write: %w", err)
	}
	if !ok {
		return ErrLocked
	}
	return nil
}

// SaveDraft stores the draft text while the learner types.
func (s *Service) SaveDraft(ctx context.Context, userID, lessonID, text string) (Writing, error) {
	if utf8.RuneCountInString(text) > maxChars {
		return Writing{}, &ValidationError{Fields: map[string]string{"text": "Bài viết quá dài"}}
	}
	if err := s.canWrite(ctx, userID, lessonID); err != nil {
		return Writing{}, err
	}
	w, err := s.d.Repo.SaveDraft(ctx, userID, lessonID, text, s.d.Now().UTC())
	if err != nil && !errors.Is(err, ErrSubmitted) {
		return Writing{}, fmt.Errorf("writing: save draft: %w", err)
	}
	return w, err
}

// Submit checks the length (5–400 words), stores the writing with the lesson's prompt as it
// is now, and queues the grading. A writing is submitted once.
func (s *Service) Submit(ctx context.Context, userID, lessonID, text string) (Writing, error) {
	text, err := checkLength(text)
	if err != nil {
		return Writing{}, err
	}
	info, err := s.d.Lessons.Info(ctx, lessonID)
	if err != nil {
		return Writing{}, err
	}
	if err := s.canWrite(ctx, userID, lessonID); err != nil {
		return Writing{}, err
	}
	w, err := s.d.Repo.Submit(ctx, Writing{
		UserID: userID, LessonID: lessonID, LessonRevision: info.Revision, LessonTitle: info.Title,
		Prompt: promptOf(info), Text: text, SubmittedAt: s.d.Now().UTC(), Gradings: 1,
	})
	if errors.Is(err, ErrSubmitted) {
		return Writing{}, err
	}
	if err != nil {
		return Writing{}, fmt.Errorf("writing: submit: %w", err)
	}
	if err := s.enqueue(ctx, w.ID); err != nil {
		return Writing{}, err
	}
	return w, nil
}

func (s *Service) enqueue(ctx context.Context, writingID string) error {
	if err := s.d.Jobs.Enqueue(ctx, job.Job{Type: job.TypeGrade, TargetID: writingID, RunAt: s.d.Now()}); err != nil {
		return fmt.Errorf("writing: queue grade: %w", err)
	}
	s.d.Notify()
	return nil
}

// Submitted reports whether the learner submitted the writing of the lesson (progress port).
func (s *Service) Submitted(ctx context.Context, userID, lessonID string) (bool, error) {
	w, ok, err := s.d.Repo.GetByLesson(ctx, userID, lessonID)
	if err != nil {
		return false, fmt.Errorf("writing: get: %w", err)
	}
	return ok && w.Status == StatusSubmitted, nil
}

// Stats counts the learner's writings submitted since `since` (nil = all) and averages the graded
// ones among them (progress port).
func (s *Service) Stats(ctx context.Context, userID string, since *time.Time) (submitted int, average *float64, err error) {
	submitted, average, err = s.d.Repo.Stats(ctx, userID, since)
	if err != nil {
		return 0, nil, fmt.Errorf("writing: stats: %w", err)
	}
	return submitted, average, nil
}

// own returns the learner's writing; another learner's writing is ErrNotFound.
func (s *Service) own(ctx context.Context, userID, id string) (Writing, error) {
	w, err := s.d.Repo.Get(ctx, id)
	if err != nil {
		return Writing{}, err
	}
	if w.UserID != userID {
		return Writing{}, ErrNotFound
	}
	return w, nil
}

// List returns the learner's submitted writings, newest first.
func (s *Service) List(ctx context.Context, userID string) ([]Summary, error) {
	list, err := s.d.Repo.List(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("writing: list: %w", err)
	}
	out := make([]Summary, len(list))
	for i, w := range list {
		out[i] = Summary{ID: w.ID, LessonID: w.LessonID, LessonTitle: w.LessonTitle, SubmittedAt: w.SubmittedAt}
		if w.Grade != nil {
			out[i].GradeStatus, out[i].Average, out[i].Seen = w.Grade.Status, Average(w.Grade), w.Grade.Seen
		}
	}
	return out, nil
}

// Detail returns one of the learner's writings.
func (s *Service) Detail(ctx context.Context, userID, id string) (Writing, error) {
	return s.own(ctx, userID, id)
}

// MarkSeen marks the result of one of the learner's writings as seen.
func (s *Service) MarkSeen(ctx context.Context, userID, id string) error {
	if _, err := s.own(ctx, userID, id); err != nil {
		return err
	}
	if err := s.d.Repo.MarkSeen(ctx, id); err != nil {
		return fmt.Errorf("writing: mark seen: %w", err)
	}
	return nil
}

// Unseen counts the learner's new results and gradings in progress.
func (s *Service) Unseen(ctx context.Context, userID string) (UnseenCount, error) {
	c, err := s.d.Repo.Unseen(ctx, userID)
	if err != nil {
		return UnseenCount{}, fmt.Errorf("writing: unseen: %w", err)
	}
	return c, nil
}
