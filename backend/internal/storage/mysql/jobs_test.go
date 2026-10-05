package mysql

import (
	"sync"
	"testing"
	"time"

	"github.com/luongtran/luna/backend/internal/job"
)

func jobsCount(t *testing.T, r *Jobs, where string, args ...any) int {
	t.Helper()
	var n int
	if err := r.db.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM jobs WHERE "+where, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func jobsOnly(t *testing.T, r *Jobs) job.Job {
	t.Helper()
	j, err := jobsScan(r.db.QueryRowContext(t.Context(), "SELECT "+jobsCols+" FROM jobs LIMIT 1"))
	if err != nil {
		t.Fatal(err)
	}
	return j
}

func TestJobsEnqueueAndClaim(t *testing.T) {
	t.Parallel()
	r := NewJobs(testDB(t))
	ctx := t.Context()
	now := time.Now().UTC().Truncate(time.Microsecond)
	lesson, target := newID(), newID()

	if _, ok, err := r.ClaimNext(ctx, now); err != nil || ok {
		t.Fatalf("empty queue: ok=%v err=%v", ok, err)
	}
	if err := r.Enqueue(ctx, job.Job{Type: job.TypeAnnotate, LessonID: lesson, TargetID: target, Revision: 3, RunAt: now}); err != nil {
		t.Fatal(err)
	}
	j, ok, err := r.ClaimNext(ctx, now)
	if err != nil || !ok {
		t.Fatalf("claim: ok=%v err=%v", ok, err)
	}
	if len(j.ID) != 24 || j.Type != job.TypeAnnotate || j.LessonID != lesson || j.TargetID != target || j.Revision != 3 ||
		j.Status != job.StatusRunning || j.Attempts != 1 || !j.RunAt.Equal(now) || j.CreatedAt.IsZero() || j.UpdatedAt.IsZero() {
		t.Fatalf("claimed = %+v", j)
	}
	if _, ok, _ := r.ClaimNext(ctx, now); ok {
		t.Fatal("a running job was claimed again")
	}
}

func TestJobsGradeJobHasNoLesson(t *testing.T) {
	t.Parallel()
	r := NewJobs(testDB(t))
	target := newID()
	if err := r.Enqueue(t.Context(), job.Job{Type: job.TypeGrade, TargetID: target, RunAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	j, ok, err := r.ClaimNext(t.Context(), time.Now().Add(time.Second))
	if err != nil || !ok || j.LessonID != "" || j.TargetID != target {
		t.Fatalf("grade job = %+v ok=%v err=%v", j, ok, err)
	}
}

func TestJobsEnqueueValidatesIDsAndDefaults(t *testing.T) {
	t.Parallel()
	r := NewJobs(testDB(t))
	if err := r.Enqueue(t.Context(), job.Job{Type: job.TypeAnnotate, LessonID: "bad"}); err == nil {
		t.Fatal("accepted a bad lesson id")
	}
	if err := r.Enqueue(t.Context(), job.Job{Type: job.TypeAnnotate, TargetID: "bad"}); err == nil {
		t.Fatal("accepted a bad target id")
	}
	if err := r.Enqueue(t.Context(), job.Job{Type: job.TypeAnnotate}); err != nil { // zero RunAt, no status
		t.Fatal(err)
	}
	if j := jobsOnly(t, r); j.Status != job.StatusPending || j.RunAt.IsZero() {
		t.Fatalf("defaults = %+v", j)
	}
	if err := r.Enqueue(t.Context(), job.Job{Type: job.TypeAnnotate, Status: job.StatusDone, RunAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	if n := jobsCount(t, r, "status = ?", string(job.StatusDone)); n != 1 {
		t.Fatalf("done jobs = %d", n)
	}
}

func TestJobsClaimOrderAndDueness(t *testing.T) {
	t.Parallel()
	r := NewJobs(testDB(t))
	ctx := t.Context()
	base := time.Now().UTC().Truncate(time.Microsecond)
	late, early, future := newID(), newID(), newID()
	for _, l := range []struct {
		lesson string
		at     time.Time
	}{{late, base.Add(-time.Minute)}, {early, base.Add(-time.Hour)}, {future, base.Add(time.Hour)}} {
		if err := r.Enqueue(ctx, job.Job{Type: job.TypeAnnotate, LessonID: l.lesson, RunAt: l.at}); err != nil {
			t.Fatal(err)
		}
	}
	var order []string
	for {
		j, ok, err := r.ClaimNext(ctx, base)
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			break
		}
		order = append(order, j.LessonID)
	}
	if len(order) != 2 || order[0] != early || order[1] != late {
		t.Fatalf("claimed order = %v, want [%s %s]", order, early, late)
	}
	if j, ok, _ := r.ClaimNext(ctx, base.Add(2*time.Hour)); !ok || j.LessonID != future {
		t.Fatalf("future job not claimed once due: %+v ok=%v", j, ok)
	}
}

func TestJobsLifecycleCompleteRetryFail(t *testing.T) {
	t.Parallel()
	r := NewJobs(testDB(t))
	ctx := t.Context()
	now := time.Now().UTC().Truncate(time.Microsecond)
	if err := r.Enqueue(ctx, job.Job{Type: job.TypePractice, LessonID: newID(), RunAt: now}); err != nil {
		t.Fatal(err)
	}
	j, _, _ := r.ClaimNext(ctx, now)

	later := now.Add(time.Minute)
	if err := r.Retry(ctx, j.ID, later, "boom"); err != nil {
		t.Fatal(err)
	}
	if got := jobsOnly(t, r); got.Status != job.StatusPending || got.Error != "boom" || !got.RunAt.Equal(later) || got.Attempts != 1 {
		t.Fatalf("after Retry = %+v", got)
	}
	if _, ok, _ := r.ClaimNext(ctx, now); ok {
		t.Fatal("retried job ran before its run time")
	}
	j2, ok, _ := r.ClaimNext(ctx, later)
	if !ok || j2.ID != j.ID || j2.Attempts != 2 {
		t.Fatalf("second claim = %+v ok=%v", j2, ok)
	}

	if err := r.Fail(ctx, j.ID, "nope"); err != nil {
		t.Fatal(err)
	}
	if got := jobsOnly(t, r); got.Status != job.StatusFailed || got.Error != "nope" {
		t.Fatalf("after Fail = %+v", got)
	}
	if err := r.Complete(ctx, j.ID); err != nil {
		t.Fatal(err)
	}
	if got := jobsOnly(t, r); got.Status != job.StatusDone || got.Error != "" {
		t.Fatalf("after Complete = %+v", got)
	}

	// An unknown id changes nothing and is not an error; a malformed one is.
	for _, f := range []func(string) error{
		func(id string) error { return r.Complete(ctx, id) },
		func(id string) error { return r.Fail(ctx, id, "x") },
		func(id string) error { return r.Retry(ctx, id, now, "x") },
	} {
		if err := f(newID()); err != nil {
			t.Fatalf("unknown id: %v", err)
		}
		if err := f("bad"); err == nil {
			t.Fatal("malformed id accepted")
		}
	}
}

func TestJobsDeletePendingAndForLesson(t *testing.T) {
	t.Parallel()
	r := NewJobs(testDB(t))
	ctx := t.Context()
	now := time.Now().UTC()
	a, b := newID(), newID()
	for _, l := range []string{a, a, a, b} {
		if err := r.Enqueue(ctx, job.Job{Type: job.TypeAnnotate, LessonID: l, RunAt: now.Add(-time.Second)}); err != nil {
			t.Fatal(err)
		}
	}
	if err := r.Enqueue(ctx, job.Job{Type: job.TypeGrade, TargetID: a, RunAt: now}); err != nil {
		t.Fatal(err)
	}
	// Make one of a's jobs running and one done.
	first, _, _ := r.ClaimNext(ctx, now)
	second, _, _ := r.ClaimNext(ctx, now)
	if first.LessonID != a && second.LessonID != a {
		t.Skip("claimed jobs are not a's; ordering assumption broke")
	}
	if err := r.Complete(ctx, second.ID); err != nil {
		t.Fatal(err)
	}

	if err := r.DeletePending(ctx, a); err != nil {
		t.Fatal(err)
	}
	if n := jobsCount(t, r, "lesson_id = ? AND status = ?", a, string(job.StatusPending)); n != 0 {
		t.Fatalf("pending left for a = %d", n)
	}
	if n := jobsCount(t, r, "lesson_id = ?", a); n == 0 {
		t.Fatal("DeletePending removed running/done jobs too")
	}
	if n := jobsCount(t, r, "lesson_id = ?", b); n != 1 {
		t.Fatalf("other lesson's jobs = %d", n)
	}

	if err := r.DeleteForLesson(ctx, a); err != nil {
		t.Fatal(err)
	}
	if n := jobsCount(t, r, "lesson_id = ?", a); n != 0 {
		t.Fatalf("jobs left for a = %d", n)
	}
	if n := jobsCount(t, r, "lesson_id = ?", b); n != 1 {
		t.Fatalf("other lesson's jobs = %d", n)
	}
	if n := jobsCount(t, r, "type = ?", string(job.TypeGrade)); n != 1 {
		t.Fatal("grade job (no lesson) was removed")
	}
	if err := r.DeletePending(ctx, "bad"); err == nil {
		t.Fatal("DeletePending accepted a bad id")
	}
	if err := r.DeleteForLesson(ctx, "bad"); err == nil {
		t.Fatal("DeleteForLesson accepted a bad id")
	}
}

func TestJobsResetRunning(t *testing.T) {
	t.Parallel()
	r := NewJobs(testDB(t))
	ctx := t.Context()
	now := time.Now().UTC()
	for range 3 {
		if err := r.Enqueue(ctx, job.Job{Type: job.TypeAnnotate, LessonID: newID(), RunAt: now.Add(-time.Second)}); err != nil {
			t.Fatal(err)
		}
	}
	r.ClaimNext(ctx, now)
	r.ClaimNext(ctx, now)
	n, err := r.ResetRunning(ctx)
	if err != nil || n != 2 {
		t.Fatalf("reset = %d, %v", n, err)
	}
	if got := jobsCount(t, r, "status = ?", string(job.StatusPending)); got != 3 {
		t.Fatalf("pending = %d", got)
	}
	if n, _ := r.ResetRunning(ctx); n != 0 {
		t.Fatalf("second reset = %d", n)
	}
}

// Two processes (here: many goroutines on separate connections) draining the queue must never
// take the same job twice.
func TestJobsClaimIsExclusiveUnderConcurrency(t *testing.T) {
	t.Parallel()
	r := NewJobs(testDB(t))
	ctx := t.Context()
	now := time.Now().UTC()
	const total = 40
	for range total {
		if err := r.Enqueue(ctx, job.Job{Type: job.TypeAnnotate, LessonID: newID(), RunAt: now.Add(-time.Second)}); err != nil {
			t.Fatal(err)
		}
	}
	var (
		mu     sync.Mutex
		seen   = map[string]int{}
		wg     sync.WaitGroup
		failed error
	)
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				j, ok, err := r.ClaimNext(ctx, now)
				mu.Lock()
				if err != nil && failed == nil {
					failed = err
				}
				if ok {
					seen[j.ID]++
				}
				mu.Unlock()
				if err != nil || !ok {
					return
				}
			}
		}()
	}
	wg.Wait()
	if failed != nil {
		t.Fatal(failed)
	}
	if len(seen) != total {
		t.Fatalf("claimed %d distinct jobs, want %d", len(seen), total)
	}
	for id, n := range seen {
		if n != 1 {
			t.Fatalf("job %s claimed %d times", id, n)
		}
	}
	if got := jobsCount(t, r, "attempts = 1 AND status = ?", string(job.StatusRunning)); got != total {
		t.Fatalf("running with 1 attempt = %d", got)
	}
}
