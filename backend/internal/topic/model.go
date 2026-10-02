// Package topic manages the topic catalogue: each topic belongs to a CEFR level and has its own
// ordered roadmap of lessons (F14). It does not import package lesson; main wires the two.
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

// Topic is a named group of lessons of one level with its ordered roadmap.
type Topic struct {
	ID          string
	Name        string
	Level       string
	Description string
	// LessonIDs is the roadmap: lessons of this topic in study order.
	LessonIDs []string
	// Words is the core vocabulary of the topic (F18), in the admin's order and spelling.
	Words []string
	// WordsSeeded is true once the startup seed considered the topic or an admin saved its words;
	// the seed never touches it again.
	WordsSeeded bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Input is what an admin submits to create or edit a topic.
type Input struct {
	Name        string
	Level       string
	Description string
}

// Summary is a topic in the admin list.
type Summary struct {
	Topic
	LessonCount int
	// Remaining counts roadmap lessons not yet studied; until learner progress exists (L)
	// every roadmap lesson counts.
	Remaining int
	Warning   bool
	// WordCount and UsedWordCount give the vocabulary coverage (F18).
	WordCount     int
	UsedWordCount int
}

// LessonRef is what the topic pages show of a lesson.
type LessonRef struct {
	ID               string
	Title            string
	Level            string
	TopicID          string
	AnnotationStatus string
	CreatedAt        time.Time
}

// Roadmap is a topic's roadmap with its lessons (deleted lessons skipped).
type Roadmap struct {
	Topic   Summary
	Lessons []LessonRef
}

// Public is a topic as learners see it.
type Public struct {
	ID          string
	Name        string
	Level       string
	Description string
	LessonCount int
}

var (
	ErrNotFound = errors.New("topic: not found")
	// ErrNameTaken is returned by repositories when the level already has a topic with that name.
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

func summarize(t Topic, lessonCount int) Summary {
	remaining := len(t.LessonIDs)
	return Summary{Topic: t, LessonCount: lessonCount, Remaining: remaining, Warning: remaining < MinRemaining}
}

// WordUse is one topic word with its coverage in the topic's lessons (F18).
type WordUse struct {
	Text string
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
