package grammar

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

// maxReportNote is the longest note a learner can attach to a report.
const maxReportNote = 300

// maxGroupNotes is how many notes a ReportGroup lists.
const maxGroupNotes = 3

// ReportInput is what a learner sends to report an exercise.
type ReportInput struct {
	ExerciseID string
	Reason     ReportReason
	Note       string
}

// ReportGroup is the open reports of one exercise, for the admin.
type ReportGroup struct {
	PointID    string
	ExerciseID string
	Count      int
	Reasons    map[ReportReason]int
	// Notes are the newest non-empty notes, at most maxGroupNotes.
	Notes    []string
	LatestAt time.Time
}

// Report stores a learner's report on an exercise of a published lesson. Reporting again reopens it.
func (s *Service) Report(ctx context.Context, userID, pointID string, in ReportInput) error {
	_, l, err := s.published(ctx, pointID)
	if err != nil {
		return err
	}
	f := fieldErrors{}
	found := false
	for _, e := range append(append([]Exercise{}, l.Content.Practice...), l.Content.Mastery...) {
		if e.ID == in.ExerciseID {
			found = true
			break
		}
	}
	if !found {
		f.add("exerciseId", "Bài tập không có trong bài này")
	}
	switch in.Reason {
	case ReasonWrongAnswer, ReasonAmbiguous, ReasonTypo, ReasonOther:
	default:
		f.add("reason", "Lý do không hợp lệ")
	}
	note := strings.TrimSpace(in.Note)
	if utf8.RuneCountInString(note) > maxReportNote {
		f.add("note", fmt.Sprintf("Tối đa %d ký tự", maxReportNote))
	}
	if len(f) > 0 {
		return &ValidationError{Fields: f}
	}
	now := s.now()
	err = s.reports.Upsert(ctx, Report{
		UserID: userID, PointID: pointID, ExerciseID: in.ExerciseID, Reason: in.Reason, Note: note,
		Status: ReportOpen, CreatedAt: now, UpdatedAt: now,
	})
	if err != nil {
		return fmt.Errorf("grammar: save report: %w", err)
	}
	return nil
}

// AdminReports returns the open reports grouped by (point, exercise), newest first.
func (s *Service) AdminReports(ctx context.Context) ([]ReportGroup, error) {
	open, err := s.reports.ListOpen(ctx)
	if err != nil {
		return nil, fmt.Errorf("grammar: list reports: %w", err)
	}
	// Newest first, whatever order the repository used.
	sort.SliceStable(open, func(i, j int) bool { return open[i].UpdatedAt.After(open[j].UpdatedAt) })
	index := map[[2]string]int{}
	groups := []ReportGroup{}
	for _, r := range open {
		key := [2]string{r.PointID, r.ExerciseID}
		i, ok := index[key]
		if !ok {
			i = len(groups)
			index[key] = i
			groups = append(groups, ReportGroup{
				PointID: r.PointID, ExerciseID: r.ExerciseID, Reasons: map[ReportReason]int{}, Notes: []string{},
				LatestAt: r.UpdatedAt,
			})
		}
		g := &groups[i]
		g.Count++
		g.Reasons[r.Reason]++
		if note := strings.TrimSpace(r.Note); note != "" && len(g.Notes) < maxGroupNotes {
			g.Notes = append(g.Notes, note)
		}
	}
	return groups, nil
}

// AdminReportsOf returns the open report groups of one point.
func (s *Service) AdminReportsOf(ctx context.Context, pointID string) ([]ReportGroup, error) {
	all, err := s.AdminReports(ctx)
	if err != nil {
		return nil, err
	}
	out := []ReportGroup{}
	for _, g := range all {
		if g.PointID == pointID {
			out = append(out, g)
		}
	}
	return out, nil
}

// ResolveReports closes the open reports on an exercise and returns how many there were.
func (s *Service) ResolveReports(ctx context.Context, pointID, exerciseID string) (int, error) {
	if _, err := s.point(pointID); err != nil {
		return 0, err
	}
	if strings.TrimSpace(exerciseID) == "" {
		return 0, &ValidationError{Fields: map[string]string{"exerciseId": "Không được để trống"}}
	}
	n, err := s.reports.Resolve(ctx, pointID, exerciseID, s.now())
	if err != nil {
		return 0, fmt.Errorf("grammar: resolve reports: %w", err)
	}
	return n, nil
}
