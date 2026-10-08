package writing

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/luongtran/luna/backend/internal/ai"
	"github.com/luongtran/luna/backend/internal/job"
)

// Caps on what the AI may store in a grade.
const (
	maxComment   = 600
	maxCorrected = 4000
)

// CheckGrade turns the AI answer into the four criteria in display order. Every score must be
// 1–5 and every text present; long texts are cut.
func CheckGrade(g ai.Grade) (Grade, error) {
	out := Grade{
		OverallVi:     cut(g.OverallVi, maxComment),
		CorrectedText: cut(g.CorrectedText, maxCorrected),
	}
	for _, c := range []struct {
		name string
		ai.Criterion
	}{
		{CriterionTask, g.Task},
		{CriterionGrammar, g.Grammar},
		{CriterionVocabulary, g.Vocabulary},
		{CriterionCoherence, g.Coherence},
	} {
		comment := cut(c.CommentVi, maxComment)
		if c.Score < 1 || c.Score > 5 || comment == "" {
			return Grade{}, ErrUnusableGrade
		}
		out.Criteria = append(out.Criteria, Criterion{Name: c.name, Score: c.Score, CommentVi: comment})
	}
	if out.OverallVi == "" || out.CorrectedText == "" {
		return Grade{}, ErrUnusableGrade
	}
	return out, nil
}

func cut(s string, limit int) string {
	s = strings.TrimSpace(s)
	if utf8.RuneCountInString(s) > limit {
		s = strings.TrimSpace(string([]rune(s)[:limit]))
	}
	return s
}

// ProcessGrade grades one submitted writing with one AI request (job handler for "grade").
// A bad answer is retried by the worker; missing configuration is permanent.
func (s *Service) ProcessGrade(ctx context.Context, j job.Job) error {
	w, err := s.d.Repo.Get(ctx, j.TargetID)
	if errors.Is(err, ErrNotFound) {
		return job.Permanent(err)
	}
	if err != nil {
		return fmt.Errorf("writing: get: %w", err)
	}
	if w.Status != StatusSubmitted || w.Grade == nil || w.Grade.Status != GradePending {
		return nil
	}
	req := ai.GradeRequest{Prompt: w.Prompt, Text: w.Text}
	switch info, err := s.d.Lessons.Info(ctx, w.LessonID); {
	case err == nil:
		req.Level, req.LessonText = info.Level, info.Content
	case !errors.Is(err, ErrNotFound):
		return fmt.Errorf("writing: lesson: %w", err)
	}

	res, err := s.d.AI.GradeWriting(ctx, req)
	if err != nil {
		if errors.Is(err, ai.ErrNotConfigured) || errors.Is(err, ai.ErrInvalidKey) {
			return job.Permanent(err)
		}
		return fmt.Errorf("writing: grade: %w", err)
	}
	g, err := CheckGrade(res)
	if err != nil {
		return err
	}
	g.Status, g.GradedAt, g.Seen = GradeDone, s.d.Now().UTC(), false
	if err := s.d.Repo.SetGrade(ctx, w.ID, g); err != nil {
		return fmt.Errorf("writing: save grade: %w", err)
	}
	return nil
}

// JobFailed records a grading that gave up, with a short Vietnamese reason; the writing stays.
func (s *Service) JobFailed(ctx context.Context, j job.Job, err error) {
	g := Grade{Status: GradeFailed, Error: failureMessage(err), GradedAt: s.d.Now().UTC(), Seen: false}
	if serr := s.d.Repo.SetGrade(ctx, j.TargetID, g); serr != nil && !errors.Is(serr, ErrNotFound) {
		s.d.Log.ErrorContext(ctx, "record grade failure", slog.String("writing_id", j.TargetID), slog.Any("error", serr))
	}
}

func failureMessage(err error) string {
	switch {
	case errors.Is(err, ai.ErrNotConfigured):
		return "AI chưa được cấu hình"
	case errors.Is(err, ai.ErrInvalidKey):
		return "Khoá API của AI không hợp lệ"
	case errors.Is(err, ai.ErrQuota):
		return "AI hết lượt, vui lòng chấm lại sau"
	case errors.Is(err, ErrUnusableGrade):
		return "AI trả về kết quả không dùng được"
	default:
		return "Không chấm được bài"
	}
}

// MaxGradings is how many times one writing can be graded: the first submission plus one more,
// either a resubmission of the edited text or a regrade after a failed grading.
const MaxGradings = 2

// GradingsUsed is how many gradings the writing used (at least one once submitted).
func GradingsUsed(w Writing) int {
	return max(w.Gradings, 1)
}

// Regrade queues the grading again for a writing whose grading failed. It uses one of the
// writing's MaxGradings gradings.
func (s *Service) Regrade(ctx context.Context, userID, id string) (Writing, error) {
	w, err := s.own(ctx, userID, id)
	if err != nil {
		return Writing{}, err
	}
	if w.Grade == nil || w.Grade.Status != GradeFailed {
		return Writing{}, ErrNotFailed
	}
	return s.regrade(ctx, w, w.Text, w.SubmittedAt)
}

// Resubmit sends the learner's edited writing to grading again, once its grading is over (done or
// failed). It uses one of the writing's MaxGradings gradings; the new grade replaces the old one.
func (s *Service) Resubmit(ctx context.Context, userID, id, text string) (Writing, error) {
	w, err := s.own(ctx, userID, id)
	if err != nil {
		return Writing{}, err
	}
	if w.Status != StatusSubmitted {
		return Writing{}, ErrNotFound
	}
	if w.Grade != nil && w.Grade.Status == GradePending {
		return Writing{}, ErrGrading
	}
	text, err = checkLength(text)
	if err != nil {
		return Writing{}, err
	}
	return s.regrade(ctx, w, text, s.d.Now().UTC())
}

func (s *Service) regrade(ctx context.Context, w Writing, text string, submittedAt time.Time) (Writing, error) {
	used := GradingsUsed(w)
	if used >= MaxGradings {
		return Writing{}, ErrNoGradings
	}
	if err := s.d.Repo.Regrade(ctx, w.ID, text, submittedAt, used+1); err != nil {
		return Writing{}, fmt.Errorf("writing: regrade: %w", err)
	}
	if err := s.enqueue(ctx, w.ID); err != nil {
		return Writing{}, err
	}
	w.Text, w.SubmittedAt, w.Gradings = text, submittedAt, used+1
	w.Grade = &Grade{Status: GradePending, Seen: true}
	return w, nil
}
