package progress

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"
)

type fakeRepo struct {
	mu   sync.Mutex
	rows map[string]StoredResult
	keys []string
}

func newFakeRepo() *fakeRepo { return &fakeRepo{rows: map[string]StoredResult{}} }

func (f *fakeRepo) Upsert(_ context.Context, userID, lessonID string, revision int, r Result) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := fmt.Sprintf("%s|%s|%d", userID, lessonID, r.SentenceIndex)
	if _, ok := f.rows[key]; !ok {
		f.keys = append(f.keys, key)
	}
	f.rows[key] = StoredResult{Result: r, Revision: revision}
	return nil
}

func (f *fakeRepo) List(_ context.Context, userID, lessonID string) ([]StoredResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	prefix := userID + "|" + lessonID + "|"
	var out []StoredResult
	// Newest first, so the service must sort by sentence index itself.
	for _, k := range slices.Backward(f.keys) {
		if strings.HasPrefix(k, prefix) {
			out = append(out, f.rows[k])
		}
	}
	return out, nil
}

func (f *fakeRepo) Delete(_ context.Context, userID, lessonID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	prefix := userID + "|" + lessonID + "|"
	f.keys = slices.DeleteFunc(f.keys, func(k string) bool { return strings.HasPrefix(k, prefix) })
	for k := range f.rows {
		if strings.HasPrefix(k, prefix) {
			delete(f.rows, k)
		}
	}
	return nil
}

type lessonInfo struct{ revision, sentences int }

type fakeLessons struct {
	mu      sync.Mutex
	lessons map[string]lessonInfo
}

func (f *fakeLessons) Info(_ context.Context, id string) (revision, sentenceCount int, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	l, ok := f.lessons[id]
	if !ok {
		return 0, 0, ErrLessonNotFound
	}
	return l.revision, l.sentences, nil
}

func (f *fakeLessons) set(id string, revision, sentences int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.lessons[id] = lessonInfo{revision, sentences}
}

func (f *fakeRepo) Totals(_ context.Context, userID string, since *time.Time) (DictationTotals, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var t DictationTotals
	lessons := map[string]bool{}
	for k, r := range f.rows {
		if strings.HasPrefix(k, userID+"|") && (since == nil || !r.CheckedAt.Before(*since)) {
			t.Sentences++
			t.CorrectWords += r.CorrectWords
			t.TotalWords += r.TotalWords
			lessons[strings.Split(k, "|")[1]] = true
		}
	}
	t.Lessons = len(lessons)
	return t, nil
}
