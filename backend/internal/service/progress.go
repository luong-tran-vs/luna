package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/luongtran/luna/backend/internal/auth"
	"github.com/luongtran/luna/backend/internal/lesson"
	"github.com/luongtran/luna/backend/internal/progress"
	"github.com/luongtran/luna/backend/internal/topic"
	"github.com/luongtran/luna/backend/internal/vocab"
)

// initProgress builds the dictation totals and the study flow (goals, the lesson being studied, its
// steps, streak, stats; L, F4, F6). It is the last one to need the others (grammar included, for
// the stats), and it hands the study service back to the writing domain, which needs it for the
// Write step.
func (c *Container) initProgress() {
	c.progress = progress.NewService(c.store.DictationResults(), progressLessons{c.lessons}, time.Now)
	c.study = progress.NewStudyService(progress.StudyDeps{
		Goals:     c.store.Goals(),
		Progress:  c.store.LessonProgress(),
		Days:      c.store.StudyDays(),
		Dictation: c.progress,
		Lessons:   progressLessons{c.lessons},
		Roadmaps:  topicRoadmaps{c.topic},
		Titles:    lessonTitles{c.lessons},
		Reviews:   dailyReviews{c.vocab},
		Timezones: c.settings,
		Quiz:      readingQuiz{c.reader},
		Writings:  c.writing,
		Grammar:   c.grammar,
		Members:   guestRoles{c.auth},
		Now:       time.Now,
	})
	c.writeSteps.svc = c.study
}

// progressLessons adapts the lesson repository to progress.Lessons.
type progressLessons struct {
	repo lesson.Repository
}

func (p progressLessons) Info(ctx context.Context, id string) (revision, sentenceCount int, err error) {
	l, err := p.repo.Get(ctx, id)
	if errors.Is(err, lesson.ErrNotFound) {
		return 0, 0, progress.ErrLessonNotFound
	}
	if err != nil {
		return 0, 0, fmt.Errorf("get lesson: %w", err)
	}
	return l.Revision, len(l.Sentences), nil
}

// topicRoadmaps adapts topic.Service to progress.Roadmaps.
type topicRoadmaps struct {
	svc *topic.Service
}

func (t topicRoadmaps) Roadmap(ctx context.Context, topicID, level string) (progress.TopicInfo, error) {
	tp, err := t.svc.Get(ctx, topicID)
	if errors.Is(err, topic.ErrNotFound) {
		return progress.TopicInfo{}, progress.ErrTopicNotFound
	}
	if err != nil {
		return progress.TopicInfo{}, fmt.Errorf("get topic: %w", err)
	}
	ids, err := t.svc.LearnerRoadmap(ctx, tp, level)
	if err != nil {
		return progress.TopicInfo{}, err
	}
	return progress.TopicInfo{ID: tp.ID, Name: tp.Name, Level: level, LessonIDs: ids}, nil
}

// dailyReviews adapts vocab.Service to progress.Reviews.
type dailyReviews struct {
	svc *vocab.Service
}

func (d dailyReviews) DueCount(ctx context.Context, userID string) (int, error) {
	list, err := d.svc.Due(ctx, userID, 1)
	if err != nil {
		return 0, fmt.Errorf("due cards: %w", err)
	}
	return list.Total, nil
}

func (d dailyReviews) DueBefore(ctx context.Context, userID string, before, createdBefore time.Time) (int, error) {
	return d.svc.DueBefore(ctx, userID, before, createdBefore)
}

func (d dailyReviews) CardCount(ctx context.Context, userID string, since *time.Time) (int, error) {
	return d.svc.Count(ctx, userID, since)
}

// readingQuiz adapts lesson.Reader to progress.ReadingQuiz (F15).
type readingQuiz struct {
	reader *lesson.Reader
}

func (q readingQuiz) Status(ctx context.Context, userID, lessonID string) (questions, answered int, err error) {
	return q.reader.QuizStatus(ctx, userID, lessonID)
}

func (q readingQuiz) Totals(ctx context.Context, userID string, since *time.Time) (answered, correct int, err error) {
	return q.reader.Totals(ctx, userID, since)
}

// guestRoles adapts auth.Service to progress.Members.
type guestRoles struct {
	svc *auth.Service
}

func (g guestRoles) IsGuest(ctx context.Context, userID string) (bool, error) {
	u, err := g.svc.UserByID(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("get user: %w", err)
	}
	return u.Role == auth.RoleGuest, nil
}
