package progress

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// StepState is how a step shows in the step indicator.
type StepState string

const (
	StateDone    StepState = "done"
	StateCurrent StepState = "current"
	StateLocked  StepState = "locked"
)

// StudyDeps are the collaborators of StudyService.
type StudyDeps struct {
	Goals     GoalRepository
	Progress  ProgressRepository
	Days      DayRepository
	Dictation *Service
	Lessons   Lessons
	Roadmaps  Roadmaps
	Titles    LessonTitles
	Reviews   Reviews
	Timezones Timezones
	// Quiz gates the Reading step on the comprehension questions (F15); nil means no questions.
	Quiz ReadingQuiz
	// Writings gates the Write step (F8); nil means a writing is never required.
	Writings Writings
	Now      func() time.Time
}

// StudyService runs the study flow (L): goals, the lesson being studied and its steps, streak.
// Lessons are studied one after another with no daily limit (updated 2026-10-02).
type StudyService struct {
	d StudyDeps
}

// NewStudyService returns a StudyService.
func NewStudyService(d StudyDeps) *StudyService {
	return &StudyService{d: d}
}

// GoalView is a goal with its topic and progress on the topic's current roadmap.
type GoalView struct {
	TopicID          string
	TopicName        string
	Level            string
	CompletedLessons int
	TotalLessons     int
	Status           GoalStatus
	EffectiveFrom    string
}

// GoalsView is the active goal and the paused ones.
type GoalsView struct {
	Active *GoalView
	Others []GoalView
}

// LessonRef identifies a lesson for the learner.
type LessonRef struct {
	ID    string
	Title string
}

// LessonStudyView is a lesson's steps for the learner, as the lesson page shows them.
type LessonStudyView struct {
	Status LessonStatus
	// Steps and CurrentStep are set for the lesson being studied and for completed lessons.
	Steps         map[Step]StepState
	CurrentStep   Step
	SentenceIndex int
	// Next is the lesson to study now, once this one is completed.
	Next          *LessonRef
	Goal          *GoalView
	GoalCompleted bool
	Streak        int
}

// CompletedLesson is a lesson in the "Đã học" list.
type CompletedLesson struct {
	LessonRef
	TopicName   string
	CompletedAt time.Time
}

// MyLessonsView is the lessons page.
type MyLessonsView struct {
	Current   *LessonRef
	Completed []CompletedLesson
	Upcoming  []LessonRef
}

// day is everything the rules need about a learner right now.
type day struct {
	userID    string
	now       time.Time
	loc       *time.Location
	today     string
	goal      *Goal
	topic     *TopicInfo
	goals     []Goal
	completed []LessonProgress
	done      map[string]bool
	state     StudyState
}

func (s *StudyService) load(ctx context.Context, userID string) (*day, error) {
	loc, err := s.d.Timezones.Location(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("progress: timezone: %w", err)
	}
	d := &day{userID: userID, now: s.d.Now(), loc: loc}
	d.today = DayKey(d.now, loc)
	if d.goals, err = s.d.Goals.List(ctx, userID); err != nil {
		return nil, fmt.Errorf("progress: goals: %w", err)
	}
	if g, ok := EffectiveGoal(d.goals); ok {
		t, err := s.d.Roadmaps.Roadmap(ctx, g.TopicID)
		switch {
		case err == nil:
			d.goal, d.topic = &g, &t
		case !errors.Is(err, ErrTopicNotFound):
			return nil, fmt.Errorf("progress: roadmap: %w", err)
		}
	}
	if d.completed, err = s.d.Progress.Completed(ctx, userID); err != nil {
		return nil, fmt.Errorf("progress: completed lessons: %w", err)
	}
	d.done = make(map[string]bool, len(d.completed))
	for _, p := range d.completed {
		d.done[p.LessonID] = true
	}
	var roadmap []string
	if d.topic != nil {
		roadmap = d.topic.LessonIDs
	}
	d.state = CurrentLesson(d.goal, roadmap, d.done)
	return d, nil
}

func goalView(g Goal, t TopicInfo, done map[string]bool) GoalView {
	v := GoalView{
		TopicID: g.TopicID, TopicName: t.Name, Level: t.Level, TotalLessons: len(t.LessonIDs),
		Status: g.Status, EffectiveFrom: g.EffectiveFrom,
	}
	for _, id := range t.LessonIDs {
		if done[id] {
			v.CompletedLessons++
		}
	}
	return v
}

// newProgress is the progress of a lesson started now.
func newProgress(d *day, lessonID string) LessonProgress {
	p := LessonProgress{UserID: d.userID, LessonID: lessonID, DayKey: d.today, StartedAt: d.now, Done: map[Step]bool{}}
	if d.goal != nil {
		p.TopicID = d.goal.TopicID
	}
	return p
}

// --- goals ---

// Goals returns the active goal and the paused ones; goals of deleted topics are left out.
func (s *StudyService) Goals(ctx context.Context, userID string) (GoalsView, error) {
	d, err := s.load(ctx, userID)
	if err != nil {
		return GoalsView{}, err
	}
	out := GoalsView{Others: []GoalView{}}
	for _, g := range d.goals {
		t, err := s.d.Roadmaps.Roadmap(ctx, g.TopicID)
		if errors.Is(err, ErrTopicNotFound) {
			continue
		}
		if err != nil {
			return GoalsView{}, fmt.Errorf("progress: roadmap: %w", err)
		}
		v := goalView(g, t, d.done)
		if g.Status == GoalActive {
			out.Active = &v
		} else {
			out.Others = append(out.Others, v)
		}
	}
	return out, nil
}

// SetGoal makes topicID the learner's goal at once: its first lesson not completed is the one to
// study now.
func (s *StudyService) SetGoal(ctx context.Context, userID, topicID string) (GoalView, error) {
	t, err := s.d.Roadmaps.Roadmap(ctx, topicID)
	if err != nil {
		return GoalView{}, fmt.Errorf("progress: roadmap: %w", err)
	}
	d, err := s.load(ctx, userID)
	if err != nil {
		return GoalView{}, err
	}
	g, err := s.d.Goals.Activate(ctx, userID, topicID, t.Level, d.today, d.now)
	if err != nil {
		return GoalView{}, fmt.Errorf("progress: activate goal: %w", err)
	}
	return goalView(g, t, d.done), nil
}

// --- the lesson page ---

// LessonStudy returns a lesson's steps for the learner. It only reads.
func (s *StudyService) LessonStudy(ctx context.Context, userID, lessonID string) (LessonStudyView, error) {
	d, err := s.load(ctx, userID)
	if err != nil {
		return LessonStudyView{}, err
	}
	return s.view(ctx, d, lessonID)
}

func (s *StudyService) view(ctx context.Context, d *day, lessonID string) (LessonStudyView, error) {
	v := LessonStudyView{Status: LessonOther, Steps: map[Step]StepState{}}
	switch {
	case d.done[lessonID]:
		v.Status = LessonCompleted
	case d.state.Kind == StudyStudying && d.state.LessonID == lessonID:
		v.Status = LessonStudying
	}
	if d.goal != nil {
		g := goalView(*d.goal, *d.topic, d.done)
		v.Goal = &g
		v.GoalCompleted = g.TotalLessons > 0 && g.CompletedLessons == g.TotalLessons
	}
	keys, err := s.d.Days.CompletedKeys(ctx, d.userID)
	if err != nil {
		return LessonStudyView{}, fmt.Errorf("progress: study days: %w", err)
	}
	v.Streak = Streak(keys, d.today)
	if v.Status == LessonCompleted && d.state.Kind == StudyStudying {
		titles, err := s.d.Titles.Titles(ctx, []string{d.state.LessonID})
		if err != nil {
			return LessonStudyView{}, fmt.Errorf("progress: lesson title: %w", err)
		}
		if title, ok := titles[d.state.LessonID]; ok {
			v.Next = &LessonRef{ID: d.state.LessonID, Title: title}
		}
	}
	if v.Status == LessonOther {
		return v, nil
	}

	p, _, err := s.d.Progress.Get(ctx, d.userID, lessonID)
	if err != nil {
		return LessonStudyView{}, fmt.Errorf("progress: lesson progress: %w", err)
	}
	v.CurrentStep = NextStep(p.Done)
	if v.Status == LessonCompleted {
		// A lesson completed before a step was added (F8 added Write) stays completed.
		v.CurrentStep = StepDone
	}
	for _, st := range Steps {
		switch {
		case p.Done[st] || v.CurrentStep == StepDone:
			v.Steps[st] = StateDone
		case st == v.CurrentStep:
			v.Steps[st] = StateCurrent
		default:
			v.Steps[st] = StateLocked
		}
	}
	if p.CurrentStep == v.CurrentStep {
		v.SentenceIndex = p.SentenceIndex
	}
	return v, nil
}

// CompleteStep records a step of the lesson being studied. Steps complete in order; completing a
// done step again changes nothing. The last step completes the lesson (goal +1, the day counts
// for the streak) and the next lesson opens at once.
func (s *StudyService) CompleteStep(ctx context.Context, userID, lessonID string, step Step) (LessonStudyView, error) {
	return s.finishStep(ctx, userID, lessonID, step, false)
}

// SkipWrite completes the Write step without a writing: writing is optional (updated
// 2026-10-02), so the learner may finish the lesson without it. Nothing is graded.
func (s *StudyService) SkipWrite(ctx context.Context, userID, lessonID string) (LessonStudyView, error) {
	return s.finishStep(ctx, userID, lessonID, StepWrite, true)
}

func (s *StudyService) finishStep(ctx context.Context, userID, lessonID string, step Step, skipWriting bool) (LessonStudyView, error) {
	if !ValidStep(step) {
		return LessonStudyView{}, &ValidationError{Fields: map[string]string{"step": "Bước không hợp lệ"}}
	}
	d, err := s.load(ctx, userID)
	if err != nil {
		return LessonStudyView{}, err
	}
	if d.done[lessonID] {
		return s.view(ctx, d, lessonID)
	}
	if d.state.Kind != StudyStudying || d.state.LessonID != lessonID {
		return LessonStudyView{}, ErrNotCurrentLesson
	}
	p, _, err := s.d.Progress.Get(ctx, userID, lessonID)
	if err != nil {
		return LessonStudyView{}, fmt.Errorf("progress: lesson progress: %w", err)
	}
	if p.Done[step] {
		return s.view(ctx, d, lessonID)
	}
	if NextStep(p.Done) != step {
		return LessonStudyView{}, ErrStepLocked
	}
	return s.completeStep(ctx, d, step, skipWriting)
}

// completeStep checks what the step needs and records it; skipWriting lets the Write step
// complete without a submitted writing.
func (s *StudyService) completeStep(ctx context.Context, d *day, step Step, skipWriting bool) (LessonStudyView, error) {
	lessonID := d.state.LessonID
	if step == StepRead && s.d.Quiz != nil {
		questions, answered, err := s.d.Quiz.Status(ctx, d.userID, lessonID)
		if err != nil {
			return LessonStudyView{}, fmt.Errorf("progress: reading quiz: %w", err)
		}
		if questions > 0 && answered < questions {
			return LessonStudyView{}, ErrReadIncomplete
		}
	}
	if step == StepWrite && s.d.Writings != nil && !skipWriting {
		submitted, err := s.d.Writings.Submitted(ctx, d.userID, lessonID)
		if err != nil {
			return LessonStudyView{}, fmt.Errorf("progress: writing: %w", err)
		}
		if !submitted {
			return LessonStudyView{}, ErrWriteIncomplete
		}
	}
	if step == StepListen {
		sum, err := s.d.Dictation.Summary(ctx, d.userID, lessonID)
		if err != nil {
			return LessonStudyView{}, fmt.Errorf("progress: dictation: %w", err)
		}
		if !sum.Completed {
			return LessonStudyView{}, ErrListenIncomplete
		}
	}
	p, ok, err := s.d.Progress.Get(ctx, d.userID, lessonID)
	if err != nil {
		return LessonStudyView{}, fmt.Errorf("progress: lesson progress: %w", err)
	}
	if !ok {
		p = newProgress(d, lessonID)
	}
	if p.Done == nil {
		p.Done = map[Step]bool{}
	}
	p.Done[step] = true
	p.CurrentStep, p.SentenceIndex = NextStep(p.Done), 0
	completed := p.CurrentStep == StepDone
	if completed {
		p.CompletedAt = d.now
	}
	if err := s.d.Progress.Upsert(ctx, p); err != nil {
		return LessonStudyView{}, fmt.Errorf("progress: save progress: %w", err)
	}
	if completed {
		if err := s.d.Days.MarkCompleted(ctx, d.userID, d.today); err != nil {
			return LessonStudyView{}, fmt.Errorf("progress: study day: %w", err)
		}
	}
	next, err := s.load(ctx, d.userID)
	if err != nil {
		return LessonStudyView{}, err
	}
	return s.view(ctx, next, lessonID)
}

// SetPosition saves the sentence of the current step of the lesson being studied; the first save
// starts the lesson.
func (s *StudyService) SetPosition(ctx context.Context, userID, lessonID string, step Step, sentence int) error {
	if !ValidStep(step) {
		return &ValidationError{Fields: map[string]string{"step": "Bước không hợp lệ"}}
	}
	d, err := s.load(ctx, userID)
	if err != nil {
		return err
	}
	if d.state.Kind != StudyStudying || d.state.LessonID != lessonID {
		return ErrNotCurrentLesson
	}
	p, ok, err := s.d.Progress.Get(ctx, userID, lessonID)
	if err != nil {
		return fmt.Errorf("progress: lesson progress: %w", err)
	}
	if NextStep(p.Done) != step {
		return ErrNotCurrentStep
	}
	_, count, err := s.d.Lessons.Info(ctx, lessonID)
	if err != nil {
		return fmt.Errorf("progress: lesson: %w", err)
	}
	if sentence < 0 || sentence >= count {
		return &ValidationError{Fields: map[string]string{"sentenceIndex": "Câu không có trong bài"}}
	}
	if !ok {
		if err := s.d.Progress.Upsert(ctx, newProgress(d, lessonID)); err != nil {
			return fmt.Errorf("progress: start lesson: %w", err)
		}
	}
	if err := s.d.Progress.SetPosition(ctx, userID, lessonID, step, sentence); err != nil {
		return fmt.Errorf("progress: save position: %w", err)
	}
	return nil
}

// --- lessons page and access ---

// MyLessons lists the lessons of the active goal's topic: the completed ones (newest first), the
// lesson being studied and the locked upcoming ones. Lessons finished in other topics show up when
// that topic is the goal again.
func (s *StudyService) MyLessons(ctx context.Context, userID string) (MyLessonsView, error) {
	d, err := s.load(ctx, userID)
	if err != nil {
		return MyLessonsView{}, err
	}
	inTopic := map[string]bool{}
	var upcoming []string
	if d.topic != nil {
		for _, id := range d.topic.LessonIDs {
			inTopic[id] = true
			if !d.done[id] && id != d.state.LessonID {
				upcoming = append(upcoming, id)
			}
		}
	}
	var completed []LessonProgress
	ids := []string{}
	for _, p := range d.completed {
		if inTopic[p.LessonID] {
			completed = append(completed, p)
			ids = append(ids, p.LessonID)
		}
	}
	if d.state.LessonID != "" {
		ids = append(ids, d.state.LessonID)
	}
	titles, err := s.d.Titles.Titles(ctx, append(ids, upcoming...))
	if err != nil {
		return MyLessonsView{}, fmt.Errorf("progress: lesson titles: %w", err)
	}

	out := MyLessonsView{Completed: []CompletedLesson{}, Upcoming: []LessonRef{}}
	if d.state.LessonID != "" {
		out.Current = &LessonRef{ID: d.state.LessonID, Title: titles[d.state.LessonID]}
	}
	topicNames := map[string]string{}
	for _, p := range completed {
		title, ok := titles[p.LessonID]
		if !ok {
			continue // lesson deleted
		}
		name, known := topicNames[p.TopicID]
		if !known && p.TopicID != "" {
			if t, err := s.d.Roadmaps.Roadmap(ctx, p.TopicID); err == nil {
				name = t.Name
			} else if !errors.Is(err, ErrTopicNotFound) {
				return MyLessonsView{}, fmt.Errorf("progress: roadmap: %w", err)
			}
			topicNames[p.TopicID] = name
		}
		out.Completed = append(out.Completed, CompletedLesson{
			LessonRef: LessonRef{ID: p.LessonID, Title: title}, TopicName: name, CompletedAt: p.CompletedAt,
		})
	}
	for _, id := range upcoming {
		if title, ok := titles[id]; ok {
			out.Upcoming = append(out.Upcoming, LessonRef{ID: id, Title: title})
		}
	}
	return out, nil
}

// CanWrite reports whether the lesson is the one being studied and its next step is Write (F8):
// only then may the writing be drafted or submitted.
func (s *StudyService) CanWrite(ctx context.Context, userID, lessonID string) (bool, error) {
	d, err := s.load(ctx, userID)
	if err != nil {
		return false, err
	}
	if d.state.Kind != StudyStudying || d.state.LessonID != lessonID {
		return false, nil
	}
	p, ok, err := s.d.Progress.Get(ctx, userID, lessonID)
	if err != nil {
		return false, fmt.Errorf("progress: lesson progress: %w", err)
	}
	return ok && NextStep(p.Done) == StepWrite, nil
}

// CanOpen reports whether a user may open a lesson's content: admins always; learners the lesson
// being studied and lessons they started before.
func (s *StudyService) CanOpen(ctx context.Context, userID string, isAdmin bool, lessonID string) (bool, error) {
	if isAdmin {
		return true, nil
	}
	if _, ok, err := s.d.Progress.Get(ctx, userID, lessonID); err != nil {
		return false, fmt.Errorf("progress: lesson progress: %w", err)
	} else if ok {
		return true, nil
	}
	d, err := s.load(ctx, userID)
	if err != nil {
		return false, err
	}
	return d.state.LessonID == lessonID, nil
}
