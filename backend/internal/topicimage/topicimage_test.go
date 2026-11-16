package topicimage

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/png"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/luongtran/luna/backend/internal/ai"
	"github.com/luongtran/luna/backend/internal/platform/httpx"
)

type fakeStore struct {
	mu     sync.Mutex
	images map[string]Image
}

func (f *fakeStore) List(context.Context) ([]Info, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []Info{}
	for id, im := range f.images {
		out = append(out, Info{TopicID: id, CreatedAt: im.CreatedAt})
	}
	return out, nil
}

func (f *fakeStore) Get(_ context.Context, id string) (Image, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	im, ok := f.images[id]
	if !ok {
		return Image{}, ErrImageNotFound
	}
	return im, nil
}

func (f *fakeStore) Save(_ context.Context, img Image) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.images[img.TopicID] = img
	return nil
}

func (f *fakeStore) Delete(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.images, id)
	return nil
}

type fakeTopics map[string]Topic

func (f fakeTopics) Get(_ context.Context, id string) (Topic, error) {
	t, ok := f[id]
	if !ok {
		return Topic{}, ErrTopicNotFound
	}
	return t, nil
}

type fakeDrawer struct {
	mu   sync.Mutex
	reqs []ai.ImageRequest
	err  error
}

func (f *fakeDrawer) GenerateImage(_ context.Context, req ai.ImageRequest) (ai.Image, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reqs = append(f.reqs, req)
	if f.err != nil {
		return ai.Image{}, f.err
	}
	return ai.Image{MIME: "image/png", Data: pngBytes()}, nil
}

func pngBytes() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 16, 9))
	img.Set(1, 1, color.RGBA{R: 200, A: 255})
	var b bytes.Buffer
	_ = png.Encode(&b, img)
	return b.Bytes()
}

func newTestService(d *fakeDrawer) (*Service, *fakeStore) {
	store := &fakeStore{images: map[string]Image{}}
	s := NewService(Deps{
		Store: store,
		Topics: fakeTopics{
			"t1": {ID: "t1", Name: "Gia đình", Words: []string{"mother", "father"}},
			"t2": {ID: "t2", Name: "Đồ ăn"},
		},
		Gemini: d,
		Log:    slog.New(slog.DiscardHandler),
		Now:    func() time.Time { return time.Date(2026, 11, 16, 9, 0, 0, 0, time.UTC) },
	})
	return s, store
}

func waitBatch(t *testing.T, s *Service) BatchStatus {
	t.Helper()
	for range 500 {
		if st := s.Batch(); !st.Running {
			return st
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatal("batch did not finish")
	return BatchStatus{}
}

func TestBatchDrawsTheTopics(t *testing.T) {
	t.Parallel()
	d := &fakeDrawer{}
	s, store := newTestService(d)
	st, err := s.StartBatch(t.Context(), BatchRequest{TopicIDs: []string{"t1", "t2", "t1", " "}, Engine: EngineGemini})
	if err != nil || !st.Running || st.Total != 2 {
		t.Fatalf("start = %+v, %v", st, err)
	}
	st = waitBatch(t, s)
	if len(st.Saved) != 2 || len(st.Failures) != 0 || st.Done() != 2 {
		t.Fatalf("status = %+v", st)
	}
	if _, err := store.Get(t.Context(), "t1"); err != nil {
		t.Fatal(err)
	}
	// The picture is asked for as a topic banner, with the topic's words.
	if r := d.reqs[0]; r.Topic == nil || r.Topic.Name == "" || ai.ImageAspect(r) != "16:9" {
		t.Fatalf("request = %+v", r)
	}
	list, _ := s.List(t.Context())
	if len(list) != 2 {
		t.Fatalf("list = %+v", list)
	}
	if err := s.Delete(t.Context(), "t1"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Image(t.Context(), "t1"); !errors.Is(err, ErrImageNotFound) {
		t.Fatalf("deleted image: %v", err)
	}
}

func TestBatchErrors(t *testing.T) {
	t.Parallel()
	s, _ := newTestService(&fakeDrawer{})
	var verr *ValidationError
	if _, err := s.StartBatch(t.Context(), BatchRequest{TopicIDs: []string{"t1"}}); !errors.As(err, &verr) || verr.Fields["engine"] == "" {
		t.Fatalf("no engine: %v", err)
	}
	if _, err := s.StartBatch(t.Context(), BatchRequest{TopicIDs: []string{"nope"}, Engine: EngineGemini}); !errors.As(err, &verr) {
		t.Fatalf("unknown topic: %v", err)
	}
	if _, err := s.StartBatch(t.Context(), BatchRequest{Engine: EngineGemini}); !errors.As(err, &verr) {
		t.Fatalf("no topic: %v", err)
	}
	if _, err := s.StartBatch(t.Context(), BatchRequest{TopicIDs: []string{"t1"}, Engine: EngineCodex}); !errors.Is(err, ErrCodexUnavailable) {
		t.Fatalf("codex: %v", err)
	}
}

func TestBatchStopsOnAnErrorEveryTopicWouldGet(t *testing.T) {
	t.Parallel()
	s, _ := newTestService(&fakeDrawer{err: ai.ErrQuota})
	if _, err := s.StartBatch(t.Context(), BatchRequest{TopicIDs: []string{"t1", "t2"}, Engine: EngineGemini}); err != nil {
		t.Fatal(err)
	}
	st := waitBatch(t, s)
	if !errors.Is(st.StopErr, ai.ErrQuota) || len(st.Saved) != 0 || st.Done() == 0 {
		t.Fatalf("status = %+v", st)
	}
}

func TestHandler(t *testing.T) {
	t.Parallel()
	s, store := newTestService(&fakeDrawer{})
	_ = store.Save(t.Context(), Image{TopicID: "t1", MIME: "image/jpeg", Data: []byte("jpg"), CreatedAt: time.UnixMilli(1000)})
	mux := http.NewServeMux()
	pass := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(httpx.WithPrincipal(r.Context(), httpx.Principal{UserID: "a", Role: httpx.RoleAdmin})))
		})
	}
	NewHandler(s, slog.New(slog.DiscardHandler)).Register(mux, pass)
	do := func(method, path, body string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		mux.ServeHTTP(rec, req)
		return rec
	}
	if rec := do(http.MethodGet, "/api/topic-images", ""); rec.Code != http.StatusOK ||
		!strings.Contains(rec.Body.String(), `"imageUrl":"/api/topic-images/t1?v=1000"`) {
		t.Fatalf("list = %d %s", rec.Code, rec.Body)
	}
	if rec := do(http.MethodGet, "/api/topic-images/t1", ""); rec.Code != http.StatusOK || rec.Body.String() != "jpg" {
		t.Fatalf("image = %d %s", rec.Code, rec.Body)
	}
	if rec := do(http.MethodGet, "/api/topic-images/t2", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("missing image = %d", rec.Code)
	}
	if rec := do(http.MethodPost, "/api/admin/topic-image-batch", `{"topicIds":["t2"],"engine":"gemini"}`); rec.Code != http.StatusAccepted {
		t.Fatalf("start = %d %s", rec.Code, rec.Body)
	}
	waitBatch(t, s)
	if rec := do(http.MethodGet, "/api/admin/topic-image-batch", ""); !strings.Contains(rec.Body.String(), `"saved":["t2"]`) {
		t.Fatalf("batch = %s", rec.Body)
	}
	if rec := do(http.MethodDelete, "/api/admin/topic-images/t1", ""); rec.Code != http.StatusNoContent {
		t.Fatalf("delete = %d", rec.Code)
	}
}
