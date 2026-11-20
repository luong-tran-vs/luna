// Package toeic is the TOEIC Listening & Reading practice (F27): the bank of question groups imported
// from JSON, the tests composed from them, the learners' attempts (saved on the server, graded on
// submit, never showing an answer before), the mistakes notebook and the goal.
package toeic

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Part is a part of the test, 1..7.
type Part int

// PartQuestions is how many questions each part has in a full test (index 0 is unused).
var PartQuestions = [8]int{0, 6, 25, 39, 30, 30, 16, 54}

// Valid reports whether p is a part of the test.
func (p Part) Valid() bool { return p >= 1 && p <= 7 }

// Section is a half of the test.
type Section string

const (
	// Listening is Part 1–4.
	Listening Section = "listening"
	// Reading is Part 5–7.
	Reading Section = "reading"
)

// SectionOf returns the section of a part.
func SectionOf(p Part) Section {
	if p <= 4 {
		return Listening
	}
	return Reading
}

// Status is the publication state of a group or a test.
type Status string

const (
	StatusDraft     Status = "draft"
	StatusPublished Status = "published"
	StatusHidden    Status = "hidden"
)

// Valid reports whether s is a known status.
func (s Status) Valid() bool {
	return s == StatusDraft || s == StatusPublished || s == StatusHidden
}

// TestKind is the size of a test.
type TestKind string

const (
	KindFull    TestKind = "full"
	KindMini50  TestKind = "mini50"
	KindMini100 TestKind = "mini100"
)

// Valid reports whether k is a known kind.
func (k TestKind) Valid() bool { return k == KindFull || k == KindMini50 || k == KindMini100 }

// Mini reports whether k is a Mini Test.
func (k TestKind) Mini() bool { return k == KindMini50 || k == KindMini100 }

// Minutes is the time of the whole test of that kind.
func (k TestKind) Minutes() int {
	switch k {
	case KindMini50:
		return 38
	case KindMini100:
		return 60
	}
	return 120
}

// Mode is how an attempt is taken.
type Mode string

const (
	// ModePractice lets the learner pick the parts and the time.
	ModePractice Mode = "practice"
	// ModeTest is the whole test as on exam day.
	ModeTest Mode = "test"
)

// SourceKind says where the questions of an attempt come from.
type SourceKind string

const (
	SourceTest     SourceKind = "test"
	SourceRandom   SourceKind = "random"
	SourcePart     SourceKind = "part"
	SourceTag      SourceKind = "tag"
	SourceMistakes SourceKind = "mistakes"
	SourceRetry    SourceKind = "retry"
)

// Accent is the voice of a speaker.
type Accent string

const (
	AccentUS Accent = "us"
	AccentGB Accent = "gb"
	AccentAU Accent = "au"
	AccentCA Accent = "ca"
)

// Valid reports whether a is a known accent.
func (a Accent) Valid() bool { return a == AccentUS || a == AccentGB || a == AccentAU || a == AccentCA }

// --- content ---

// Group is a question group: one question (Part 1, 2, 5) or a text or talk with its questions.
// The json names are the import format (contracts/content-format.md).
type Group struct {
	ID        string     `json:"-"`
	Key       string     `json:"key"`
	Part      Part       `json:"part"`
	Level     int        `json:"level"`
	Status    Status     `json:"status,omitempty"`
	Source    string     `json:"source,omitempty"`
	Version   int        `json:"version,omitempty"`
	Scene     string     `json:"scene,omitempty"`
	Script    []Turn     `json:"script,omitempty"`
	Graphic   *Graphic   `json:"graphic,omitempty"`
	Docs      []Doc      `json:"docs,omitempty"`
	Questions []Question `json:"questions"`
	CreatedAt time.Time  `json:"-"`
	UpdatedAt time.Time  `json:"-"`
}

// Content is the part of a group the import writes (what is stored as its body).
type Content struct {
	Scene     string     `json:"scene,omitempty"`
	Script    []Turn     `json:"script,omitempty"`
	Graphic   *Graphic   `json:"graphic,omitempty"`
	Docs      []Doc      `json:"docs,omitempty"`
	Questions []Question `json:"questions"`
}

// Content returns the body of g.
func (g Group) Content() Content {
	return Content{Scene: g.Scene, Script: g.Script, Graphic: g.Graphic, Docs: g.Docs, Questions: g.Questions}
}

// SetContent replaces the body of g.
func (g *Group) SetContent(c Content) {
	g.Scene, g.Script, g.Graphic, g.Docs, g.Questions = c.Scene, c.Script, c.Graphic, c.Docs, c.Questions
}

// Turn is a line of a talk or conversation.
type Turn struct {
	Speaker     string `json:"speaker"`
	Accent      Accent `json:"accent"`
	Text        string `json:"text"`
	TranslateVi string `json:"translateVi,omitempty"`
}

// Graphic is the table, schedule, price list, floor plan or bar chart of a "Look at the graphic" question.
type Graphic struct {
	Kind    string        `json:"kind"`
	Title   string        `json:"title,omitempty"`
	Columns []string      `json:"columns,omitempty"`
	Rows    [][]string    `json:"rows,omitempty"`
	Cells   []GraphicCell `json:"cells,omitempty"`
	Bars    []Bar         `json:"bars,omitempty"`
}

// GraphicCell is a room of a floor plan.
type GraphicCell struct {
	Label string `json:"label"`
	Row   int    `json:"row"`
	Col   int    `json:"col"`
	Span  int    `json:"span,omitempty"`
}

// Bar is a bar of a bar chart.
type Bar struct {
	Label string  `json:"label"`
	Value float64 `json:"value"`
}

// Doc is a text of Part 6/7: an email, a notice, a web page...
type Doc struct {
	Kind        string  `json:"kind"`
	Title       string  `json:"title,omitempty"`
	Meta        []KV    `json:"meta,omitempty"`
	Body        []Block `json:"body"`
	ImageSlot   string  `json:"imageSlot,omitempty"`
	TranslateVi string  `json:"translateVi,omitempty"`
}

// KV is a header line of a text (From, To, Subject, Date, URL...).
type KV struct {
	K string `json:"k"`
	V string `json:"v"`
}

// Block is a paragraph, heading, list, table or chat message of a text. Text may hold **bold** and
// the blanks [[n]] of Part 6 (n is the question's number in the group).
type Block struct {
	Kind  string   `json:"kind"`
	Text  string   `json:"text,omitempty"`
	Items []string `json:"items,omitempty"`
	Table *Table   `json:"table,omitempty"`
	Who   string   `json:"who,omitempty"`
	Time  string   `json:"time,omitempty"`
}

// Table is a table inside a text.
type Table struct {
	Columns []string   `json:"columns"`
	Rows    [][]string `json:"rows"`
}

// Question is a question of a group.
type Question struct {
	No          int      `json:"no"`
	Prompt      string   `json:"prompt,omitempty"`
	Options     []string `json:"options"`
	Answer      int      `json:"answer"`
	Evidence    []Ref    `json:"evidence,omitempty"`
	ExplainVi   string   `json:"explainVi"`
	TranslateVi string   `json:"translateVi,omitempty"`
	Tags        []string `json:"tags"`
	Vocab       []Vocab  `json:"vocab,omitempty"`
}

// Ref points at the words that give the answer away: a block of a text (Doc ≥ 0) or a line of the
// script (Doc = -1).
type Ref struct {
	Doc   int    `json:"doc"`
	Block int    `json:"block"`
	Turn  int    `json:"turn"`
	Text  string `json:"text"`
}

// Vocab is a word worth noting in a question.
type Vocab struct {
	Word      string `json:"word"`
	IPA       string `json:"ipa,omitempty"`
	MeaningVi string `json:"meaningVi"`
}

// QuestionID is the stable id of a question: the group key and its number in the group.
func QuestionID(key string, no int) string { return key + "#" + strconv.Itoa(no) }

// SplitQuestionID returns the group key and the number of a question id.
func SplitQuestionID(id string) (key string, no int, ok bool) {
	i := strings.LastIndexByte(id, '#')
	if i <= 0 {
		return "", 0, false
	}
	n, err := strconv.Atoi(id[i+1:])
	if err != nil || n < 1 {
		return "", 0, false
	}
	return id[:i], n, true
}

// Question returns the question numbered no of g.
func (g Group) Question(no int) (Question, bool) {
	for _, q := range g.Questions {
		if q.No == no {
			return q, true
		}
	}
	return Question{}, false
}

// Image is a picture of a group: the Part 1 photo ("photo") or the picture of a text ("doc-<n>").
type Image struct {
	GroupKey  string
	Slot      string
	MIME      string
	Data      []byte
	Origin    string
	CreatedAt time.Time
}

// SlotPhoto is the slot of the Part 1 photo.
const SlotPhoto = "photo"

// Test is a fixed test (full or Mini) composed of groups.
type Test struct {
	ID        string
	Key       string
	Title     string
	Kind      TestKind
	Order     int
	Groups    []string
	Status    Status
	CreatedAt time.Time
	UpdatedAt time.Time
}

// PartCheck is how many questions a test has and needs in one part.
type PartCheck struct {
	Part Part `json:"part"`
	Have int  `json:"have"`
	Need int  `json:"need"`
}

// PhotoGap is a Part 1 question still without a photo.
type PhotoGap struct {
	GroupKey string `json:"groupKey"`
	Number   int    `json:"number"`
}

// TestCheck is what a test still misses before it can be published.
type TestCheck struct {
	Parts            []PartCheck `json:"parts"`
	MissingGroups    []string    `json:"missingGroups"`
	DraftGroups      []string    `json:"draftGroups"`
	MissingPhotos    []PhotoGap  `json:"missingPhotos"`
	Graphics         int         `json:"graphics"`
	Docs             int         `json:"docs"`
	ListeningMinutes float64     `json:"listeningMinutes"`
	// Complete is true when all seven parts are there (a full test can be taken as on exam day).
	Complete bool `json:"complete"`
	// Ready is true when the test can be published: every part it has is whole, Part 5–7 are there,
	// every group is published and every photo drawn.
	Ready bool `json:"ready"`
}

// --- learners ---

// Source is where the questions of an attempt come from.
type Source struct {
	Kind        SourceKind `json:"kind"`
	TestKey     string     `json:"testKey,omitempty"`
	Part        Part       `json:"part,omitempty"`
	Tag         string     `json:"tag,omitempty"`
	FromAttempt string     `json:"fromAttempt,omitempty"`
	Size        int        `json:"size,omitempty"`
	Count       int        `json:"count,omitempty"`
}

// AttemptStatus is the state of an attempt.
type AttemptStatus string

const (
	AttemptInProgress AttemptStatus = "in_progress"
	AttemptSubmitted  AttemptStatus = "submitted"
)

// Attempt is one sitting of a learner: the questions it was built with, the answers so far and, once
// submitted, the result.
type Attempt struct {
	ID     string
	UserID string
	Mode   Mode
	Source Source
	Title  string
	Parts  []Part
	// GroupKeys are the groups of the attempt, fixed when it starts.
	GroupKeys []string
	// QuestionIDs are the questions that count, in order.
	QuestionIDs []string
	// Numbers is the number shown for each question.
	Numbers          map[string]int
	TimeLimitMs      int64
	RemainingMs      int64
	ElapsedMs        int64
	Phase            Section
	ListeningSegment int
	Position         string
	Answers          map[string]int
	Flags            []string
	Status           AttemptStatus
	Counted          bool
	Result           *Result
	StartedAt        time.Time
	SavedAt          time.Time
	SubmittedAt      time.Time
}

// State is the part of an attempt that changes while it is taken (stored as one body).
type State struct {
	Title            string         `json:"title"`
	Parts            []Part         `json:"parts"`
	Numbers          map[string]int `json:"numbers"`
	TimeLimitMs      int64          `json:"timeLimitMs"`
	RemainingMs      int64          `json:"remainingMs"`
	ElapsedMs        int64          `json:"elapsedMs"`
	Phase            Section        `json:"phase,omitempty"`
	ListeningSegment int            `json:"listeningSegment"`
	Position         string         `json:"position,omitempty"`
	Answers          map[string]int `json:"answers"`
	Flags            []string       `json:"flags"`
}

// State returns the changing part of a.
func (a Attempt) State() State {
	return State{
		Title: a.Title, Parts: a.Parts, Numbers: a.Numbers, TimeLimitMs: a.TimeLimitMs, RemainingMs: a.RemainingMs,
		ElapsedMs: a.ElapsedMs, Phase: a.Phase, ListeningSegment: a.ListeningSegment, Position: a.Position,
		Answers: a.Answers, Flags: a.Flags,
	}
}

// SetState sets the changing part of a.
func (a *Attempt) SetState(s State) {
	a.Title, a.Parts, a.Numbers, a.TimeLimitMs, a.RemainingMs = s.Title, s.Parts, s.Numbers, s.TimeLimitMs, s.RemainingMs
	a.ElapsedMs, a.Phase, a.ListeningSegment, a.Position = s.ElapsedMs, s.Phase, s.ListeningSegment, s.Position
	a.Answers, a.Flags = s.Answers, s.Flags
}

// Outcome of a question in a result.
const (
	OutcomeRight   = "right"
	OutcomeWrong   = "wrong"
	OutcomeSkipped = "skipped"
)

// Result is the grade of a submitted attempt.
type Result struct {
	Correct    int               `json:"correct"`
	Wrong      int               `json:"wrong"`
	Skipped    int               `json:"skipped"`
	ByQuestion map[string]string `json:"byQuestion"`
	Listening  *SectionScore     `json:"listening,omitempty"`
	Reading    *SectionScore     `json:"reading,omitempty"`
	// Total is the estimated 990 score; 0 when the attempt does not have all seven parts.
	Total      int            `json:"total"`
	DurationMs int64          `json:"durationMs"`
	Version    map[string]int `json:"version"`
}

// SectionScore is the grade of one section.
type SectionScore struct {
	Correct int `json:"correct"`
	Total   int `json:"total"`
	Scaled  int `json:"scaled"`
}

// Mistake is a question of the learner's mistakes notebook.
type Mistake struct {
	UserID      string
	QuestionID  string
	GroupKey    string
	Part        Part
	Tags        []string
	Streak      int
	LastWrongAt time.Time
	AddedAt     time.Time
}

// Goal is the learner's target score and exam date.
type Goal struct {
	UserID    string
	Target    int
	ExamDate  string
	UpdatedAt time.Time
}

// GoalTargets are the targets a learner may choose.
var GoalTargets = []int{450, 600, 750, 900}

// --- errors ---

var (
	// ErrNotFound is returned for an unknown group, test, attempt, image or mistake.
	ErrNotFound = errors.New("toeic: not found")
	// ErrExists is returned when a test key is taken.
	ErrExists = errors.New("toeic: already exists")
	// ErrMembersOnly is returned when a guest asks for what only members get.
	ErrMembersOnly = errors.New("toeic: members only")
	// ErrAttemptSubmitted is returned when a submitted attempt is changed.
	ErrAttemptSubmitted = errors.New("toeic: attempt already submitted")
	// ErrNotEnough is returned when the published questions cannot fill a round.
	ErrNotEnough = errors.New("toeic: not enough questions")
)

// ErrAttemptInProgress is returned when the learner starts a test that already has an attempt in
// progress; ID is that attempt.
type ErrAttemptInProgress struct{ ID string }

func (e *ErrAttemptInProgress) Error() string { return "toeic: attempt in progress " + e.ID }

// ErrTestNotReady is returned when a test that misses questions or photos is published.
type ErrTestNotReady struct{ Check TestCheck }

func (e *ErrTestNotReady) Error() string { return "toeic: test not ready" }

// ValidationError lists the invalid fields with a message in Vietnamese.
type ValidationError struct{ Fields map[string]string }

func (e *ValidationError) Error() string { return fmt.Sprintf("toeic: invalid %v", e.Fields) }

// invalid returns a ValidationError of one field.
func invalid(field, msg string) error {
	return &ValidationError{Fields: map[string]string{field: msg}}
}
