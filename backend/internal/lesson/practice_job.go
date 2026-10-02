package lesson

import (
	"context"
	"errors"
	"fmt"

	"github.com/luongtran/luna/backend/internal/ai"
	"github.com/luongtran/luna/backend/internal/job"
)

// ProcessPractice asks the AI provider once for the vocabulary practice of a lesson (F17),
// and keeps the parts that pass the checks. Nothing usable is a permanent
// failure, so a generation never costs more than one successful AI request.
func (s *Service) ProcessPractice(ctx context.Context, j job.Job) error {
	l, ok, err := s.current(ctx, j)
	if !ok {
		return err
	}
	if l.AnnotationStatus != StatusDone || len(l.Annotations) == 0 {
		return job.Permanent(ErrAnnotationNotDone)
	}
	words := practiceWords(l)
	res, err := s.AI.Practice(ctx, ai.PracticeRequest{
		Level: string(l.Level), Title: l.Title, Sentences: sentenceTexts(l.Sentences), Words: words,
	})
	if err != nil {
		if errors.Is(err, ai.ErrNotConfigured) || errors.Is(err, ai.ErrInvalidKey) {
			return job.Permanent(err)
		}
		return err
	}
	p, err := CleanPractice(res, words)
	if err != nil {
		return job.Permanent(err)
	}
	// Not saved when the content or practice changed meanwhile: then there is nothing to do.
	if _, err := s.Lessons.SavePractice(ctx, l.ID, l.Revision, l.PracticeVersion, p); err != nil {
		return fmt.Errorf("lesson: save practice: %w", err)
	}
	return nil
}

// RegeneratePractice queues a new practice for a lesson whose annotations are done (F17). The
// current practice stays visible until the new one is saved.
func (s *Service) RegeneratePractice(ctx context.Context, id string) (Lesson, error) {
	l, err := s.Lessons.Get(ctx, id)
	if err != nil {
		return Lesson{}, err
	}
	if l.AnnotationStatus != StatusDone || len(l.Annotations) == 0 {
		return Lesson{}, ErrAnnotationNotDone
	}
	if l.PracticeStatus == StatusRunning {
		return Lesson{}, ErrPracticeRunning
	}
	if _, err := s.Lessons.SetStatus(ctx, id, l.Revision, job.TypePractice, StatusRunning, ""); err != nil {
		return Lesson{}, fmt.Errorf("lesson: set practice status: %w", err)
	}
	if err := s.enqueue(ctx, id, l.Revision, job.TypePractice); err != nil {
		return Lesson{}, err
	}
	return s.Lessons.Get(ctx, id)
}

// QueueMissingPractice queues a practice for every lesson annotated before F17, which never got
// one, and returns how many were queued. Each lesson is queued once: its practice status leaves
// StatusNone, so a failed generation is not queued again here (the admin can regenerate it).
func (s *Service) QueueMissingPractice(ctx context.Context) (int, error) {
	refs, err := s.Lessons.WithoutPractice(ctx)
	if err != nil {
		return 0, fmt.Errorf("lesson: lessons without practice: %w", err)
	}
	queued := 0
	for _, r := range refs {
		ok, err := s.Lessons.SetStatus(ctx, r.ID, r.Revision, job.TypePractice, StatusRunning, "")
		if err != nil {
			return queued, fmt.Errorf("lesson: set practice status: %w", err)
		}
		if !ok { // edited meanwhile: the new revision gets its own practice
			continue
		}
		if err := s.enqueue(ctx, r.ID, r.Revision, job.TypePractice); err != nil {
			return queued, err
		}
		queued++
	}
	return queued, nil
}
