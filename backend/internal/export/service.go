package export

import (
	"context"
	"fmt"
	"time"
)

// Service builds exports. The user is always the session user.
type Service struct {
	repo     Repository
	settings Settings
	now      func() time.Time
}

// NewService returns a Service.
func NewService(repo Repository, settings Settings, now func() time.Time) *Service {
	return &Service{repo: repo, settings: settings, now: now}
}

// Build gathers the user's data and the download file name (luna-export-YYYYMMDD.json, the date
// in the learner's timezone). ErrNotFound when the account no longer exists.
func (s *Service) Build(ctx context.Context, userID string) (Export, string, error) {
	account, err := s.repo.Account(ctx, userID)
	if err != nil {
		return Export{}, "", fmt.Errorf("export: account: %w", err)
	}
	values, err := s.settings.Values(ctx, userID)
	if err != nil {
		return Export{}, "", fmt.Errorf("export: settings: %w", err)
	}
	loc, err := s.settings.Location(ctx, userID)
	if err != nil {
		return Export{}, "", fmt.Errorf("export: timezone: %w", err)
	}
	now := s.now().In(loc)
	e := Export{Version: Version, ExportedAt: now, Account: account, Settings: values}

	for _, part := range []struct {
		collection string
		dst        *[]Doc
	}{
		{CollCards, &e.Cards},
		{CollReviewLogs, &e.ReviewLogs},
		{CollGoals, &e.Goals},
		{CollLessonProgress, &e.LessonProgress},
		{CollStudyDays, &e.StudyDays},
		{CollDictationResults, &e.DictationResults},
		{CollReadingAnswers, &e.ReadingAnswers},
		{CollWritings, &e.Writings},
		{CollGrammarProgress, &e.GrammarProgress},
	} {
		docs, err := s.repo.UserDocs(ctx, part.collection, userID)
		if err != nil {
			return Export{}, "", fmt.Errorf("export: %s: %w", part.collection, err)
		}
		*part.dst = withoutUser(docs)
	}

	if e.Lessons, err = s.lessons(ctx, e.LessonProgress); err != nil {
		return Export{}, "", err
	}
	return e, "luna-export-" + now.Format("20060102") + ".json", nil
}

// withoutUser drops the userId of each document (every document is the learner's) and never
// returns nil, so empty parts are [] in JSON.
func withoutUser(docs []Doc) []Doc {
	out := make([]Doc, 0, len(docs))
	for _, d := range docs {
		delete(d, "userId")
		out = append(out, d)
	}
	return out
}

// lessons returns the lessons the learner started, in progress order; a lesson deleted since is
// {id, deleted: true}.
func (s *Service) lessons(ctx context.Context, progress []Doc) ([]Doc, error) {
	var ids []string
	seen := map[string]bool{}
	for _, p := range progress {
		id, ok := p["lessonId"].(string)
		if ok && id != "" && !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	out := make([]Doc, 0, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	found, err := s.repo.Lessons(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("export: lessons: %w", err)
	}
	byID := make(map[string]Doc, len(found))
	for _, l := range found {
		if id, ok := l["_id"].(string); ok {
			byID[id] = l
		}
	}
	for _, id := range ids {
		if l, ok := byID[id]; ok {
			out = append(out, l)
		} else {
			out = append(out, Doc{"id": id, "deleted": true})
		}
	}
	return out, nil
}
