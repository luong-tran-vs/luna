package grammar

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/luongtran/luna/backend/internal/ai"
)

// Example picture batches (added 2026-11-06): the admin draws the pictures of a lesson's examples,
// or of every lesson of a level, one picture after another, each saved as soon as it is drawn. Like
// the word bank's picture batch it runs inside this server process, not through the job queue: the
// local server and the production one share the queue, and only the local one runs Codex.

// MaxImageBatch is the most pictures one batch draws, so a click cannot run up the bill.
const MaxImageBatch = 300

// imageBatchTimeout bounds the drawing of one picture of a batch.
const imageBatchTimeout = 5 * time.Minute

// ErrImageBatchRunning means an example picture batch is running already; one runs at a time.
var ErrImageBatchRunning = errors.New("grammar: an image batch is running")

// ImageBatchRequest starts a batch over the examples of the given points' lessons.
type ImageBatchRequest struct {
	PointIDs []string
	// Keys, when set, limits the batch to these examples (one point): "Vẽ lại" of one example.
	Keys []string
	// Replace draws the examples that have a picture too, replacing it; without it they are left out.
	Replace bool
	Engine  ImageEngine
	// Key and Label tell the pages what the batch is for: a point ("point:<id>", `bài "…"`) or a
	// level ("level:A1", "các bài ngữ pháp A1").
	Key   string
	Label string
}

// ImageBatchItem is an example of a batch, with why it got no picture when it has none.
type ImageBatchItem struct {
	PointID string
	Key     string
	En      string
	Err     error
}

// ImageBatchStatus is how a batch is going. The zero value means no batch ran since the server started.
type ImageBatchStatus struct {
	Running  bool
	Stopping bool
	Stopped  bool
	Engine   ImageEngine
	Key      string
	Label    string
	Total    int
	// Points counts the lessons the examples come from.
	Points   int
	Drawing  *ImageBatchItem
	Saved    []ImageBatchItem
	Failures []ImageBatchItem
	StopErr  error
	// StartedAt and FinishedAt bound the batch.
	StartedAt  time.Time
	FinishedAt time.Time
}

// Done is how many examples were dealt with, with or without a picture.
func (b ImageBatchStatus) Done() int { return len(b.Saved) + len(b.Failures) }

type imageRun struct {
	status ImageBatchStatus
	stop   context.CancelFunc
}

// batchItem is an example to draw.
type batchItem struct {
	PointID string
	Key     string
	En, Vi  string
	// Picture, set for a picture exercise, describes its picture; En is then the exercise sentence.
	Picture string
	// Point is the point's English title, which the prompt names.
	Point string
}

// StartImageBatch picks the examples to draw and draws them one after another in the background.
func (s *Service) StartImageBatch(ctx context.Context, req ImageBatchRequest) (ImageBatchStatus, error) {
	if s.images.Store == nil {
		return ImageBatchStatus{}, ErrImagesUnavailable
	}
	drawer, err := s.imageDrawer(req.Engine)
	if err != nil {
		return ImageBatchStatus{}, err
	}
	engine := req.Engine
	if engine == "" {
		engine = EngineGemini
	}
	items, points, err := s.batchItems(ctx, req)
	if err != nil {
		return ImageBatchStatus{}, err
	}
	if len(items) == 0 {
		return ImageBatchStatus{}, &ValidationError{Fields: map[string]string{"pointIds": "Mọi câu ví dụ đã có ảnh"}}
	}
	if len(items) > MaxImageBatch {
		return ImageBatchStatus{}, &ValidationError{Fields: map[string]string{
			"pointIds": fmt.Sprintf("Mỗi lượt tối đa %d ảnh, lượt này cần %d. Hãy tạo theo từng bài.", MaxImageBatch, len(items)),
		}}
	}

	s.imgMu.Lock()
	defer s.imgMu.Unlock()
	if s.imgRun != nil && s.imgRun.status.Running {
		return ImageBatchStatus{}, ErrImageBatchRunning
	}
	// The batch outlives the request that started it.
	base := context.WithoutCancel(ctx)
	feed, stop := context.WithCancel(base)
	run := &imageRun{stop: stop, status: ImageBatchStatus{
		Running: true, Engine: engine, Key: strings.TrimSpace(req.Key), Label: strings.TrimSpace(req.Label),
		Total: len(items), Points: points, StartedAt: s.now(),
	}}
	s.imgRun = run
	slog.InfoContext(ctx, "grammar: image batch started", slog.String("engine", string(engine)),
		slog.String("label", run.status.Label), slog.Int("pictures", len(items)))
	go s.runImageBatch(base, feed, run, drawer, engine, items)
	return run.snapshot(), nil
}

// batchItems lists the examples a batch draws, in the order of the points, and how many lessons they come from.
func (s *Service) batchItems(ctx context.Context, req ImageBatchRequest) ([]batchItem, int, error) {
	only := map[string]bool{}
	for _, k := range req.Keys {
		only[strings.TrimSpace(k)] = true
	}
	seen := map[string]bool{}
	var out []batchItem
	points := 0
	for _, id := range req.PointIDs {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		p, err := s.point(id)
		// Only the points that can be drawn get pictures (added 2026-11-06).
		if err != nil || !p.Drawable {
			continue
		}
		l, err := s.lessons.Get(ctx, id)
		if errors.Is(err, ErrNotFound) {
			continue
		}
		if err != nil {
			return nil, 0, err
		}
		have, err := s.imageKeys(ctx, id)
		if err != nil {
			return nil, 0, err
		}
		n := len(out)
		for _, e := range l.Content.Examples {
			k := ExampleKey(e.En)
			if (len(only) > 0 && !only[k]) || (have[k] && !req.Replace) || strings.TrimSpace(e.En) == "" {
				continue
			}
			have[k] = true // a sentence given twice is drawn once
			out = append(out, batchItem{PointID: id, Key: k, En: e.En, Vi: e.Vi, Point: pointName(p)})
		}
		// The pictures of the picture exercises, which the learner sees once drawn.
		for _, list := range [][]Exercise{l.Content.Practice, l.Content.Mastery} {
			for _, e := range list {
				if e.Kind != KindPicture || strings.TrimSpace(e.Picture) == "" {
					continue
				}
				k := PictureKey(e)
				if (len(only) > 0 && !only[k]) || (have[k] && !req.Replace) {
					continue
				}
				have[k] = true
				out = append(out, batchItem{PointID: id, Key: k, En: e.Text, Picture: e.Picture, Point: pointName(p)})
			}
		}
		if len(out) > n {
			points++
		}
	}
	return out, points, nil
}

// pointName is how the prompt names a point: its English title, else the Vietnamese one.
func pointName(p Point) string {
	if t := strings.TrimSpace(p.TitleEn); t != "" {
		return t
	}
	return p.TitleVi
}

// runImageBatch draws the examples in order, one at a time, until they run out or feed ends.
func (s *Service) runImageBatch(ctx, feed context.Context, run *imageRun, drawer ai.ImageProvider, engine ImageEngine, items []batchItem) {
	for _, it := range items {
		if feed.Err() != nil {
			break
		}
		item := ImageBatchItem{PointID: it.PointID, Key: it.Key, En: it.En}
		s.imgMu.Lock()
		run.status.Drawing = &item
		s.imgMu.Unlock()

		dctx, cancel := context.WithTimeout(ctx, imageBatchTimeout)
		err := s.drawExample(dctx, drawer, engine, it)
		cancel()

		s.imgMu.Lock()
		st := &run.status
		st.Drawing = nil
		switch {
		case err == nil:
			st.Saved = append(st.Saved, item)
		case endsImageBatch(err):
			item.Err = err
			st.Failures = append(st.Failures, item)
			if st.StopErr == nil {
				st.StopErr = err
			}
			run.stop()
		default:
			slog.WarnContext(ctx, "grammar: batch picture failed", slog.String("point", it.PointID), slog.Any("error", err))
			item.Err = err
			st.Failures = append(st.Failures, item)
		}
		s.imgMu.Unlock()
	}

	s.imgMu.Lock()
	run.stop()
	st := &run.status
	st.Running, st.Stopped, st.Stopping = false, st.Stopping, false
	st.FinishedAt = s.now()
	s.imgMu.Unlock()
	slog.InfoContext(ctx, "grammar: image batch finished", slog.String("engine", string(engine)),
		slog.Int("saved", len(st.Saved)), slog.Int("failed", len(st.Failures)), slog.Int("total", st.Total),
		slog.Bool("stopped", st.Stopped), slog.Any("stop_error", st.StopErr))
}

// endsImageBatch tells the errors every next picture of the batch would get too.
func endsImageBatch(err error) bool {
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

// StopImageBatch asks the running batch to stop: the picture being drawn finishes, no new one starts.
func (s *Service) StopImageBatch() ImageBatchStatus {
	s.imgMu.Lock()
	defer s.imgMu.Unlock()
	if s.imgRun == nil {
		return ImageBatchStatus{}
	}
	if s.imgRun.status.Running {
		s.imgRun.status.Stopping = true
		s.imgRun.stop()
	}
	return s.imgRun.snapshot()
}

// ImageBatch returns how the running, or last, batch is going.
func (s *Service) ImageBatch() ImageBatchStatus {
	s.imgMu.Lock()
	defer s.imgMu.Unlock()
	if s.imgRun == nil {
		return ImageBatchStatus{}
	}
	return s.imgRun.snapshot()
}

// snapshot copies the status; the caller holds imgMu.
func (r *imageRun) snapshot() ImageBatchStatus {
	st := r.status
	if st.Drawing != nil {
		d := *st.Drawing
		st.Drawing = &d
	}
	st.Saved = append([]ImageBatchItem(nil), st.Saved...)
	st.Failures = append([]ImageBatchItem(nil), st.Failures...)
	return st
}
