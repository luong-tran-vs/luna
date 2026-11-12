package lesson

import (
	"errors"
	"testing"
)

func TestRemoveScenesTakesTheScenesOfTheLessonsAway(t *testing.T) {
	t.Parallel()
	e := newImageEnv(t)
	a, b, bare := e.annotated(t), e.annotated(t), e.annotated(t)
	for _, l := range []Lesson{a, b} {
		if err := e.svc.UploadImage(t.Context(), l.ID, SceneKey, testPNG(80, 45)); err != nil {
			t.Fatal(err)
		}
	}
	n, err := e.svc.RemoveScenes(t.Context(), []string{a.ID, bare.ID, a.ID, b.ID})
	if err != nil || n != 2 {
		t.Fatalf("removed %d, %v", n, err)
	}
	scenes, _ := e.svc.HasScenes(t.Context(), []string{a.ID, b.ID})
	if scenes[a.ID] || scenes[b.ID] {
		t.Fatalf("scenes left = %v", scenes)
	}
	// Nothing left: removing again is no error.
	if n, err := e.svc.RemoveScenes(t.Context(), []string{a.ID}); err != nil || n != 0 {
		t.Fatalf("again: %d, %v", n, err)
	}
	var verr *ValidationError
	if _, err := e.svc.RemoveScenes(t.Context(), nil); !errors.As(err, &verr) {
		t.Fatalf("no lessons: %v", err)
	}
}

func TestRemoveScenesWaitsForTheBatch(t *testing.T) {
	t.Parallel()
	e := newImageEnv(t)
	slow := &blockingImageAI{release: make(chan struct{})}
	e.svc.ImageAI = slow
	a := e.annotated(t)
	if _, err := e.svc.StartSceneBatch(t.Context(), SceneBatchRequest{LessonIDs: []string{a.ID}}); err != nil {
		t.Fatal(err)
	}
	if _, err := e.svc.RemoveScenes(t.Context(), []string{a.ID}); !errors.Is(err, ErrSceneBatchRunning) {
		t.Fatalf("during a batch: %v", err)
	}
	close(slow.release)
	waitSceneBatch(t, e.svc)
}
