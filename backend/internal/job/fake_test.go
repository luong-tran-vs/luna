package job

import (
	"context"
	"slices"
	"strconv"
	"sync"
	"time"
)

// fakeRepo is an in-memory Repository.
type fakeRepo struct {
	mu     sync.Mutex
	jobs   map[string]*Job
	nextID int
	resets int
}

func newFakeRepo() *fakeRepo { return &fakeRepo{jobs: map[string]*Job{}} }

func (f *fakeRepo) Enqueue(_ context.Context, j Job) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.nextID++
	j.ID = strconv.Itoa(f.nextID)
	if j.Status == "" {
		j.Status = StatusPending
	}
	f.jobs[j.ID] = &j
	return nil
}

func (f *fakeRepo) ClaimNext(_ context.Context, now time.Time) (Job, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var due []*Job
	for _, j := range f.jobs {
		if j.Status == StatusPending && !j.RunAt.After(now) {
			due = append(due, j)
		}
	}
	if len(due) == 0 {
		return Job{}, false, nil
	}
	slices.SortFunc(due, func(a, b *Job) int { return a.RunAt.Compare(b.RunAt) })
	j := due[0]
	j.Status = StatusRunning
	j.Attempts++
	return *j, true, nil
}

func (f *fakeRepo) set(id string, fn func(*Job)) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if j, ok := f.jobs[id]; ok {
		fn(j)
	}
	return nil
}

func (f *fakeRepo) Complete(_ context.Context, id string) error {
	return f.set(id, func(j *Job) { j.Status = StatusDone })
}

func (f *fakeRepo) Retry(_ context.Context, id string, runAt time.Time, msg string) error {
	return f.set(id, func(j *Job) { j.Status, j.RunAt, j.Error = StatusPending, runAt, msg })
}

func (f *fakeRepo) Fail(_ context.Context, id, msg string) error {
	return f.set(id, func(j *Job) { j.Status, j.Error = StatusFailed, msg })
}

func (f *fakeRepo) DeletePending(context.Context, string) error   { return nil }
func (f *fakeRepo) DeleteForLesson(context.Context, string) error { return nil }

func (f *fakeRepo) ResetRunning(context.Context) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.resets++
	var n int64
	for _, j := range f.jobs {
		if j.Status == StatusRunning {
			j.Status = StatusPending
			n++
		}
	}
	return n, nil
}

func (f *fakeRepo) get(id string) Job {
	f.mu.Lock()
	defer f.mu.Unlock()
	return *f.jobs[id]
}

type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}
