package toeic

import (
	"context"
	"fmt"
	"strings"
)

// maxDeleteKeys bounds one delete request.
const maxDeleteKeys = 500

// DeleteBlocked is a group that was not deleted, and why.
type DeleteBlocked struct {
	Key    string `json:"key"`
	Reason string `json:"reason"`
}

// DeleteReport says what a delete did: the groups removed (with their pictures), the ones kept
// because something uses them, and the keys that do not exist.
type DeleteReport struct {
	Deleted []string        `json:"deleted"`
	Blocked []DeleteBlocked `json:"blocked"`
	Missing []string        `json:"missing"`
}

// DeleteGroups removes groups from the bank, with their pictures. A group that a test holds (in any
// status) or that a learner's submitted attempt has is kept and reported: deleting it would break
// the test or the learner's results and mistakes notebook; it can be hidden instead.
func (s *Service) DeleteGroups(ctx context.Context, keys []string) (DeleteReport, error) {
	rep := DeleteReport{Deleted: []string{}, Blocked: []DeleteBlocked{}, Missing: []string{}}
	keys = uniqueKeys(keys)
	if len(keys) == 0 {
		return rep, invalid("keys", "Chọn ít nhất một cụm để xoá")
	}
	if len(keys) > maxDeleteKeys {
		return rep, invalid("keys", fmt.Sprintf("Mỗi lần xoá tối đa %d cụm", maxDeleteKeys))
	}
	found, err := s.d.Groups.ByKeys(ctx, keys)
	if err != nil {
		return rep, err
	}
	exists := map[string]bool{}
	for _, g := range found {
		exists[g.Key] = true
	}
	tests, err := s.d.Tests.List(ctx, false)
	if err != nil {
		return rep, err
	}
	inTests := map[string][]string{}
	for _, t := range tests {
		for _, k := range t.Groups {
			inTests[k] = append(inTests[k], t.Key)
		}
	}

	var del []string
	for _, k := range keys {
		if !exists[k] {
			rep.Missing = append(rep.Missing, k)
			continue
		}
		if ts := inTests[k]; len(ts) > 0 {
			rep.Blocked = append(rep.Blocked, DeleteBlocked{Key: k,
				Reason: fmt.Sprintf("Đang thuộc đề %s: gỡ khỏi đề trước", strings.Join(ts, ", "))})
			continue
		}
		done, err := s.attemptsWith(ctx, []string{k})
		if err != nil {
			return rep, err
		}
		if len(done) > 0 {
			rep.Blocked = append(rep.Blocked, DeleteBlocked{Key: k,
				Reason: fmt.Sprintf("Đã có %d bài làm của người học: hãy ẩn thay vì xoá", len(done))})
			continue
		}
		del = append(del, k)
	}
	if len(del) == 0 {
		return rep, nil
	}

	pics, err := s.d.Images.Has(ctx, del)
	if err != nil {
		return rep, err
	}
	for k, slots := range pics {
		for _, slot := range slots {
			if err := s.d.Images.Delete(ctx, k, slot); err != nil {
				return rep, err
			}
		}
	}
	if err := s.d.Groups.Delete(ctx, del); err != nil {
		return rep, err
	}
	rep.Deleted = del
	return rep, nil
}

// uniqueKeys is the keys trimmed, without empty ones and repeats, in order.
func uniqueKeys(keys []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, k := range keys {
		k = strings.TrimSpace(k)
		if k != "" && !seen[k] {
			seen[k] = true
			out = append(out, k)
		}
	}
	return out
}
