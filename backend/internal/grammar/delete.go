package grammar

import (
	"context"
	"errors"
	"fmt"
)

// ErrPublished means the lesson is published, so it must be unpublished before it is deleted
// (added 2026-10-26).
var ErrPublished = errors.New("grammar: lesson is published")

// Delete removes the draft lesson of a point, which then has no lesson until one is generated or
// imported (added 2026-10-26). A published lesson is refused with ErrPublished: learners see it,
// so it is unpublished first. Learner progress and reports on the point are kept.
func (s *Service) Delete(ctx context.Context, pointID string) error {
	l, err := s.AdminGet(ctx, pointID)
	if err != nil {
		return err
	}
	if l.Status == StatusPublished {
		return ErrPublished
	}
	if err := s.lessons.Delete(ctx, pointID); err != nil {
		return fmt.Errorf("grammar: delete lesson: %w", err)
	}
	return nil
}
