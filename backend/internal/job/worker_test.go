package job

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"
)

type harness struct {
	repo   *fakeRepo
	clock  *fakeClock
	worker *Worker
	calls  int
	failed []error
	mu     sync.Mutex
}

func newHarness(t *testing.T, handler func(ctx context.Context, j Job) error) *harness {
	t.Helper()
	h := &harness{repo: newFakeRepo(), clock: &fakeClock{now: time.Date(2026, 9, 29, 8, 0, 0, 0, time.UTC)}}
	h.worker = NewWorker(h.repo, map[Type]Handler{
		TypeTTS: func(ctx context.Context, j Job) error {
			h.mu.Lock()
			h.calls++
			h.mu.Unlock()
			return handler(ctx, j)
		},
	}, func(_ context.Context, _ Job, err error) {
		h.mu.Lock()
		h.failed = append(h.failed, err)
		h.mu.Unlock()
	}, h.clock.Now, slog.New(slog.DiscardHandler))
	return h
}

func (h *harness) enqueue(t *testing.T, typ Type) string {
	t.Helper()
	if err := h.repo.Enqueue(t.Context(), Job{Type: typ, LessonID: "l1", Revision: 1, RunAt: h.clock.Now()}); err != nil {
		t.Fatal(err)
	}
	return "1"
}

func TestWorkerSuccess(t *testing.T) {
	t.Parallel()
	h := newHarness(t, func(context.Context, Job) error { return nil })
	id := h.enqueue(t, TypeTTS)

	if !h.worker.RunOnce(t.Context()) {
		t.Fatal("RunOnce found no job")
	}
	if got := h.repo.get(id); got.Status != StatusDone || got.Attempts != 1 {
		t.Fatalf("job = %+v, want done after 1 attempt", got)
	}
	if h.worker.RunOnce(t.Context()) {
		t.Fatal("RunOnce ran a job twice")
	}
}

func TestWorkerRetriesWithBackoffThenFails(t *testing.T) {
	t.Parallel()
	boom := errors.New("kokoro down")
	h := newHarness(t, func(context.Context, Job) error { return boom })
	id := h.enqueue(t, TypeTTS)
	start := h.clock.Now()

	h.worker.RunOnce(t.Context())
	if got := h.repo.get(id); got.Status != StatusPending || !got.RunAt.Equal(start.Add(30*time.Second)) || got.Error == "" {
		t.Fatalf("after 1st failure: %+v", got)
	}
	if h.worker.RunOnce(t.Context()) {
		t.Fatal("job ran before its backoff elapsed")
	}

	h.clock.Advance(30 * time.Second)
	h.worker.RunOnce(t.Context())
	if got := h.repo.get(id); got.Status != StatusPending || !got.RunAt.Equal(h.clock.Now().Add(2*time.Minute)) {
		t.Fatalf("after 2nd failure: %+v", got)
	}

	h.clock.Advance(2 * time.Minute)
	h.worker.RunOnce(t.Context())
	if got := h.repo.get(id); got.Status != StatusFailed || got.Attempts != MaxAttempts {
		t.Fatalf("after 3rd failure: %+v", got)
	}
	if len(h.failed) != 1 || !errors.Is(h.failed[0], boom) {
		t.Fatalf("onFailed calls = %v", h.failed)
	}
}

func TestWorkerPermanentErrorFailsImmediately(t *testing.T) {
	t.Parallel()
	h := newHarness(t, func(context.Context, Job) error { return Permanent(errors.New("no api key")) })
	id := h.enqueue(t, TypeTTS)

	h.worker.RunOnce(t.Context())
	if got := h.repo.get(id); got.Status != StatusFailed || got.Attempts != 1 {
		t.Fatalf("job = %+v, want failed after 1 attempt", got)
	}
	if len(h.failed) != 1 {
		t.Fatalf("onFailed not called")
	}
}

func TestWorkerUnknownTypeFails(t *testing.T) {
	t.Parallel()
	h := newHarness(t, func(context.Context, Job) error { return nil })
	id := h.enqueue(t, TypeAnnotate) // no handler registered in the harness

	h.worker.RunOnce(t.Context())
	if got := h.repo.get(id); got.Status != StatusFailed {
		t.Fatalf("job = %+v, want failed", got)
	}
}

func TestWorkerRunResetsRunningAndStops(t *testing.T) {
	t.Parallel()
	done := make(chan struct{})
	h := newHarness(t, func(context.Context, Job) error {
		close(done)
		return nil
	})
	// A job left running by a crashed process.
	_ = h.repo.Enqueue(t.Context(), Job{Type: TypeTTS, Status: StatusRunning, RunAt: h.clock.Now()})

	ctx, cancel := context.WithCancel(t.Context())
	stopped := make(chan struct{})
	go func() {
		h.worker.Run(ctx)
		close(stopped)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("reset job was not processed")
	}
	cancel()
	select {
	case <-stopped:
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not stop after cancel")
	}
	if h.repo.resets != 1 {
		t.Fatalf("ResetRunning called %d times", h.repo.resets)
	}
}

func TestWorkerNotifyWakesIdleWorker(t *testing.T) {
	t.Parallel()
	done := make(chan struct{})
	h := newHarness(t, func(context.Context, Job) error {
		close(done)
		return nil
	})
	h.worker.idle = time.Hour // without Notify the job would wait an hour

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	go h.worker.Run(ctx)
	time.Sleep(50 * time.Millisecond) // let Run go idle

	h.enqueue(t, TypeTTS)
	h.worker.Notify()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Notify did not wake the worker")
	}
}
