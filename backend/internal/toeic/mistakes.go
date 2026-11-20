package toeic

import (
	"context"
	"errors"
	"fmt"
	"sort"
)

// streakToLeave is how many right answers in a row take a question out of the notebook.
const streakToLeave = 2

// applyMistake updates an entry of the notebook with an answer and reports whether to keep it. In a
// submitted attempt a wrong or skipped answer resets the streak and a right one changes nothing; in
// the notebook's own round (drill) a right answer counts toward leaving.
func applyMistake(entry *Mistake, right, drill bool) (keep bool) {
	switch {
	case !right:
		entry.Streak = 0
	case drill:
		entry.Streak++
	}
	return entry.Streak < streakToLeave
}

// recordSubmitted puts the wrong and skipped questions of a submitted attempt in the notebook.
func (s *Service) recordSubmitted(ctx context.Context, a Attempt, groups map[string]Group) error {
	for _, id := range a.QuestionIDs {
		outcome := a.Result.ByQuestion[id]
		if outcome == OutcomeRight || outcome == "" {
			continue
		}
		if err := s.markWrong(ctx, a.UserID, id, groups); err != nil {
			return err
		}
	}
	return nil
}

// markWrong adds a question to the notebook or resets its streak.
func (s *Service) markWrong(ctx context.Context, userID, id string, groups map[string]Group) error {
	key, no, ok := SplitQuestionID(id)
	g, found := groups[key]
	if !ok || !found {
		return nil
	}
	q, found := g.Question(no)
	if !found {
		return nil
	}
	now := s.now()
	m, err := s.d.Mistakes.Get(ctx, userID, id)
	if errors.Is(err, ErrNotFound) {
		m = Mistake{UserID: userID, QuestionID: id, AddedAt: now}
	} else if err != nil {
		return err
	}
	m.GroupKey, m.Part, m.Tags, m.LastWrongAt = key, g.Part, q.Tags, now
	applyMistake(&m, false, false)
	return s.d.Mistakes.Upsert(ctx, m)
}

// MistakeItem is a question of the notebook.
type MistakeItem struct {
	QuestionID string   `json:"questionId"`
	GroupKey   string   `json:"groupKey"`
	Part       Part     `json:"part"`
	Tags       []string `json:"tags"`
	Streak     int      `json:"streak"`
	Prompt     string   `json:"prompt"`
}

// MistakesView is the notebook, with how many questions each part has.
type MistakesView struct {
	Items  []MistakeItem `json:"items"`
	Counts map[Part]int  `json:"counts"`
	Need   int           `json:"need"`
}

// Mistakes returns the notebook, filtered by part (0: all).
func (s *Service) Mistakes(ctx context.Context, userID string, part Part) (MistakesView, error) {
	if part != 0 && !part.Valid() {
		return MistakesView{}, invalid("part", "Part phải từ 1 đến 7")
	}
	all, err := s.d.Mistakes.List(ctx, userID, 0)
	if err != nil {
		return MistakesView{}, err
	}
	v := MistakesView{Items: []MistakeItem{}, Counts: map[Part]int{}, Need: streakToLeave}
	var keys []string
	for _, m := range all {
		v.Counts[m.Part]++
		if part == 0 || m.Part == part {
			keys = append(keys, m.GroupKey)
		}
	}
	groups, err := s.groupsOf(ctx, keys)
	if err != nil {
		return MistakesView{}, err
	}
	for _, m := range all {
		if part != 0 && m.Part != part {
			continue
		}
		item := MistakeItem{QuestionID: m.QuestionID, GroupKey: m.GroupKey, Part: m.Part, Tags: m.Tags, Streak: m.Streak}
		if item.Tags == nil {
			item.Tags = []string{}
		}
		if g, ok := groups[m.GroupKey]; ok {
			_, no, _ := SplitQuestionID(m.QuestionID)
			if q, ok := g.Question(no); ok && q.Prompt != "" {
				item.Prompt = q.Prompt
			} else {
				item.Prompt = groupTitle(g)
			}
		}
		v.Items = append(v.Items, item)
	}
	return v, nil
}

// RoundItem is a question of a notebook round, with its whole group shown.
type RoundItem struct {
	QuestionID string     `json:"questionId"`
	Streak     int        `json:"streak"`
	Group      PaperGroup `json:"group"`
}

// MistakeRound returns up to count questions of the notebook, wrong longest ago first.
func (s *Service) MistakeRound(ctx context.Context, userID string, part Part, count int) ([]RoundItem, error) {
	if part != 0 && !part.Valid() {
		return nil, invalid("part", "Part phải từ 1 đến 7")
	}
	if count <= 0 {
		count = 10
	}
	count = min(count, 30)
	list, err := s.d.Mistakes.List(ctx, userID, part)
	if err != nil {
		return nil, err
	}
	sort.SliceStable(list, func(i, j int) bool { return list[i].LastWrongAt.Before(list[j].LastWrongAt) })
	keys := make([]string, 0, len(list))
	for _, m := range list {
		keys = append(keys, m.GroupKey)
	}
	groups, err := s.groupsOf(ctx, keys)
	if err != nil {
		return nil, err
	}
	out := []RoundItem{}
	for _, m := range list {
		if len(out) == count {
			break
		}
		g, ok := groups[m.GroupKey]
		if !ok || g.Status != StatusPublished {
			continue
		}
		pv, err := s.paperView(ctx, []Group{g}, []string{m.QuestionID}, map[string]int{m.QuestionID: len(out) + 1})
		if err != nil {
			return nil, err
		}
		if len(pv) == 0 {
			continue
		}
		out = append(out, RoundItem{QuestionID: m.QuestionID, Streak: m.Streak, Group: pv[0]})
	}
	return out, nil
}

// CheckReply is the instant grade of a notebook question.
type CheckReply struct {
	Right       bool   `json:"right"`
	Answer      int    `json:"answer"`
	Streak      int    `json:"streak"`
	Removed     bool   `json:"removed"`
	ExplainVi   string `json:"explainVi"`
	Evidence    []Ref  `json:"evidence"`
	TranslateVi string `json:"translateVi,omitempty"`
}

// CheckMistake grades one notebook question at once: two right answers in a row take it out.
func (s *Service) CheckMistake(ctx context.Context, userID, qid string, choice int) (CheckReply, error) {
	m, err := s.d.Mistakes.Get(ctx, userID, qid)
	if err != nil {
		return CheckReply{}, err
	}
	key, no, ok := SplitQuestionID(qid)
	if !ok {
		return CheckReply{}, ErrNotFound
	}
	g, err := s.GetGroup(ctx, key)
	if err != nil {
		return CheckReply{}, err
	}
	q, ok := g.Question(no)
	if !ok {
		return CheckReply{}, ErrNotFound
	}
	if choice < 0 || choice >= len(q.Options) {
		return CheckReply{}, invalid("choice", fmt.Sprintf("Chọn từ 0 đến %d", len(q.Options)-1))
	}
	right := choice == q.Answer
	keep := applyMistake(&m, right, true)
	if !right {
		m.LastWrongAt = s.now()
	}
	if keep {
		err = s.d.Mistakes.Upsert(ctx, m)
	} else {
		err = s.d.Mistakes.Delete(ctx, userID, qid)
	}
	if err != nil {
		return CheckReply{}, err
	}
	reply := CheckReply{Right: right, Answer: q.Answer, Streak: m.Streak, Removed: !keep, ExplainVi: q.ExplainVi,
		Evidence: q.Evidence, TranslateVi: q.TranslateVi}
	if reply.Evidence == nil {
		reply.Evidence = []Ref{}
	}
	return reply, nil
}

// FinishMistakes ends a notebook round: the day counts as a study day.
func (s *Service) FinishMistakes(ctx context.Context, userID string) error {
	return s.d.StudyDays.MarkStudied(ctx, userID)
}
