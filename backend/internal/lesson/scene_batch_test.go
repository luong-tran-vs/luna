package lesson

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/luongtran/luna/backend/internal/ai"
)

// waitSceneBatch waits for the scene batch to end.
func waitSceneBatch(t *testing.T, svc *Service) SceneBatchStatus {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if st := svc.SceneBatch(); !st.Running {
			return st
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("scene batch still running")
	return SceneBatchStatus{}
}

func TestSceneBatchDrawsMissingScenesInOrder(t *testing.T) {
	t.Parallel()
	e := newImageEnv(t)
	first, second, pictured := e.annotated(t), e.annotated(t), e.annotated(t)
	if err := e.svc.UploadImage(t.Context(), pictured.ID, SceneKey, testPNG(80, 45)); err != nil {
		t.Fatal(err)
	}
	fresh := e.create(t) // not annotated: skipped

	st, err := e.svc.StartSceneBatch(t.Context(), SceneBatchRequest{
		LessonIDs: []string{first.ID, pictured.ID, fresh.ID, second.ID, first.ID}, Key: "A1-3", Label: "Gia đình",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !st.Running || st.Total != 2 || st.Engine != EngineGemini || st.Label != "Gia đình" || len(st.Skipped) != 1 || st.Skipped[0].ID != fresh.ID {
		t.Fatalf("started = %+v", st)
	}
	st = waitSceneBatch(t, e.svc)
	if len(st.Saved) != 2 || st.Saved[0].ID != first.ID || st.Saved[1].ID != second.ID || len(st.Failures) != 0 || st.FinishedAt.IsZero() {
		t.Fatalf("finished = %+v", st)
	}
	scenes, _ := e.svc.HasScenes(t.Context(), []string{first.ID, second.ID, fresh.ID})
	if !scenes[first.ID] || !scenes[second.ID] || scenes[fresh.ID] {
		t.Fatalf("scenes = %v", scenes)
	}
	if n := len(e.draw.requests); n != 2 {
		t.Fatalf("%d drawings, want 2 (the pictured lesson is skipped)", n)
	}

	// Run again: nothing is left to draw.
	var verr *ValidationError
	if _, err := e.svc.StartSceneBatch(t.Context(), SceneBatchRequest{LessonIDs: []string{first.ID, second.ID}}); !errors.As(err, &verr) {
		t.Fatalf("nothing to draw: %v", err)
	}
}

func TestSceneBatchStopsOnQuota(t *testing.T) {
	t.Parallel()
	e := newImageEnv(t)
	a, b := e.annotated(t), e.annotated(t)
	e.draw.failErr = ai.ErrQuota
	if _, err := e.svc.StartSceneBatch(t.Context(), SceneBatchRequest{LessonIDs: []string{a.ID, b.ID}}); err != nil {
		t.Fatal(err)
	}
	st := waitSceneBatch(t, e.svc)
	if !errors.Is(st.StopErr, ai.ErrQuota) || len(st.Failures) != 1 || len(e.draw.requests) != 1 {
		t.Fatalf("status = %+v, %d requests", st, len(e.draw.requests))
	}
}

// blockingImageAI draws once release is closed.
type blockingImageAI struct {
	fakeImageAI
	release chan struct{}
}

func (f *blockingImageAI) GenerateImage(ctx context.Context, req ai.ImageRequest) (ai.Image, error) {
	<-f.release
	return f.fakeImageAI.GenerateImage(ctx, req)
}

func TestSceneBatchOneAtATimeAndStop(t *testing.T) {
	t.Parallel()
	e := newImageEnv(t)
	a, b := e.annotated(t), e.annotated(t)
	slow := &blockingImageAI{release: make(chan struct{})}
	e.svc.ImageAI = slow
	if _, err := e.svc.StartSceneBatch(t.Context(), SceneBatchRequest{LessonIDs: []string{a.ID, b.ID}}); err != nil {
		t.Fatal(err)
	}
	if _, err := e.svc.StartSceneBatch(t.Context(), SceneBatchRequest{LessonIDs: []string{a.ID}}); !errors.Is(err, ErrSceneBatchRunning) {
		t.Fatalf("second batch: %v", err)
	}
	for len(e.svc.SceneBatch().Drawing) == 0 {
		time.Sleep(time.Millisecond)
	}
	// Gemini draws one scene at a time.
	if d := e.svc.SceneBatch().Drawing; len(d) != 1 || d[0].ID != a.ID {
		t.Fatalf("drawing = %+v", d)
	}
	if st := e.svc.StopSceneBatch(); !st.Stopping {
		t.Fatalf("stop = %+v", st)
	}
	close(slow.release)
	st := waitSceneBatch(t, e.svc)
	// The lesson being drawn finished; the next one never started.
	if !st.Stopped || len(st.Saved) != 1 || st.Saved[0].ID != a.ID {
		t.Fatalf("stopped = %+v", st)
	}
}

// gateImageAI draws once released, telling each drawing it starts.
type gateImageAI struct {
	fakeImageAI
	started chan struct{}
	release chan struct{}
}

func (g *gateImageAI) GenerateImage(ctx context.Context, req ai.ImageRequest) (ai.Image, error) {
	g.started <- struct{}{}
	select {
	case <-g.release:
	case <-ctx.Done():
		return ai.Image{}, ctx.Err()
	}
	return g.fakeImageAI.GenerateImage(ctx, req)
}

func TestSceneBatchWithCodexDrawsSeveralAtOnce(t *testing.T) {
	t.Parallel()
	e := newImageEnv(t)
	a, b, c := e.annotated(t), e.annotated(t), e.annotated(t)
	gate := &gateImageAI{started: make(chan struct{}, 10), release: make(chan struct{})}
	e.svc.CodexImageAI, e.svc.CodexConcurrency = gate, 2
	if _, err := e.svc.StartSceneBatch(t.Context(), SceneBatchRequest{LessonIDs: []string{a.ID, b.ID, c.ID}, Engine: EngineCodex}); err != nil {
		t.Fatal(err)
	}
	<-gate.started
	<-gate.started
	if d := e.svc.SceneBatch().Drawing; len(d) != 2 || d[0].ID == d[1].ID || d[0].ID == c.ID || d[1].ID == c.ID {
		t.Fatalf("drawing at once = %+v", d)
	}
	for range 3 {
		gate.release <- struct{}{}
	}
	st := waitSceneBatch(t, e.svc)
	if len(st.Saved) != 3 || len(st.Drawing) != 0 || st.Engine != EngineCodex {
		t.Fatalf("finished = %+v", st)
	}
}

func TestSceneBatchRefusals(t *testing.T) {
	t.Parallel()
	e := newImageEnv(t)
	a := e.annotated(t)
	if _, err := e.svc.StartSceneBatch(t.Context(), SceneBatchRequest{LessonIDs: []string{a.ID}, Engine: EngineCodex}); !errors.Is(err, ErrCodexUnavailable) {
		t.Fatalf("codex: %v", err)
	}
	var verr *ValidationError
	if _, err := e.svc.StartSceneBatch(t.Context(), SceneBatchRequest{LessonIDs: []string{e.create(t).ID}}); !errors.As(err, &verr) {
		t.Fatalf("only unannotated lessons: %v", err)
	}
	if st := e.svc.SceneBatch(); st.Running || st.Total != 0 {
		t.Fatalf("a batch ran: %+v", st)
	}
}
