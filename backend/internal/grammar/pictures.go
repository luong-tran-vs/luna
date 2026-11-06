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

// Picture exercises added to a lesson (added 2026-11-06): a lesson written before the picture kind
// gets its picture exercises from one AI request, without being written again. The rest of the
// lesson, its edits and the checks the admin confirmed stay.

// How many picture exercises a lesson of a point that can be drawn has in each part.
const (
	PicturePractice = 3
	PictureMastery  = 1
)

// picturesNeeded is how many picture exercises each part still needs, within its limit.
func picturesNeeded(c Content) (practice, mastery int) {
	need := func(list []Exercise, want, limit int) int {
		have := 0
		for _, e := range list {
			if e.Kind == KindPicture {
				have++
			}
		}
		return max(0, min(want-have, limit-len(list)))
	}
	return need(c.Practice, PicturePractice, maxPractice), need(c.Mastery, PictureMastery, maxMastery)
}

// hasAllPictures reports whether both parts have their picture exercises.
func hasAllPictures(c Content) bool {
	count := func(list []Exercise) int {
		n := 0
		for _, e := range list {
			if e.Kind == KindPicture {
				n++
			}
		}
		return n
	}
	return count(c.Practice) >= PicturePractice && count(c.Mastery) >= PictureMastery
}

// AddPictures has the AI write the picture exercises a lesson is missing and adds them at the end of
// practice and mastery. added counts them. The new exercises are checked by the AI when the lesson
// was checked before; the other checks stay. A published lesson stays published: the learner sees a
// picture exercise once its picture is drawn.
func (s *Service) AddPictures(ctx context.Context, pointID string) (l Lesson, added int, err error) {
	p, err := s.point(pointID)
	if err != nil {
		return Lesson{}, 0, err
	}
	if !p.Drawable {
		return Lesson{}, 0, &ValidationError{Fields: map[string]string{"pointId": "Điểm ngữ pháp này không vẽ được nên không có bài nhìn ảnh"}}
	}
	l, err = s.lessons.Get(ctx, pointID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return Lesson{}, 0, err
		}
		return Lesson{}, 0, fmt.Errorf("grammar: get lesson: %w", err)
	}
	np, nm := picturesNeeded(l.Content)
	if np+nm == 0 {
		msg := "Bài đã có đủ bài nhìn ảnh"
		if !hasAllPictures(l.Content) {
			msg = "Bài đã đủ số bài tập tối đa. Xoá bớt bài tập rồi thử lại."
		}
		return Lesson{}, 0, &ValidationError{Fields: map[string]string{"pointId": msg}}
	}

	seen := map[string]bool{}
	var existing []string
	for _, e := range append(append([]Exercise(nil), l.Content.Practice...), l.Content.Mastery...) {
		seen[strings.ToLower(strings.TrimSpace(e.Text))] = true
		existing = append(existing, e.Text)
	}

	aiCtx, cancel := context.WithTimeout(ctx, generateTimeout)
	defer cancel()
	raw, err := s.ai.GrammarPictures(aiCtx, ai.GrammarPicturesRequest{
		Level: p.Level, TitleVi: p.TitleVi, TitleEn: p.TitleEn, Pattern: p.Pattern, HintVi: p.HintVi,
		Practice: np, Mastery: nm, Existing: existing,
	})
	if err != nil {
		if errors.Is(err, ai.ErrNotConfigured) || errors.Is(err, ai.ErrInvalidKey) || errors.Is(err, ai.ErrQuota) {
			return Lesson{}, 0, err
		}
		return Lesson{}, 0, fmt.Errorf("%w: %w", errAI, err)
	}
	// Only valid picture exercises with a new sentence are kept, as many as the part needs.
	fresh := func(in []ai.GrammarExercise, n int) []Exercise {
		var out []Exercise
		for _, e := range normalizeExercises(exercisesFromAI(in), "new") {
			key := strings.ToLower(e.Text)
			if len(out) == n || e.Kind != KindPicture || seen[key] || exerciseProblems(e) != nil {
				continue
			}
			seen[key] = true
			e.ID = "" // numbered after the lesson's own exercises
			out = append(out, e)
		}
		return out
	}
	newPractice, newMastery := fresh(raw.Practice, np), fresh(raw.Mastery, nm)
	if len(newPractice)+len(newMastery) == 0 {
		slog.WarnContext(ctx, "grammar pictures from the AI are unusable", slog.String("point", pointID))
		return Lesson{}, 0, ErrUnusable
	}

	content := cloneContent(l.Content)
	content.Practice = normalizeExercises(append(content.Practice, newPractice...), "p")
	content.Mastery = normalizeExercises(append(content.Mastery, newMastery...), "m")
	if fields := Validate(content); fields != nil {
		slog.WarnContext(ctx, "grammar lesson invalid after adding pictures", slog.String("point", pointID), slog.Any("problems", fields))
		return Lesson{}, 0, ErrUnusable
	}
	var fresher []Exercise
	fresher = append(fresher, content.Practice[len(content.Practice)-len(newPractice):]...)
	fresher = append(fresher, content.Mastery[len(content.Mastery)-len(newMastery):]...)

	l.Content = content
	if !l.CheckedAt.IsZero() {
		l.Checks = append(append([]Check(nil), l.Checks...), s.checkNew(ctx, p, fresher)...)
		if unconfirmed(l.Checks) > 0 {
			l.VerifiedAt = time.Time{}
		}
	}
	l, err = s.saveLesson(ctx, l)
	if err != nil {
		return Lesson{}, 0, err
	}
	return l, len(fresher), nil
}

// checkNew has the AI solve the given exercises and returns their flags. An exercise the AI did not
// answer, or every one when the AI fails, is flagged as unchecked: the admin looks at it.
func (s *Service) checkNew(ctx context.Context, p Point, exercises []Exercise) []Check {
	aiCtx, cancel := context.WithTimeout(ctx, checkTimeout)
	defer cancel()
	byID, err := s.solve(aiCtx, p, exercises)
	if err != nil {
		slog.WarnContext(ctx, "grammar check of new pictures failed", slog.String("point", p.ID), slog.Any("error", err))
	}
	var out []Check
	for _, e := range exercises {
		sol, ok := byID[e.ID]
		if !ok {
			out = append(out, Check{ExerciseID: e.ID, Kind: CheckUnchecked, NoteVi: missingNote})
			continue
		}
		if c, flagged := compareSolution(e, sol); flagged {
			out = append(out, c)
		}
	}
	return out
}

// --- picture exercises of a whole level, in the background ---

// ErrPictureBatchRunning means a batch adding picture exercises runs already; one runs at a time.
var ErrPictureBatchRunning = errors.New("grammar: a picture exercise batch is running")

// PictureBatchRequest adds the missing picture exercises to the lessons of the given points.
type PictureBatchRequest struct {
	PointIDs []string
	// Key and Label tell the pages what the batch is for ("level:A1", "các bài ngữ pháp A1").
	Key   string
	Label string
}

// PictureBatchItem is a lesson of a batch: how many exercises it got, or why it got none.
type PictureBatchItem struct {
	PointID string
	Title   string
	Added   int
	Err     error
}

// PictureBatchStatus is how a batch is going. The zero value means no batch ran since the server started.
type PictureBatchStatus struct {
	Running  bool
	Stopping bool
	Stopped  bool
	Key      string
	Label    string
	Total    int
	Current  *PictureBatchItem
	Saved    []PictureBatchItem
	Failures []PictureBatchItem
	StopErr  error
	// StartedAt and FinishedAt bound the batch.
	StartedAt  time.Time
	FinishedAt time.Time
}

// Done is how many lessons were dealt with.
func (b PictureBatchStatus) Done() int { return len(b.Saved) + len(b.Failures) }

type pictureRun struct {
	status PictureBatchStatus
	stop   context.CancelFunc
}

// StartPictureBatch picks the lessons that miss picture exercises and adds them one lesson after
// another in the background, inside this server process like the picture batches.
func (s *Service) StartPictureBatch(ctx context.Context, req PictureBatchRequest) (PictureBatchStatus, error) {
	var items []PictureBatchItem
	seen := map[string]bool{}
	for _, id := range req.PointIDs {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		p, err := s.point(id)
		if err != nil || !p.Drawable {
			continue
		}
		l, err := s.lessons.Get(ctx, id)
		if errors.Is(err, ErrNotFound) {
			continue
		}
		if err != nil {
			return PictureBatchStatus{}, fmt.Errorf("grammar: get lesson: %w", err)
		}
		if np, nm := picturesNeeded(l.Content); np+nm == 0 {
			continue
		}
		items = append(items, PictureBatchItem{PointID: id, Title: p.TitleVi})
	}
	if len(items) == 0 {
		return PictureBatchStatus{}, &ValidationError{Fields: map[string]string{"pointIds": "Mọi bài vẽ được đã có đủ bài nhìn ảnh"}}
	}

	s.picMu.Lock()
	defer s.picMu.Unlock()
	if s.picRun != nil && s.picRun.status.Running {
		return PictureBatchStatus{}, ErrPictureBatchRunning
	}
	base := context.WithoutCancel(ctx)
	feed, stop := context.WithCancel(base)
	run := &pictureRun{stop: stop, status: PictureBatchStatus{
		Running: true, Key: strings.TrimSpace(req.Key), Label: strings.TrimSpace(req.Label),
		Total: len(items), StartedAt: s.now(),
	}}
	s.picRun = run
	slog.InfoContext(ctx, "grammar: picture exercise batch started", slog.String("label", run.status.Label), slog.Int("lessons", len(items)))
	go s.runPictureBatch(base, feed, run, items)
	return run.snapshot(), nil
}

func (s *Service) runPictureBatch(ctx, feed context.Context, run *pictureRun, items []PictureBatchItem) {
	for _, it := range items {
		if feed.Err() != nil {
			break
		}
		cur := it
		s.picMu.Lock()
		run.status.Current = &cur
		s.picMu.Unlock()

		_, added, err := s.AddPictures(ctx, it.PointID)

		s.picMu.Lock()
		st := &run.status
		st.Current = nil
		it.Added, it.Err = added, err
		switch {
		case err == nil:
			st.Saved = append(st.Saved, it)
		case errors.Is(err, ai.ErrNotConfigured), errors.Is(err, ai.ErrInvalidKey), errors.Is(err, ai.ErrQuota):
			st.Failures = append(st.Failures, it)
			if st.StopErr == nil {
				st.StopErr = err
			}
			run.stop()
		default:
			slog.WarnContext(ctx, "grammar: batch picture exercises failed", slog.String("point", it.PointID), slog.Any("error", err))
			st.Failures = append(st.Failures, it)
		}
		s.picMu.Unlock()
	}

	s.picMu.Lock()
	run.stop()
	st := &run.status
	st.Running, st.Stopped, st.Stopping = false, st.Stopping, false
	st.FinishedAt = s.now()
	s.picMu.Unlock()
	slog.InfoContext(ctx, "grammar: picture exercise batch finished", slog.Int("saved", len(st.Saved)),
		slog.Int("failed", len(st.Failures)), slog.Int("total", st.Total), slog.Bool("stopped", st.Stopped))
}

// StopPictureBatch asks the running batch to stop: the lesson being done finishes, no new one starts.
func (s *Service) StopPictureBatch() PictureBatchStatus {
	s.picMu.Lock()
	defer s.picMu.Unlock()
	if s.picRun == nil {
		return PictureBatchStatus{}
	}
	if s.picRun.status.Running {
		s.picRun.status.Stopping = true
		s.picRun.stop()
	}
	return s.picRun.snapshot()
}

// PictureBatch returns how the running, or last, batch is going.
func (s *Service) PictureBatch() PictureBatchStatus {
	s.picMu.Lock()
	defer s.picMu.Unlock()
	if s.picRun == nil {
		return PictureBatchStatus{}
	}
	return s.picRun.snapshot()
}

// snapshot copies the status; the caller holds picMu.
func (r *pictureRun) snapshot() PictureBatchStatus {
	st := r.status
	if st.Current != nil {
		c := *st.Current
		st.Current = &c
	}
	st.Saved = append([]PictureBatchItem(nil), st.Saved...)
	st.Failures = append([]PictureBatchItem(nil), st.Failures...)
	return st
}
