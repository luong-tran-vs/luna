package lesson

import (
	"context"

	"github.com/luongtran/luna/backend/internal/job"
)

// Info is the part of a lesson an admin edits without changing its content.
type Info struct {
	Title   string
	Level   Level
	TopicID string
	Source  string
	License string
}

// Repository stores lessons. Implementations live in internal/storage.
type Repository interface {
	// Create stores l and returns it with its ID.
	Create(ctx context.Context, l Lesson) (Lesson, error)
	// Get returns ErrNotFound when the lesson does not exist (or the id is malformed).
	Get(ctx context.Context, id string) (Lesson, error)
	List(ctx context.Context, f Filter) ([]Summary, error)
	// Summaries returns the lessons that exist among ids, in any order.
	Summaries(ctx context.Context, ids []string) ([]Summary, error)
	UpdateInfo(ctx context.Context, id string, info Info) error
	// ReplaceContent overwrites content, revision, sentences, annotations and statuses.
	ReplaceContent(ctx context.Context, l Lesson) error
	// SetStatus changes one work status if the lesson is still at revision; ok reports that.
	SetStatus(ctx context.Context, id string, revision int, t job.Type, st Status, errMsg string) (bool, error)
	// SaveAudio stores sentence audio paths and marks audio done if still at revision.
	SaveAudio(ctx context.Context, id string, revision int, paths []string) (bool, error)
	// SaveAnnotations stores AI annotations and marks them done if still at revision.
	SaveAnnotations(ctx context.Context, id string, revision int, anns []Annotation) (bool, error)
	// ReplaceAnnotations stores admin-edited annotations and marks them done.
	ReplaceAnnotations(ctx context.Context, id string, anns []Annotation) error
	Delete(ctx context.Context, id string) error
}

// Topics is what lessons need from topics (implemented over topic.Service in main).
type Topics interface {
	// Get returns ErrTopicNotFound when the topic does not exist.
	Get(ctx context.Context, id string) (TopicRef, error)
	// Names returns every topic by id.
	Names(ctx context.Context) (map[string]TopicRef, error)
	// RoadmapLessonIDs is the set of lessons in any topic roadmap.
	RoadmapLessonIDs(ctx context.Context) (map[string]bool, error)
	// MoveLesson takes a lesson out of topic from's roadmap and, if it was there, appends it
	// to topic to's roadmap.
	MoveLesson(ctx context.Context, lessonID, from, to string) error
}
