package topicimage

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

// Topic picture batches: the admin picks topics (one to draw again, or every topic lacking a
// picture) and the server draws them in the background, saving each picture as soon as it is drawn.
// Like the other picture batches it runs inside this server process, not through the job queue.

// MaxBatch is the most topics one batch draws.
const MaxBatch = ai.MaxImageBatch

// MaxStyle bounds the admin's picture style.
const MaxStyle = 500

// batchDrawTimeout bounds one picture of a batch.
const batchDrawTimeout = 5 * time.Minute

// BatchRequest starts a batch over the given topics.
type BatchRequest struct {
	TopicIDs []string
	Engine   Engine
	Style    string
}

// BatchFailure is a topic of the batch that got no picture.
type BatchFailure struct {
	TopicID string
	Err     error
}

// BatchStatus is how a batch is going. The zero value means no batch ran since the server started.
type BatchStatus struct {
	Running    bool
	Stopping   bool
	Stopped    bool
	Engine     Engine
	Total      int
	Drawing    []string
	Saved      []string
	Failures   []BatchFailure
	StopErr    error
	StartedAt  time.Time
	FinishedAt time.Time
}

// Done is how many topics were dealt with, with or without a picture.
func (b BatchStatus) Done() int { return len(b.Saved) + len(b.Failures) }

type batchRun struct {
	status BatchStatus
	stop   context.CancelFunc
}

// StartBatch checks the request and draws the topics in the background.
func (s *Service) StartBatch(ctx context.Context, req BatchRequest) (BatchStatus, error) {
	if s.d.Store == nil || s.d.Topics == nil {
		return BatchStatus{}, ErrImagesUnavailable
	}
	style := strings.TrimSpace(req.Style)
	if utf8.RuneCountInString(style) > MaxStyle {
		return BatchStatus{}, &ValidationError{Fields: map[string]string{"style": fmt.Sprintf("Mô tả ảnh tối đa %d ký tự", MaxStyle)}}
	}
	topics, err := s.batchTopics(ctx, req.TopicIDs)
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
	run := &batchRun{stop: stop, status: BatchStatus{Running: true, Engine: engine, Total: len(topics), StartedAt: s.d.Now()}}
	s.run = run
	s.d.Log.InfoContext(ctx, "topicimage: batch started", slog.String("engine", string(engine)),
		slog.Int("topics", len(topics)), slog.Int("at_once", workers))
	go s.runBatch(base, feed, run, drawer, engine, style, topics, workers)
	return run.snapshot(), nil
}

// batchTopics reads the topics of a request, without repeats, keeping the order.
func (s *Service) batchTopics(ctx context.Context, ids []string) ([]Topic, error) {
	seen := map[string]bool{}
	out := make([]Topic, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		t, err := s.d.Topics.Get(ctx, id)
		if errors.Is(err, ErrTopicNotFound) {
			return nil, &ValidationError{Fields: map[string]string{"topicIds": "Có chủ đề không tồn tại"}}
		}
		if err != nil {
			return nil, fmt.Errorf("topicimage: topic: %w", err)
		}
		out = append(out, t)
	}
	if len(out) == 0 {
		return nil, &ValidationError{Fields: map[string]string{"topicIds": "Chưa chọn chủ đề nào"}}
	}
	if len(out) > MaxBatch {
		return nil, &ValidationError{Fields: map[string]string{"topicIds": fmt.Sprintf("Mỗi lượt tối đa %d ảnh", MaxBatch)}}
	}
	return out, nil
}

// runBatch hands the topics out in order to the workers until they run out or feed ends.
func (s *Service) runBatch(ctx, feed context.Context, run *batchRun, drawer ai.ImageProvider, engine Engine, style string, topics []Topic, workers int) {
	next := make(chan Topic)
	go func() {
		defer close(next)
		for _, t := range topics {
			select {
			case next <- t:
			case <-feed.Done():
				return
			}
		}
	}()
	var wg sync.WaitGroup
	for range workers {
		wg.Go(func() {
			for t := range next {
				if feed.Err() != nil {
					continue // stopped: let the feeder end
				}
				s.drawInBatch(ctx, run, drawer, engine, style, t)
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
	s.d.Log.InfoContext(ctx, "topicimage: batch finished", slog.String("engine", string(engine)),
		slog.Int("saved", len(st.Saved)), slog.Int("failed", len(st.Failures)), slog.Int("total", st.Total),
		slog.Bool("stopped", st.Stopped), slog.Any("stop_error", st.StopErr))
}

func (s *Service) drawInBatch(ctx context.Context, run *batchRun, drawer ai.ImageProvider, engine Engine, style string, t Topic) {
	s.mu.Lock()
	run.status.Drawing = append(run.status.Drawing, t.ID)
	s.mu.Unlock()

	dctx, cancel := context.WithTimeout(ctx, batchDrawTimeout)
	err := s.draw(dctx, drawer, engine, t, style)
	cancel()

	s.mu.Lock()
	defer s.mu.Unlock()
	st := &run.status
	st.Drawing = slices.DeleteFunc(st.Drawing, func(id string) bool { return id == t.ID })
	switch {
	case err == nil:
		st.Saved = append(st.Saved, t.ID)
	case endsBatch(err):
		// The other topics would fail alike: stop handing them out.
		if st.StopErr == nil {
			st.StopErr = err
		}
		st.Failures = append(st.Failures, BatchFailure{TopicID: t.ID, Err: err})
		run.stop()
	default:
		s.d.Log.WarnContext(ctx, "topicimage: batch picture failed", slog.String("topic", t.ID), slog.Any("error", err))
		st.Failures = append(st.Failures, BatchFailure{TopicID: t.ID, Err: err})
	}
}

// endsBatch tells the errors every next topic of the batch would get too.
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

// StopBatch asks the running batch to stop: the topics being drawn finish, no new one starts.
func (s *Service) StopBatch() BatchStatus {
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
func (s *Service) Batch() BatchStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.run == nil {
		return BatchStatus{}
	}
	return s.run.snapshot()
}

// snapshot copies the status; the caller holds mu.
func (r *batchRun) snapshot() BatchStatus {
	st := r.status
	st.Drawing = slices.Clone(st.Drawing)
	st.Saved = slices.Clone(st.Saved)
	st.Failures = slices.Clone(st.Failures)
	return st
}
