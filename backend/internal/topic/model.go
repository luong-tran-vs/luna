// Package topic manages the topic catalogue (F14). A topic is shared by every CEFR level: lessons
// carry their own level, and each (topic, level) pair has its own ordered roadmap. It does not
// import package lesson; main wires the two.
package topic

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
)

// Levels lists the CEFR levels in order.
var Levels = []string{"A1", "A2", "B1", "B2", "C1", "C2"}

// ValidLevel reports whether s is one of Levels.
func ValidLevel(s string) bool { return slices.Contains(Levels, s) }

// levelRank orders levels A1 → C2.
func levelRank(s string) int { return slices.Index(Levels, s) }

// MinRemaining is the number of not-yet-studied roadmap lessons below which admins are warned.
const MinRemaining = 3

// DefaultName is the topic of lessons that had none before topics existed.
const DefaultName = "Chung"

// Word is one core word of a topic (F18). Level is the lowest level it is meant for; "" means
// every level.
type Word struct {
	Text  string
	Level string
}

// FitsLevel reports whether the word may be given to a lesson of level.
func (w Word) FitsLevel(level string) bool {
	return w.Level == "" || levelRank(w.Level) <= levelRank(level)
}

// Texts returns the words' texts in order.
func Texts(words []Word) []string {
	out := make([]string, len(words))
	for i, w := range words {
		out[i] = w.Text
	}
	return out
}

// Place is where a lesson belongs: its topic and its level, which together name a roadmap.
type Place struct {
	TopicID string
	Level   string
}

// Topic is a named group of lessons, shared by every level, with one roadmap per level.
type Topic struct {
	ID          string
	Name        string
	Description string
	// Roadmaps maps a level to the lessons of this topic and level in study order; a level
	// without a roadmap may be absent.
	Roadmaps map[string][]string
	// Words is the core vocabulary of the topic (F18), in the admin's order and spelling.
	Words []Word
	// WordsSeeded is true once the startup seed considered the topic or an admin saved its words;
	// the seed never touches it again.
	WordsSeeded bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Roadmap returns the roadmap of level (nil when there is none).
func (t Topic) Roadmap(level string) []string { return t.Roadmaps[level] }

// AllRoadmapLessons returns the lessons of every roadmap of the topic, A1 → C2.
func (t Topic) AllRoadmapLessons() []string {
	var out []string
	for _, l := range Levels {
		out = append(out, t.Roadmaps[l]...)
	}
	return out
}

// Input is what an admin submits to create or edit a topic.
type Input struct {
	Name        string
	Description string
}

// LevelSummary is one level of a topic in the admin list: its lessons and roadmap.
type LevelSummary struct {
	Level        string
	LessonCount  int
	RoadmapCount int
	// Remaining counts roadmap lessons not yet studied; until learner progress exists (L)
	// every roadmap lesson counts.
	Remaining int
	Warning   bool
}

// Summary is a topic in the admin list.
type Summary struct {
	Topic
	// LessonCount counts the lessons of every level.
	LessonCount int
	// Levels lists, A1 → C2, the levels that have lessons or a roadmap.
	Levels []LevelSummary
	// WordCount and UsedWordCount give the vocabulary coverage (F18).
	WordCount     int
	UsedWordCount int
}

// Level returns the summary of level (an empty roadmap when the topic has nothing there).
func (s Summary) Level(level string) LevelSummary {
	for _, l := range s.Levels {
		if l.Level == level {
			return l
		}
	}
	return LevelSummary{Level: level, Warning: true}
}

// LessonRef is what the topic pages show of a lesson.
type LessonRef struct {
	ID               string
	Title            string
	Level            string
	TopicID          string
	AnnotationStatus string
	// Draft lessons are hidden from learners until published.
	Draft     bool
	CreatedAt time.Time
}

// Roadmap is the roadmap of one (topic, level) with its lessons (deleted lessons skipped).
type Roadmap struct {
	Topic   Summary
	Level   string
	Lessons []LessonRef
}

// Public is a topic as learners see it at one level.
type Public struct {
	ID          string
	Name        string
	Level       string
	Description string
	LessonCount int
}

var (
	ErrNotFound = errors.New("topic: not found")
	// ErrNameTaken is returned by repositories when a topic already has that name.
	ErrNameTaken = errors.New("topic: name taken")
)

// InUseError is returned when deleting a topic that still has lessons.
type InUseError struct {
	Count int
}

func (e *InUseError) Error() string { return fmt.Sprintf("topic: still has %d lessons", e.Count) }

// ValidationError lists invalid input fields with Vietnamese messages for the admin.
type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string { return fmt.Sprintf("topic: invalid input %v", e.Fields) }

// NameKey is the form used to compare names: lowercase, trimmed, single-spaced.
func NameKey(name string) string {
	return strings.ToLower(strings.Join(strings.Fields(name), " "))
}

// summarize counts the lessons of t per level (counts: level → lessons of the topic).
func summarize(t Topic, counts map[string]int) Summary {
	s := Summary{Topic: t, Levels: []LevelSummary{}}
	for _, level := range Levels {
		n, roadmap := counts[level], len(t.Roadmaps[level])
		s.LessonCount += n
		if n == 0 && roadmap == 0 {
			continue
		}
		s.Levels = append(s.Levels, LevelSummary{
			Level: level, LessonCount: n, RoadmapCount: roadmap, Remaining: roadmap, Warning: roadmap < MinRemaining,
		})
	}
	return s
}

// WordUse is one topic word with its coverage in the topic's lessons (F18).
type WordUse struct {
	Text  string
	Level string
	// Used is true when at least one lesson of the topic contains the word.
	Used bool
	// LessonCount is the number of lessons of the topic containing it.
	LessonCount int
}

// LessonText is what coverage needs from a lesson: its content and the base forms of its
// single-word annotations (lowercase word → lemma).
type LessonText struct {
	Content string
	Lemmas  map[string]string
}
