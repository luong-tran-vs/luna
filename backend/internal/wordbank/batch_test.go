package wordbank

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/luongtran/luna/backend/internal/ai"
)

// gateAI draws only when released, telling which word it started.
type gateAI struct {
	started chan string
	release chan struct{}
}

func newGateAI() *gateAI {
	return &gateAI{started: make(chan string, 10), release: make(chan struct{})}
}

func (g *gateAI) GenerateImage(ctx context.Context, req ai.ImageRequest) (ai.Image, error) {
	g.started <- req.Word
	select {
	case <-g.release:
	case <-ctx.Done():
		return ai.Image{}, ctx.Err()
	}
	return ai.Image{MIME: "image/png", Data: testPNG(40, 30)}, nil
}

func addWords(t *testing.T, e *testEnv, lemmas ...string) {
	t.Helper()
	for _, l := range lemmas {
		if _, _, err := e.svc.Add(t.Context(), Input{Lemma: l}); err != nil {
			t.Fatal(err)
		}
	}
}

// waitBatch waits for the batch to end.
func waitBatch(t *testing.T, s *Service) BatchStatus {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if st := s.Batch(); !st.Running {
			return st
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("batch still running")
	return BatchStatus{}
}

func savedLemmas(st BatchStatus) []string {
	out := make([]string, len(st.Saved))
	for i, w := range st.Saved {
		out[i] = w.Lemma
	}
	return out
}

func TestBatchDrawsTheFilteredWordsLackingAPicture(t *testing.T) {
	t.Parallel()
	e := newEnv()
	addWords(t, e, "house", "tablix", "good morning", "apple")
	if _, err := e.svc.GenerateImage(t.Context(), "tablix", "", ""); err != nil {
		t.Fatal(err)
	}
	e.draw.reqs = nil

	// Topic t1 holds house and tablix; tablix has its picture already.
	st, err := e.svc.StartBatch(t.Context(), BatchRequest{TopicID: "t1", Style: "watercolor"})
	if err != nil {
		t.Fatal(err)
	}
	if !st.Running || st.Total != 1 || st.Engine != EngineGemini || st.Ticked || st.TopicName != "Nhà cửa" || st.Topics != nil {
		t.Fatalf("started = %+v", st)
	}
	st = waitBatch(t, e.svc)
	if got := savedLemmas(st); !slices.Equal(got, []string{"house"}) || len(st.Failures) != 0 || st.Done() != 1 {
		t.Fatalf("finished = %+v", st)
	}
	if !st.Saved[0].HasImage() || st.FinishedAt.IsZero() {
		t.Errorf("saved word = %+v", st.Saved[0])
	}
	if len(e.draw.reqs) != 1 || e.draw.reqs[0].Style != "watercolor" {
		t.Errorf("requests = %+v", e.draw.reqs)
	}

	// Without a filter: every word still lacking one, by lemma.
	if _, err := e.svc.StartBatch(t.Context(), BatchRequest{}); err != nil {
		t.Fatal(err)
	}
	if got := savedLemmas(waitBatch(t, e.svc)); !slices.Equal(got, []string{"apple", "good morning"}) {
		t.Errorf("saved = %v", got)
	}
	var verr *ValidationError
	if _, err := e.svc.StartBatch(t.Context(), BatchRequest{}); !errors.As(err, &verr) || verr.Fields["lemmas"] == "" {
		t.Errorf("nothing left to draw: %v", err)
	}
}

func TestBatchOfTickedWordsRunsOneAtATimeAndStops(t *testing.T) {
	t.Parallel()
	e := newEnv()
	addWords(t, e, "house", "apple", "tablix")
	gate := newGateAI()
	e.svc.imageAI = gate

	if _, err := e.svc.StartBatch(t.Context(), BatchRequest{Lemmas: []string{"Tablix", "house", "tablix", "apple"}}); err != nil {
		t.Fatal(err)
	}
	if st := e.svc.Batch(); !st.Ticked || st.TopicName != "" || !slices.Equal(st.Topics, []string{"Nhà cửa"}) {
		t.Fatalf("ticked scope = %+v", st)
	}
	if first := <-gate.started; first != "tablix" {
		t.Fatalf("first word = %q, want the ticked order", first)
	}
	if _, err := e.svc.StartBatch(t.Context(), BatchRequest{Lemmas: []string{"apple"}}); !errors.Is(err, ErrBatchRunning) {
		t.Fatalf("second batch: %v", err)
	}
	if st := e.svc.Batch(); st.Total != 3 || !slices.Equal(st.Drawing, []string{"tablix"}) {
		t.Fatalf("running = %+v", st)
	}
	if st := e.svc.StopBatch(); !st.Stopping {
		t.Fatalf("stop = %+v", st)
	}
	gate.release <- struct{}{} // the word being drawn finishes
	st := waitBatch(t, e.svc)
	if got := savedLemmas(st); !slices.Equal(got, []string{"tablix"}) || !st.Stopped || st.Stopping {
		t.Fatalf("stopped = %+v", st)
	}
	select {
	case w := <-gate.started:
		t.Errorf("%q started after stopping", w)
	default:
	}
}

func TestBatchStopsAtAnErrorEveryWordWouldGet(t *testing.T) {
	t.Parallel()
	e := newEnv()
	addWords(t, e, "apple", "house", "tablix")
	e.draw.err = ai.ErrQuota
	if _, err := e.svc.StartBatch(t.Context(), BatchRequest{}); err != nil {
		t.Fatal(err)
	}
	st := waitBatch(t, e.svc)
	if !errors.Is(st.StopErr, ai.ErrQuota) || len(st.Failures) != 1 || st.Failures[0].Lemma != "apple" || st.Stopped {
		t.Fatalf("finished = %+v", st)
	}

	// Other failures skip the word.
	e.draw.err = errFake
	if _, err := e.svc.StartBatch(t.Context(), BatchRequest{}); err != nil {
		t.Fatal(err)
	}
	if st := waitBatch(t, e.svc); len(st.Failures) != 3 || st.StopErr != nil || !errors.Is(st.Failures[2].Err, ErrDrawFailed) {
		t.Fatalf("skipping = %+v", st)
	}
}

func TestBatchWithCodexDrawsSeveralAtOnce(t *testing.T) {
	t.Parallel()
	e := newEnv()
	addWords(t, e, "apple", "house", "tablix")
	gate := newGateAI()
	e.svc.codexAI, e.svc.codexConcurrency = gate, 2
	if _, err := e.svc.StartBatch(t.Context(), BatchRequest{Engine: EngineCodex}); err != nil {
		t.Fatal(err)
	}
	got := []string{<-gate.started, <-gate.started}
	slices.Sort(got)
	if !slices.Equal(got, []string{"apple", "house"}) {
		t.Fatalf("drawing at once = %v", got)
	}
	for range 3 {
		gate.release <- struct{}{}
	}
	if st := waitBatch(t, e.svc); len(st.Saved) != 3 || st.Engine != EngineCodex {
		t.Fatalf("finished = %+v", st)
	}
}

func TestBatchRejects(t *testing.T) {
	t.Parallel()
	e := newEnv()
	many := make([]string, MaxBatch+1)
	for i := range many {
		many[i] = fmt.Sprintf("word%d", i)
	}
	var verr *ValidationError
	if _, err := e.svc.StartBatch(t.Context(), BatchRequest{Lemmas: many}); !errors.As(err, &verr) || verr.Fields["lemmas"] == "" {
		t.Errorf("too many: %v", err)
	}
	if _, err := e.svc.StartBatch(t.Context(), BatchRequest{Engine: EngineCodex}); !errors.Is(err, ErrCodexUnavailable) {
		t.Errorf("no codex: %v", err)
	}
	if _, err := e.svc.StartBatch(t.Context(), BatchRequest{TopicID: "nope"}); !errors.As(err, &verr) || verr.Fields["topicId"] == "" {
		t.Errorf("no topic: %v", err)
	}
	if st := e.svc.Batch(); st.Running || st.Total != 0 {
		t.Errorf("no batch ran: %+v", st)
	}
}

func TestBatchScopeNamesTheChosenTopicOrTheFewestTopics(t *testing.T) {
	t.Parallel()
	e := newEnv()
	e.topics.lists = append(e.topics.lists, TopicList{TopicRef: TopicRef{ID: "t3", Name: "Ăn uống"}, Words: []string{"apple", "house"}})
	addWords(t, e, "house", "tablix", "good morning", "apple")
	tests := []struct {
		name      string
		req       BatchRequest
		topicName string
		topics    []string
	}{
		// house is in all three topics: ticked in Nhà cửa's list, the batch is Nhà cửa's.
		{"ticked in the filter topic", BatchRequest{Lemmas: []string{"house", "tablix"}, TopicID: "t1"}, "Nhà cửa", nil},
		{"ticked across topics", BatchRequest{Lemmas: []string{"house", "tablix", "good morning"}, TopicID: "t1"}, "", []string{"Nhà cửa", "Chào hỏi"}},
		{"fewest topics, most covering first", BatchRequest{Lemmas: []string{"apple", "house", "good morning"}}, "", []string{"Chào hỏi", "Ăn uống"}},
		// Three topics hold two words each: Chào hỏi by name, then one word each for Nhà cửa (tablix) and Ăn uống (apple).
		{"every missing word", BatchRequest{}, "", []string{"Chào hỏi", "Nhà cửa", "Ăn uống"}},
	}
	for _, tt := range tests {
		sc := e.svc.batchScope(t.Context(), tt.req, mustWords(t, e, tt.req))
		if sc.topicName != tt.topicName || !slices.Equal(sc.topics, tt.topics) {
			t.Errorf("%s: scope = %+v, want %q %v", tt.name, sc, tt.topicName, tt.topics)
		}
	}
}

func mustWords(t *testing.T, e *testEnv, req BatchRequest) []string {
	t.Helper()
	lemmas, err := e.svc.batchWords(t.Context(), req)
	if err != nil {
		t.Fatal(err)
	}
	return lemmas
}
