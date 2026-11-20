package toeic

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"
)

// maxTimeLimitMin bounds the time a learner may give a practice round.
const maxTimeLimitMin = 180

// StartReq is what the learner starts an attempt with.
type StartReq struct {
	Source       Source `json:"source"`
	Mode         Mode   `json:"mode"`
	Parts        []Part `json:"parts"`
	TimeLimitMin int    `json:"timeLimitMin"`
}

// AttemptView is an attempt as the exam room gets it: never an answer while it is in progress.
type AttemptView struct {
	ID               string         `json:"id"`
	Title            string         `json:"title"`
	Mode             Mode           `json:"mode"`
	Status           AttemptStatus  `json:"status"`
	Source           Source         `json:"source"`
	Phase            Section        `json:"phase,omitempty"`
	Parts            []Part         `json:"parts"`
	TimeLimitMs      int64          `json:"timeLimitMs"`
	RemainingMs      int64          `json:"remainingMs"`
	ElapsedMs        int64          `json:"elapsedMs"`
	ListeningSegment int            `json:"listeningSegment"`
	Position         string         `json:"position"`
	Answers          map[string]int `json:"answers"`
	Flags            []string       `json:"flags"`
	Paper            []PaperGroup   `json:"paper"`
}

// Start begins an attempt. A learner has at most one attempt in progress per test: starting the
// test again gives ErrAttemptInProgress with that attempt.
func (s *Service) Start(ctx context.Context, userID string, isGuest bool, req StartReq) (AttemptView, error) {
	if err := checkStartReq(&req); err != nil {
		return AttemptView{}, err
	}
	if err := s.checkStart(ctx, isGuest, req.Source); err != nil {
		return AttemptView{}, err
	}
	if req.Source.Kind == SourceTest {
		prev, err := s.d.Attempts.InProgress(ctx, userID, req.Source.TestKey)
		if err == nil {
			return AttemptView{}, &ErrAttemptInProgress{ID: prev.ID}
		}
		if !errors.Is(err, ErrNotFound) {
			return AttemptView{}, err
		}
	}
	p, err := s.buildPaper(ctx, userID, req.Source, req.Mode, req.Parts)
	if err != nil {
		return AttemptView{}, err
	}
	now := s.now()
	limit := int64(req.TimeLimitMin) * int64(time.Minute/time.Millisecond)
	a := Attempt{
		UserID: userID, Mode: req.Mode, Source: req.Source, Title: p.title, Parts: p.parts(),
		QuestionIDs: p.questionIDs, Numbers: p.numbers, TimeLimitMs: limit, RemainingMs: limit,
		Answers: map[string]int{}, Flags: []string{}, Status: AttemptInProgress, StartedAt: now, SavedAt: now,
	}
	for _, g := range p.groups {
		a.GroupKeys = append(a.GroupKeys, g.Key)
	}
	if len(a.QuestionIDs) > 0 {
		a.Position = a.QuestionIDs[0]
	}
	a.Phase = Reading
	if len(a.Parts) > 0 && a.Parts[0] <= 4 {
		a.Phase = Listening
	}
	a, err = s.d.Attempts.Create(ctx, a)
	if err != nil {
		return AttemptView{}, err
	}
	return s.viewOf(ctx, a, p.groups)
}

// checkStartReq validates and normalises a start request.
func checkStartReq(req *StartReq) error {
	f := map[string]string{}
	switch req.Source.Kind {
	case SourceTest:
		if req.Source.TestKey == "" {
			f["source.testKey"] = "Cần mã đề"
		}
	case SourcePart, SourceRetry, SourceRandom, SourceTag, SourceMistakes:
	default:
		f["source.kind"] = "Nguồn câu hỏi không hợp lệ"
	}
	if req.Mode == "" {
		req.Mode = ModePractice
	}
	if req.Mode != ModePractice && req.Mode != ModeTest {
		f["mode"] = "Chế độ phải là practice hoặc test"
	}
	if req.Source.Kind == SourceTest && req.Mode == ModePractice && len(req.Parts) == 0 {
		f["parts"] = "Chọn ít nhất một Part"
	}
	for _, p := range req.Parts {
		if !p.Valid() {
			f["parts"] = "Part phải từ 1 đến 7"
		}
	}
	slices.Sort(req.Parts)
	req.Parts = slices.Compact(req.Parts)
	if req.TimeLimitMin < 0 || req.TimeLimitMin > maxTimeLimitMin {
		f["timeLimitMin"] = fmt.Sprintf("Thời gian từ 0 đến %d phút", maxTimeLimitMin)
	}
	if len(f) > 0 {
		return &ValidationError{Fields: f}
	}
	return nil
}

// Open returns an attempt to go on with. A timed attempt whose time ran out while the learner was
// away is submitted first.
func (s *Service) Open(ctx context.Context, userID, id string) (AttemptView, error) {
	a, err := s.d.Attempts.Get(ctx, userID, id)
	if err != nil {
		return AttemptView{}, err
	}
	if a.Status == AttemptInProgress && a.TimeLimitMs > 0 && a.RemainingMs <= 0 {
		if _, err := s.Submit(ctx, userID, id, nil); err != nil {
			return AttemptView{}, err
		}
		if a, err = s.d.Attempts.Get(ctx, userID, id); err != nil {
			return AttemptView{}, err
		}
	}
	groups, _, err := s.attemptGroups(ctx, a)
	if err != nil {
		return AttemptView{}, err
	}
	return s.viewOf(ctx, a, groups)
}

func (s *Service) viewOf(ctx context.Context, a Attempt, groups []Group) (AttemptView, error) {
	pv, err := s.paperView(ctx, groups, a.QuestionIDs, a.Numbers)
	if err != nil {
		return AttemptView{}, err
	}
	v := AttemptView{
		ID: a.ID, Title: a.Title, Mode: a.Mode, Status: a.Status, Source: a.Source, Phase: a.Phase, Parts: a.Parts,
		TimeLimitMs: a.TimeLimitMs, RemainingMs: a.RemainingMs, ElapsedMs: a.ElapsedMs, ListeningSegment: a.ListeningSegment,
		Position: a.Position, Answers: a.Answers, Flags: a.Flags, Paper: pv,
	}
	if v.Answers == nil {
		v.Answers = map[string]int{}
	}
	if v.Flags == nil {
		v.Flags = []string{}
	}
	return v, nil
}

// Progress is what the exam room saves; nil fields are left as they are.
type Progress struct {
	// Answers are merged; -1 clears a choice.
	Answers          map[string]int `json:"answers"`
	Flags            []string       `json:"flags"`
	Position         *string        `json:"position"`
	RemainingMs      *int64         `json:"remainingMs"`
	ElapsedMs        *int64         `json:"elapsedMs"`
	Phase            *Section       `json:"phase"`
	ListeningSegment *int           `json:"listeningSegment"`
}

// SaveProgress stores the progress of an attempt in progress.
func (s *Service) SaveProgress(ctx context.Context, userID, id string, p Progress) error {
	a, err := s.d.Attempts.Get(ctx, userID, id)
	if err != nil {
		return err
	}
	if a.Status != AttemptInProgress {
		return ErrAttemptSubmitted
	}
	if err := apply(&a, p); err != nil {
		return err
	}
	a.SavedAt = s.now()
	return s.d.Attempts.Save(ctx, a)
}

// apply merges p into a: answers only for the attempt's questions, the time left only going down,
// the elapsed time only going up, and no way back from the reading to the listening.
func apply(a *Attempt, p Progress) error {
	in := make(map[string]bool, len(a.QuestionIDs))
	for _, id := range a.QuestionIDs {
		in[id] = true
	}
	f := map[string]string{}
	for id, v := range p.Answers {
		switch {
		case !in[id]:
			f["answers."+id] = "Câu không có trong bài"
		case v < -1 || v > 3:
			f["answers."+id] = "Phương án không hợp lệ"
		}
	}
	if p.Phase != nil && *p.Phase != Listening && *p.Phase != Reading {
		f["phase"] = "Phần thi không hợp lệ"
	} else if p.Phase != nil && *p.Phase == Listening && a.Phase == Reading {
		f["phase"] = "Không quay lại phần Nghe được"
	}
	if len(f) > 0 {
		return &ValidationError{Fields: f}
	}
	if a.Answers == nil {
		a.Answers = map[string]int{}
	}
	for id, v := range p.Answers {
		if v < 0 {
			delete(a.Answers, id)
		} else {
			a.Answers[id] = v
		}
	}
	if p.Flags != nil {
		flags := []string{}
		for _, id := range p.Flags {
			if in[id] && !slices.Contains(flags, id) {
				flags = append(flags, id)
			}
		}
		a.Flags = flags
	}
	if p.Position != nil && in[*p.Position] {
		a.Position = *p.Position
	}
	if p.RemainingMs != nil && a.TimeLimitMs > 0 && *p.RemainingMs >= 0 && *p.RemainingMs < a.RemainingMs {
		a.RemainingMs = *p.RemainingMs
	}
	if p.ElapsedMs != nil && *p.ElapsedMs > a.ElapsedMs {
		a.ElapsedMs = *p.ElapsedMs
	}
	if p.Phase != nil {
		a.Phase = *p.Phase
	}
	if p.ListeningSegment != nil && *p.ListeningSegment > a.ListeningSegment {
		a.ListeningSegment = *p.ListeningSegment
	}
	return nil
}

// Submit grades an attempt with its last progress, updates the mistakes notebook and counts the day
// as a study day. Submitting again gives the same result.
func (s *Service) Submit(ctx context.Context, userID, id string, last *Progress) (ResultView, error) {
	a, err := s.d.Attempts.Get(ctx, userID, id)
	if err != nil {
		return ResultView{}, err
	}
	if a.Status == AttemptSubmitted {
		return s.Result(ctx, userID, id)
	}
	if last != nil {
		if err := apply(&a, *last); err != nil {
			return ResultView{}, err
		}
	}
	_, byKey, err := s.attemptGroups(ctx, a)
	if err != nil {
		return ResultView{}, err
	}
	r := grade(byKey, a)
	a.Counted, err = s.counts(ctx, a)
	if err != nil {
		return ResultView{}, err
	}
	now := s.now()
	a.Status, a.Result, a.SubmittedAt, a.SavedAt = AttemptSubmitted, &r, now, now
	if err := s.d.Attempts.Submit(ctx, a); err != nil {
		return ResultView{}, err
	}
	if err := s.recordSubmitted(ctx, a, byKey); err != nil {
		return ResultView{}, err
	}
	if err := s.d.StudyDays.MarkStudied(ctx, userID); err != nil {
		return ResultView{}, fmt.Errorf("mark studied: %w", err)
	}
	return s.Result(ctx, userID, id)
}

// counts reports whether a goes on the score chart: the first whole sitting of a test (FR-004a), or
// any random test.
func (s *Service) counts(ctx context.Context, a Attempt) (bool, error) {
	if !fullParts(a.Parts) {
		return false, nil
	}
	switch a.Source.Kind {
	case SourceRandom:
		return true, nil
	case SourceTest:
		if a.Mode != ModeTest {
			return false, nil
		}
		prev, err := s.d.Attempts.List(ctx, a.UserID, AttemptFilter{TestKey: a.Source.TestKey, Counted: true, Limit: 1})
		if err != nil {
			return false, err
		}
		return len(prev) == 0, nil
	}
	return false, nil
}
