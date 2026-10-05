// Package grammar is the grammar syllabus: the points a learner meets at each CEFR level, in the
// order they are best taught. Lessons are assigned one point, and the AI that writes or annotates
// a lesson is told to teach exactly that point, so the grammar notes follow a plan instead of the
// model's choice of the moment.
//
// The syllabus is data (syllabus.json, embedded): edit it to add or reorder points. A point's id
// is kept by the lessons that use it, so never reuse or rename an id.
package grammar

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"slices"
	"sync"
)

// Levels are the CEFR levels in teaching order.
var Levels = []string{"A1", "A2", "B1", "B2", "C1", "C2"}

// Point is one grammar point of the syllabus.
type Point struct {
	ID      string `json:"id"`
	Level   string `json:"level"`
	TitleVi string `json:"titleVi"`
	// TitleEn and Pattern are what the AI is told to teach.
	TitleEn string `json:"titleEn"`
	Pattern string `json:"pattern"`
	// HintVi says in one line when the point is used.
	HintVi   string   `json:"hintVi"`
	Examples []string `json:"examples"`
}

// Syllabus is the full list of points, in teaching order within each level.
type Syllabus struct {
	points []Point
	byID   map[string]int
}

//go:embed syllabus.json
var embedded []byte

// Parse reads and checks a syllabus: every point needs an id, a known level, titles, a pattern and
// an example, and ids are unique.
func Parse(data []byte) (*Syllabus, error) {
	var file struct {
		Points []Point `json:"points"`
	}
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("grammar: parse syllabus: %w", err)
	}
	s := &Syllabus{points: file.Points, byID: make(map[string]int, len(file.Points))}
	for i, p := range file.Points {
		switch {
		case p.ID == "":
			return nil, fmt.Errorf("grammar: point %d has no id", i)
		case !slices.Contains(Levels, p.Level):
			return nil, fmt.Errorf("grammar: point %q has unknown level %q", p.ID, p.Level)
		case p.TitleVi == "" || p.TitleEn == "" || p.Pattern == "" || p.HintVi == "":
			return nil, fmt.Errorf("grammar: point %q is missing a title, pattern or hint", p.ID)
		case len(p.Examples) == 0:
			return nil, fmt.Errorf("grammar: point %q has no example", p.ID)
		}
		if _, dup := s.byID[p.ID]; dup {
			return nil, fmt.Errorf("grammar: duplicate id %q", p.ID)
		}
		s.byID[p.ID] = i
	}
	return s, nil
}

var (
	defaultOnce     sync.Once
	defaultSyllabus *Syllabus
	defaultErr      error
)

// Default returns the embedded syllabus. A broken embedded file is a build mistake that the tests
// catch, so it panics rather than letting the app run without a syllabus.
func Default() *Syllabus {
	defaultOnce.Do(func() { defaultSyllabus, defaultErr = Parse(embedded) })
	if defaultErr != nil {
		panic(defaultErr)
	}
	return defaultSyllabus
}

// All returns every point, in teaching order. The slice is a copy.
func (s *Syllabus) All() []Point { return slices.Clone(s.points) }

// ByLevel returns the points of one level, in teaching order; empty for an unknown level.
func (s *Syllabus) ByLevel(level string) []Point {
	var out []Point
	for _, p := range s.points {
		if p.Level == level {
			out = append(out, p)
		}
	}
	return out
}

// Get returns the point with the given id.
func (s *Syllabus) Get(id string) (Point, bool) {
	i, ok := s.byID[id]
	if !ok {
		return Point{}, false
	}
	return s.points[i], true
}
