package progress

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	maxTyped = 1000
	maxWords = 1000
)

// Input is one checked sentence as sent by the Listening page. The comparison runs in the
// browser; the backend only checks that the numbers are in range.
type Input struct {
	SentenceIndex int
	Typed         string
	CorrectWords  int
	TotalWords    int
}

// Service records dictation results. The user is always the session user.
type Service struct {
	repo    DictationRepository
	lessons Lessons
	now     func() time.Time
}

// NewService returns a Service.
func NewService(repo DictationRepository, lessons Lessons, now func() time.Time) *Service {
	return &Service{repo: repo, lessons: lessons, now: now}
}

// Record stores the latest result of one sentence against the current lesson revision and
// returns the new summary.
func (s *Service) Record(ctx context.Context, userID, lessonID string, in Input) (Summary, error) {
	revision, count, err := s.lessons.Info(ctx, lessonID)
	if err != nil {
		return Summary{}, fmt.Errorf("progress: lesson: %w", err)
	}
	in.Typed = strings.TrimSpace(in.Typed)
	if err := validate(in, count); err != nil {
		return Summary{}, err
	}
	r := Result{
		SentenceIndex: in.SentenceIndex,
		Typed:         in.Typed,
		CorrectWords:  in.CorrectWords,
		TotalWords:    in.TotalWords,
		CheckedAt:     s.now(),
	}
	if err := s.repo.Upsert(ctx, userID, lessonID, revision, r); err != nil {
		return Summary{}, fmt.Errorf("progress: upsert: %w", err)
	}
	return s.summarize(ctx, userID, lessonID, revision, count)
}

// Summary totals userID's results for the current revision of the lesson.
func (s *Service) Summary(ctx context.Context, userID, lessonID string) (Summary, error) {
	revision, count, err := s.lessons.Info(ctx, lessonID)
	if err != nil {
		return Summary{}, fmt.Errorf("progress: lesson: %w", err)
	}
	return s.summarize(ctx, userID, lessonID, revision, count)
}

func (s *Service) summarize(ctx context.Context, userID, lessonID string, revision, count int) (Summary, error) {
	stored, err := s.repo.List(ctx, userID, lessonID)
	if err != nil {
		return Summary{}, fmt.Errorf("progress: list: %w", err)
	}
	sum := Summary{SentenceCount: count, Results: []Result{}}
	for _, r := range stored {
		// Results of an older revision belong to content that no longer exists.
		if r.Revision != revision || r.SentenceIndex < 0 || r.SentenceIndex >= count {
			continue
		}
		sum.Results = append(sum.Results, r.Result)
		sum.CorrectWords += r.CorrectWords
		sum.TotalWords += r.TotalWords
	}
	sort.Slice(sum.Results, func(i, j int) bool { return sum.Results[i].SentenceIndex < sum.Results[j].SentenceIndex })
	sum.CheckedCount = len(sum.Results)
	if sum.TotalWords > 0 {
		sum.Rate = float64(sum.CorrectWords) / float64(sum.TotalWords)
	}
	sum.Completed = count > 0 && sum.CheckedCount == count
	return sum, nil
}

func validate(in Input, sentenceCount int) error {
	fields := map[string]string{}
	if in.SentenceIndex < 0 || in.SentenceIndex >= sentenceCount {
		fields["sentenceIndex"] = "Câu không có trong bài"
	}
	if n := utf8.RuneCountInString(in.Typed); n < 1 || n > maxTyped {
		fields["typed"] = "Câu đã gõ cần từ 1 đến 1000 ký tự"
	}
	if in.TotalWords < 1 || in.TotalWords > maxWords {
		fields["totalWords"] = "Số từ không hợp lệ"
	}
	if in.CorrectWords < 0 || in.CorrectWords > in.TotalWords {
		fields["correctWords"] = "Số từ đúng không hợp lệ"
	}
	if len(fields) > 0 {
		return &ValidationError{Fields: fields}
	}
	return nil
}

// Totals sums the user's dictation results over every lesson.
func (s *Service) Totals(ctx context.Context, userID string) (DictationTotals, error) {
	t, err := s.repo.Totals(ctx, userID)
	if err != nil {
		return DictationTotals{}, fmt.Errorf("progress: dictation totals: %w", err)
	}
	return t, nil
}
