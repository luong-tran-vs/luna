package toeic

import (
	"context"
	"errors"
	"math"
	"slices"
	"time"
)

// TestState is where the learner stands on a test of the library.
type TestState struct {
	Status    string     `json:"status"` // "new" | "in_progress" | "submitted"
	AttemptID string     `json:"attemptId,omitempty"`
	Position  int        `json:"position,omitempty"`
	Total     *int       `json:"total"`
	Correct   *int       `json:"correct"`
	Questions int        `json:"questions,omitempty"`
	At        *time.Time `json:"at"`
}

// TestCard is a test of the library.
type TestCard struct {
	Key       string    `json:"key"`
	Title     string    `json:"title"`
	Kind      TestKind  `json:"kind"`
	Questions int       `json:"questions"`
	Minutes   int       `json:"minutes"`
	Parts     int       `json:"parts"`
	Complete  bool      `json:"complete"`
	Locked    bool      `json:"locked"`
	State     TestState `json:"state"`
}

// GoalView is the learner's goal with the days left before the exam.
type GoalView struct {
	Target   int    `json:"target"`
	ExamDate string `json:"examDate,omitempty"`
	DaysLeft *int   `json:"daysLeft"`
}

// ScoreLine is a scored attempt (the latest estimate).
type ScoreLine struct {
	AttemptID string    `json:"attemptId"`
	Total     int       `json:"total"`
	Listening int       `json:"listening"`
	Reading   int       `json:"reading"`
	At        time.Time `json:"at"`
}

// RecentLine is a recent attempt of the side column.
type RecentLine struct {
	AttemptID string    `json:"attemptId"`
	Title     string    `json:"title"`
	Correct   int       `json:"correct"`
	Total     int       `json:"total"`
	At        time.Time `json:"at"`
}

// LibraryView is the TOEIC home: the tests, the goal and the latest results.
type LibraryView struct {
	Tests    []TestCard   `json:"tests"`
	Goal     *GoalView    `json:"goal"`
	Latest   *ScoreLine   `json:"latest"`
	Recent   []RecentLine `json:"recent"`
	Mistakes *int         `json:"mistakes"`
	// PartCounts are the published questions of each part (index 1..7), for the part practice.
	PartCounts [8]int `json:"partCounts"`
	Guest      bool   `json:"guest"`
}

// minutesOf is the time of a test card: the kind's time for a whole test, else ~0.6 min per question.
func minutesOf(t Test, c TestCheck, questions int) int {
	if c.Complete {
		return t.Kind.Minutes()
	}
	return int(math.Ceil(float64(questions) * 0.6))
}

// Library returns the tests with the learner's state on each.
func (s *Service) Library(ctx context.Context, userID string, isGuest bool) (LibraryView, error) {
	v := LibraryView{Tests: []TestCard{}, Recent: []RecentLine{}, Guest: isGuest}
	tests, err := s.d.Tests.List(ctx, true)
	if err != nil {
		return v, err
	}
	guestKey := guestTestOf(tests)
	attempts, err := s.d.Attempts.List(ctx, userID, AttemptFilter{})
	if err != nil {
		return v, err
	}
	for _, t := range tests {
		ta, err := s.testAdminOf(ctx, t)
		if err != nil {
			return v, err
		}
		card := TestCard{Key: t.Key, Title: t.Title, Kind: t.Kind, Complete: ta.Check.Complete, Locked: locked(t, isGuest, guestKey),
			State: TestState{Status: "new"}}
		for _, pc := range ta.Check.Parts {
			card.Questions += pc.Have
			if pc.Have > 0 {
				card.Parts++
			}
		}
		card.Minutes = minutesOf(t, ta.Check, card.Questions)
		card.State = stateOf(t.Key, attempts)
		v.Tests = append(v.Tests, card)
	}
	for _, a := range attempts {
		if a.Status != AttemptSubmitted || a.Result == nil {
			continue
		}
		if a.Counted && v.Latest == nil && a.Result.Listening != nil && a.Result.Reading != nil {
			v.Latest = &ScoreLine{AttemptID: a.ID, Total: a.Result.Total, Listening: a.Result.Listening.Scaled,
				Reading: a.Result.Reading.Scaled, At: a.SubmittedAt}
		}
		if len(v.Recent) < 5 {
			v.Recent = append(v.Recent, RecentLine{AttemptID: a.ID, Title: a.Title, Correct: a.Result.Correct,
				Total: len(a.QuestionIDs), At: a.SubmittedAt})
		}
	}
	if v.Goal, err = s.goalView(ctx, userID); err != nil {
		return v, err
	}
	if !isGuest {
		n, err := s.d.Mistakes.Count(ctx, userID)
		if err != nil {
			return v, err
		}
		v.Mistakes = &n
	}
	published, err := s.d.Groups.Published(ctx, 0)
	if err != nil {
		return v, err
	}
	for _, g := range published {
		v.PartCounts[g.Part] += len(g.Questions)
	}
	return v, nil
}

// stateOf is the learner's state on a test from their attempts (newest first).
func stateOf(key string, attempts []Attempt) TestState {
	for _, a := range attempts {
		if a.Source.Kind != SourceTest || a.Source.TestKey != key {
			continue
		}
		if a.Status == AttemptInProgress {
			return TestState{Status: "in_progress", AttemptID: a.ID, Position: a.Numbers[a.Position], Questions: len(a.QuestionIDs)}
		}
	}
	for _, a := range attempts {
		if a.Source.Kind != SourceTest || a.Source.TestKey != key || a.Status != AttemptSubmitted || a.Result == nil {
			continue
		}
		st := TestState{Status: "submitted", AttemptID: a.ID, Questions: len(a.QuestionIDs)}
		at := a.SubmittedAt
		st.At = &at
		if a.Result.Total > 0 {
			t := a.Result.Total
			st.Total = &t
		} else {
			c := a.Result.Correct
			st.Correct = &c
		}
		return st
	}
	return TestState{Status: "new"}
}

// goalView returns the learner's goal, nil when none.
func (s *Service) goalView(ctx context.Context, userID string) (*GoalView, error) {
	g, err := s.d.Goals.Get(ctx, userID)
	if errors.Is(err, ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	v := &GoalView{Target: g.Target, ExamDate: g.ExamDate}
	if g.ExamDate != "" {
		loc, err := s.d.Timezones.Location(ctx, userID)
		if err != nil {
			return nil, err
		}
		exam, err := time.ParseInLocation(time.DateOnly, g.ExamDate, loc)
		if err == nil {
			now := s.d.Now().In(loc)
			today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
			days := int(math.Round(exam.Sub(today).Hours() / 24))
			v.DaysLeft = &days
		}
	}
	return v, nil
}

// PartInfo is a part of a test with its questions and question types.
type PartInfo struct {
	Part      Part     `json:"part"`
	Questions int      `json:"questions"`
	Tags      []string `json:"tags"`
}

// AttemptLine is a sitting of a test in its detail page.
type AttemptLine struct {
	ID          string    `json:"id"`
	Mode        Mode      `json:"mode"`
	Parts       []Part    `json:"parts"`
	Correct     int       `json:"correct"`
	Total       int       `json:"total"`
	Scaled      *int      `json:"scaled"`
	Retake      bool      `json:"retake"`
	DurationMs  int64     `json:"durationMs"`
	SubmittedAt time.Time `json:"submittedAt"`
}

// TestDetailView is the detail page of a test.
type TestDetailView struct {
	Key        string        `json:"key"`
	Title      string        `json:"title"`
	Kind       TestKind      `json:"kind"`
	Minutes    int           `json:"minutes"`
	Questions  int           `json:"questions"`
	Complete   bool          `json:"complete"`
	Locked     bool          `json:"locked"`
	Parts      []PartInfo    `json:"parts"`
	Attempts   []AttemptLine `json:"attempts"`
	InProgress *string       `json:"inProgress"`
}

// TestDetail returns a published test with the learner's sittings; a guest sees a locked test's
// name only.
func (s *Service) TestDetail(ctx context.Context, userID string, isGuest bool, key string) (TestDetailView, error) {
	t, err := s.d.Tests.Get(ctx, key)
	if err != nil {
		return TestDetailView{}, err
	}
	if t.Status != StatusPublished {
		return TestDetailView{}, ErrNotFound
	}
	guestKey, err := s.guestTestKey(ctx)
	if err != nil {
		return TestDetailView{}, err
	}
	v := TestDetailView{Key: t.Key, Title: t.Title, Kind: t.Kind, Locked: locked(t, isGuest, guestKey), Parts: []PartInfo{}, Attempts: []AttemptLine{}}
	if v.Locked {
		return v, nil
	}
	ta, err := s.testAdminOf(ctx, t)
	if err != nil {
		return TestDetailView{}, err
	}
	v.Complete = ta.Check.Complete
	byPart := map[Part]*PartInfo{}
	for _, g := range ta.Groups {
		pi := byPart[g.Part]
		if pi == nil {
			pi = &PartInfo{Part: g.Part, Tags: []string{}}
			byPart[g.Part] = pi
		}
		pi.Questions += len(g.Questions)
		for _, tag := range groupTags(g) {
			if !slices.Contains(pi.Tags, tag) {
				pi.Tags = append(pi.Tags, tag)
			}
		}
	}
	for p := Part(1); p <= 7; p++ {
		if pi := byPart[p]; pi != nil {
			v.Parts = append(v.Parts, *pi)
			v.Questions += pi.Questions
		}
	}
	v.Minutes = minutesOf(t, ta.Check, v.Questions)
	list, err := s.d.Attempts.List(ctx, userID, AttemptFilter{TestKey: key})
	if err != nil {
		return TestDetailView{}, err
	}
	// Oldest first to tell the retakes, then shown newest first.
	slices.Reverse(list)
	first := true
	for _, a := range list {
		if a.Status == AttemptInProgress {
			id := a.ID
			v.InProgress = &id
			continue
		}
		if a.Result == nil {
			continue
		}
		line := AttemptLine{ID: a.ID, Mode: a.Mode, Parts: a.Parts, Correct: a.Result.Correct, Total: len(a.QuestionIDs),
			Retake: !first, DurationMs: a.Result.DurationMs, SubmittedAt: a.SubmittedAt}
		if a.Result.Total > 0 {
			t := a.Result.Total
			line.Scaled = &t
		}
		first = false
		v.Attempts = append(v.Attempts, line)
	}
	slices.Reverse(v.Attempts)
	return v, nil
}

// SaveWord adds a word of an answer to the learner's vocabulary notebook (FR-028).
func (s *Service) SaveWord(ctx context.Context, userID string, w CardInput) error {
	f := map[string]string{}
	if len([]rune(w.Word)) == 0 || len([]rune(w.Word)) > 60 {
		f["word"] = "Từ cần 1–60 ký tự"
	}
	if w.MeaningVi == "" {
		f["meaningVi"] = "Cần nghĩa tiếng Việt"
	}
	if len(f) > 0 {
		return &ValidationError{Fields: f}
	}
	return s.d.Cards.SaveWord(ctx, userID, w)
}

// Image returns a picture of a group; only of a published group unless isAdmin.
func (s *Service) Image(ctx context.Context, groupKey, slot string, isAdmin bool) (Image, error) {
	if !isAdmin {
		g, err := s.GetGroup(ctx, groupKey)
		if err != nil {
			return Image{}, err
		}
		if g.Status != StatusPublished {
			return Image{}, ErrNotFound
		}
	}
	return s.d.Images.Get(ctx, groupKey, slot)
}
