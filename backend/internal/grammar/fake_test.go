package grammar

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/luongtran/luna/backend/internal/ai"
)

type fakeLessons struct {
	mu      sync.Mutex
	lessons map[string]Lesson
}

func (f *fakeLessons) Get(_ context.Context, id string) (Lesson, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	l, ok := f.lessons[id]
	if !ok {
		return Lesson{}, ErrNotFound
	}
	return l, nil
}

func (f *fakeLessons) Save(_ context.Context, l Lesson) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.lessons[l.PointID] = l
	return nil
}

func (f *fakeLessons) List(context.Context) ([]LessonSummary, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []LessonSummary
	for _, l := range f.lessons {
		out = append(out, LessonSummary{PointID: l.PointID, Status: l.Status, Edited: l.Edited, Flags: unconfirmed(l.Checks), Verified: !l.VerifiedAt.IsZero(), Checked: !l.CheckedAt.IsZero(), UpdatedAt: l.UpdatedAt, PublishedAt: l.PublishedAt})
	}
	return out, nil
}

type fakeProgress struct {
	mu   sync.Mutex
	recs map[[2]string]Progress
}

func (f *fakeProgress) Get(_ context.Context, userID, pointID string) (Progress, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	p, ok := f.recs[[2]string{userID, pointID}]
	if !ok {
		return Progress{}, ErrNotFound
	}
	return p, nil
}

func (f *fakeProgress) List(_ context.Context, userID string) ([]Progress, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []Progress
	for k, p := range f.recs {
		if k[0] == userID {
			out = append(out, p)
		}
	}
	return out, nil
}

func (f *fakeProgress) Save(_ context.Context, p Progress) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.recs[[2]string{p.UserID, p.PointID}] = p
	return nil
}

type fakeAI struct {
	ai.Disabled
	mu      sync.Mutex
	content ai.GrammarLessonContent
	err     error
	calls   int
	req     ai.GrammarLessonRequest

	// solve answers SolveGrammarExercises; nil solves every exercise like the fixtures key.
	solve      func(ai.SolveRequest) ([]ai.Solution, error)
	solveCalls int
	solveReq   ai.SolveRequest
}

func (f *fakeAI) GrammarLesson(_ context.Context, req ai.GrammarLessonRequest) (ai.GrammarLessonContent, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	f.req = req
	return f.content, f.err
}

type fakeCounter struct{ counts map[string]int }

func (f fakeCounter) CountByGrammarPoint(context.Context, string) (map[string]int, error) {
	return f.counts, nil
}

var testNow = time.Date(2026, 10, 5, 8, 0, 0, 0, time.UTC)

type env struct {
	svc      *Service
	lessons  *fakeLessons
	progress *fakeProgress
	reports  *fakeReports
	ai       *fakeAI
	now      time.Time
}

func newEnv(t *testing.T) *env {
	t.Helper()
	e := &env{
		lessons:  &fakeLessons{lessons: map[string]Lesson{}},
		progress: &fakeProgress{recs: map[[2]string]Progress{}},
		reports:  &fakeReports{},
		ai:       &fakeAI{content: aiContent()},
		now:      testNow,
	}
	e.svc = NewService(e.lessons, e.progress, e.reports, e.ai, fakeCounter{counts: map[string]int{"a1-to-be": 3}}, func() time.Time { return e.now })
	return e
}

func goodChoice(n string) ai.GrammarExercise {
	return ai.GrammarExercise{
		Kind: "choice", PromptVi: "Chọn", Text: "She ___ a teacher " + n + ".", Options: []string{"am", "is", "are", "be"},
		AnswerIndex: 1, ExplanationVi: "Chủ ngữ she đi với is.",
	}
}

func goodFill(n string) ai.GrammarExercise {
	return ai.GrammarExercise{
		Kind: "fill", PromptVi: "Điền", Text: "I ___ a student " + n + ".", Answers: []string{"am"}, ExplanationVi: "I đi với am.",
	}
}

func goodReorder(n string) ai.GrammarExercise {
	return ai.GrammarExercise{
		Kind: "reorder", PromptVi: "Sắp xếp", Text: "Cô ấy là giáo viên " + n, Sentence: "She is a teacher",
		Words: []string{"teacher", "a", "She", "is", "are"}, ExplanationVi: "Câu khẳng định với is.",
	}
}

// mod returns e changed by f.
func mod(e ai.GrammarExercise, f func(*ai.GrammarExercise)) ai.GrammarExercise {
	f(&e)
	return e
}

// exercises returns n valid exercises that mix the three kinds; prefix keeps the texts apart.
func exercises(prefix string, n int) []ai.GrammarExercise {
	var out []ai.GrammarExercise
	for i := range n {
		tag := prefix + string(rune('a'+i))
		switch i % 3 {
		case 0:
			out = append(out, goodChoice(tag))
		case 1:
			out = append(out, goodFill(tag))
		default:
			out = append(out, goodReorder(tag))
		}
	}
	return out
}

func aiContent() ai.GrammarLessonContent {
	return ai.GrammarLessonContent{
		Objective:   "Bạn có thể nói về bản thân.",
		Explanation: []string{"Động từ to be có ba dạng."},
		Usage:       []string{"Dùng để nói về danh tính."},
		Structures:  []ai.GrammarStructure{{Label: "Khẳng định", Pattern: "S + am/is/are", Example: "I am a student."}},
		Examples: []ai.GrammarExample{
			{En: "I am a student.", Vi: "Tôi là học sinh."}, {En: "She is a nurse.", Vi: "Cô ấy là y tá."},
			{En: "They are friends.", Vi: "Họ là bạn."},
		},
		Mistakes: []ai.GrammarMistake{{Wrong: "She are a teacher.", Right: "She is a teacher.", NoteVi: "She đi với is."}},
		Practice: exercises("p", 8),
		Mastery:  exercises("m", 6),
	}
}

func ids(list []Exercise) []string {
	out := make([]string, len(list))
	for i, e := range list {
		out[i] = e.ID
	}
	return out
}

func (f *fakeAI) SolveGrammarExercises(_ context.Context, req ai.SolveRequest) ([]ai.Solution, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.solveCalls++
	f.solveReq = req
	if f.solve != nil {
		return f.solve(req)
	}
	return solveLikeFixtures(req), nil
}

// solveLikeFixtures answers every exercise the way goodChoice, goodFill and goodReorder are keyed.
func solveLikeFixtures(req ai.SolveRequest) []ai.Solution {
	var out []ai.Solution
	for _, e := range req.Exercises {
		s := ai.Solution{ExerciseID: e.ID}
		switch e.Kind {
		case "choice":
			one := 1
			s.ChoiceIndex = &one
		case "fill":
			s.Answer = "am"
		case "reorder":
			s.Sentence = "She is a teacher"
		}
		out = append(out, s)
	}
	return out
}

type fakeReports struct {
	mu   sync.Mutex
	rows []Report
}

func (f *fakeReports) Upsert(_ context.Context, r Report) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i, old := range f.rows {
		if old.UserID == r.UserID && old.PointID == r.PointID && old.ExerciseID == r.ExerciseID {
			r.ID, r.CreatedAt = old.ID, old.CreatedAt
			f.rows[i] = r
			return nil
		}
	}
	r.ID = fmt.Sprintf("r%d", len(f.rows)+1)
	f.rows = append(f.rows, r)
	return nil
}

func (f *fakeReports) ListOpen(context.Context) ([]Report, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []Report
	for _, r := range f.rows {
		if r.Status == ReportOpen {
			out = append(out, r)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].UpdatedAt.After(out[j].UpdatedAt) })
	return out, nil
}

func (f *fakeReports) Resolve(_ context.Context, pointID, exerciseID string, at time.Time) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for i, r := range f.rows {
		if r.PointID == pointID && r.ExerciseID == exerciseID && r.Status == ReportOpen {
			f.rows[i].Status, f.rows[i].ResolvedAt = ReportResolved, at
			n++
		}
	}
	return n, nil
}
