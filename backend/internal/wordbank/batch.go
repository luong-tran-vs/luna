package wordbank

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

// Picture batches (added 2026-11-04): the admin picks words, or every word lacking a picture, and
// the server draws them one after another in the background, saving each picture as soon as it
// is drawn. A batch runs inside this server process, not through the job queue: the local server
// and the production one share the queue, and only the local one runs Codex. A batch cut short
// (the server stopped) is simply started again: the words that got a picture are no longer missing.

// MaxBatch is the most words one batch draws: the bound shared by the picture batches.
const MaxBatch = ai.MaxImageBatch

// batchDrawTimeout bounds one picture of a batch.
const batchDrawTimeout = 5 * time.Minute

// ErrBatchRunning means a picture batch is running already; one runs at a time.
var ErrBatchRunning = errors.New("wordbank: a picture batch is running")

// BatchRequest starts a batch: the ticked Lemmas, or, without any, the words lacking a picture
// that match Search and TopicID (the list's filter), the first MaxBatch by lemma.
type BatchRequest struct {
	Lemmas  []string
	Search  string
	TopicID string
	Engine  Engine
	Style   string
}

// BatchFailure is a word of the batch that got no picture.
type BatchFailure struct {
	Lemma string
	Err   error
}

// BatchStatus is how a batch is going. The zero value means no batch ran since the server started.
type BatchStatus struct {
	Running bool
	// Stopping is true once the admin asked to stop: the words being drawn finish, no new one starts.
	Stopping bool
	// Stopped is true when the batch ended because the admin stopped it.
	Stopped bool
	Engine  Engine
	// Ticked is true for the words the admin ticked, false for the words lacking a picture that
	// matched the filter (Search, and the topic).
	Ticked bool
	// TopicName is the topic every word of the batch comes from (the filter's), "" when they
	// come from several: Topics then names the fewest topics holding them all.
	TopicName string
	Search    string
	Topics    []string
	Total     int
	// Drawing are the words being drawn now.
	Drawing []string
	// Saved are the words that got their picture, in the order they were saved.
	Saved    []Word
	Failures []BatchFailure
	// StopErr is why the batch ended early by itself: an error the next words would get too (no
	// quota, AI or Codex not set up, no image tool); nil otherwise.
	StopErr    error
	StartedAt  time.Time
	FinishedAt time.Time
}

// Done is how many words were dealt with, with or without a picture.
func (b BatchStatus) Done() int { return len(b.Saved) + len(b.Failures) }

type batchRun struct {
	status BatchStatus
	// stop ends the handing out of words.
	stop context.CancelFunc
}

// StartBatch checks the request, picks the words and starts drawing them in the background.
func (s *Service) StartBatch(ctx context.Context, req BatchRequest) (BatchStatus, error) {
	style, err := checkStyle(req.Style)
	if err != nil {
		return BatchStatus{}, err
	}
	drawer, workers, err := s.imageDrawer(req.Engine)
	if err != nil {
		return BatchStatus{}, err
	}
	lemmas, err := s.batchWords(ctx, req)
	if err != nil {
		return BatchStatus{}, err
	}
	scope := s.batchScope(ctx, req, lemmas)
	if len(lemmas) == 0 {
		return BatchStatus{}, &ValidationError{Fields: map[string]string{"lemmas": "Không có từ nào cần sinh ảnh"}}
	}
	engine := req.Engine
	if engine == "" {
		engine = EngineGemini
	}

	s.batchMu.Lock()
	defer s.batchMu.Unlock()
	if s.batch != nil && s.batch.status.Running {
		return BatchStatus{}, ErrBatchRunning
	}
	// The batch outlives the request that started it.
	base := context.WithoutCancel(ctx)
	feed, stop := context.WithCancel(base)
	st := BatchStatus{Running: true, Engine: engine, Total: len(lemmas), StartedAt: s.now(), Ticked: len(req.Lemmas) > 0, Topics: scope.topics}
	st.TopicName = scope.topicName
	if !st.Ticked {
		st.Search = strings.Join(strings.Fields(req.Search), " ")
	}
	b := &batchRun{stop: stop, status: st}
	s.batch = b
	s.log.InfoContext(ctx, "wordbank: picture batch started", slog.String("engine", string(engine)),
		slog.Int("words", len(lemmas)), slog.Int("at_once", workers))
	go s.runBatch(base, feed, b, drawer, engine, style, lemmas, workers)
	return b.snapshot(), nil
}

// batchWords lists the words a batch draws, in order, without repeats.
func (s *Service) batchWords(ctx context.Context, req BatchRequest) ([]string, error) {
	if len(req.Lemmas) > 0 {
		seen := map[string]bool{}
		var out []string
		for _, l := range req.Lemmas {
			if k := Normalize(l); k != "" && !seen[k] {
				seen[k] = true
				out = append(out, k)
			}
		}
		if len(out) > MaxBatch {
			return nil, &ValidationError{Fields: map[string]string{"lemmas": fmt.Sprintf("Mỗi lượt sinh tối đa %d ảnh", MaxBatch)}}
		}
		return out, nil
	}
	search := strings.Join(strings.Fields(req.Search), " ")
	if utf8.RuneCountInString(search) > MaxSearch {
		return nil, &ValidationError{Fields: map[string]string{"q": fmt.Sprintf("Từ khoá tối đa %d ký tự", MaxSearch)}}
	}
	q := ListQuery{Search: search, Missing: MissingImage, Limit: MaxBatch}
	if id := strings.TrimSpace(req.TopicID); id != "" {
		topics, err := s.topics.All(ctx)
		if err != nil {
			return nil, fmt.Errorf("wordbank: topic words: %w", err)
		}
		t, ok := findTopic(topics, id)
		if !ok {
			return nil, &ValidationError{Fields: map[string]string{"topicId": msgNoTopic}}
		}
		q.Lemmas = lemmasOf(t)
	}
	words, _, err := s.repo.List(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("wordbank: list: %w", err)
	}
	out := make([]string, len(words))
	for i, w := range words {
		out[i] = w.Lemma
	}
	return out, nil
}

// runBatch hands the words out in order to the workers until they run out or feed ends.
func (s *Service) runBatch(ctx, feed context.Context, b *batchRun, drawer ai.ImageProvider, engine Engine, style string, lemmas []string, workers int) {
	next := make(chan string)
	go func() {
		defer close(next)
		for _, l := range lemmas {
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
				s.drawInBatch(ctx, b, drawer, engine, style, l)
			}
		})
	}
	wg.Wait()

	s.batchMu.Lock()
	b.stop()
	st := &b.status
	st.Running, st.Stopped, st.Stopping = false, st.Stopping, false
	st.FinishedAt = s.now()
	s.batchMu.Unlock()
	s.log.InfoContext(ctx, "wordbank: picture batch finished", slog.String("engine", string(engine)),
		slog.Int("saved", len(st.Saved)), slog.Int("failed", len(st.Failures)), slog.Int("total", st.Total),
		slog.Bool("stopped", st.Stopped), slog.Any("stop_error", st.StopErr))
}

func (s *Service) drawInBatch(ctx context.Context, b *batchRun, drawer ai.ImageProvider, engine Engine, style, lemma string) {
	s.batchMu.Lock()
	b.status.Drawing = append(b.status.Drawing, lemma)
	s.batchMu.Unlock()

	dctx, cancel := context.WithTimeout(ctx, batchDrawTimeout)
	w, err := s.draw(dctx, drawer, engine, lemma, style)
	cancel()

	s.batchMu.Lock()
	defer s.batchMu.Unlock()
	st := &b.status
	st.Drawing = slices.DeleteFunc(st.Drawing, func(l string) bool { return l == lemma })
	switch {
	case err == nil:
		st.Saved = append(st.Saved, w)
	case endsBatch(err):
		// The other words would fail alike: stop handing them out.
		if st.StopErr == nil {
			st.StopErr = err
		}
		st.Failures = append(st.Failures, BatchFailure{Lemma: lemma, Err: err})
		b.stop()
	default:
		s.log.WarnContext(ctx, "wordbank: batch picture failed", slog.String("lemma", lemma), slog.Any("error", err))
		st.Failures = append(st.Failures, BatchFailure{Lemma: lemma, Err: err})
	}
}

// endsBatch tells the errors every next word of the batch would get too.
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

// StopBatch asks the running batch to stop: the words being drawn finish, no new one starts.
func (s *Service) StopBatch() BatchStatus {
	s.batchMu.Lock()
	defer s.batchMu.Unlock()
	if s.batch == nil {
		return BatchStatus{}
	}
	if s.batch.status.Running {
		s.batch.status.Stopping = true
		s.batch.stop()
	}
	return s.batch.snapshot()
}

// Batch returns how the running, or last, batch is going.
func (s *Service) Batch() BatchStatus {
	s.batchMu.Lock()
	defer s.batchMu.Unlock()
	if s.batch == nil {
		return BatchStatus{}
	}
	return s.batch.snapshot()
}

// snapshot copies the status; the caller holds batchMu.
func (b *batchRun) snapshot() BatchStatus {
	st := b.status
	st.Drawing = slices.Clone(st.Drawing)
	st.Saved = slices.Clone(st.Saved)
	st.Failures = slices.Clone(st.Failures)
	return st
}

type batchScope struct {
	topicName string
	topics    []string
}

// batchScope tells the admin what the batch draws: the filter's topic when every word is in it
// (always so for the words lacking a picture of a topic; for ticked words when the admin ticked
// them in that topic's list), otherwise the fewest topics that hold all the words, the most
// covering first. A word is often in several topics ("kind" in Tính cách and others), so naming
// every topic holding a word would name far more topics than the admin chose from. Without the
// topics the batch still runs; it only names less.
func (s *Service) batchScope(ctx context.Context, req BatchRequest, lemmas []string) batchScope {
	topics, err := s.topics.All(ctx)
	if err != nil {
		s.log.WarnContext(ctx, "wordbank: batch topics", slog.Any("error", err))
		return batchScope{}
	}
	chosen, hasChosen := findTopic(topics, strings.TrimSpace(req.TopicID))
	left := map[string]bool{}
	holders := topicsByLemma(topics)
	for _, l := range lemmas {
		if len(holders[l]) > 0 {
			left[l] = true
		}
	}
	var out batchScope
	if hasChosen {
		in := lemmasOf(chosen)
		if !slices.ContainsFunc(lemmas, func(l string) bool { return !slices.Contains(in, l) }) {
			return batchScope{topicName: chosen.Name}
		}
		if takeTopic(left, in) > 0 {
			out.topics = append(out.topics, chosen.Name)
		}
	}
	// Greedy cover: the topic holding most of the words left, by name on a tie.
	for len(left) > 0 {
		best, bestN := -1, 0
		for i, t := range topics {
			n := countLeft(left, lemmasOf(t))
			if n > bestN || (n == bestN && n > 0 && t.Name < topics[best].Name) {
				best, bestN = i, n
			}
		}
		if best < 0 {
			break
		}
		takeTopic(left, lemmasOf(topics[best]))
		out.topics = append(out.topics, topics[best].Name)
	}
	return out
}

// countLeft counts the words of a topic still left, each once (a list may repeat a word).
func countLeft(left map[string]bool, words []string) int {
	seen := map[string]bool{}
	for _, l := range words {
		if left[l] {
			seen[l] = true
		}
	}
	return len(seen)
}

// takeTopic removes the topic's words from left and returns how many it removed.
func takeTopic(left map[string]bool, words []string) int {
	n := 0
	for _, l := range words {
		if left[l] {
			delete(left, l)
			n++
		}
	}
	return n
}
