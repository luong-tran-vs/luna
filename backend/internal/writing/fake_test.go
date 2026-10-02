package writing

import (
	"context"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/luongtran/luna/backend/internal/ai"
	"github.com/luongtran/luna/backend/internal/job"
)

// fakeRepo is an in-memory Repository with the unique (user, lesson) key of "writings".
type fakeRepo struct {
	mu     sync.Mutex
	byID   map[string]Writing
	nextID int
}

func newFakeRepo() *fakeRepo { return &fakeRepo{byID: map[string]Writing{}} }

func (f *fakeRepo) find(userID, lessonID string) (Writing, bool) {
	for _, w := range f.byID {
		if w.UserID == userID && w.LessonID == lessonID {
			return w, true
		}
	}
	return Writing{}, false
}

func (f *fakeRepo) GetByLesson(_ context.Context, userID, lessonID string) (Writing, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	w, ok := f.find(userID, lessonID)
	return w, ok, nil
}

func (f *fakeRepo) Get(_ context.Context, id string) (Writing, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	w, ok := f.byID[id]
	if !ok {
		return Writing{}, ErrNotFound
	}
	return w, nil
}

func (f *fakeRepo) SaveDraft(_ context.Context, userID, lessonID, text string, now time.Time) (Writing, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	w, ok := f.find(userID, lessonID)
	if ok && w.Status == StatusSubmitted {
		return Writing{}, ErrSubmitted
	}
	if !ok {
		f.nextID++
		w = Writing{ID: "w" + strconv.Itoa(f.nextID), UserID: userID, LessonID: lessonID, Status: StatusDraft, CreatedAt: now}
	}
	w.Text, w.UpdatedAt = text, now
	f.byID[w.ID] = w
	return w, nil
}

func (f *fakeRepo) Submit(_ context.Context, in Writing) (Writing, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	w, ok := f.find(in.UserID, in.LessonID)
	if ok && w.Status == StatusSubmitted {
		return Writing{}, ErrSubmitted
	}
	if !ok {
		f.nextID++
		w = Writing{ID: "w" + strconv.Itoa(f.nextID), UserID: in.UserID, LessonID: in.LessonID, CreatedAt: in.SubmittedAt}
	}
	w.LessonRevision, w.LessonTitle, w.Prompt, w.Text = in.LessonRevision, in.LessonTitle, in.Prompt, in.Text
	w.Status, w.SubmittedAt, w.UpdatedAt = StatusSubmitted, in.SubmittedAt, in.SubmittedAt
	w.Grade = &Grade{Status: GradePending, Seen: true}
	f.byID[w.ID] = w
	return w, nil
}

func (f *fakeRepo) SetGrade(_ context.Context, id string, g Grade) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	w, ok := f.byID[id]
	if !ok {
		return ErrNotFound
	}
	w.Grade = &g
	f.byID[id] = w
	return nil
}

func (f *fakeRepo) MarkSeen(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	w, ok := f.byID[id]
	if !ok {
		return ErrNotFound
	}
	if w.Grade != nil {
		g := *w.Grade
		g.Seen = true
		w.Grade = &g
	}
	f.byID[id] = w
	return nil
}

func (f *fakeRepo) submitted(userID string) []Writing {
	var out []Writing
	for _, w := range f.byID {
		if w.UserID == userID && w.Status == StatusSubmitted {
			out = append(out, w)
		}
	}
	slices.SortFunc(out, func(a, b Writing) int { return b.SubmittedAt.Compare(a.SubmittedAt) })
	return out
}

func (f *fakeRepo) List(_ context.Context, userID string) ([]Writing, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.submitted(userID), nil
}

func (f *fakeRepo) Unseen(_ context.Context, userID string) (UnseenCount, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out UnseenCount
	for _, w := range f.submitted(userID) {
		switch {
		case w.Grade.Status == GradePending:
			out.Pending++
		case !w.Grade.Seen:
			out.Unseen++
			if out.Latest == nil {
				out.Latest = &Latest{ID: w.ID, Status: w.Grade.Status}
			}
		}
	}
	return out, nil
}

func (f *fakeRepo) Stats(_ context.Context, userID string) (int, *float64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	list := f.submitted(userID)
	sum, n := 0.0, 0
	for _, w := range list {
		if a := Average(w.Grade); a != nil {
			sum += *a
			n++
		}
	}
	if n == 0 {
		return len(list), nil, nil
	}
	avg := sum / float64(n)
	return len(list), &avg, nil
}

// fakeLessons knows lesson "l1" (A1 with a prompt) and "l2" (B1 without one).
type fakeLessons map[string]LessonInfo

func newFakeLessons() fakeLessons {
	return fakeLessons{
		"l1": {Title: "My family", Level: "A1", Content: "Tom has a big family.", WritingPrompt: "Write about your family.", Revision: 2},
		"l2": {Title: "At work", Level: "B1", Content: "I work in an office.", Revision: 1},
	}
}

func (f fakeLessons) Info(_ context.Context, id string) (LessonInfo, error) {
	l, ok := f[id]
	if !ok {
		return LessonInfo{}, ErrNotFound
	}
	return l, nil
}

// fakeSteps allows writing for the "user/lesson" pairs in open.
type fakeSteps struct {
	mu   sync.Mutex
	open map[string]bool
}

func (f *fakeSteps) CanWrite(_ context.Context, userID, lessonID string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.open[userID+"/"+lessonID], nil
}

func (f *fakeSteps) set(userID, lessonID string, open bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.open[userID+"/"+lessonID] = open
}

// fakeJobs records enqueued jobs.
type fakeJobs struct {
	mu   sync.Mutex
	jobs []job.Job
}

func (f *fakeJobs) Enqueue(_ context.Context, j job.Job) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.jobs = append(f.jobs, j)
	return nil
}

func (f *fakeJobs) all() []job.Job {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.jobs)
}

// fakeGrader returns a fixed grade or error and records the request.
type fakeGrader struct {
	mu    sync.Mutex
	grade ai.Grade
	err   error
	calls int
	req   ai.GradeRequest
}

func (f *fakeGrader) GradeWriting(_ context.Context, req ai.GradeRequest) (ai.Grade, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	f.req = req
	return f.grade, f.err
}

func goodGrade() ai.Grade {
	return ai.Grade{
		Task: ai.Criterion{Score: 4, CommentVi: "Đúng đề."}, Grammar: ai.Criterion{Score: 3, CommentVi: "Sai thì."},
		Vocabulary: ai.Criterion{Score: 4, CommentVi: "Đủ từ."}, Coherence: ai.Criterion{Score: 4, CommentVi: "Mạch lạc."},
		OverallVi: "Khá tốt.", CorrectedText: "My family has four members.",
	}
}

type env struct {
	svc     *Service
	repo    *fakeRepo
	steps   *fakeSteps
	jobs    *fakeJobs
	grader  *fakeGrader
	notify  int
	now     time.Time
	lessons fakeLessons
}

func newEnv() *env {
	e := &env{
		repo: newFakeRepo(), steps: &fakeSteps{open: map[string]bool{}}, jobs: &fakeJobs{},
		grader: &fakeGrader{grade: goodGrade()}, lessons: newFakeLessons(),
		now: time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC),
	}
	e.svc = NewService(Deps{
		Repo: e.repo, Lessons: e.lessons, Steps: e.steps, Jobs: e.jobs, AI: e.grader,
		Notify: func() { e.notify++ }, Now: func() time.Time { return e.now },
	})
	e.steps.set("u1", "l1", true)
	return e
}

// Explain is unused by writings (F9 explains words).
func (f *fakeGrader) Explain(context.Context, ai.ExplainRequest) (ai.Explanation, error) {
	return ai.Explanation{}, nil
}

// Practice is unused by writings (F17 is lesson practice).
func (f *fakeGrader) Practice(context.Context, ai.PracticeRequest) (ai.Practice, error) {
	return ai.Practice{}, nil
}
