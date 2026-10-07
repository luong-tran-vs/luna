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
	// GrammarPointID is "" to clear the assigned point.
	GrammarPointID string
}

// RevisionRef names one revision of a lesson.
type RevisionRef struct {
	ID       string
	Revision int
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
	// Every method that writes content (ReplaceContent, SaveAnnotations, SavePractice, ReplaceExtras,
	// ReplaceAnnotations) also drops the review, whose flags point into the old arrays (F22).
	// ReplaceContent overwrites content, revision, sentences, annotations and statuses.
	ReplaceContent(ctx context.Context, l Lesson) error
	// SetStatus changes one work status if the lesson is still at revision; ok reports that.
	SetStatus(ctx context.Context, id string, revision int, t job.Type, st Status, errMsg string) (bool, error)
	// SaveAnnotations stores AI annotations and extras, marks them done, clears the extras
	// edited flag and bumps QuizVersion, if still at revision. It also drops the practice and
	// marks it running, since a new practice is queued for the new annotations (F17).
	SaveAnnotations(ctx context.Context, id string, revision int, anns []Annotation, extras Extras) (bool, error)
	// SavePractice stores a practice, marks it done and bumps PracticeVersion, if the lesson is
	// still at revision and PracticeVersion still equals prevVersion.
	SavePractice(ctx context.Context, id string, revision, prevVersion int, p Practice) (bool, error)
	// WithoutPractice lists the lessons whose annotations are done but whose practice was never
	// queued (lessons annotated before F17).
	WithoutPractice(ctx context.Context) ([]RevisionRef, error)
	// ReplaceExtras stores admin-edited extras and sets the edited flag; bumpQuiz bumps
	// QuizVersion (the questions changed).
	ReplaceExtras(ctx context.Context, id string, extras Extras, bumpQuiz bool) error
	// ReplaceAnnotations stores admin-edited annotations and marks them done.
	ReplaceAnnotations(ctx context.Context, id string, anns []Annotation) error
	// SaveReview stores the AI check of a lesson (F22) if it is still at revision; ok is false when the
	// lesson was rewritten meanwhile. r is nil to clear the review. It leaves updated_at alone.
	SaveReview(ctx context.Context, id string, revision int, r *Review) (bool, error)
	Delete(ctx context.Context, id string) error
	// CountByGrammarPoint returns how many lessons use each grammar point; topicID "" counts
	// every topic. Lessons without a point are not counted.
	CountByGrammarPoint(ctx context.Context, topicID string) (map[string]int, error)
}

// Topics is what lessons need from topics (implemented over topic.Service in main).
type Topics interface {
	// Get returns ErrTopicNotFound when the topic does not exist.
	Get(ctx context.Context, id string) (TopicRef, error)
	// Names returns every topic by id.
	Names(ctx context.Context) (map[string]TopicRef, error)
	// RoadmapLessonIDs is the set of lessons in any topic roadmap.
	RoadmapLessonIDs(ctx context.Context) (map[string]bool, error)
	// AppendLesson adds a lesson at the end of the roadmap p unless it is already there.
	AppendLesson(ctx context.Context, p Place, lessonID string) error
	// MoveLesson takes a lesson out of roadmap from and, if it was there, appends it to roadmap to.
	MoveLesson(ctx context.Context, lessonID string, from, to Place) error
	// Position is the 1-based place of a lesson in the roadmap p, 0 when it is not there or the
	// topic does not exist.
	Position(ctx context.Context, p Place, lessonID string) (int, error)
}
