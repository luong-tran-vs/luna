package verbs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/luongtran/luna/backend/internal/ai"
)

// Verb picture batches: the admin picks verbs (all those lacking a picture, or one to draw again)
// and the server draws them in the background, saving each picture as soon as it is drawn. Like the
// word bank's batch it runs inside this server process, not through the job queue: the local server
// and the production one share the queue, and only the local one runs Codex.

// MaxBatch is the most verbs one batch draws: every irregular verb fits in one.
const MaxBatch = 300

// MaxStyle bounds the admin's picture style.
const MaxStyle = 500

// batchDrawTimeout bounds one picture of a batch.
const batchDrawTimeout = 5 * time.Minute

// ErrBatchRunning means a verb picture batch is running already; one runs at a time.
var ErrBatchRunning = errors.New("verbs: an image batch is running")

// BatchRequest starts a batch over the given verbs.
type BatchRequest struct {
	Items  []DrawItem
	Engine ImageEngine
	Style  string
}

// BatchFailure is a verb of the batch that got no picture.
type BatchFailure struct {
	Base string
	Err  error
}

// BatchStatus is how a batch is going. The zero value means no batch ran since the server started.
type BatchStatus struct {
	Running    bool
	Stopping   bool
	Stopped    bool
	Engine     ImageEngine
	Total      int
	Drawing    []string
	Saved      []string
	Failures   []BatchFailure
	StopErr    error
	StartedAt  time.Time
	FinishedAt time.Time
}

// Done is how many verbs were dealt with, with or without a picture.
func (b BatchStatus) Done() int { return len(b.Saved) + len(b.Failures) }

type imageRun struct {
	status BatchStatus
	stop   context.CancelFunc
}

// StartBatch checks the request and draws the verbs one after another in the background.
func (s *Images) StartBatch(ctx context.Context, req BatchRequest) (BatchStatus, error) {
	if s.d.Store == nil {
		return BatchStatus{}, ErrImagesUnavailable
	}
	style := strings.TrimSpace(req.Style)
	if utf8.RuneCountInString(style) > MaxStyle {
		return BatchStatus{}, &ValidationError{Fields: map[string]string{"style": fmt.Sprintf("Mô tả ảnh tối đa %d ký tự", MaxStyle)}}
	}
	items, err := batchItems(req.Items)
	if err != nil {
		return BatchStatus{}, err
	}
	drawer, workers, err := s.drawer(req.Engine)
	if err != nil {
		return BatchStatus{}, err
	}
	engine := req.Engine

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.run != nil && s.run.status.Running {
		return BatchStatus{}, ErrBatchRunning
	}
	// The batch outlives the request that started it.
	base := context.WithoutCancel(ctx)
	feed, stop := context.WithCancel(base)
	run := &imageRun{stop: stop, status: BatchStatus{Running: true, Engine: engine, Total: len(items), StartedAt: s.d.Now()}}
	s.run = run
	s.d.Log.InfoContext(ctx, "verbs: image batch started", slog.String("engine", string(engine)),
		slog.Int("verbs", len(items)), slog.Int("at_once", workers))
	go s.runBatch(base, feed, run, drawer, engine, style, items, workers)
	return run.snapshot(), nil
}

// batchItems checks the verbs of a request and drops repeats, keeping the order.
func batchItems(in []DrawItem) ([]DrawItem, error) {
	seen := map[string]bool{}
	out := make([]DrawItem, 0, len(in))
	for _, it := range in {
		b, err := checkBase(it.Base)
		if err != nil {
			return nil, err
		}
		if seen[b] {
			continue
		}
		seen[b] = true
		it.Base = b
		it.MeaningVi = clip(strings.TrimSpace(it.MeaningVi), 200)
		it.Sentence = clip(strings.TrimSpace(it.Sentence), 300)
		out = append(out, it)
	}
	if len(out) == 0 {
		return nil, &ValidationError{Fields: map[string]string{"items": "Chưa chọn động từ nào"}}
	}
	if len(out) > MaxBatch {
		return nil, &ValidationError{Fields: map[string]string{"items": fmt.Sprintf("Mỗi lượt tối đa %d ảnh", MaxBatch)}}
	}
	return out, nil
}

// clip cuts s to at most n characters.
func clip(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n])
}

// runBatch hands the verbs out in order to the workers until they run out or feed ends.
func (s *Images) runBatch(ctx, feed context.Context, run *imageRun, drawer ai.ImageProvider, engine ImageEngine, style string, items []DrawItem, workers int) {
	next := make(chan DrawItem)
	go func() {
		defer close(next)
		for _, it := range items {
			select {
			case next <- it:
			case <-feed.Done():
				return
			}
		}
	}()
	var wg sync.WaitGroup
	for range workers {
		wg.Go(func() {
			for it := range next {
				if feed.Err() != nil {
					continue // stopped: let the feeder end
				}
				s.drawInBatch(ctx, run, drawer, engine, style, it)
			}
		})
	}
	wg.Wait()

	s.mu.Lock()
	run.stop()
	st := &run.status
	st.Running, st.Stopped, st.Stopping = false, st.Stopping, false
	st.FinishedAt = s.d.Now()
	s.mu.Unlock()
	s.d.Log.InfoContext(ctx, "verbs: image batch finished", slog.String("engine", string(engine)),
		slog.Int("saved", len(st.Saved)), slog.Int("failed", len(st.Failures)), slog.Int("total", st.Total),
		slog.Bool("stopped", st.Stopped), slog.Any("stop_error", st.StopErr))
}

func (s *Images) drawInBatch(ctx context.Context, run *imageRun, drawer ai.ImageProvider, engine ImageEngine, style string, it DrawItem) {
	s.mu.Lock()
	run.status.Drawing = append(run.status.Drawing, it.Base)
	s.mu.Unlock()

	dctx, cancel := context.WithTimeout(ctx, batchDrawTimeout)
	err := s.draw(dctx, drawer, engine, it, style)
	cancel()

	s.mu.Lock()
	defer s.mu.Unlock()
	st := &run.status
	st.Drawing = slices.DeleteFunc(st.Drawing, func(b string) bool { return b == it.Base })
	switch {
	case err == nil:
		st.Saved = append(st.Saved, it.Base)
	case endsBatch(err):
		// The other verbs would fail alike: stop handing them out.
		if st.StopErr == nil {
			st.StopErr = err
		}
		st.Failures = append(st.Failures, BatchFailure{Base: it.Base, Err: err})
		run.stop()
	default:
		s.d.Log.WarnContext(ctx, "verbs: batch picture failed", slog.String("base", it.Base), slog.Any("error", err))
		st.Failures = append(st.Failures, BatchFailure{Base: it.Base, Err: err})
	}
}

// endsBatch tells the errors every next verb of the batch would get too.
func endsBatch(err error) bool {
	for _, e := range []error{
		ai.ErrNotConfigured, ai.ErrInvalidKey, ai.ErrQuota, ai.ErrNoImageTool,
		ErrCodexNotLoggedIn, ErrCodexUnavailable, ErrImagesUnavailable,
	} {
		if errors.Is(err, e) {
			return true
		}
	}
	return false
}

// StopBatch asks the running batch to stop: the verbs being drawn finish, no new one starts.
func (s *Images) StopBatch() BatchStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.run == nil {
		return BatchStatus{}
	}
	if s.run.status.Running {
		s.run.status.Stopping = true
		s.run.stop()
	}
	return s.run.snapshot()
}

// Batch returns how the running, or last, batch is going.
func (s *Images) Batch() BatchStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.run == nil {
		return BatchStatus{}
	}
	return s.run.snapshot()
}

// snapshot copies the status; the caller holds mu.
func (r *imageRun) snapshot() BatchStatus {
	st := r.status
	st.Drawing = slices.Clone(st.Drawing)
	st.Saved = slices.Clone(st.Saved)
	st.Failures = slices.Clone(st.Failures)
	return st
}
