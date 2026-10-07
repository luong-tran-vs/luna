package topic

import "context"

// Repository stores topics and their roadmaps. Implementations live in internal/storage.
type Repository interface {
	// Create stores t and returns it with its ID; ErrNameTaken for a duplicate name.
	Create(ctx context.Context, t Topic) (Topic, error)
	// Get returns ErrNotFound when the topic does not exist (or the id is malformed).
	Get(ctx context.Context, id string) (Topic, error)
	// List returns every topic.
	List(ctx context.Context) ([]Topic, error)
	// Update changes name and description; ErrNameTaken, ErrNotFound.
	Update(ctx context.Context, id string, in Input) (Topic, error)
	Delete(ctx context.Context, id string) error
	// SetLessons replaces the roadmap of level.
	SetLessons(ctx context.Context, id, level string, lessonIDs []string) error
	// RemoveLesson takes a lesson out of the roadmap of level; removed is false when it was not there.
	RemoveLesson(ctx context.Context, id, level, lessonID string) (removed bool, err error)
	// AppendLesson adds a lesson at the end of the roadmap of level unless it is already there.
	AppendLesson(ctx context.Context, id, level, lessonID string) error
	// SetWords replaces the word list and marks it seeded (an admin edit is never overwritten).
	SetWords(ctx context.Context, id string, words []Word) (Topic, error)
}

// Lessons is what topics need from lessons (implemented over lesson.Repository in main).
type Lessons interface {
	// CountByTopic counts lessons per topic id and level.
	CountByTopic(ctx context.Context) (map[string]map[string]int, error)
	// PlaceOf returns the topic and level of each existing lesson among ids.
	PlaceOf(ctx context.Context, ids []string) (map[string]Place, error)
	// Refs returns the existing lessons among ids, in any order.
	Refs(ctx context.Context, ids []string) ([]LessonRef, error)
	// Texts returns the lessons of each topic among topicIDs (F18 coverage).
	Texts(ctx context.Context, topicIDs []string) (map[string][]LessonText, error)
}
