package topic

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	maxName        = 60
	maxDescription = 200
)

// Service manages topics and their roadmaps.
type Service struct {
	repo    Repository
	lessons Lessons
	ai      WordSuggester
	now     func() time.Time
}

// NewService returns a Service.
func NewService(repo Repository, lessons Lessons, suggester WordSuggester, now func() time.Time) *Service {
	return &Service{repo: repo, lessons: lessons, ai: suggester, now: now}
}

func cleanInput(in Input) Input {
	return Input{
		Name:        strings.Join(strings.Fields(in.Name), " "),
		Description: strings.TrimSpace(in.Description),
	}
}

func validate(in Input) error {
	fields := map[string]string{}
	switch n := utf8.RuneCountInString(in.Name); {
	case n == 0:
		fields["name"] = "Vui lòng nhập tên chủ đề"
	case n > maxName:
		fields["name"] = fmt.Sprintf("Tên chủ đề tối đa %d ký tự", maxName)
	}
	if utf8.RuneCountInString(in.Description) > maxDescription {
		fields["description"] = fmt.Sprintf("Mô tả tối đa %d ký tự", maxDescription)
	}
	if len(fields) > 0 {
		return &ValidationError{Fields: fields}
	}
	return nil
}

func nameTaken() error {
	return &ValidationError{Fields: map[string]string{"name": "Chủ đề này đã có"}}
}

// checkLevel rejects a missing or unknown level with the error of field.
func checkLevel(level, field string) error {
	if !ValidLevel(level) {
		return &ValidationError{Fields: map[string]string{field: "Vui lòng chọn trình độ từ A1 đến C2"}}
	}
	return nil
}

func sortTopics(topics []Topic) {
	slices.SortFunc(topics, func(a, b Topic) int { return strings.Compare(NameKey(a.Name), NameKey(b.Name)) })
}

// List returns every topic by name, with lesson counts and roadmap warnings per level.
func (s *Service) List(ctx context.Context) ([]Summary, error) {
	topics, err := s.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("topic: list: %w", err)
	}
	counts, err := s.lessons.CountByTopic(ctx)
	if err != nil {
		return nil, fmt.Errorf("topic: count lessons: %w", err)
	}
	sortTopics(topics)
	out := make([]Summary, len(topics))
	for i, t := range topics {
		out[i] = summarize(t, counts[t.ID])
	}
	if err := s.fillCoverage(ctx, out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Service) summary(ctx context.Context, t Topic) (Summary, error) {
	counts, err := s.lessons.CountByTopic(ctx)
	if err != nil {
		return Summary{}, fmt.Errorf("topic: count lessons: %w", err)
	}
	out := []Summary{summarize(t, counts[t.ID])}
	if err := s.fillCoverage(ctx, out); err != nil {
		return Summary{}, err
	}
	return out[0], nil
}

// Create adds a topic; the name must be unique.
func (s *Service) Create(ctx context.Context, in Input) (Summary, error) {
	in = cleanInput(in)
	if err := validate(in); err != nil {
		return Summary{}, err
	}
	now := s.now()
	t, err := s.repo.Create(ctx, Topic{
		Name: in.Name, Description: in.Description, Roadmaps: map[string][]string{}, CreatedAt: now, UpdatedAt: now,
	})
	if errors.Is(err, ErrNameTaken) {
		return Summary{}, nameTaken()
	}
	if err != nil {
		return Summary{}, fmt.Errorf("topic: create: %w", err)
	}
	return summarize(t, nil), nil
}

// Update edits a topic's name and description.
func (s *Service) Update(ctx context.Context, id string, in Input) (Summary, error) {
	in = cleanInput(in)
	if err := validate(in); err != nil {
		return Summary{}, err
	}
	t, err := s.repo.Update(ctx, id, in)
	switch {
	case errors.Is(err, ErrNameTaken):
		return Summary{}, nameTaken()
	case err != nil:
		return Summary{}, err
	}
	return s.summary(ctx, t)
}

// Delete removes a topic without lessons at any level; *InUseError while lessons remain.
func (s *Service) Delete(ctx context.Context, id string) error {
	if _, err := s.repo.Get(ctx, id); err != nil {
		return err
	}
	counts, err := s.lessons.CountByTopic(ctx)
	if err != nil {
		return fmt.Errorf("topic: count lessons: %w", err)
	}
	n := 0
	for _, c := range counts[id] {
		n += c
	}
	if n > 0 {
		return &InUseError{Count: n}
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("topic: delete: %w", err)
	}
	return nil
}

// Public lists, for learners, the topics that have a roadmap at level, with its number of lessons.
func (s *Service) Public(ctx context.Context, level string) ([]Public, error) {
	if err := checkLevel(level, "level"); err != nil {
		return nil, err
	}
	topics, err := s.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("topic: list: %w", err)
	}
	sortTopics(topics)
	var ids []string
	for _, t := range topics {
		ids = append(ids, t.Roadmap(level)...)
	}
	published, err := s.published(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := []Public{}
	for _, t := range topics {
		n := 0
		for _, lid := range t.Roadmap(level) {
			if published[lid] {
				n++
			}
		}
		if n > 0 {
			out = append(out, Public{ID: t.ID, Name: t.Name, Level: level, Description: t.Description, LessonCount: n})
		}
	}
	return out, nil
}

// LearnerRoadmap is the roadmap of t at level as learners see it: published lessons only, in order.
func (s *Service) LearnerRoadmap(ctx context.Context, t Topic, level string) ([]string, error) {
	ids := t.Roadmap(level)
	published, err := s.published(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := []string{}
	for _, lid := range ids {
		if published[lid] {
			out = append(out, lid)
		}
	}
	return out, nil
}

// published is the set of existing, non-draft lessons among ids.
func (s *Service) published(ctx context.Context, ids []string) (map[string]bool, error) {
	out := map[string]bool{}
	if len(ids) == 0 {
		return out, nil
	}
	refs, err := s.lessons.Refs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("topic: roadmap lessons: %w", err)
	}
	for _, r := range refs {
		if !r.Draft {
			out[r.ID] = true
		}
	}
	return out, nil
}

// Roadmap returns the roadmap of a topic at level with its lessons in order; deleted lessons are
// skipped.
func (s *Service) Roadmap(ctx context.Context, id, level string) (Roadmap, error) {
	if err := checkLevel(level, "level"); err != nil {
		return Roadmap{}, err
	}
	t, err := s.repo.Get(ctx, id)
	if err != nil {
		return Roadmap{}, err
	}
	sum, err := s.summary(ctx, t)
	if err != nil {
		return Roadmap{}, err
	}
	ids := t.Roadmap(level)
	refs, err := s.lessons.Refs(ctx, ids)
	if err != nil {
		return Roadmap{}, fmt.Errorf("topic: roadmap lessons: %w", err)
	}
	byID := make(map[string]LessonRef, len(refs))
	for _, r := range refs {
		byID[r.ID] = r
	}
	out := Roadmap{Topic: sum, Level: level, Lessons: []LessonRef{}}
	for _, lid := range ids {
		if r, ok := byID[lid]; ok {
			out.Lessons = append(out.Lessons, r)
		}
	}
	return out, nil
}

// SetRoadmap replaces the roadmap of a topic at level. Lessons must be unique, exist and belong
// to the topic at that level.
func (s *Service) SetRoadmap(ctx context.Context, id, level string, lessonIDs []string) (Roadmap, error) {
	if err := checkLevel(level, "level"); err != nil {
		return Roadmap{}, err
	}
	if _, err := s.repo.Get(ctx, id); err != nil {
		return Roadmap{}, err
	}
	bad := func(msg string) error { return &ValidationError{Fields: map[string]string{"lessonIds": msg}} }
	seen := map[string]bool{}
	for _, lid := range lessonIDs {
		if seen[lid] {
			return Roadmap{}, bad("Lộ trình có bài bị trùng")
		}
		seen[lid] = true
	}
	places, err := s.lessons.PlaceOf(ctx, lessonIDs)
	if err != nil {
		return Roadmap{}, fmt.Errorf("topic: lesson topics: %w", err)
	}
	for _, lid := range lessonIDs {
		switch p, ok := places[lid]; {
		case !ok:
			return Roadmap{}, bad("Lộ trình có bài không tồn tại")
		case p.TopicID != id:
			return Roadmap{}, bad("Chỉ thêm được bài của chủ đề này")
		case p.Level != level:
			return Roadmap{}, bad("Chỉ thêm được bài " + level + " của chủ đề này")
		}
	}
	if lessonIDs == nil {
		lessonIDs = []string{}
	}
	if err := s.repo.SetLessons(ctx, id, level, lessonIDs); err != nil {
		return Roadmap{}, fmt.Errorf("topic: save roadmap: %w", err)
	}
	return s.Roadmap(ctx, id, level)
}

// --- used by lessons (lesson.Topics port, adapted in main) ---

// Get returns a topic or ErrNotFound.
func (s *Service) Get(ctx context.Context, id string) (Topic, error) {
	return s.repo.Get(ctx, id)
}

// All returns every topic.
func (s *Service) All(ctx context.Context) ([]Topic, error) {
	topics, err := s.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("topic: list: %w", err)
	}
	return topics, nil
}

// RoadmapLessonIDs is the set of lessons in any roadmap.
func (s *Service) RoadmapLessonIDs(ctx context.Context) (map[string]bool, error) {
	topics, err := s.All(ctx)
	if err != nil {
		return nil, err
	}
	out := map[string]bool{}
	for _, t := range topics {
		for _, lid := range t.AllRoadmapLessons() {
			out[lid] = true
		}
	}
	return out, nil
}

// AppendLesson adds a lesson at the end of the roadmap of p unless it is already there;
// ErrNotFound when the topic does not exist.
func (s *Service) AppendLesson(ctx context.Context, p Place, lessonID string) error {
	if _, err := s.repo.Get(ctx, p.TopicID); err != nil {
		return err
	}
	if err := s.repo.AppendLesson(ctx, p.TopicID, p.Level, lessonID); err != nil {
		return fmt.Errorf("topic: append to roadmap: %w", err)
	}
	return nil
}

// MoveLesson takes a lesson out of from's roadmap and, if it was there, appends it to to's
// roadmap. The removal comes first: a failure in between leaves the lesson out of both
// roadmaps rather than in two.
func (s *Service) MoveLesson(ctx context.Context, lessonID string, from, to Place) error {
	if from.TopicID == "" || from == to {
		return nil
	}
	removed, err := s.repo.RemoveLesson(ctx, from.TopicID, from.Level, lessonID)
	if err != nil {
		return fmt.Errorf("topic: remove from roadmap: %w", err)
	}
	if !removed {
		return nil
	}
	if err := s.repo.AppendLesson(ctx, to.TopicID, to.Level, lessonID); err != nil {
		return fmt.Errorf("topic: append to roadmap: %w", err)
	}
	return nil
}
