package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/luongtran/luna/backend/internal/lesson"
	"github.com/luongtran/luna/backend/internal/progress"
	"github.com/luongtran/luna/backend/internal/writing"
)

// initWriting builds the writing service: the Write step and its AI grading (F8). It needs the
// study service (the Write step) and the study service needs it: writeSteps is handed over now
// and given the study service by initProgress.
func (c *Container) initWriting() {
	c.writeSteps = &writingSteps{}
	c.writing = writing.NewService(writing.Deps{
		Repo:    c.store.Writings(),
		Lessons: writingLessons{c.lessons},
		Steps:   c.writeSteps,
		Jobs:    c.jobs,
		AI:      c.ai,
		Notify:  func() { c.worker.Notify() },
		Now:     time.Now,
		Log:     c.log,
	})
}

// writingLessons adapts the lesson repository to writing.Lessons (F8).
type writingLessons struct {
	repo lesson.Repository
}

func (w writingLessons) Info(ctx context.Context, id string) (writing.LessonInfo, error) {
	l, err := w.repo.Get(ctx, id)
	if errors.Is(err, lesson.ErrNotFound) {
		return writing.LessonInfo{}, writing.ErrNotFound
	}
	if err != nil {
		return writing.LessonInfo{}, fmt.Errorf("get lesson: %w", err)
	}
	return writing.LessonInfo{
		Title: l.Title, Level: string(l.Level), Content: l.Content, WritingPrompt: l.Extras.WritingPrompt, Revision: l.Revision,
	}, nil
}

// writingSteps adapts progress.StudyService to writing.Steps; svc is set once it exists.
type writingSteps struct {
	svc *progress.StudyService
}

func (w *writingSteps) CanWrite(ctx context.Context, userID, lessonID string) (bool, error) {
	return w.svc.CanWrite(ctx, userID, lessonID)
}
