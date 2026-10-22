package lesson

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/luongtran/luna/backend/internal/ai"
)

// The check of the Speaking step (added 2026-10-22): the learner answers up to three questions
// aloud, the browser recognises the speech, and one AI request corrects the answers.

// Limits of a speaking check.
const (
	maxSpeakAnswers = speakQuestionsCount
	maxSpeakText    = 300
	// SpeakChecksPerDay is how many checks one learner may ask for in one lesson a day; admins have
	// no limit.
	SpeakChecksPerDay = 5
)

// ErrSpeakLimit means the learner checked this lesson's speaking SpeakChecksPerDay times today.
var ErrSpeakLimit = errors.New("lesson: speaking check limit reached")

// SpeakResult is the check of one answer; Corrected equals Said when the answer is fine.
type SpeakResult struct {
	Said      string
	Corrected string
	NoteVi    string
}

// speakLimiter counts the checks of each learner and lesson per day, in memory: a restart only
// gives a few more.
type speakLimiter struct {
	mu   sync.Mutex
	day  string
	used map[string]int
}

func (l *speakLimiter) take(key string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if day := now.UTC().Format(time.DateOnly); day != l.day || l.used == nil {
		l.day, l.used = day, map[string]int{}
	}
	if l.used[key] >= SpeakChecksPerDay {
		return false
	}
	l.used[key]++
	return true
}

// giveBack returns a check that did not happen (the AI failed), so it is not counted.
func (l *speakLimiter) giveBack(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.used[key] > 0 {
		l.used[key]--
	}
}

// CheckSpeaking corrects the learner's spoken answers to the Speaking step's questions in one AI
// request. Answers left empty come back as they are, and no request is made when all are empty.
func (r *Reader) CheckSpeaking(ctx context.Context, userID, lessonID string, answers []ai.SpeakAnswer, unlimited bool) ([]SpeakResult, error) {
	answers = slices.Clone(answers)
	fields := map[string]string{}
	if len(answers) == 0 || len(answers) > maxSpeakAnswers {
		fields["answers"] = fmt.Sprintf("Cần từ 1 đến %d câu trả lời", maxSpeakAnswers)
	}
	for i := range answers {
		answers[i].Question, answers[i].Said = collapse(answers[i].Question), collapse(answers[i].Said)
		if utf8.RuneCountInString(answers[i].Question) > maxSpeakText || utf8.RuneCountInString(answers[i].Said) > maxSpeakText {
			fields[fmt.Sprintf("answers.%d", i)] = fmt.Sprintf("Tối đa %d ký tự", maxSpeakText)
		}
	}
	if len(fields) > 0 {
		return nil, &ValidationError{Fields: fields}
	}
	l, err := r.lessons.Get(ctx, lessonID)
	if err != nil {
		return nil, err
	}
	out := make([]SpeakResult, len(answers))
	var asked []ai.SpeakAnswer
	var index []int
	for i, a := range answers {
		out[i] = SpeakResult{Said: a.Said, Corrected: a.Said}
		if a.Said != "" {
			asked = append(asked, a)
			index = append(index, i)
		}
	}
	if len(asked) == 0 {
		return out, nil
	}
	checker, ok := r.ai.(ai.SpeakChecker)
	if !ok {
		return nil, ai.ErrNotConfigured
	}
	key := userID + "/" + lessonID
	if !unlimited && !r.speakLimit.take(key, time.Now()) {
		return nil, ErrSpeakLimit
	}
	res, err := checker.CheckSpeaking(ctx, ai.SpeakCheckRequest{Level: string(l.Level), Answers: asked})
	if err != nil {
		if !unlimited {
			r.speakLimit.giveBack(key)
		}
		return nil, err
	}
	for _, f := range res {
		if f.Index < 0 || f.Index >= len(index) {
			continue
		}
		i := index[f.Index]
		c := limited(f.Corrected, maxSpeakText)
		if c != "" && sentenceKey(c) != sentenceKey(out[i].Said) {
			out[i].Corrected, out[i].NoteVi = c, limited(f.NoteVi, maxExplanation)
		}
	}
	return out, nil
}

// sentenceKey compares sentences ignoring case, spaces and the final punctuation.
func sentenceKey(s string) string {
	return strings.TrimRight(textKey(s), ".!?")
}
