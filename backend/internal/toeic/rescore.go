package toeic

import (
	"context"
	"errors"
	"slices"
)

// attemptsWith returns the submitted attempts having any of the groups, each once.
func (s *Service) attemptsWith(ctx context.Context, keys []string) ([]Attempt, error) {
	var out []Attempt
	seen := map[string]bool{}
	for _, k := range keys {
		list, err := s.d.Attempts.WithGroup(ctx, k)
		if err != nil {
			return nil, err
		}
		for _, a := range list {
			if !seen[a.ID] && a.Status == AttemptSubmitted {
				seen[a.ID] = true
				out = append(out, a)
			}
		}
	}
	return out, nil
}

// countRescore returns how many submitted attempts replacing the groups would grade again.
func (s *Service) countRescore(ctx context.Context, keys []string) (int, error) {
	if len(keys) == 0 {
		return 0, nil
	}
	list, err := s.attemptsWith(ctx, keys)
	return len(list), err
}

// rescore grades again every submitted attempt having a replaced group, then brings the notebooks
// in line (research R8): for each learner and question of those groups, the answer of the latest
// attempt with it decides: right under the new key takes the question out, wrong adds it.
func (s *Service) rescore(ctx context.Context, keys []string) (int, error) {
	list, err := s.attemptsWith(ctx, keys)
	if err != nil {
		return 0, err
	}
	type latest struct {
		a       Attempt
		outcome string
	}
	last := map[[2]string]latest{}
	for _, a := range list {
		groups, err := s.groupsOf(ctx, a.GroupKeys)
		if err != nil {
			return 0, err
		}
		r := grade(groups, a)
		if a.Result != nil {
			r.DurationMs = a.Result.DurationMs
		}
		if err := s.d.Attempts.UpdateResult(ctx, a.ID, r); err != nil {
			return 0, err
		}
		for _, id := range a.QuestionIDs {
			key, _, ok := SplitQuestionID(id)
			if !ok || !slices.Contains(keys, key) {
				continue
			}
			k := [2]string{a.UserID, id}
			if prev, ok := last[k]; !ok || a.SubmittedAt.After(prev.a.SubmittedAt) {
				last[k] = latest{a: a, outcome: r.ByQuestion[id]}
			}
		}
	}
	for k, l := range last {
		userID, id := k[0], k[1]
		if l.outcome == OutcomeRight {
			if err := s.d.Mistakes.Delete(ctx, userID, id); err != nil {
				return 0, err
			}
			continue
		}
		if _, err := s.d.Mistakes.Get(ctx, userID, id); err == nil {
			continue
		} else if !errors.Is(err, ErrNotFound) {
			return 0, err
		}
		key, _, _ := SplitQuestionID(id)
		groups, err := s.groupsOf(ctx, []string{key})
		if err != nil {
			return 0, err
		}
		if err := s.markWrong(ctx, userID, id, groups); err != nil {
			return 0, err
		}
	}
	return len(list), nil
}
