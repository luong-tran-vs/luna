package grammar

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/png"
	"sync"
	"testing"
	"time"

	"github.com/luongtran/luna/backend/internal/ai"
)

type fakeGrammarImages struct {
	mu     sync.Mutex
	images map[string]ExampleImage
}

func (f *fakeGrammarImages) Keys(_ context.Context, pointID string) ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []string
	for _, img := range f.images {
		if img.PointID == pointID {
			out = append(out, img.Key)
		}
	}
	return out, nil
}

func (f *fakeGrammarImages) Get(_ context.Context, pointID, key string) (ExampleImage, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	img, ok := f.images[pointID+"/"+key]
	if !ok {
		return ExampleImage{}, ErrImageNotFound
	}
	return img, nil
}

func (f *fakeGrammarImages) Save(_ context.Context, img ExampleImage) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.images[img.PointID+"/"+img.Key] = img
	return nil
}

func (f *fakeGrammarImages) Delete(_ context.Context, pointID, key string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.images, pointID+"/"+key)
	return nil
}

func (f *fakeGrammarImages) DeleteForPoint(_ context.Context, pointID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for k, img := range f.images {
		if img.PointID == pointID {
			delete(f.images, k)
		}
	}
	return nil
}

// fakeDrawer draws a 800×600 PNG, or fails with failErr.
type fakeDrawer struct {
	mu       sync.Mutex
	requests []ai.ImageRequest
	failErr  error
}

func (f *fakeDrawer) GenerateImage(_ context.Context, req ai.ImageRequest) (ai.Image, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.requests = append(f.requests, req)
	if f.failErr != nil {
		return ai.Image{}, f.failErr
	}
	img := image.NewNRGBA(image.Rect(0, 0, 800, 600))
	img.Set(1, 1, color.NRGBA{R: 200, A: 255})
	var b bytes.Buffer
	_ = png.Encode(&b, img)
	return ai.Image{MIME: "image/png", Data: b.Bytes()}, nil
}

func (f *fakeDrawer) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.requests)
}

// imageEnv is an env whose a1-to-be lesson has three examples, with pictures drawn by draw.
func newImageEnv(t *testing.T) (*env, *fakeGrammarImages, *fakeDrawer) {
	t.Helper()
	e := newEnv(t)
	if _, _, err := e.svc.Generate(t.Context(), "a1-to-be", false); err != nil {
		t.Fatal(err)
	}
	store := &fakeGrammarImages{images: map[string]ExampleImage{}}
	draw := &fakeDrawer{}
	e.svc.WithImages(ImageDeps{Store: store, Gemini: draw})
	return e, store, draw
}

func waitImageBatch(t *testing.T, svc *Service) ImageBatchStatus {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if st := svc.ImageBatch(); !st.Running {
			return st
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("image batch still running")
	return ImageBatchStatus{}
}

func TestExampleKeyIgnoresCaseAndSpacing(t *testing.T) {
	t.Parallel()
	if ExampleKey("She is  a nurse.") != ExampleKey(" she is a NURSE. ") || ExampleKey("She is a nurse.") == ExampleKey("He is a nurse.") {
		t.Fatal("keys")
	}
	if len(ExampleKey("x")) != 16 {
		t.Fatalf("key %q", ExampleKey("x"))
	}
}

func TestImageBatchDrawsTheMissingExamples(t *testing.T) {
	t.Parallel()
	e, store, draw := newImageEnv(t)
	// The second example already has its picture.
	_ = store.Save(t.Context(), ExampleImage{PointID: "a1-to-be", Key: ExampleKey("She is a nurse."), MIME: "image/jpeg", Data: []byte("x")})

	st, err := e.svc.StartImageBatch(t.Context(), ImageBatchRequest{PointIDs: []string{"a1-to-be", "nope"}, Key: "point:a1-to-be", Label: "bài to be"})
	if err != nil {
		t.Fatal(err)
	}
	if !st.Running || st.Total != 2 || st.Points != 1 || st.Engine != EngineGemini {
		t.Fatalf("started = %+v", st)
	}
	st = waitImageBatch(t, e.svc)
	if len(st.Saved) != 2 || st.Saved[0].En != "I am a student." || st.Saved[1].En != "They are friends." {
		t.Fatalf("finished = %+v", st)
	}
	req := draw.requests[0]
	if req.Example == nil || req.Example.En != "I am a student." || req.Example.Vi != "Tôi là học sinh." || req.Example.Point == "" {
		t.Fatalf("request = %+v", req.Example)
	}

	l, _ := e.svc.AdminGet(t.Context(), "a1-to-be")
	keys, err := e.svc.ExampleImageKeys(t.Context(), "a1-to-be", l.Content)
	if err != nil || keys[0] == "" || keys[1] == "" || keys[2] == "" {
		t.Fatalf("keys = %q, %v", keys, err)
	}

	// Nothing left: a new batch is refused; with replace, every example is drawn again.
	var verr *ValidationError
	if _, err := e.svc.StartImageBatch(t.Context(), ImageBatchRequest{PointIDs: []string{"a1-to-be"}}); !errors.As(err, &verr) {
		t.Fatalf("nothing to draw: %v", err)
	}
	if _, err := e.svc.StartImageBatch(t.Context(), ImageBatchRequest{PointIDs: []string{"a1-to-be"}, Keys: []string{keys[2]}, Replace: true}); err != nil {
		t.Fatal(err)
	}
	if st := waitImageBatch(t, e.svc); st.Total != 1 || len(st.Saved) != 1 || st.Saved[0].Key != keys[2] {
		t.Fatalf("redraw one = %+v", st)
	}
	if draw.count() != 3 {
		t.Fatalf("%d drawings", draw.count())
	}
}

func TestImageBatchStopsOnQuotaAndRefusesCodexOff(t *testing.T) {
	t.Parallel()
	e, _, draw := newImageEnv(t)
	if _, err := e.svc.StartImageBatch(t.Context(), ImageBatchRequest{PointIDs: []string{"a1-to-be"}, Engine: EngineCodex}); !errors.Is(err, ErrCodexUnavailable) {
		t.Fatalf("codex: %v", err)
	}
	draw.failErr = ai.ErrQuota
	if _, err := e.svc.StartImageBatch(t.Context(), ImageBatchRequest{PointIDs: []string{"a1-to-be"}}); err != nil {
		t.Fatal(err)
	}
	st := waitImageBatch(t, e.svc)
	if !errors.Is(st.StopErr, ai.ErrQuota) || len(st.Failures) != 1 || draw.count() != 1 {
		t.Fatalf("status = %+v", st)
	}
}

func TestExampleImagesOnlyForPublishedLessonsAndGoWithTheLesson(t *testing.T) {
	t.Parallel()
	e, store, _ := newImageEnv(t)
	key := ExampleKey("I am a student.")
	_ = store.Save(t.Context(), ExampleImage{PointID: "a1-to-be", Key: key, MIME: "image/jpeg", Data: []byte("x")})
	if _, err := e.svc.Image(t.Context(), "a1-to-be", key, false); !errors.Is(err, ErrImageNotFound) {
		t.Fatalf("draft lesson to a learner: %v", err)
	}
	if img, err := e.svc.Image(t.Context(), "a1-to-be", key, true); err != nil || string(img.Data) != "x" {
		t.Fatalf("admin: %v", err)
	}
	if err := e.svc.Delete(t.Context(), "a1-to-be"); err != nil {
		t.Fatal(err)
	}
	if keys, _ := store.Keys(t.Context(), "a1-to-be"); len(keys) != 0 {
		t.Fatalf("pictures kept: %v", keys)
	}
}
