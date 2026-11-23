package wordbank

import (
	"errors"
	"slices"
	"testing"
	"time"
)

func topicPics(t *testing.T, e *testEnv, id string) TopicPictures {
	t.Helper()
	list, err := e.svc.TopicsPictures(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range list {
		if p.TopicID == id {
			return p
		}
	}
	return TopicPictures{}
}

func TestRedrawRoundsOfATopic(t *testing.T) {
	t.Parallel()
	e := newEnv()
	addWords(t, e, "house", "tablix", "good morning", "apple")
	for _, l := range []string{"house", "tablix", "good morning"} {
		if _, err := e.svc.GenerateImage(t.Context(), l, "", ""); err != nil {
			t.Fatal(err)
		}
	}
	// t1 holds house and tablix, both drawn already; apple is in no topic.
	if p := topicPics(t, e, "t1"); p.Words != 2 || p.Missing != 0 || p.Rounds != 0 || p.Fresh != 0 {
		t.Fatalf("before = %+v", p)
	}
	var verr *ValidationError
	if _, err := e.svc.StartBatch(t.Context(), BatchRequest{TopicIDs: []string{"t1"}}); !errors.As(err, &verr) {
		t.Fatalf("missing mode with nothing missing: %v", err)
	}

	// "Vẽ lại hết": every word again, and round 1 starts.
	e.now = e.now.Add(time.Hour)
	gate := newGateAI()
	e.svc.imageAI = gate
	st, err := e.svc.StartBatch(t.Context(), BatchRequest{TopicIDs: []string{"t1"}, Mode: BatchRedraw})
	if err != nil {
		t.Fatal(err)
	}
	if st.Total != 2 || st.Mode != BatchRedraw || st.TopicName != "Nhà cửa" {
		t.Fatalf("started = %+v", st)
	}
	// Stopped after the first word: the round is half done.
	if first := <-gate.started; first != "house" {
		t.Fatalf("first = %q", first)
	}
	e.svc.StopBatch()
	gate.release <- struct{}{}
	waitBatch(t, e.svc)
	p := topicPics(t, e, "t1")
	if p.Rounds != 1 || !p.LastAt.Equal(e.now) || p.Fresh != 1 || p.Words != 2 {
		t.Fatalf("half done = %+v", p)
	}
	// Chào hỏi holds house too: its picture is new, but Chào hỏi had no round.
	if p := topicPics(t, e, "t2"); p.Rounds != 0 || p.Fresh != 0 {
		t.Errorf("other topic = %+v", p)
	}

	// "Vẽ tiếp" draws only the word the round has not reached.
	e.now = e.now.Add(time.Hour)
	e.svc.imageAI = e.draw
	if got := mustWords(t, e, BatchRequest{TopicIDs: []string{"t1"}, Mode: BatchContinue}); !slices.Equal(got, []string{"tablix"}) {
		t.Fatalf("continue words = %v", got)
	}
	if _, err := e.svc.StartBatch(t.Context(), BatchRequest{TopicIDs: []string{"t1"}, Mode: BatchContinue}); err != nil {
		t.Fatal(err)
	}
	waitBatch(t, e.svc)
	if p := topicPics(t, e, "t1"); p.Rounds != 1 || p.Fresh != 2 {
		t.Fatalf("done = %+v", p)
	}
	if _, err := e.svc.StartBatch(t.Context(), BatchRequest{TopicIDs: []string{"t1"}, Mode: BatchContinue}); !errors.As(err, &verr) {
		t.Errorf("nothing left to continue: %v", err)
	}

	// A second "Vẽ lại hết" counts round 2; no word is fresh until drawn again.
	e.now = e.now.Add(time.Hour)
	gate = newGateAI()
	e.svc.imageAI = gate
	if _, err := e.svc.StartBatch(t.Context(), BatchRequest{TopicIDs: []string{"t1"}, Mode: BatchRedraw}); err != nil {
		t.Fatal(err)
	}
	if p := topicPics(t, e, "t1"); p.Rounds != 2 || p.Fresh != 0 {
		t.Errorf("round 2 = %+v", p)
	}
	<-gate.started
	e.svc.StopBatch()
	gate.release <- struct{}{}
	waitBatch(t, e.svc)
}

func TestRedrawNeedsTopicsAndAKnownMode(t *testing.T) {
	t.Parallel()
	e := newEnv()
	addWords(t, e, "house")
	var verr *ValidationError
	for _, req := range []BatchRequest{
		{Mode: BatchRedraw},
		{Mode: BatchRedraw, Lemmas: []string{"house"}, TopicIDs: []string{"t1"}},
		{Mode: "other", TopicIDs: []string{"t1"}},
		{Mode: BatchContinue, TopicIDs: []string{"nope"}},
	} {
		if _, err := e.svc.batchWords(t.Context(), req); !errors.As(err, &verr) {
			t.Errorf("%+v: %v", req, err)
		}
	}
	if r, _ := e.repo.TopicRedraws(t.Context()); len(r) != 0 {
		t.Errorf("a refused batch counted a round: %v", r)
	}
}
