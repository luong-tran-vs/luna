package lesson

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/luongtran/luna/backend/internal/ai"
)

// Scene picture batches (added 2026-11-06): from the course page the admin draws the scene picture
// of every lesson of a topic that has none, one lesson after another (CODEX_IMAGE_CONCURRENCY at once
// with Codex since 2026-11-11, as the word bank does), each saved as soon as it is drawn. Like the word bank's picture batch it runs inside this server process, not through the
// job queue: the local server and the production one share the queue, and only the local one runs
// Codex. A batch cut short is simply started again: the lessons that got a picture are skipped.

// MaxSceneBatch is the most lessons one batch takes, so a click cannot run up the bill.
const MaxSceneBatch = 100

// sceneBatchTimeout bounds the drawing of one scene of a batch.
const sceneBatchTimeout = 5 * time.Minute

// ErrSceneBatchRunning means a scene batch is running already; one runs at a time.
var ErrSceneBatchRunning = errors.New("lesson: a scene batch is running")

// SceneBatchRequest starts a batch over the lessons of a topic.
type SceneBatchRequest struct {
	LessonIDs []string
	Engine    ImageEngine
	// Replace draws the lessons that have a scene picture too, replacing it (the lesson page's
	// "Tạo lại ảnh cảnh"); without it they are left out.
	Replace bool
	// Key and Label tell the pages what the batch is for: a topic of the course ("A1-3", `chủ đề
	// "Gia đình"`) or one lesson ("lesson:<id>", `bài "…"`).
	Key   string
	Label string
}

// SceneBatchLesson is a lesson of a batch, with why it got no picture when it has none.
type SceneBatchLesson struct {
	ID    string
	Title string
	Err   error
}

// SceneBatchStatus is how a batch is going. The zero value means no batch ran since the server started.
type SceneBatchStatus struct {
	Running bool
	// Stopping is true once the admin asked to stop: the lessons being drawn finish, no new one starts.
	Stopping bool
	Stopped  bool
	Engine   ImageEngine
	Key      string
	Label    string
	// Total counts the lessons to draw (the skipped ones aside).
	Total int
	// Drawing are the lessons being drawn now (several at once with Codex since 2026-11-11).
	Drawing []SceneBatchLesson
	Saved   []SceneBatchLesson
	// Failures are lessons whose drawing failed; Skipped those not drawn at all (not annotated yet).
	Failures []SceneBatchLesson
	Skipped  []SceneBatchLesson
	// StopErr is why the batch ended early by itself: an error the next lessons would get too.
	StopErr    error
	StartedAt  time.Time
	FinishedAt time.Time
}

// Done is how many lessons were dealt with, with or without a picture.
func (b SceneBatchStatus) Done() int { return len(b.Saved) + len(b.Failures) }

type sceneRun struct {
	status SceneBatchStatus
	stop   context.CancelFunc
}

// StartSceneBatch picks the lessons without a scene picture among req.LessonIDs and draws them one
// after another in the background. A lesson not annotated yet is skipped.
func (s *Service) StartSceneBatch(ctx context.Context, req SceneBatchRequest) (SceneBatchStatus, error) {
	if s.ImageStore == nil {
		return SceneBatchStatus{}, ErrImagesUnavailable
	}
	drawer, err := s.imageDrawer(req.Engine)
	if err != nil {
		return SceneBatchStatus{}, err
	}
	engine := req.Engine
	if engine == "" {
		engine = EngineGemini
	}
	ids := uniqueIDs(req.LessonIDs)
	if len(ids) > MaxSceneBatch {
		return SceneBatchStatus{}, &ValidationError{Fields: map[string]string{"lessonIds": fmt.Sprintf("Mỗi lượt tối đa %d bài", MaxSceneBatch)}}
	}
	scenes, err := s.HasScenes(ctx, ids)
	if err != nil {
		return SceneBatchStatus{}, err
	}
	st := SceneBatchStatus{Running: true, Engine: engine, Key: strings.TrimSpace(req.Key), Label: strings.TrimSpace(req.Label)}
	var todo []Lesson
	for _, id := range ids {
		if scenes[id] && !req.Replace {
			continue
		}
		l, err := s.Lessons.Get(ctx, id)
		if errors.Is(err, ErrNotFound) {
			continue
		}
		if err != nil {
			return SceneBatchStatus{}, err
		}
		if l.AnnotationStatus != StatusDone {
			st.Skipped = append(st.Skipped, SceneBatchLesson{ID: l.ID, Title: l.Title, Err: ErrAnnotationNotDone})
			continue
		}
		todo = append(todo, l)
	}
	if len(todo) == 0 {
		msg := "Mọi bài đã có ảnh cảnh"
		if len(st.Skipped) > 0 {
			msg = "Không có bài nào tạo được ảnh cảnh: các bài còn thiếu chưa chú thích xong"
		}
		return SceneBatchStatus{}, &ValidationError{Fields: map[string]string{"lessonIds": msg}}
	}
	st.Total = len(todo)
	workers := s.sceneWorkers(engine)

	s.sceneMu.Lock()
	defer s.sceneMu.Unlock()
	if s.sceneRun != nil && s.sceneRun.status.Running {
		return SceneBatchStatus{}, ErrSceneBatchRunning
	}
	// The batch outlives the request that started it.
	base := context.WithoutCancel(ctx)
	feed, stop := context.WithCancel(base)
	st.StartedAt = s.Now()
	run := &sceneRun{status: st, stop: stop}
	s.sceneRun = run
	s.Log.InfoContext(ctx, "lesson: scene batch started", slog.String("engine", string(engine)),
		slog.String("label", st.Label), slog.Int("lessons", len(todo)), slog.Int("skipped", len(st.Skipped)),
		slog.Int("at_once", workers))
	go s.runSceneBatch(base, feed, run, drawer, engine, todo, workers)
	return run.snapshot(), nil
}

// sceneWorkers is how many scenes a batch draws at once: CodexConcurrency with Codex (added
// 2026-11-11), one with Gemini.
func (s *Service) sceneWorkers(engine ImageEngine) int {
	if engine == EngineCodex {
		return max(1, s.CodexConcurrency)
	}
	return 1
}

// runSceneBatch hands the lessons out in order to the workers until they run out or feed ends.
func (s *Service) runSceneBatch(ctx, feed context.Context, run *sceneRun, drawer ai.ImageProvider, engine ImageEngine, lessons []Lesson, workers int) {
	next := make(chan Lesson)
	go func() {
		defer close(next)
		for _, l := range lessons {
			select {
			case next <- l:
			case <-feed.Done():
				return
			}
		}
	}()
	var wg sync.WaitGroup
	for range workers {
		wg.Go(func() {
			for l := range next {
				if feed.Err() != nil {
					continue // stopped: let the feeder end
				}
				s.drawSceneInBatch(ctx, run, drawer, engine, l)
			}
		})
	}
	wg.Wait()

	s.sceneMu.Lock()
	run.stop()
	st := &run.status
	st.Running, st.Stopped, st.Stopping = false, st.Stopping, false
	st.FinishedAt = s.Now()
	s.sceneMu.Unlock()
	s.Log.InfoContext(ctx, "lesson: scene batch finished", slog.String("engine", string(engine)),
		slog.Int("saved", len(st.Saved)), slog.Int("failed", len(st.Failures)), slog.Int("total", st.Total),
		slog.Bool("stopped", st.Stopped), slog.Any("stop_error", st.StopErr))
}

func (s *Service) drawSceneInBatch(ctx context.Context, run *sceneRun, drawer ai.ImageProvider, engine ImageEngine, l Lesson) {
	item := SceneBatchLesson{ID: l.ID, Title: l.Title}
	s.sceneMu.Lock()
	run.status.Drawing = append(run.status.Drawing, item)
	s.sceneMu.Unlock()

	dctx, cancel := context.WithTimeout(ctx, sceneBatchTimeout)
	// The lesson is read again: it may have changed since the batch started.
	err := s.drawBatchScene(dctx, l.ID, drawer, engine)
	cancel()

	s.sceneMu.Lock()
	defer s.sceneMu.Unlock()
	st := &run.status
	st.Drawing = slices.DeleteFunc(st.Drawing, func(d SceneBatchLesson) bool { return d.ID == l.ID })
	switch {
	case err == nil:
		st.Saved = append(st.Saved, item)
	case endsSceneBatch(err):
		item.Err = err
		st.Failures = append(st.Failures, item)
		if st.StopErr == nil {
			st.StopErr = err
		}
		run.stop()
	default:
		s.Log.WarnContext(ctx, "lesson: batch scene failed", slog.String("lesson_id", l.ID), slog.Any("error", err))
		item.Err = err
		st.Failures = append(st.Failures, item)
	}
}

func (s *Service) drawBatchScene(ctx context.Context, id string, drawer ai.ImageProvider, engine ImageEngine) error {
	l, err := s.Lessons.Get(ctx, id)
	if err != nil {
		return err
	}
	return s.drawScene(ctx, l, drawer, engine)
}

// endsSceneBatch tells the errors every next lesson of the batch would get too.
func endsSceneBatch(err error) bool {
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

// StopSceneBatch asks the running batch to stop: the lessons being drawn finish, no new one starts.
func (s *Service) StopSceneBatch() SceneBatchStatus {
	s.sceneMu.Lock()
	defer s.sceneMu.Unlock()
	if s.sceneRun == nil {
		return SceneBatchStatus{}
	}
	if s.sceneRun.status.Running {
		s.sceneRun.status.Stopping = true
		s.sceneRun.stop()
	}
	return s.sceneRun.snapshot()
}

// SceneBatch returns how the running, or last, batch is going.
func (s *Service) SceneBatch() SceneBatchStatus {
	s.sceneMu.Lock()
	defer s.sceneMu.Unlock()
	if s.sceneRun == nil {
		return SceneBatchStatus{}
	}
	return s.sceneRun.snapshot()
}

// snapshot copies the status; the caller holds sceneMu.
func (r *sceneRun) snapshot() SceneBatchStatus {
	st := r.status
	st.Drawing = slices.Clone(st.Drawing)
	st.Saved = append([]SceneBatchLesson(nil), st.Saved...)
	st.Failures = append([]SceneBatchLesson(nil), st.Failures...)
	st.Skipped = append([]SceneBatchLesson(nil), st.Skipped...)
	return st
}

// uniqueIDs trims ids and drops empty and repeated ones, keeping their order.
func uniqueIDs(ids []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, id := range ids {
		if id = strings.TrimSpace(id); id != "" && !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}
