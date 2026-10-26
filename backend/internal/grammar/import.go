package grammar

import (
	"context"
	"errors"
	"fmt"
)

// ErrHasLesson means the point already has a lesson, which an import replaces only when asked
// (added 2026-10-26).
var ErrHasLesson = errors.New("grammar: point has a lesson")

// Import saves a lesson written outside the app, by an admin or an agent (added 2026-10-26), as a
// draft of the point: no AI is called. It passes the checks of an admin's edit (Normalize,
// Validate) and is marked edited, so a new generation asks first. A point that has a lesson gets
// ErrHasLesson unless replace, which puts the new lesson back to draft. dryRun only checks it.
// replaced tells whether a lesson was there.
func (s *Service) Import(ctx context.Context, pointID string, content Content, replace, dryRun bool) (l Lesson, replaced bool, err error) {
	if _, err := s.point(pointID); err != nil {
		return Lesson{}, false, err
	}
	content = Normalize(content)
	if fields := Validate(content); fields != nil {
		return Lesson{}, false, &ValidationError{Fields: fields}
	}
	now := s.now()
	l = Lesson{PointID: pointID, Status: StatusDraft, Content: content, Edited: true, Imported: true, CreatedAt: now, UpdatedAt: now}
	old, err := s.lessons.Get(ctx, pointID)
	switch {
	case err == nil:
		if !replace {
			return Lesson{}, false, ErrHasLesson
		}
		l.CreatedAt, replaced = old.CreatedAt, true
	case errors.Is(err, ErrNotFound):
	default:
		return Lesson{}, false, fmt.Errorf("grammar: get lesson: %w", err)
	}
	if dryRun {
		return l, replaced, nil
	}
	if err := s.lessons.Save(ctx, l); err != nil {
		return Lesson{}, false, fmt.Errorf("grammar: save lesson: %w", err)
	}
	return l, replaced, nil
}
