package grammar

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/luongtran/luna/backend/internal/ai"
)

// generateTimeout bounds the AI call that writes one grammar lesson.
const generateTimeout = 120 * time.Second

// ProgressNew is the status of a point the learner has not started; it is never stored.
const ProgressNew ProgressStatus = "new"

var (
	// ErrExists means the point already has a lesson that was edited or published, so a new
	// generation needs force.
	ErrExists = errors.New("grammar: lesson exists")
	// ErrUnusable means the AI answered but too little of it passed the checks.
	ErrUnusable = errors.New("grammar: AI returned no usable lesson")
	// errAI wraps every failure of the AI call so handlers can tell it from storage errors.
	errAI = errors.New("grammar: generate")
)

// LessonCounter counts the topic lessons that use each grammar point (implemented over the lesson
// repository in internal/service, so this package does not import the lesson package).
type LessonCounter interface {
	CountByGrammarPoint(ctx context.Context, topicID string) (map[string]int, error)
}

// Service serves the grammar lessons: the admin side (generate, edit, publish) and the learner
// side (list, study, record attempts).
type Service struct {
	lessons  LessonRepository
	progress ProgressRepository
	reports  ReportRepository
	ai       ai.Provider
	counter  LessonCounter
	syllabus *Syllabus
	now      func() time.Time
}

// NewService returns a Service over the embedded syllabus.
func NewService(lessons LessonRepository, progress ProgressRepository, reports ReportRepository, provider ai.Provider, counter LessonCounter,
	now func() time.Time,
) *Service {
	return &Service{lessons: lessons, progress: progress, reports: reports, ai: provider, counter: counter, syllabus: Default(), now: now}
}

// point returns the syllabus point or ErrNotFound.
func (s *Service) point(id string) (Point, error) {
	p, ok := s.syllabus.Get(id)
	if !ok {
		return Point{}, ErrNotFound
	}
	return p, nil
}

// --- admin ---

// AdminList returns a summary of every lesson that exists, in syllabus order.
func (s *Service) AdminList(ctx context.Context) ([]LessonSummary, error) {
	all, err := s.lessons.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("grammar: list lessons: %w", err)
	}
	byID := make(map[string]LessonSummary, len(all))
	for _, l := range all {
		byID[l.PointID] = l
	}
	out := make([]LessonSummary, 0, len(all))
	for _, p := range s.syllabus.All() {
		if l, ok := byID[p.ID]; ok {
			out = append(out, l)
		}
	}
	return out, nil
}

// AdminGet returns the lesson of a point, whatever its status.
func (s *Service) AdminGet(ctx context.Context, pointID string) (Lesson, error) {
	if _, err := s.point(pointID); err != nil {
		return Lesson{}, err
	}
	return s.lessons.Get(ctx, pointID)
}

// Generate asks the AI (one request) for the lesson of a point, checks it and stores it as an
// unedited draft. created is true when the point had no lesson. A lesson that was edited or
// published is replaced only with force (ErrExists otherwise).
func (s *Service) Generate(ctx context.Context, pointID string, force bool) (l Lesson, created bool, err error) {
	p, err := s.point(pointID)
	if err != nil {
		return Lesson{}, false, err
	}
	old, err := s.lessons.Get(ctx, pointID)
	switch {
	case err == nil:
		if (old.Edited || old.Status == StatusPublished) && !force {
			return Lesson{}, false, ErrExists
		}
	case errors.Is(err, ErrNotFound):
		created = true
	default:
		return Lesson{}, false, fmt.Errorf("grammar: get lesson: %w", err)
	}

	aiCtx, cancel := context.WithTimeout(ctx, generateTimeout)
	defer cancel()
	raw, err := s.ai.GrammarLesson(aiCtx, ai.GrammarLessonRequest{
		Level: p.Level, TitleVi: p.TitleVi, TitleEn: p.TitleEn, Pattern: p.Pattern, HintVi: p.HintVi, Examples: p.Examples,
	})
	if err != nil {
		if errors.Is(err, ai.ErrNotConfigured) || errors.Is(err, ai.ErrInvalidKey) || errors.Is(err, ai.ErrQuota) {
			return Lesson{}, false, err
		}
		return Lesson{}, false, fmt.Errorf("%w: %w", errAI, err)
	}
	content := FilterGenerated(fromAI(raw))
	if Validate(content) != nil {
		return Lesson{}, false, ErrUnusable
	}

	now := s.now()
	l = Lesson{PointID: pointID, Status: StatusDraft, Content: content, CreatedAt: now, UpdatedAt: now}
	if !created {
		l.CreatedAt = old.CreatedAt
	}
	if err := s.lessons.Save(ctx, l); err != nil {
		return Lesson{}, false, fmt.Errorf("grammar: save lesson: %w", err)
	}
	// A failed check must not lose the lesson: the admin can run it again.
	if checked, err := s.runCheck(ctx, p, l); err != nil {
		slog.WarnContext(ctx, "grammar check after generate failed", slog.String("point", pointID), slog.Any("error", err))
	} else {
		l = checked
	}
	return l, created, nil
}

// Save replaces the content of an existing lesson with the admin's version. The status stays.
func (s *Service) Save(ctx context.Context, pointID string, content Content) (Lesson, error) {
	l, err := s.AdminGet(ctx, pointID)
	if err != nil {
		return Lesson{}, err
	}
	content = Normalize(content)
	if fields := Validate(content); fields != nil {
		return Lesson{}, &ValidationError{Fields: fields}
	}
	l.Content = content
	l.Edited = true
	l.Checks, l.CheckedAt, l.VerifiedAt = nil, time.Time{}, time.Time{}
	l.UpdatedAt = s.now()
	if err := s.lessons.Save(ctx, l); err != nil {
		return Lesson{}, fmt.Errorf("grammar: save lesson: %w", err)
	}
	return l, nil
}

// Publish opens the lesson to learners after checking it against every limit. A lesson whose AI check
// left flags needs acknowledgeFlags (FlagsError otherwise).
func (s *Service) Publish(ctx context.Context, pointID string, acknowledgeFlags bool) (Lesson, error) {
	l, err := s.AdminGet(ctx, pointID)
	if err != nil {
		return Lesson{}, err
	}
	l.Content = Normalize(l.Content)
	if n := unconfirmed(l.Checks); n > 0 && !acknowledgeFlags {
		return Lesson{}, &FlagsError{Count: n}
	}
	if fields := Validate(l.Content); fields != nil {
		return Lesson{}, &ValidationError{Fields: fields}
	}
	now := s.now()
	if l.Status != StatusPublished {
		l.PublishedAt = now
	}
	l.Status = StatusPublished
	l.UpdatedAt = now
	if err := s.lessons.Save(ctx, l); err != nil {
		return Lesson{}, fmt.Errorf("grammar: save lesson: %w", err)
	}
	return l, nil
}

// Unpublish hides the lesson from learners again; their progress stays.
func (s *Service) Unpublish(ctx context.Context, pointID string) (Lesson, error) {
	l, err := s.AdminGet(ctx, pointID)
	if err != nil {
		return Lesson{}, err
	}
	l.Status = StatusDraft
	l.PublishedAt = time.Time{}
	l.UpdatedAt = s.now()
	if err := s.lessons.Save(ctx, l); err != nil {
		return Lesson{}, fmt.Errorf("grammar: save lesson: %w", err)
	}
	return l, nil
}

// fromAI converts a provider answer to Content; the exercises are checked later.
func fromAI(r ai.GrammarLessonContent) Content {
	c := Content{Objective: r.Objective, Explanation: r.Explanation, Usage: r.Usage}
	for _, s := range r.Structures {
		c.Structures = append(c.Structures, Structure(s))
	}
	for _, e := range r.Examples {
		c.Examples = append(c.Examples, Example(e))
	}
	for _, m := range r.Mistakes {
		c.Mistakes = append(c.Mistakes, Mistake(m))
	}
	c.Practice = exercisesFromAI(r.Practice)
	c.Mastery = exercisesFromAI(r.Mastery)
	return c
}

func exercisesFromAI(in []ai.GrammarExercise) []Exercise {
	out := make([]Exercise, len(in))
	for i, e := range in {
		out[i] = Exercise{
			Kind: Kind(e.Kind), PromptVi: e.PromptVi, Text: e.Text, Options: e.Options, AnswerIndex: e.AnswerIndex,
			Answers: e.Answers, Words: e.Words, Sentence: e.Sentence, ExplanationVi: e.ExplanationVi,
		}
	}
	return out
}

// --- learner ---

// PointStatus is one syllabus point in the learner's list.
type PointStatus struct {
	Point
	// Available is true when the point has a published lesson.
	Available   bool
	Status      ProgressStatus
	BestMastery int
}

// Overview is the learner's list of points and the suggested next one.
type Overview struct {
	Points []PointStatus
	// Next is the first available point that is not mastered yet, "" when there is none.
	Next string
}

// List returns the points of a level ("" for every level) in syllabus order, with the learner's
// status, and the next point to study.
func (s *Service) List(ctx context.Context, userID, level string) (Overview, error) {
	if level != "" && !slices.Contains(Levels, level) {
		return Overview{}, &ValidationError{Fields: map[string]string{"level": "Trình độ không hợp lệ"}}
	}
	lessons, err := s.lessons.List(ctx)
	if err != nil {
		return Overview{}, fmt.Errorf("grammar: list lessons: %w", err)
	}
	published := map[string]bool{}
	for _, l := range lessons {
		if l.Status == StatusPublished {
			published[l.PointID] = true
		}
	}
	records, err := s.progress.List(ctx, userID)
	if err != nil {
		return Overview{}, fmt.Errorf("grammar: list progress: %w", err)
	}
	byPoint := make(map[string]Progress, len(records))
	for _, p := range records {
		byPoint[p.PointID] = p
	}

	points := s.syllabus.All()
	if level != "" {
		points = s.syllabus.ByLevel(level)
	}
	out := Overview{Points: make([]PointStatus, 0, len(points))}
	for _, p := range points {
		ps := PointStatus{Point: p, Available: published[p.ID], Status: ProgressNew}
		if rec, ok := byPoint[p.ID]; ok {
			ps.Status, ps.BestMastery = rec.Status, rec.BestMastery
		}
		if ps.Available && ps.Status != ProgressMastered && out.Next == "" {
			out.Next = p.ID
		}
		out.Points = append(out.Points, ps)
	}
	return out, nil
}

// Detail is the study page of a published point.
type Detail struct {
	Point       Point
	Content     Content
	Progress    Progress
	LessonCount int
}

// published returns the published lesson of a point, ErrNotFound for anything else.
func (s *Service) published(ctx context.Context, pointID string) (Point, Lesson, error) {
	p, err := s.point(pointID)
	if err != nil {
		return Point{}, Lesson{}, err
	}
	l, err := s.lessons.Get(ctx, pointID)
	if err != nil {
		return Point{}, Lesson{}, err
	}
	if l.Status != StatusPublished {
		return Point{}, Lesson{}, ErrNotFound
	}
	return p, l, nil
}

func (s *Service) progressOf(ctx context.Context, userID, pointID string) (Progress, error) {
	rec, err := s.progress.Get(ctx, userID, pointID)
	switch {
	case err == nil:
		if rec.Weak == nil {
			rec.Weak = []string{}
		}
		return rec, nil
	case errors.Is(err, ErrNotFound):
		return Progress{UserID: userID, PointID: pointID, Status: ProgressNew, Weak: []string{}}, nil
	default:
		return Progress{}, fmt.Errorf("grammar: get progress: %w", err)
	}
}

// Get returns the study page of a published point with the learner's progress.
func (s *Service) Get(ctx context.Context, userID, pointID string) (Detail, error) {
	p, l, err := s.published(ctx, pointID)
	if err != nil {
		return Detail{}, err
	}
	rec, err := s.progressOf(ctx, userID, pointID)
	if err != nil {
		return Detail{}, err
	}
	counts, err := s.counter.CountByGrammarPoint(ctx, "")
	if err != nil {
		return Detail{}, fmt.Errorf("grammar: count lessons: %w", err)
	}
	return Detail{Point: p, Content: expandFillAnswers(l.Content), Progress: rec, LessonCount: counts[pointID]}, nil
}

// Attempt kinds.
const (
	AttemptPractice = "practice"
	AttemptMastery  = "mastery"
)

// Attempt is a finished round of exercises, scored in the browser.
type Attempt struct {
	Kind    string
	Correct int
	Total   int
	// Wrong lists the ids of the exercises answered wrongly.
	Wrong []string
}

// RecordAttempt checks an attempt against the lesson and updates the learner's progress.
// passed is true when a mastery check reached MasteryPercent.
func (s *Service) RecordAttempt(ctx context.Context, userID, pointID string, a Attempt) (rec Progress, passed bool, err error) {
	_, l, err := s.published(ctx, pointID)
	if err != nil {
		return Progress{}, false, err
	}
	if fields := checkAttempt(l.Content, a); fields != nil {
		return Progress{}, false, &ValidationError{Fields: fields}
	}
	rec, err = s.progressOf(ctx, userID, pointID)
	if err != nil {
		return Progress{}, false, err
	}

	now := s.now()
	percent := (a.Correct*100 + a.Total/2) / a.Total
	rec.UserID, rec.PointID, rec.UpdatedAt = userID, pointID, now
	rec.Weak = slices.Clone(a.Wrong)
	if rec.Weak == nil {
		rec.Weak = []string{}
	}
	if rec.Status != ProgressMastered {
		rec.Status = ProgressLearning
	}
	if a.Kind == AttemptPractice {
		rec.PracticeAttempts++
		rec.LastPractice = percent
	} else {
		rec.MasteryAttempts++
		rec.BestMastery = max(rec.BestMastery, percent)
		passed = a.Correct*100 >= MasteryPercent*a.Total
		if passed && rec.Status != ProgressMastered {
			rec.Status = ProgressMastered
			rec.MasteredAt = now
		}
	}
	if err := s.progress.Save(ctx, rec); err != nil {
		return Progress{}, false, fmt.Errorf("grammar: save progress: %w", err)
	}
	return rec, passed, nil
}

func checkAttempt(c Content, a Attempt) map[string]string {
	f := fieldErrors{}
	var list []Exercise
	switch a.Kind {
	case AttemptPractice:
		list = c.Practice
	case AttemptMastery:
		list = c.Mastery
	default:
		f.add("kind", "Loại lượt làm không hợp lệ")
		return f
	}
	if a.Total != len(list) {
		f.add("total", fmt.Sprintf("Số bài phải là %d", len(list)))
	}
	if a.Correct < 0 || a.Correct > a.Total {
		f.add("correct", "Số câu đúng phải từ 0 đến tổng số bài")
	}
	ids := map[string]bool{}
	for _, e := range list {
		ids[e.ID] = true
	}
	seen := map[string]bool{}
	for _, id := range a.Wrong {
		if !ids[id] || seen[id] {
			f.add("wrong", "Danh sách câu sai không hợp lệ")
		}
		seen[id] = true
	}
	if len(a.Wrong) != a.Total-a.Correct {
		f.add("wrong", "Số câu sai không khớp với số câu đúng")
	}
	if len(f) == 0 {
		return nil
	}
	return f
}
