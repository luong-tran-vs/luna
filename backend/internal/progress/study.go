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
	// ReviewLimit is the learner's daily card limit; nil means DefaultReviewLimit (F12 sets it).
	ReviewLimit func(ctx context.Context, userID string) int
	Now         func() time.Time
}

// StudyService runs the daily flow (L): goals, today's lesson and its steps, streak.
type StudyService struct {
	d StudyDeps
}

// NewStudyService returns a StudyService.
func NewStudyService(d StudyDeps) *StudyService {
	if d.ReviewLimit == nil {
		d.ReviewLimit = func(context.Context, string) int { return DefaultReviewLimit }
	}
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

// SetGoalResult is the new active goal; StartsTomorrow when today's lesson had started.
type SetGoalResult struct {
	Active         GoalView
	EffectiveFrom  string
	StartsTomorrow bool
}

// LessonRef identifies a lesson for the learner.
type LessonRef struct {
	ID    string
	Title string
}

// TodayView is what the today page shows.
type TodayView struct {
	Kind          TodayKind
	Goal          *GoalView
	Lesson        *LessonRef
	Steps         map[Step]StepState
	CurrentStep   Step
	SentenceIndex int
	ReviewCount   int
	Streak        int
	GoalCompleted bool
}

// CompletedLesson is a lesson in the "Đã học" list.
type CompletedLesson struct {
	LessonRef
	TopicName   string
	CompletedAt time.Time
}

// MyLessonsView is the lessons page.
type MyLessonsView struct {
	Today     *LessonRef
	Completed []CompletedLesson
	Upcoming  []LessonRef
}

// day is everything the rules need about a learner right now.
type day struct {
	userID    string
	now       time.Time
	loc       *time.Location
	today     string
	studyDay  *StudyDay
	goal      *Goal
	topic     *TopicInfo
	goals     []Goal
	completed []LessonProgress
	done      map[string]bool
	state     TodayState
}

func (s *StudyService) load(ctx context.Context, userID string) (*day, error) {
	loc, err := s.d.Timezones.Location(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("progress: timezone: %w", err)
	}
	d := &day{userID: userID, now: s.d.Now(), loc: loc}
	latest, err := s.d.Days.LatestKey(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("progress: latest study day: %w", err)
	}
	d.today = EffectiveDayKey(DayKey(d.now, loc), latest)
	if d.studyDay, err = s.d.Days.Get(ctx, userID, d.today); err != nil {
		return nil, fmt.Errorf("progress: study day: %w", err)
	}
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
	d.state = TodayLesson(d.goal, roadmap, d.done, d.studyDay)
	return d, nil
}

func (s *StudyService) startOfDay(d *day) time.Time {
	y, m, dd := d.now.In(d.loc).Date()
	return time.Date(y, m, dd, 0, 0, 0, 0, d.loc)
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

// SetGoal makes topicID the learner's goal. When today's lesson has started, the learner
// finishes it today and the new topic gives lessons from tomorrow.
func (s *StudyService) SetGoal(ctx context.Context, userID, topicID string) (SetGoalResult, error) {
	t, err := s.d.Roadmaps.Roadmap(ctx, topicID)
	if err != nil {
		return SetGoalResult{}, fmt.Errorf("progress: roadmap: %w", err)
	}
	d, err := s.load(ctx, userID)
	if err != nil {
		return SetGoalResult{}, err
	}
	started := d.studyDay != nil && d.studyDay.LessonID != ""
	effective := d.today
	if started {
		effective = NextDayKey(d.today)
	}
	g, err := s.d.Goals.Activate(ctx, userID, topicID, t.Level, effective, d.now)
	if err != nil {
		return SetGoalResult{}, fmt.Errorf("progress: activate goal: %w", err)
	}
	return SetGoalResult{Active: goalView(g, t, d.done), EffectiveFrom: effective, StartsTomorrow: started}, nil
}

// --- today ---

// Today returns today's lesson and steps. With no card to review, opening the review step
// completes it (and fixes today's lesson).
func (s *StudyService) Today(ctx context.Context, userID string) (TodayView, error) {
	d, err := s.load(ctx, userID)
	if err != nil {
		return TodayView{}, err
	}
	v, err := s.view(ctx, d)
	if err != nil {
		return TodayView{}, err
	}
	if v.Kind == TodayStudying && v.CurrentStep == StepReview && v.ReviewCount == 0 {
		return s.completeStep(ctx, d, StepReview)
	}
	return v, nil
}

func (s *StudyService) view(ctx context.Context, d *day) (TodayView, error) {
	v := TodayView{Kind: d.state.Kind, Steps: map[Step]StepState{}}
	for _, st := range Steps {
		v.Steps[st] = StateLocked
	}
	if d.goal != nil {
		g := goalView(*d.goal, *d.topic, d.done)
		v.Goal = &g
		v.GoalCompleted = g.TotalLessons > 0 && g.CompletedLessons == g.TotalLessons
	}
	keys, err := s.d.Days.CompletedKeys(ctx, d.userID)
	if err != nil {
		return TodayView{}, fmt.Errorf("progress: study days: %w", err)
	}
	v.Streak = Streak(keys, d.today)
	if d.state.LessonID == "" {
		return v, nil
	}

	titles, err := s.d.Titles.Titles(ctx, []string{d.state.LessonID})
	if err != nil {
		return TodayView{}, fmt.Errorf("progress: lesson title: %w", err)
	}
	v.Lesson = &LessonRef{ID: d.state.LessonID, Title: titles[d.state.LessonID]}
	p, _, err := s.d.Progress.Get(ctx, d.userID, d.state.LessonID)
	if err != nil {
		return TodayView{}, fmt.Errorf("progress: lesson progress: %w", err)
	}
	v.CurrentStep = NextStep(p.Done)
	for _, st := range Steps {
		switch {
		case p.Done[st]:
			v.Steps[st] = StateDone
		case st == v.CurrentStep:
			v.Steps[st] = StateCurrent
		}
	}
	if p.CurrentStep == v.CurrentStep {
		v.SentenceIndex = p.SentenceIndex
	}
	if v.CurrentStep == StepReview {
		if v.ReviewCount, err = s.reviewCount(ctx, d); err != nil {
			return TodayView{}, err
		}
	}
	return v, nil
}

// reviewCount is the number of cards for today's review step: due cards within the quota left.
func (s *StudyService) reviewCount(ctx context.Context, d *day) (int, error) {
	reviewed, err := s.d.Reviews.ReviewedSince(ctx, d.userID, s.startOfDay(d))
	if err != nil {
		return 0, fmt.Errorf("progress: reviewed cards: %w", err)
	}
	due, err := s.d.Reviews.DueCount(ctx, d.userID)
	if err != nil {
		return 0, fmt.Errorf("progress: due cards: %w", err)
	}
	return min(due, ReviewQuota(s.d.ReviewLimit(ctx, d.userID), reviewed)), nil
}

// CompleteStep records a step of today's lesson. Steps complete in order; completing a done
// step again changes nothing. The last step completes the lesson (goal +1, streak +1).
func (s *StudyService) CompleteStep(ctx context.Context, userID string, step Step) (TodayView, error) {
	if !ValidStep(step) {
		return TodayView{}, &ValidationError{Fields: map[string]string{"step": "Bước không hợp lệ"}}
	}
	d, err := s.load(ctx, userID)
	if err != nil {
		return TodayView{}, err
	}
	switch d.state.Kind {
	case TodayDone:
		return s.view(ctx, d)
	case TodayStudying:
	default:
		return TodayView{}, ErrNoLesson
	}
	p, _, err := s.d.Progress.Get(ctx, userID, d.state.LessonID)
	if err != nil {
		return TodayView{}, fmt.Errorf("progress: lesson progress: %w", err)
	}
	if p.Done[step] {
		return s.view(ctx, d)
	}
	if NextStep(p.Done) != step {
		return TodayView{}, ErrStepLocked
	}
	return s.completeStep(ctx, d, step)
}

func (s *StudyService) completeStep(ctx context.Context, d *day, step Step) (TodayView, error) {
	lessonID := d.state.LessonID
	if step == StepRead && s.d.Quiz != nil {
		questions, answered, err := s.d.Quiz.Status(ctx, d.userID, lessonID)
		if err != nil {
			return TodayView{}, fmt.Errorf("progress: reading quiz: %w", err)
		}
		if questions > 0 && answered < questions {
			return TodayView{}, ErrReadIncomplete
		}
	}
	if step == StepListen {
		sum, err := s.d.Dictation.Summary(ctx, d.userID, lessonID)
		if err != nil {
			return TodayView{}, fmt.Errorf("progress: dictation: %w", err)
		}
		if !sum.Completed {
			return TodayView{}, ErrListenIncomplete
		}
	}
	if err := s.d.Days.Start(ctx, d.userID, d.today, lessonID); err != nil {
		return TodayView{}, fmt.Errorf("progress: start day: %w", err)
	}
	p, ok, err := s.d.Progress.Get(ctx, d.userID, lessonID)
	if err != nil {
		return TodayView{}, fmt.Errorf("progress: lesson progress: %w", err)
	}
	if !ok {
		p = LessonProgress{UserID: d.userID, LessonID: lessonID, DayKey: d.today, StartedAt: d.now}
		if d.goal != nil {
			p.TopicID = d.goal.TopicID
		}
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
		return TodayView{}, fmt.Errorf("progress: save progress: %w", err)
	}
	reviewed, err := s.d.Reviews.ReviewedSince(ctx, d.userID, s.startOfDay(d))
	if err != nil {
		return TodayView{}, fmt.Errorf("progress: reviewed cards: %w", err)
	}
	if err := s.d.Days.Update(ctx, d.userID, d.today, reviewed, completed); err != nil {
		return TodayView{}, fmt.Errorf("progress: update day: %w", err)
	}
	next, err := s.load(ctx, d.userID)
	if err != nil {
		return TodayView{}, err
	}
	return s.view(ctx, next)
}

// SetPosition saves the sentence of the current step of today's lesson.
func (s *StudyService) SetPosition(ctx context.Context, userID string, step Step, sentence int) error {
	if !ValidStep(step) {
		return &ValidationError{Fields: map[string]string{"step": "Bước không hợp lệ"}}
	}
	d, err := s.load(ctx, userID)
	if err != nil {
		return err
	}
	if d.state.Kind != TodayStudying {
		return ErrNoLesson
	}
	p, _, err := s.d.Progress.Get(ctx, userID, d.state.LessonID)
	if err != nil {
		return fmt.Errorf("progress: lesson progress: %w", err)
	}
	if NextStep(p.Done) != step {
		return ErrNotCurrentStep
	}
	_, count, err := s.d.Lessons.Info(ctx, d.state.LessonID)
	if err != nil {
		return fmt.Errorf("progress: lesson: %w", err)
	}
	if sentence < 0 || sentence >= count {
		return &ValidationError{Fields: map[string]string{"sentenceIndex": "Câu không có trong bài"}}
	}
	if err := s.d.Progress.SetPosition(ctx, userID, d.state.LessonID, step, sentence); err != nil {
		return fmt.Errorf("progress: save position: %w", err)
	}
	return nil
}

// --- lessons page and access ---

// MyLessons lists today's lesson, the completed ones (newest first) and the locked upcoming
// lessons of the active goal.
func (s *StudyService) MyLessons(ctx context.Context, userID string) (MyLessonsView, error) {
	d, err := s.load(ctx, userID)
	if err != nil {
		return MyLessonsView{}, err
	}
	ids := []string{}
	for _, p := range d.completed {
		ids = append(ids, p.LessonID)
	}
	if d.state.LessonID != "" {
		ids = append(ids, d.state.LessonID)
	}
	var upcoming []string
	if d.topic != nil {
		for _, id := range d.topic.LessonIDs {
			if !d.done[id] && id != d.state.LessonID {
				upcoming = append(upcoming, id)
			}
		}
	}
	titles, err := s.d.Titles.Titles(ctx, append(ids, upcoming...))
	if err != nil {
		return MyLessonsView{}, fmt.Errorf("progress: lesson titles: %w", err)
	}

	out := MyLessonsView{Completed: []CompletedLesson{}, Upcoming: []LessonRef{}}
	if d.state.LessonID != "" {
		out.Today = &LessonRef{ID: d.state.LessonID, Title: titles[d.state.LessonID]}
	}
	topicNames := map[string]string{}
	for _, p := range d.completed {
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

// CanOpen reports whether a user may open a lesson's content: admins always; learners today's
// lesson and lessons they started before.
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
