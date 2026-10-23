package ai

import "context"

// CurriculumPoint is a grammar point of the level, as the AI sees it when it drafts a course.
type CurriculumPoint struct {
	ID      string
	Title   string
	Pattern string
}

// CurriculumIdea is a topic suggested for the level, with why it comes at that place.
type CurriculumIdea struct {
	Name string
	Why  string
}

// OutlineRequest asks for the topics of a level's course (added 2026-10-23): their order, what the
// learner can do after each and the grammar points each teaches.
type OutlineRequest struct {
	Level  string
	Points []CurriculumPoint
	// Topics are the names of the catalogue's topics, to be preferred.
	Topics      []string
	Suggestions []CurriculumIdea
	// Earlier lists the topics of the approved courses below the level, "A1: Gia đình".
	Earlier        []string
	MinTopics      int
	MaxTopics      int
	PreviousErrors []string
}

// OutlineTopic is one topic of a drafted course.
type OutlineTopic struct {
	Name    string   `json:"name"`
	CanDoVi string   `json:"canDoVi"`
	Grammar []string `json:"grammar"`
}

// SlotsRequest asks for the lessons of one topic of a course: Main main lessons then Extra extra
// ones, each with a title, a story idea and its grammar points.
type SlotsRequest struct {
	Level  string
	Topic  OutlineTopic
	Points []CurriculumPoint
	Main   int
	Extra  int
	// Avoid are lesson titles used elsewhere in the courses.
	Avoid []string
}

// SlotDraft is one lesson of a drafted topic.
type SlotDraft struct {
	Title   string   `json:"title"`
	Idea    string   `json:"idea"`
	Grammar []string `json:"grammar"`
}

// CurriculumWriter drafts courses. The Gemini client implements it; without it the admin cannot
// draft a course with the AI.
type CurriculumWriter interface {
	OutlineCurriculum(ctx context.Context, req OutlineRequest) ([]OutlineTopic, error)
	WriteTopicSlots(ctx context.Context, req SlotsRequest) ([]SlotDraft, error)
}
