package progress

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// ActionKind is how the home button reads: "Bắt đầu" before any step is done, "Tiếp tục" after.
type ActionKind string

const (
	ActionStart    ActionKind = "start"
	ActionContinue ActionKind = "continue"
)

// Action is the home button of the lesson being studied: it opens Step.
type Action struct {
	Kind ActionKind
	Step Step
}

// TomorrowCutoff is 00:00 of the day after tomorrow in loc: cards due before it are due by the
// end of tomorrow.
func TomorrowCutoff(now time.Time, loc *time.Location) time.Time {
	y, m, d := now.In(loc).Date()
	return time.Date(y, m, d+2, 0, 0, 0, 0, loc)
}

// NextAction is the home button for the lesson being studied; nil once every step is done.
func NextAction(done map[Step]bool, current Step) *Action {
	if current == StepDone || !ValidStep(current) {
		return nil
	}
	a := &Action{Kind: ActionStart, Step: current}
	for _, s := range Steps {
		if done[s] {
			a.Kind = ActionContinue
		}
	}
	return a
}

// SkillCounts are the lessons of the current roadmap with the read / listen / write step done.
type SkillCounts struct {
	Read, Listen, Write, Total int
}

// DashboardLesson is the lesson being studied with its topic.
type DashboardLesson struct {
	ID, Title, TopicName, Level string
}

// DashboardView is what the home page shows (F6).
type DashboardView struct {
	Kind          StudyKind
	Goal          *GoalView
	GoalCompleted bool
	Skills        *SkillCounts
	Lesson        *DashboardLesson
	Steps         map[Step]StepState
	CurrentStep   Step
	Action        *Action
	Streak        int
	TomorrowCards int
}

// Dashboard returns the home page numbers. It only reads.
func (s *StudyService) Dashboard(ctx context.Context, userID string) (DashboardView, error) {
	d, err := s.load(ctx, userID)
	if err != nil {
		return DashboardView{}, err
	}
	v, err := s.view(ctx, d, d.state.LessonID)
	if err != nil {
		return DashboardView{}, err
	}
	out := DashboardView{
		Kind: d.state.Kind, Goal: v.Goal, GoalCompleted: v.GoalCompleted, Steps: v.Steps, Streak: v.Streak,
	}
	if d.state.LessonID != "" {
		out.CurrentStep = v.CurrentStep
	} else {
		for _, st := range Steps {
			out.Steps[st] = StateLocked
		}
	}
	if d.topic != nil {
		c, err := s.d.Progress.StepCounts(ctx, userID, d.topic.LessonIDs, nil)
		if err != nil {
			return DashboardView{}, fmt.Errorf("progress: step counts: %w", err)
		}
		out.Skills = &SkillCounts{Read: c.Read, Listen: c.Listen, Write: c.Write, Total: len(d.topic.LessonIDs)}
	}
	if id := d.state.LessonID; id != "" {
		p, _, err := s.d.Progress.Get(ctx, userID, id)
		if err != nil {
			return DashboardView{}, fmt.Errorf("progress: lesson progress: %w", err)
		}
		titles, err := s.d.Titles.Titles(ctx, []string{id})
		if err != nil {
			return DashboardView{}, fmt.Errorf("progress: lesson title: %w", err)
		}
		l := &DashboardLesson{ID: id, Title: titles[id]}
		if err := s.lessonTopic(ctx, d, p.TopicID, l); err != nil {
			return DashboardView{}, err
		}
		out.Lesson = l
		out.Action = NextAction(p.Done, v.CurrentStep)
	}
	y, m, dd := d.now.In(d.loc).Date()
	tomorrow := time.Date(y, m, dd+1, 0, 0, 0, 0, d.loc)
	if out.TomorrowCards, err = s.d.Reviews.DueBefore(ctx, userID, TomorrowCutoff(d.now, d.loc), tomorrow); err != nil {
		return DashboardView{}, fmt.Errorf("progress: cards due tomorrow: %w", err)
	}
	return out, nil
}

// lessonTopic sets the topic of the lesson being studied: the topic it was started in, else the goal's.
func (s *StudyService) lessonTopic(ctx context.Context, d *day, topicID string, l *DashboardLesson) error {
	if topicID == "" || d.topic != nil && topicID == d.topic.ID {
		if d.topic != nil {
			l.TopicName, l.Level = d.topic.Name, d.topic.Level
		}
		return nil
	}
	t, err := s.d.Roadmaps.Roadmap(ctx, topicID)
	switch {
	case err == nil:
		l.TopicName, l.Level = t.Name, t.Level
	case !errors.Is(err, ErrTopicNotFound):
		return fmt.Errorf("progress: lesson topic: %w", err)
	}
	return nil
}
