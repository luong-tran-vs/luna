package lesson

import (
	"context"
	"maps"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/luongtran/luna/backend/internal/ai"
	"github.com/luongtran/luna/backend/internal/job"
)

// fakeLessons is an in-memory Repository.
type fakeLessons struct {
	mu     sync.Mutex
	byID   map[string]Lesson
	nextID int
}

func newFakeLessons() *fakeLessons { return &fakeLessons{byID: map[string]Lesson{}} }

func (f *fakeLessons) Create(_ context.Context, l Lesson) (Lesson, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.nextID++
	l.ID = "l" + strconv.Itoa(f.nextID)
	f.byID[l.ID] = l
	return l, nil
}

func (f *fakeLessons) Get(_ context.Context, id string) (Lesson, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	l, ok := f.byID[id]
	if !ok {
		return Lesson{}, ErrNotFound
	}
	l.Sentences = slices.Clone(l.Sentences)
	l.Annotations = slices.Clone(l.Annotations)
	return l, nil
}

func summaryOf(l Lesson) Summary {
	return Summary{
		ID: l.ID, Title: l.Title, Level: l.Level, TopicID: l.TopicID,
		AnnotationStatus: l.AnnotationStatus, CreatedAt: l.CreatedAt,
	}
}

func (f *fakeLessons) List(_ context.Context, flt Filter) ([]Summary, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []Summary
	for _, l := range f.byID {
		if (flt.Level == "" || l.Level == flt.Level) && (flt.TopicID == "" || l.TopicID == flt.TopicID) {
			out = append(out, summaryOf(l))
		}
	}
	slices.SortFunc(out, func(a, b Summary) int { return b.CreatedAt.Compare(a.CreatedAt) })
	return out, nil
}

func (f *fakeLessons) Summaries(_ context.Context, ids []string) ([]Summary, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []Summary
	for _, id := range ids {
		if l, ok := f.byID[id]; ok {
			out = append(out, summaryOf(l))
		}
	}
	return out, nil
}

func (f *fakeLessons) update(id string, fn func(*Lesson) bool) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	l, ok := f.byID[id]
	if !ok {
		return false, ErrNotFound
	}
	if !fn(&l) {
		return false, nil
	}
	f.byID[id] = l
	return true, nil
}

func (f *fakeLessons) UpdateInfo(_ context.Context, id string, in Info) error {
	_, err := f.update(id, func(l *Lesson) bool {
		l.Title, l.Level, l.TopicID, l.Source, l.License = in.Title, in.Level, in.TopicID, in.Source, in.License
		return true
	})
	return err
}

func (f *fakeLessons) ReplaceContent(_ context.Context, next Lesson) error {
	_, err := f.update(next.ID, func(l *Lesson) bool { *l = next; return true })
	return err
}

func (f *fakeLessons) SetStatus(_ context.Context, id string, rev int, t job.Type, st Status, msg string) (bool, error) {
	return f.update(id, func(l *Lesson) bool {
		if l.Revision != rev {
			return false
		}
		switch t {
		case job.TypePractice:
			l.PracticeStatus, l.PracticeError = st, msg
		default:
			l.AnnotationStatus, l.AnnotationError = st, msg
		}
		return true
	})
}

func (f *fakeLessons) SaveAnnotations(_ context.Context, id string, rev int, anns []Annotation, x Extras) (bool, error) {
	return f.update(id, func(l *Lesson) bool {
		if l.Revision != rev {
			return false
		}
		l.Annotations, l.AnnotationStatus, l.AnnotationError = anns, StatusDone, ""
		l.Extras, l.ExtrasEditedByAdmin = x, false
		l.QuizVersion++
		l.Practice, l.PracticeStatus, l.PracticeError = nil, StatusRunning, ""
		return true
	})
}

func (f *fakeLessons) ReplaceAnnotations(_ context.Context, id string, anns []Annotation) error {
	_, err := f.update(id, func(l *Lesson) bool {
		l.Annotations, l.AnnotationStatus, l.AnnotationError = anns, StatusDone, ""
		return true
	})
	return err
}

func (f *fakeLessons) ReplaceExtras(_ context.Context, id string, x Extras, bumpQuiz bool) error {
	_, err := f.update(id, func(l *Lesson) bool {
		l.Extras, l.ExtrasEditedByAdmin = x, true
		if bumpQuiz {
			l.QuizVersion++
		}
		return true
	})
	return err
}

func (f *fakeLessons) Delete(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.byID, id)
	return nil
}

// fakeTopics is an in-memory Topics port: topic-a1 (A1 Family), topic-b1 (B1 Work), and a
// roadmap per topic.
type fakeTopics struct {
	mu         sync.Mutex
	topics     map[string]TopicRef
	roadmaps   map[string][]string
	moves      []string // "lesson:from>to"
	failAppend error
}

func newFakeTopics() *fakeTopics {
	return &fakeTopics{
		topics: map[string]TopicRef{
			"topic-a1": {ID: "topic-a1", Name: "Family", Level: "A1"},
			"topic-b1": {ID: "topic-b1", Name: "Work", Level: "B1"},
		},
		roadmaps: map[string][]string{},
	}
}

func (f *fakeTopics) Get(_ context.Context, id string) (TopicRef, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	t, ok := f.topics[id]
	if !ok {
		return TopicRef{}, ErrTopicNotFound
	}
	return t, nil
}

func (f *fakeTopics) Names(context.Context) (map[string]TopicRef, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return maps.Clone(f.topics), nil
}

func (f *fakeTopics) RoadmapLessonIDs(context.Context) (map[string]bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := map[string]bool{}
	for _, ids := range f.roadmaps {
		for _, id := range ids {
			out[id] = true
		}
	}
	return out, nil
}

func (f *fakeTopics) MoveLesson(_ context.Context, lessonID, from, to string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.moves = append(f.moves, lessonID+":"+from+">"+to)
	if i := slices.Index(f.roadmaps[from], lessonID); i >= 0 {
		f.roadmaps[from] = slices.Delete(f.roadmaps[from], i, i+1)
		f.roadmaps[to] = append(f.roadmaps[to], lessonID)
	}
	return nil
}

func (f *fakeTopics) setRoadmap(topicID string, ids ...string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.roadmaps[topicID] = ids
}

// fakeJobs records enqueued jobs and deletions.
type fakeJobs struct {
	mu            sync.Mutex
	enqueueErr    error
	jobs          []job.Job
	deletedPend   []string
	deletedLesson []string
}

func (f *fakeJobs) Enqueue(_ context.Context, j job.Job) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.enqueueErr != nil {
		return f.enqueueErr
	}
	f.jobs = append(f.jobs, j)
	return nil
}

func (f *fakeJobs) ClaimNext(context.Context, time.Time) (job.Job, bool, error) {
	return job.Job{}, false, nil
}
func (f *fakeJobs) Complete(context.Context, string) error                 { return nil }
func (f *fakeJobs) Retry(context.Context, string, time.Time, string) error { return nil }
func (f *fakeJobs) Fail(context.Context, string, string) error             { return nil }
func (f *fakeJobs) ResetRunning(context.Context) (int64, error)            { return 0, nil }
func (f *fakeJobs) DeletePending(_ context.Context, id string) error {
	f.record(&f.deletedPend, id)
	return nil
}

func (f *fakeJobs) DeleteForLesson(_ context.Context, id string) error {
	f.record(&f.deletedLesson, id)
	return nil
}

func (f *fakeJobs) record(list *[]string, id string) {
	f.mu.Lock()
	*list = append(*list, id)
	f.mu.Unlock()
}

func (f *fakeJobs) all() []job.Job {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.jobs)
}

// fakeAI returns fixed annotations and counts calls.
type fakeAI struct {
	mu     sync.Mutex
	calls  int
	result []ai.Annotation
	extras ai.LessonExtras
	err    error

	annotateReq ai.AnnotateRequest

	drafts   []ai.LessonDraft
	genErr   error
	genBlock bool
	genCalls int
	genReq   ai.GenerateRequest

	explanation  ai.Explanation
	explainErr   error
	explainGate  chan struct{}
	explainCalls int
	explainReq   ai.ExplainRequest

	practice      ai.Practice
	practiceErr   error
	practiceCalls int
	practiceReq   ai.PracticeRequest
}

// Annotate returns result as annotations plus the configured extras, and records the request.
func (f *fakeAI) Annotate(_ context.Context, req ai.AnnotateRequest) (ai.LessonExtras, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	f.annotateReq = req
	x := f.extras
	x.Annotations = f.result
	return x, f.err
}

// GenerateLessons returns the configured drafts or error and records the request. With
// block set it waits until the context ends.
func (f *fakeAI) GenerateLessons(ctx context.Context, req ai.GenerateRequest) ([]ai.LessonDraft, error) {
	f.mu.Lock()
	f.genCalls++
	f.genReq = req
	block, drafts, err := f.genBlock, f.drafts, f.genErr
	f.mu.Unlock()
	if block {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	return drafts, err
}

func (f *fakeAI) lastRequest() (ai.GenerateRequest, int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.genReq, f.genCalls
}

// AppendLesson adds a lesson at the end of a topic roadmap unless it is there; failAppend
// makes it fail.
func (f *fakeTopics) AppendLesson(_ context.Context, topicID, lessonID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failAppend != nil {
		return f.failAppend
	}
	if !slices.Contains(f.roadmaps[topicID], lessonID) {
		f.roadmaps[topicID] = append(f.roadmaps[topicID], lessonID)
	}
	return nil
}

func (f *fakeTopics) roadmap(topicID string) []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.roadmaps[topicID])
}

// fakeAnswers is an in-memory AnswerRepository with the unique key of reading_answers.
type fakeAnswers struct {
	mu   sync.Mutex
	rows []Answer
}

func newFakeAnswers() *fakeAnswers { return &fakeAnswers{} }

func (f *fakeAnswers) Insert(_ context.Context, a Answer) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, r := range f.rows {
		if r.UserID == a.UserID && r.LessonID == a.LessonID && r.QuizVersion == a.QuizVersion && r.QuestionIndex == a.QuestionIndex {
			return &AlreadyAnsweredError{Answer: r}
		}
	}
	f.rows = append(f.rows, a)
	return nil
}

func (f *fakeAnswers) List(_ context.Context, userID, lessonID string, version int) ([]Answer, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []Answer
	for _, r := range f.rows {
		if r.UserID == userID && r.LessonID == lessonID && r.QuizVersion == version {
			out = append(out, r)
		}
	}
	slices.SortFunc(out, func(a, b Answer) int { return a.QuestionIndex - b.QuestionIndex })
	return out, nil
}

func (f *fakeAnswers) Totals(_ context.Context, userID string) (answered, correct int, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, r := range f.rows {
		if r.UserID == userID {
			answered++
			if r.Correct {
				correct++
			}
		}
	}
	return answered, correct, nil
}

// GradeWriting is unused by lessons (F8 grades writings).
func (f *fakeAI) GradeWriting(context.Context, ai.GradeRequest) (ai.Grade, error) {
	return ai.Grade{}, nil
}

// Explain returns the configured explanation or error and counts calls. With explainGate set it
// waits on the channel (or the context) first.
func (f *fakeAI) Explain(ctx context.Context, req ai.ExplainRequest) (ai.Explanation, error) {
	f.mu.Lock()
	f.explainCalls++
	f.explainReq = req
	gate, res, err := f.explainGate, f.explanation, f.explainErr
	f.mu.Unlock()
	if gate != nil {
		select {
		case <-gate:
		case <-ctx.Done():
			return ai.Explanation{}, ctx.Err()
		}
	}
	return res, err
}

func (f *fakeAI) explains() (int, ai.ExplainRequest) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.explainCalls, f.explainReq
}

// fakeAsks is an in-memory AskRepository.
type fakeAsks struct {
	mu   sync.Mutex
	rows map[AskKey]AskResult
	puts int
}

func newFakeAsks() *fakeAsks { return &fakeAsks{rows: map[AskKey]AskResult{}} }

func (f *fakeAsks) Get(_ context.Context, key AskKey) (AskResult, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	r, ok := f.rows[key]
	return r, ok, nil
}

func (f *fakeAsks) Put(_ context.Context, r AskResult) (AskResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.puts++
	if old, ok := f.rows[r.AskKey]; ok {
		return old, nil
	}
	f.rows[r.AskKey] = r
	return r, nil
}

func (f *fakeAsks) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.rows)
}

func (f *fakeLessons) SavePractice(_ context.Context, id string, rev, prevVersion int, p Practice) (bool, error) {
	return f.update(id, func(l *Lesson) bool {
		if l.Revision != rev || l.PracticeVersion != prevVersion {
			return false
		}
		l.Practice, l.PracticeStatus, l.PracticeError = &p, StatusDone, ""
		l.PracticeVersion++
		return true
	})
}

func (f *fakeLessons) WithoutPractice(context.Context) ([]RevisionRef, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []RevisionRef
	for _, l := range f.byID {
		if l.AnnotationStatus == StatusDone && l.PracticeStatus == StatusNone {
			out = append(out, RevisionRef{ID: l.ID, Revision: l.Revision})
		}
	}
	return out, nil
}

// Position is the 1-based place of a lesson in the topic roadmap, 0 when absent.
func (f *fakeTopics) Position(_ context.Context, topicID, lessonID string) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Index(f.roadmaps[topicID], lessonID) + 1, nil
}

// Practice returns the configured practice or error, records the request and counts calls.
func (f *fakeAI) Practice(_ context.Context, req ai.PracticeRequest) (ai.Practice, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.practiceCalls++
	f.practiceReq = req
	return f.practice, f.practiceErr
}

func (f *fakeAI) practices() (int, ai.PracticeRequest) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.practiceCalls, f.practiceReq
}

func (f *fakeAI) SuggestWords(context.Context, ai.SuggestWordsRequest) ([]string, error) {
	return nil, ai.ErrNotConfigured
}
