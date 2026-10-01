package topic

import (
	"cmp"
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
	now     func() time.Time
}

// NewService returns a Service.
func NewService(repo Repository, lessons Lessons, now func() time.Time) *Service {
	return &Service{repo: repo, lessons: lessons, now: now}
}

func cleanInput(in Input) Input {
	return Input{
		Name:        strings.Join(strings.Fields(in.Name), " "),
		Level:       strings.ToUpper(strings.TrimSpace(in.Level)),
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
	if !ValidLevel(in.Level) {
		fields["level"] = "Vui lòng chọn trình độ từ A1 đến C2"
	}
	if utf8.RuneCountInString(in.Description) > maxDescription {
		fields["description"] = fmt.Sprintf("Mô tả tối đa %d ký tự", maxDescription)
	}
	if len(fields) > 0 {
		return &ValidationError{Fields: fields}
	}
	return nil
}

func nameTaken(level string) error {
	return &ValidationError{Fields: map[string]string{"name": "Chủ đề này đã có ở trình độ " + level}}
}

func sortTopics(topics []Topic) {
	slices.SortFunc(topics, func(a, b Topic) int {
		if c := cmp.Compare(levelRank(a.Level), levelRank(b.Level)); c != 0 {
			return c
		}
		return strings.Compare(NameKey(a.Name), NameKey(b.Name))
	})
}

// List returns the topics of level ("" = all), A1 → C2 then by name, with lesson counts and
// roadmap warnings.
func (s *Service) List(ctx context.Context, level string) ([]Summary, error) {
	if level != "" && !ValidLevel(level) {
		return nil, &ValidationError{Fields: map[string]string{"level": "Trình độ không hợp lệ"}}
	}
	topics, err := s.repo.List(ctx, level)
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
	return out, nil
}

func (s *Service) summary(ctx context.Context, t Topic) (Summary, error) {
	counts, err := s.lessons.CountByTopic(ctx)
	if err != nil {
		return Summary{}, fmt.Errorf("topic: count lessons: %w", err)
	}
	return summarize(t, counts[t.ID]), nil
}

// Create adds a topic; the name must be unique within its level.
func (s *Service) Create(ctx context.Context, in Input) (Summary, error) {
	in = cleanInput(in)
	if err := validate(in); err != nil {
		return Summary{}, err
	}
	now := s.now()
	t, err := s.repo.Create(ctx, Topic{
		Name: in.Name, Level: in.Level, Description: in.Description, LessonIDs: []string{}, CreatedAt: now, UpdatedAt: now,
	})
	if errors.Is(err, ErrNameTaken) {
		return Summary{}, nameTaken(in.Level)
	}
	if err != nil {
		return Summary{}, fmt.Errorf("topic: create: %w", err)
	}
	return summarize(t, 0), nil
}

// Update edits a topic. Changing its level changes the level of all its lessons.
func (s *Service) Update(ctx context.Context, id string, in Input) (Summary, error) {
	in = cleanInput(in)
	if err := validate(in); err != nil {
		return Summary{}, err
	}
	cur, err := s.repo.Get(ctx, id)
	if err != nil {
		return Summary{}, err
	}
	t, err := s.repo.Update(ctx, id, in)
	switch {
	case errors.Is(err, ErrNameTaken):
		return Summary{}, nameTaken(in.Level)
	case err != nil:
		return Summary{}, fmt.Errorf("topic: update: %w", err)
	}
	if cur.Level != in.Level {
		if err := s.lessons.SetLevelByTopic(ctx, id, in.Level); err != nil {
			return Summary{}, fmt.Errorf("topic: update lesson levels: %w", err)
		}
	}
	return s.summary(ctx, t)
}

// Delete removes a topic without lessons; *InUseError while lessons remain.
func (s *Service) Delete(ctx context.Context, id string) error {
	if _, err := s.repo.Get(ctx, id); err != nil {
		return err
	}
	counts, err := s.lessons.CountByTopic(ctx)
	if err != nil {
		return fmt.Errorf("topic: count lessons: %w", err)
	}
	if n := counts[id]; n > 0 {
		return &InUseError{Count: n}
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("topic: delete: %w", err)
	}
	return nil
}

// Public lists the topics of a level for learners, with the number of roadmap lessons.
func (s *Service) Public(ctx context.Context, level string) ([]Public, error) {
	if level != "" && !ValidLevel(level) {
		return nil, &ValidationError{Fields: map[string]string{"level": "Trình độ không hợp lệ"}}
	}
	topics, err := s.repo.List(ctx, level)
	if err != nil {
		return nil, fmt.Errorf("topic: list: %w", err)
	}
	sortTopics(topics)
	out := make([]Public, len(topics))
	for i, t := range topics {
		out[i] = Public{ID: t.ID, Name: t.Name, Level: t.Level, Description: t.Description, LessonCount: len(t.LessonIDs)}
	}
	return out, nil
}

// Roadmap returns a topic's roadmap with its lessons in order; deleted lessons are skipped.
func (s *Service) Roadmap(ctx context.Context, id string) (Roadmap, error) {
	t, err := s.repo.Get(ctx, id)
	if err != nil {
		return Roadmap{}, err
	}
	sum, err := s.summary(ctx, t)
	if err != nil {
		return Roadmap{}, err
	}
	refs, err := s.lessons.Refs(ctx, t.LessonIDs)
	if err != nil {
		return Roadmap{}, fmt.Errorf("topic: roadmap lessons: %w", err)
	}
	byID := make(map[string]LessonRef, len(refs))
	for _, r := range refs {
		byID[r.ID] = r
	}
	out := Roadmap{Topic: sum, Lessons: []LessonRef{}}
	for _, lid := range t.LessonIDs {
		if r, ok := byID[lid]; ok {
			out.Lessons = append(out.Lessons, r)
		}
	}
	return out, nil
}

// SetRoadmap replaces a topic's roadmap. Lessons must be unique, exist and belong to the topic.
func (s *Service) SetRoadmap(ctx context.Context, id string, lessonIDs []string) (Roadmap, error) {
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
	topicOf, err := s.lessons.TopicOf(ctx, lessonIDs)
	if err != nil {
		return Roadmap{}, fmt.Errorf("topic: lesson topics: %w", err)
	}
	for _, lid := range lessonIDs {
		switch tid, ok := topicOf[lid]; {
		case !ok:
			return Roadmap{}, bad("Lộ trình có bài không tồn tại")
		case tid != id:
			return Roadmap{}, bad("Chỉ thêm được bài của chủ đề này")
		}
	}
	if lessonIDs == nil {
		lessonIDs = []string{}
	}
	if err := s.repo.SetLessons(ctx, id, lessonIDs); err != nil {
		return Roadmap{}, fmt.Errorf("topic: save roadmap: %w", err)
	}
	return s.Roadmap(ctx, id)
}

// --- used by lessons (lesson.Topics port, adapted in main) ---

// Get returns a topic or ErrNotFound.
func (s *Service) Get(ctx context.Context, id string) (Topic, error) {
	return s.repo.Get(ctx, id)
}

// All returns every topic.
func (s *Service) All(ctx context.Context) ([]Topic, error) {
	topics, err := s.repo.List(ctx, "")
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
		for _, lid := range t.LessonIDs {
			out[lid] = true
		}
	}
	return out, nil
}

// AppendLesson adds a lesson at the end of a topic roadmap unless it is already there;
// ErrNotFound when the topic does not exist.
func (s *Service) AppendLesson(ctx context.Context, topicID, lessonID string) error {
	if _, err := s.repo.Get(ctx, topicID); err != nil {
		return err
	}
	if err := s.repo.AppendLesson(ctx, topicID, lessonID); err != nil {
		return fmt.Errorf("topic: append to roadmap: %w", err)
	}
	return nil
}

// MoveLesson takes a lesson out of from's roadmap and, if it was there, appends it to to's
// roadmap. The removal comes first: a failure in between leaves the lesson out of both
// roadmaps rather than in two.
func (s *Service) MoveLesson(ctx context.Context, lessonID, from, to string) error {
	if from == "" || from == to {
		return nil
	}
	removed, err := s.repo.RemoveLesson(ctx, from, lessonID)
	if err != nil {
		return fmt.Errorf("topic: remove from roadmap: %w", err)
	}
	if !removed {
		return nil
	}
	if err := s.repo.AppendLesson(ctx, to, lessonID); err != nil {
		return fmt.Errorf("topic: append to roadmap: %w", err)
	}
	return nil
}
