package topic

import "context"

// Repository stores topics and their roadmaps. Implementations live in internal/storage.
type Repository interface {
	// Create stores t and returns it with its ID; ErrNameTaken for a duplicate (level, name).
	Create(ctx context.Context, t Topic) (Topic, error)
	// Get returns ErrNotFound when the topic does not exist (or the id is malformed).
	Get(ctx context.Context, id string) (Topic, error)
	// List returns the topics of level ("" = every level).
	List(ctx context.Context, level string) ([]Topic, error)
	// Update changes name, level and description; ErrNameTaken, ErrNotFound.
	Update(ctx context.Context, id string, in Input) (Topic, error)
	Delete(ctx context.Context, id string) error
	// SetLessons replaces the roadmap.
	SetLessons(ctx context.Context, id string, lessonIDs []string) error
	// RemoveLesson takes a lesson out of the roadmap; removed is false when it was not there.
	RemoveLesson(ctx context.Context, id, lessonID string) (removed bool, err error)
	// AppendLesson adds a lesson at the end of the roadmap unless it is already there.
	AppendLesson(ctx context.Context, id, lessonID string) error
}

// Lessons is what topics need from lessons (implemented over lesson.Repository in main).
type Lessons interface {
	// CountByTopic counts lessons per topic id.
	CountByTopic(ctx context.Context) (map[string]int, error)
	// TopicOf returns the topic id of each existing lesson among ids.
	TopicOf(ctx context.Context, ids []string) (map[string]string, error)
	// Refs returns the existing lessons among ids, in any order.
	Refs(ctx context.Context, ids []string) ([]LessonRef, error)
	// SetLevelByTopic sets the level of every lesson of a topic.
	SetLevelByTopic(ctx context.Context, topicID, level string) error
}
