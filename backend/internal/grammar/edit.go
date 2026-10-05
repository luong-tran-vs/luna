package grammar

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// ErrNotChecked means the lesson was never checked by the AI, so there is nothing to verify.
var ErrNotChecked = errors.New("grammar: lesson not checked")

// unconfirmed counts the checks an admin has not confirmed yet.
func unconfirmed(checks []Check) int {
	n := 0
	for _, c := range checks {
		if !c.Confirmed {
			n++
		}
	}
	return n
}

// exerciseRef finds an exercise by id: the list it is in (the lesson's slice) and its index.
func exerciseRef(c *Content, id string) (list *[]Exercise, idx int, prefix string, ok bool) {
	for i, e := range c.Practice {
		if e.ID == id {
			return &c.Practice, i, "p", true
		}
	}
	for i, e := range c.Mastery {
		if e.ID == id {
			return &c.Mastery, i, "m", true
		}
	}
	return nil, 0, "", false
}

// ConfirmCheck marks the check of one exercise as looked at and kept. ErrNotFound when the
// exercise has no check.
func (s *Service) ConfirmCheck(ctx context.Context, pointID, exerciseID string) (Lesson, error) {
	l, err := s.AdminGet(ctx, pointID)
	if err != nil {
		return Lesson{}, err
	}
	found := false
	checks := make([]Check, len(l.Checks))
	copy(checks, l.Checks)
	for i := range checks {
		if checks[i].ExerciseID == exerciseID {
			checks[i].Confirmed, found = true, true
		}
	}
	if !found {
		return Lesson{}, ErrNotFound
	}
	l.Checks = checks
	return s.saveLesson(ctx, l)
}

// Verify confirms the whole check: every flag counts as looked at. ErrNotChecked when the AI check
// has not run.
func (s *Service) Verify(ctx context.Context, pointID string) (Lesson, error) {
	l, err := s.AdminGet(ctx, pointID)
	if err != nil {
		return Lesson{}, err
	}
	if l.CheckedAt.IsZero() {
		return Lesson{}, ErrNotChecked
	}
	checks := make([]Check, len(l.Checks))
	copy(checks, l.Checks)
	for i := range checks {
		checks[i].Confirmed = true
	}
	l.Checks, l.VerifiedAt = checks, s.now()
	return s.saveLesson(ctx, l)
}

// ReplaceExercise replaces one exercise in place (same position and id). Its kind cannot change.
// Only that exercise's check goes; the other checks and CheckedAt stay.
func (s *Service) ReplaceExercise(ctx context.Context, pointID, exerciseID string, ex Exercise) (Lesson, error) {
	l, err := s.AdminGet(ctx, pointID)
	if err != nil {
		return Lesson{}, err
	}
	content := cloneContent(l.Content)
	list, idx, prefix, ok := exerciseRef(&content, exerciseID)
	if !ok {
		return Lesson{}, ErrNotFound
	}
	ex.ID = exerciseID
	if strings.TrimSpace(string(ex.Kind)) != string((*list)[idx].Kind) {
		return Lesson{}, &ValidationError{Fields: map[string]string{"kind": "Không được đổi loại bài tập"}}
	}
	ex = normalizeExercises([]Exercise{ex}, prefix)[0]
	ex.ID = exerciseID
	if problems := exerciseProblems(ex); problems != nil {
		fields := make(map[string]string, len(problems))
		for k, v := range problems {
			fields["exercise."+k] = v
		}
		return Lesson{}, &ValidationError{Fields: fields}
	}
	(*list)[idx] = ex
	l.Content = content
	l.Checks = withoutCheck(l.Checks, exerciseID)
	l.VerifiedAt, l.Edited = time.Time{}, true
	return s.saveLesson(ctx, l)
}

// DeleteExercise removes one exercise, keeping the ids of the others, and closes its open reports.
// A lesson cannot drop below the minimum number of exercises.
func (s *Service) DeleteExercise(ctx context.Context, pointID, exerciseID string) (Lesson, error) {
	l, err := s.AdminGet(ctx, pointID)
	if err != nil {
		return Lesson{}, err
	}
	content := cloneContent(l.Content)
	list, idx, prefix, ok := exerciseRef(&content, exerciseID)
	if !ok {
		return Lesson{}, ErrNotFound
	}
	*list = append((*list)[:idx], (*list)[idx+1:]...)
	f := fieldErrors{}
	if prefix == "p" {
		f.count("content.practice", len(*list), minPractice, maxPractice, "bài luyện tập")
	} else {
		f.count("content.mastery", len(*list), minMastery, maxMastery, "bài kiểm tra")
	}
	if len(f) > 0 {
		return Lesson{}, &ValidationError{Fields: f}
	}
	l.Content = content
	l.Checks = withoutCheck(l.Checks, exerciseID)
	l.VerifiedAt, l.Edited = time.Time{}, true
	l, err = s.saveLesson(ctx, l)
	if err != nil {
		return Lesson{}, err
	}
	if _, err := s.reports.Resolve(ctx, pointID, exerciseID, s.now()); err != nil {
		return Lesson{}, fmt.Errorf("grammar: resolve reports: %w", err)
	}
	return l, nil
}

func (s *Service) saveLesson(ctx context.Context, l Lesson) (Lesson, error) {
	l.UpdatedAt = s.now()
	if err := s.lessons.Save(ctx, l); err != nil {
		return Lesson{}, fmt.Errorf("grammar: save lesson: %w", err)
	}
	return l, nil
}

func withoutCheck(checks []Check, exerciseID string) []Check {
	out := make([]Check, 0, len(checks))
	for _, c := range checks {
		if c.ExerciseID != exerciseID {
			out = append(out, c)
		}
	}
	return out
}

// cloneContent copies the exercise lists, so edits never reach a slice another reader holds.
func cloneContent(c Content) Content {
	c.Practice = append([]Exercise(nil), c.Practice...)
	c.Mastery = append([]Exercise(nil), c.Mastery...)
	return c
}
