package grammar

import "slices"

// CourseTopic is the first topic of a level's fixed course that teaches a point (added 2026-10-22).
type CourseTopic struct {
	Number int
	Name   string
}

// CourseOrder returns, for a level, the course topic of each point it teaches and the points in
// course order; ok is false when the level has no course. It is the curriculum package's, given
// by the caller so that grammar does not import it.
type CourseOrder func(level string) (topics map[string]CourseTopic, order []string, ok bool)

// SetCourse makes List follow a level's course.
func (s *Service) SetCourse(c CourseOrder) { s.course = c }

// inCourseOrder puts the points the course teaches first, in course order, each with its topic;
// the others follow in syllabus order.
func inCourseOrder(points []PointStatus, topics map[string]CourseTopic, order []string) []PointStatus {
	rank := func(id string) int {
		if i := slices.Index(order, id); i >= 0 {
			return i
		}
		return len(order)
	}
	out := slices.Clone(points)
	for i := range out {
		if t, ok := topics[out[i].ID]; ok {
			out[i].Course = &t
		}
	}
	slices.SortStableFunc(out, func(a, b PointStatus) int { return rank(a.ID) - rank(b.ID) })
	return out
}
