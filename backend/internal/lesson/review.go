package lesson

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/luongtran/luna/backend/internal/ai"
)

// FlagArea is the part of a lesson a Flag points into.
type FlagArea string

const (
	// AreaSentence points into Lesson.Sentences.
	AreaSentence FlagArea = "sentence"
	// AreaAnnotation points into Lesson.Annotations.
	AreaAnnotation FlagArea = "annotation"
	// AreaQuestion points into Lesson.Extras.Questions.
	AreaQuestion FlagArea = "question"
	// AreaTranslation points into Lesson.Practice.Translations.
	AreaTranslation FlagArea = "translation"
)

// areaOrder sorts flags by area.
var areaOrder = []FlagArea{AreaSentence, AreaAnnotation, AreaQuestion, AreaTranslation}

// ValidFlagArea reports whether a is one of the four areas.
func ValidFlagArea(a FlagArea) bool { return slices.Contains(areaOrder, a) }

// FlagKind says why the AI check flagged an item.
type FlagKind string

const (
	// FlagMismatch means the second AI chose another option than the stored answer of a question.
	FlagMismatch FlagKind = "mismatch"
	// FlagAmbiguous means the second AI finds a question unclear, with several right answers or none.
	FlagAmbiguous FlagKind = "ambiguous"
	// FlagWrong means the second AI finds a sentence, annotation or translation wrong.
	FlagWrong FlagKind = "wrong"
	// FlagUnchecked means the second AI did not answer a question, so the admin has to check it.
	FlagUnchecked FlagKind = "unchecked"
)

// Flag marks one item of a lesson that the AI check doubts. Index is the item's place in the array
// of its area. Confirmed is set once the admin has looked at it.
type Flag struct {
	Area      FlagArea
	Index     int
	Kind      FlagKind
	NoteVi    string
	Confirmed bool
}

// Review is the result of the AI check of a lesson (F22). Every write to the lesson's content drops
// it, because the flags point into arrays that have changed.
type Review struct {
	CheckedAt time.Time
	// VerifiedAt is the zero time until the admin marks the check as done.
	VerifiedAt time.Time
	Flags      []Flag
}

// Open counts the flags the admin has not confirmed yet.
func (r *Review) Open() int {
	if r == nil {
		return 0
	}
	n := 0
	for _, f := range r.Flags {
		if !f.Confirmed {
			n++
		}
	}
	return n
}

// SummaryOf returns the Flags, Checked and Verified fields of a Summary for review r (nil when
// the lesson was never checked).
func SummaryOf(r *Review) (flags int, checked, verified bool) {
	if r == nil {
		return 0, false, false
	}
	return r.Open(), true, !r.VerifiedAt.IsZero()
}

var (
	// ErrNotChecked means the lesson has no AI check to verify.
	ErrNotChecked = errors.New("lesson: lesson was not checked")
	// ErrFlagNotFound means the lesson has no flag at that area and index.
	ErrFlagNotFound = errors.New("lesson: flag not found")
	// errReviewAI wraps every error of the AI call so handlers can tell it from storage errors.
	errReviewAI = errors.New("lesson: review")
)

// FlagsError means flags are still open when the admin marks the check as done.
type FlagsError struct{ Count int }

func (e *FlagsError) Error() string { return fmt.Sprintf("lesson: %d unconfirmed flags", e.Count) }

// reviewTimeout bounds the AI call that reads over one lesson.
const reviewTimeout = 120 * time.Second

// Check has a second AI read over the AI-written parts of a lesson: its sentences, annotations,
// comprehension questions (answered without their keys) and the translations of the practice. It
// flags what the AI doubts and replaces the lesson's review. An AI failure returns an error and
// leaves the lesson unchanged.
func (s *Service) Check(ctx context.Context, id string) (Lesson, error) {
	l, err := s.Lessons.Get(ctx, id)
	if err != nil {
		return Lesson{}, err
	}
	if l.AnnotationStatus != StatusDone {
		return Lesson{}, ErrAnnotationNotDone
	}
	aiCtx, cancel := context.WithTimeout(ctx, reviewTimeout)
	defer cancel()
	res, err := s.AI.ReviewLesson(aiCtx, reviewRequest(l))
	if err != nil {
		if errors.Is(err, ai.ErrNotConfigured) || errors.Is(err, ai.ErrInvalidKey) || errors.Is(err, ai.ErrQuota) {
			return Lesson{}, err
		}
		return Lesson{}, fmt.Errorf("%w: %w", errReviewAI, err)
	}
	// A false result means the lesson changed during the AI call: the reply is stale, so the
	// lesson is returned as it is now, without a review.
	if _, err := s.Lessons.SaveReview(ctx, id, l.Revision, &Review{CheckedAt: s.Now(), Flags: flagsOf(l, res)}); err != nil {
		return Lesson{}, fmt.Errorf("lesson: save review: %w", err)
	}
	return s.Lessons.Get(ctx, id)
}

func reviewRequest(l Lesson) ai.ReviewRequest {
	req := ai.ReviewRequest{Level: string(l.Level), Title: l.Title, Sentences: sentenceTexts(l.Sentences)}
	for i, a := range l.Annotations {
		req.Annotations = append(req.Annotations, ai.ReviewAnnotation{
			Index: i, Text: a.Text, Lemma: a.Lemma, MeaningVi: a.MeaningVi, SentenceIndex: a.SentenceIndex,
		})
	}
	for i, q := range l.Extras.Questions {
		req.Questions = append(req.Questions, ai.ReviewQuestion{Index: i, Prompt: q.Prompt, Options: q.Options})
	}
	if l.Practice != nil {
		for i, t := range l.Practice.Translations {
			req.Translations = append(req.Translations, ai.ReviewTranslation{Index: i, Vi: t.Vi, En: t.En})
		}
	}
	return req
}

// flagsOf compares what the reviewer said with the lesson. Items the reviewer names outside the
// lesson are dropped; the flags come sorted by area, then index.
func flagsOf(l Lesson, res ai.ReviewResult) []Flag {
	flags := []Flag{}
	seen := map[Flag]bool{}
	add := func(f Flag) {
		key := Flag{Area: f.Area, Index: f.Index}
		if !seen[key] {
			seen[key] = true
			flags = append(flags, f)
		}
	}
	issues := func(area FlagArea, items []ai.ReviewIssue, size int, fallback string) {
		for _, it := range items {
			if it.Index < 0 || it.Index >= size {
				continue
			}
			add(Flag{Area: area, Index: it.Index, Kind: FlagWrong, NoteVi: noteOr(it.NoteVi, fallback)})
		}
	}
	issues(AreaSentence, res.Sentences, len(l.Sentences), "AI thấy câu này sai hoặc không tự nhiên.")
	issues(AreaAnnotation, res.Annotations, len(l.Annotations), "AI thấy nghĩa của chú thích này không đúng với câu.")
	translations := 0
	if l.Practice != nil {
		translations = len(l.Practice.Translations)
	}
	issues(AreaTranslation, res.Translations, translations,
		"AI thấy câu tiếng Việt và câu tiếng Anh không cùng nghĩa, hoặc câu tiếng Anh sai.")

	answers := map[int]ai.ReviewAnswer{}
	for _, a := range res.Answers {
		if _, dup := answers[a.Index]; !dup {
			answers[a.Index] = a
		}
	}
	for i, q := range l.Extras.Questions {
		a, answered := answers[i]
		note := strings.TrimSpace(a.NoteVi)
		switch {
		case a.Ambiguous && answered:
			add(Flag{Area: AreaQuestion, Index: i, Kind: FlagAmbiguous, NoteVi: noteOr(note,
				"AI thấy câu này mơ hồ: có thể có nhiều hơn một đáp án đúng, hoặc không có đáp án nào.")})
		case !answered || a.ChoiceIndex == nil:
			add(Flag{Area: AreaQuestion, Index: i, Kind: FlagUnchecked, NoteVi: "AI không trả lời câu này, hãy tự kiểm tra."})
		case *a.ChoiceIndex != q.AnswerIndex:
			got := "chọn một đáp án không có trong câu"
			if c := *a.ChoiceIndex; c >= 0 && c < len(q.Options) {
				got = fmt.Sprintf("chọn \"%s\"", q.Options[c])
			}
			msg := fmt.Sprintf("AI %s, khác đáp án đã lưu.", got)
			if note != "" {
				msg += " " + note
			}
			add(Flag{Area: AreaQuestion, Index: i, Kind: FlagMismatch, NoteVi: msg})
		}
	}
	slices.SortStableFunc(flags, func(a, b Flag) int {
		if c := slices.Index(areaOrder, a.Area) - slices.Index(areaOrder, b.Area); c != 0 {
			return c
		}
		return a.Index - b.Index
	})
	return flags
}

func noteOr(note, fallback string) string {
	if note = strings.TrimSpace(note); note != "" {
		return note
	}
	return fallback
}

// ConfirmFlag marks the flag at area and index as looked at by the admin. A lesson without that flag
// (or without a check) returns ErrFlagNotFound.
func (s *Service) ConfirmFlag(ctx context.Context, id string, area FlagArea, index int) (Lesson, error) {
	l, err := s.Lessons.Get(ctx, id)
	if err != nil {
		return Lesson{}, err
	}
	if l.Review == nil {
		return Lesson{}, ErrFlagNotFound
	}
	r := *l.Review
	r.Flags = slices.Clone(r.Flags)
	i := slices.IndexFunc(r.Flags, func(f Flag) bool { return f.Area == area && f.Index == index })
	if i < 0 {
		return Lesson{}, ErrFlagNotFound
	}
	r.Flags[i].Confirmed = true
	ok, err := s.Lessons.SaveReview(ctx, id, l.Revision, &r)
	if err != nil {
		return Lesson{}, fmt.Errorf("lesson: save review: %w", err)
	}
	if !ok { // the lesson was rewritten meanwhile, so the flag no longer exists
		return Lesson{}, ErrFlagNotFound
	}
	return s.Lessons.Get(ctx, id)
}

// Verify marks the AI check as done. It returns ErrNotChecked before the first check and a
// *FlagsError while flags are still unconfirmed.
func (s *Service) Verify(ctx context.Context, id string) (Lesson, error) {
	l, err := s.Lessons.Get(ctx, id)
	if err != nil {
		return Lesson{}, err
	}
	if l.Review == nil {
		return Lesson{}, ErrNotChecked
	}
	if n := l.Review.Open(); n > 0 {
		return Lesson{}, &FlagsError{Count: n}
	}
	r := *l.Review
	r.Flags = slices.Clone(r.Flags)
	r.VerifiedAt = s.Now()
	ok, err := s.Lessons.SaveReview(ctx, id, l.Revision, &r)
	if err != nil {
		return Lesson{}, fmt.Errorf("lesson: save review: %w", err)
	}
	if !ok {
		return Lesson{}, ErrNotChecked
	}
	return s.Lessons.Get(ctx, id)
}
