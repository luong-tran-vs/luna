package progress

import (
	"context"
	"fmt"
)

// PathLessonState is how a lesson shows on the learning path.
type PathLessonState string

const (
	PathDone    PathLessonState = "done"
	PathCurrent PathLessonState = "current"
	PathLocked  PathLessonState = "locked"
	// PathMembersOnly is a lesson a guest cannot study.
	PathMembersOnly PathLessonState = "membersOnly"
)

// PathLesson is one step of a roadmap.
type PathLesson struct {
	ID    string
	Title string
	State PathLessonState
}

// PathRoadmap is one topic's roadmap within a milestone; Active when it is the learner's goal.
type PathRoadmap struct {
	TopicID          string
	TopicName        string
	TopicDescription string
	Active           bool
	Completed        int
	Lessons          []PathLesson
}

// Milestone is one level ("Phần"): every topic with lessons at that level, one roadmap after another.
type Milestone struct {
	Level     string
	Active    bool
	Completed int
	Total     int
	Roadmaps  []PathRoadmap
}

// PathView is the "Học" tab: one milestone per level (all of them, even with no lesson yet).
// TopicID and TopicName are the active goal's topic, "" when the learner has no goal.
type PathView struct {
	TopicID    string
	TopicName  string
	Milestones []Milestone
	Streak     int
}

// Path returns every roadmap of every level, in level then topic order. Only the goal's roadmap has
// a lesson to study now; elsewhere lessons already completed are done and the rest locked until the
// learner makes that roadmap the goal.
func (s *StudyService) Path(ctx context.Context, userID string) (PathView, error) {
	d, err := s.load(ctx, userID)
	if err != nil {
		return PathView{}, err
	}
	keys, err := s.d.Days.CompletedKeys(ctx, userID)
	if err != nil {
		return PathView{}, fmt.Errorf("progress: study days: %w", err)
	}
	byLevel, err := s.d.Roadmaps.LevelRoadmaps(ctx)
	if err != nil {
		return PathView{}, fmt.Errorf("progress: roadmaps: %w", err)
	}
	out := PathView{Milestones: make([]Milestone, 0, len(goalLevels)), Streak: Streak(keys, d.today)}
	if d.goal != nil {
		out.TopicID, out.TopicName = d.goal.TopicID, d.topic.Name
	}

	var ids []string
	for _, level := range goalLevels {
		m := Milestone{Level: level, Active: d.goal != nil && d.goal.Level == level, Roadmaps: []PathRoadmap{}}
		roadmaps := byLevel[level]
		// The goal's roadmap shows even before it has a lesson.
		if m.Active && !hasTopic(roadmaps, d.goal.TopicID) {
			roadmaps = append(roadmaps, TopicInfo{
				ID: d.goal.TopicID, Name: d.topic.Name, Description: d.topic.Description, Level: level,
			})
		}
		for _, t := range roadmaps {
			r := PathRoadmap{
				TopicID: t.ID, TopicName: t.Name, TopicDescription: t.Description,
				Active:  m.Active && t.ID == d.goal.TopicID,
				Lessons: make([]PathLesson, 0, len(t.LessonIDs)),
			}
			for i, id := range t.LessonIDs {
				r.Lessons = append(r.Lessons, PathLesson{ID: id, State: pathState(d, r.Active, i, id)})
				if d.done[id] {
					r.Completed++
				}
			}
			m.Completed += r.Completed
			m.Total += len(t.LessonIDs)
			m.Roadmaps = append(m.Roadmaps, r)
			ids = append(ids, t.LessonIDs...)
		}
		out.Milestones = append(out.Milestones, m)
	}

	titles, err := s.d.Titles.Titles(ctx, ids)
	if err != nil {
		return PathView{}, fmt.Errorf("progress: lesson titles: %w", err)
	}
	for i := range out.Milestones {
		for j := range out.Milestones[i].Roadmaps {
			r := &out.Milestones[i].Roadmaps[j]
			for k := range r.Lessons {
				r.Lessons[k].Title = titles[r.Lessons[k].ID]
			}
		}
	}
	return out, nil
}

func hasTopic(roadmaps []TopicInfo, topicID string) bool {
	for _, t := range roadmaps {
		if t.ID == topicID {
			return true
		}
	}
	return false
}

// pathState is the state of the index-th lesson of a roadmap. A guest studies only the first lesson
// of each roadmap: on other roadmaps that one stays locked (open once it is the goal).
func pathState(d *day, active bool, index int, lessonID string) PathLessonState {
	switch {
	case d.done[lessonID]:
		return PathDone
	case active && lessonID == d.state.LessonID:
		return PathCurrent
	case d.guest && (active || index > 0):
		return PathMembersOnly
	default:
		return PathLocked
	}
}
