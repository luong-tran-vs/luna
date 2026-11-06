package grammar

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"sync"
	"time"

	"github.com/luongtran/luna/backend/internal/ai"
)

// generateTimeout bounds the AI call that writes one grammar lesson.
const generateTimeout = 5 * time.Minute

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

// StudyDays counts today as a study day for the streak (implemented over progress in
// internal/service).
type StudyDays interface {
	MarkStudied(ctx context.Context, userID string) error
}

// Service serves the grammar lessons: the admin side (generate, edit, publish) and the learner
// side (list, study, record attempts).
type Service struct {
	lessons  LessonRepository
	progress ProgressRepository
	reports  ReportRepository
	ai       ai.Provider
	counter  LessonCounter
	// studyDays counts a practice or test for the streak; nil counts none.
	studyDays StudyDays
	syllabus  *Syllabus
	now       func() time.Time
	// course orders a level's points as its fixed course teaches them; nil keeps the syllabus order.
	course CourseOrder
	// images keeps and draws the pictures of the examples (images.go); imgMu guards imgRun, the
	// running or last picture batch (images_batch.go).
	images ImageDeps
	imgMu  sync.Mutex
	imgRun *imageRun
	// picMu guards picRun, the running or last batch adding picture exercises (pictures.go).
	picMu  sync.Mutex
	picRun *pictureRun
}

// NewService returns a Service over the embedded syllabus.
func NewService(lessons LessonRepository, progress ProgressRepository, reports ReportRepository, provider ai.Provider, counter LessonCounter,
	studyDays StudyDays, now func() time.Time,
) *Service {
	return &Service{
		lessons: lessons, progress: progress, reports: reports, ai: provider, counter: counter, studyDays: studyDays, syllabus: Default(), now: now,
	}
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
		l, ok := byID[p.ID]
		if !ok {
			continue
		}
		if p.Drawable {
			if err := s.countPictures(ctx, &l); err != nil {
				return nil, err
			}
		}
		out = append(out, l)
	}
	return out, nil
}

// countPictures fills in the picture exercises of a lesson summary and how many are drawn.
func (s *Service) countPictures(ctx context.Context, sum *LessonSummary) error {
	l, err := s.lessons.Get(ctx, sum.PointID)
	if errors.Is(err, ErrNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("grammar: get lesson: %w", err)
	}
	have, err := s.imageKeys(ctx, sum.PointID)
	if err != nil {
		return err
	}
	for _, e := range append(append([]Exercise(nil), l.Content.Practice...), l.Content.Mastery...) {
		if e.Kind != KindPicture {
			continue
		}
		sum.Pictures++
		if have[PictureKey(e)] {
			sum.PicturesDrawn++
		}
	}
	sum.PicturesComplete = hasAllPictures(l.Content)
	return nil
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
	req := ai.GrammarLessonRequest{
		Level: p.Level, TitleVi: p.TitleVi, TitleEn: p.TitleEn, Pattern: p.Pattern, HintVi: p.HintVi, Examples: p.Examples,
		Pictures: p.Drawable,
	}
	for _, m := range MistakesOf(pointID) {
		req.CommonMistakes = append(req.CommonMistakes, ai.GrammarMistake{Wrong: m.Wrong, Right: m.Right, NoteVi: m.NoteVi})
	}
	raw, err := s.ai.GrammarLesson(aiCtx, req)
	if err != nil {
		if errors.Is(err, ai.ErrNotConfigured) || errors.Is(err, ai.ErrInvalidKey) || errors.Is(err, ai.ErrQuota) {
			return Lesson{}, false, err
		}
		return Lesson{}, false, fmt.Errorf("%w: %w", errAI, err)
	}
	content := FilterGenerated(fromAI(raw))
	if problems := Validate(content); problems != nil {
		slog.WarnContext(ctx, "grammar lesson from the AI is unusable", slog.String("point", pointID), slog.Any("problems", problems))
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

// Save replaces the content of an existing lesson with the admin's version. The status stays. The AI
// check is about the exercises only, so it is dropped only when an exercise changed (added
// 2026-11-02: before, any save dropped it, even for a fix in the theory).
func (s *Service) Save(ctx context.Context, pointID string, content Content) (Lesson, error) {
	l, err := s.AdminGet(ctx, pointID)
	if err != nil {
		return Lesson{}, err
	}
	content = Normalize(content)
	if fields := Validate(content); fields != nil {
		return Lesson{}, &ValidationError{Fields: fields}
	}
	exercisesChanged := !sameExercises(l.Content.Practice, content.Practice) || !sameExercises(l.Content.Mastery, content.Mastery)
	l.Content = content
	l.Edited = true
	if exercisesChanged {
		l.Checks, l.CheckedAt, l.VerifiedAt = nil, time.Time{}, time.Time{}
	}
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
			Answers: e.Answers, Words: e.Words, Sentence: e.Sentence, ExplanationVi: e.ExplanationVi, Picture: e.Picture,
		}
		// Models sometimes put the choices of a picture exercise in words (seen 2026-11-06).
		if out[i].Kind == KindPicture && len(out[i].Options) == 0 && len(out[i].Words) > 0 {
			out[i].Options = out[i].Words
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
	// WeakCount is how many practice exercises the latest practice round got wrong, to redo.
	WeakCount int
	// Course is the first topic of the level's course that teaches the point; nil when none.
	Course *CourseTopic
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
			ps.Status, ps.BestMastery, ps.WeakCount = rec.Status, rec.BestMastery, len(rec.Weak)
		}
		out.Points = append(out.Points, ps)
	}
	if level != "" && s.course != nil {
		if topics, order, ok := s.course(ctx, level); ok {
			out.Points = inCourseOrder(out.Points, topics, order)
		}
	}
	for _, ps := range out.Points {
		if ps.Available && ps.Status != ProgressMastered {
			out.Next = ps.ID
			break
		}
	}
	return out, nil
}

// MasteredCount counts the points the learner mastered at or after since (nil = every mastered
// point), for the stats page. Mastering a point (passing its mastery check) is what finishes a
// grammar lesson, and MasteredAt is set once, on the first pass, so it dates the finish.
func (s *Service) MasteredCount(ctx context.Context, userID string, since *time.Time) (int, error) {
	records, err := s.progress.List(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("grammar: list progress: %w", err)
	}
	n := 0
	for _, p := range records {
		if p.Status == ProgressMastered && (since == nil || !p.MasteredAt.Before(*since)) {
			n++
		}
	}
	return n, nil
}

// Detail is the study page of a published point.
type Detail struct {
	Point Point
	// Pictures gives the picture key of each picture exercise of Content, by exercise id.
	Pictures    map[string]string
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
	content, pictures, err := s.learnerContent(ctx, pointID, l.Content)
	if err != nil {
		return Detail{}, err
	}
	return Detail{Point: p, Content: expandFillAnswers(content), Pictures: pictures, Progress: rec, LessonCount: counts[pointID]}, nil
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
	content, _, err := s.learnerContent(ctx, pointID, l.Content)
	if err != nil {
		return Progress{}, false, err
	}
	if fields := checkAttempt(content, a); fields != nil {
		return Progress{}, false, &ValidationError{Fields: fields}
	}
	rec, err = s.progressOf(ctx, userID, pointID)
	if err != nil {
		return Progress{}, false, err
	}

	now := s.now()
	percent := (a.Correct*100 + a.Total/2) / a.Total
	rec.UserID, rec.PointID, rec.UpdatedAt = userID, pointID, now
	// Only a practice round decides what to redo: the wrong answers of a test are reviewed on its
	// result screen and cannot be redone as practice, so a test leaves the list as it was.
	if a.Kind == AttemptPractice {
		rec.Weak = slices.Clone(a.Wrong)
	}
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
	// A practice or test done counts the day for the streak.
	if s.studyDays != nil {
		if err := s.studyDays.MarkStudied(ctx, userID); err != nil {
			return Progress{}, false, fmt.Errorf("grammar: study day: %w", err)
		}
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

// sameExercises says whether two lists hold the same exercises in the same order, every field
// alike (a missing list and an empty one count as the same).
func sameExercises(a, b []Exercise) bool {
	return slices.EqualFunc(a, b, func(x, y Exercise) bool {
		return x.ID == y.ID && x.Kind == y.Kind && x.PromptVi == y.PromptVi && x.Text == y.Text &&
			slices.Equal(x.Options, y.Options) && x.AnswerIndex == y.AnswerIndex && slices.Equal(x.Answers, y.Answers) &&
			slices.Equal(x.Words, y.Words) && x.Sentence == y.Sentence && x.ExplanationVi == y.ExplanationVi && x.Picture == y.Picture
	})
}
