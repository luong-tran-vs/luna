package toeic

import (
	"context"
	"fmt"
	"slices"
	"strings"
)

// Report status of one group of an import.
const (
	ReportOK        = "ok"
	ReportDuplicate = "duplicate"
	ReportReplace   = "replace"
	ReportError     = "error"
)

// GroupReport is what an import says about one group.
type GroupReport struct {
	Key       string            `json:"key"`
	Part      Part              `json:"part"`
	Questions int               `json:"questions"`
	Status    string            `json:"status"`
	Fields    map[string]string `json:"fields,omitempty"`
}

// ImportReport is the answer of an import or a dry run.
type ImportReport struct {
	DryRun     bool     `json:"dryRun"`
	Created    []string `json:"created"`
	Replaced   []string `json:"replaced"`
	Duplicates []string `json:"duplicates"`
	// Rescored is how many submitted attempts were (or, in a dry run, would be) graded again.
	Rescored int `json:"rescored"`
	// ListeningMinutes is the reading time of the listening groups of the file, nil without any.
	ListeningMinutes *float64      `json:"listeningMinutes"`
	Reports          []GroupReport `json:"reports"`
}

// Import checks the groups of a file and, unless dryRun, stores them: new groups as drafts, known
// keys replaced when replace is set (skipped as duplicates otherwise). A file with any error is
// refused whole with a ValidationError.
func (s *Service) Import(ctx context.Context, gs []Group, replace, dryRun bool) (ImportReport, error) {
	rep := ImportReport{DryRun: dryRun, Created: []string{}, Replaced: []string{}, Duplicates: []string{}, Reports: []GroupReport{}}
	if len(gs) == 0 {
		return rep, invalid("groups", "File không có cụm nào")
	}
	for i := range gs {
		Normalize(&gs[i])
	}
	fields := map[string]string{}
	if err := Validate(gs); err != nil {
		fields = err.(*ValidationError).Fields
	}

	keys := make([]string, len(gs))
	for i, g := range gs {
		keys[i] = g.Key
	}
	existing, err := s.d.Groups.ByKeys(ctx, keys)
	if err != nil {
		return rep, err
	}
	old := make(map[string]Group, len(existing))
	for _, g := range existing {
		old[g.Key] = g
	}
	inPublished, err := s.publishedTestGroups(ctx)
	if err != nil {
		return rep, err
	}

	var save []Group
	var replacedKeys []string
	for i, g := range gs {
		prefix := fmt.Sprintf("groups[%d].", i)
		r := GroupReport{Key: g.Key, Part: g.Part, Questions: len(g.Questions), Status: ReportOK}
		prev, known := old[g.Key]
		switch {
		case known && !replace:
			r.Status = ReportDuplicate
		case known:
			r.Status = ReportReplace
			if prev.Part != g.Part {
				fields[prefix+"part"] = fmt.Sprintf("Cụm đã có ở Part %d, không đổi Part được", prev.Part)
			}
			if tests := inPublished[g.Key]; len(tests) > 0 && len(prev.Questions) != len(g.Questions) {
				fields[prefix+"questions"] = fmt.Sprintf("Cụm đang ở đề đã đăng (%s): phải giữ %d câu",
					strings.Join(tests, ", "), len(prev.Questions))
			}
		}
		for k, v := range fields {
			if rest, ok := strings.CutPrefix(k, prefix); ok {
				if r.Fields == nil {
					r.Fields = map[string]string{}
				}
				r.Fields[rest] = v
			}
		}
		if len(r.Fields) > 0 {
			r.Status = ReportError
		}
		rep.Reports = append(rep.Reports, r)
		switch r.Status {
		case ReportDuplicate:
			rep.Duplicates = append(rep.Duplicates, g.Key)
		case ReportReplace:
			rep.Replaced = append(rep.Replaced, g.Key)
			replacedKeys = append(replacedKeys, g.Key)
			save = append(save, g)
		case ReportOK:
			rep.Created = append(rep.Created, g.Key)
			save = append(save, g)
		}
	}
	if slices.ContainsFunc(gs, func(g Group) bool { return g.Part.Valid() && g.Part <= 4 }) {
		m := estimateListening(gs)
		rep.ListeningMinutes = &m
	}

	if len(fields) > 0 {
		if dryRun {
			return rep, nil
		}
		return rep, &ValidationError{Fields: fields}
	}
	if dryRun {
		n, err := s.countRescore(ctx, replacedKeys)
		if err != nil {
			return rep, err
		}
		rep.Rescored = n
		return rep, nil
	}
	if len(save) == 0 {
		return rep, nil
	}
	now := s.now()
	for i := range save {
		save[i].Status, save[i].Version = StatusDraft, 1
		save[i].CreatedAt, save[i].UpdatedAt = now, now
	}
	if _, _, err := s.d.Groups.Upsert(ctx, save); err != nil {
		return rep, err
	}
	if len(replacedKeys) > 0 {
		n, err := s.rescore(ctx, replacedKeys)
		if err != nil {
			return rep, err
		}
		rep.Rescored = n
	}
	return rep, nil
}

// publishedTestGroups returns, by group key, the published tests that have the group.
func (s *Service) publishedTestGroups(ctx context.Context) (map[string][]string, error) {
	tests, err := s.d.Tests.List(ctx, true)
	if err != nil {
		return nil, err
	}
	out := map[string][]string{}
	for _, t := range tests {
		for _, k := range t.Groups {
			out[k] = append(out[k], t.Key)
		}
	}
	return out, nil
}
