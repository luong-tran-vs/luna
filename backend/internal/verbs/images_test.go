package verbs

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"image/png"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/luongtran/luna/backend/internal/ai"
	"github.com/luongtran/luna/backend/internal/platform/httpx"
)

type fakeImageStore struct {
	mu   sync.Mutex
	imgs map[string]VerbImage
}

func newFakeImageStore() *fakeImageStore { return &fakeImageStore{imgs: map[string]VerbImage{}} }

func (f *fakeImageStore) List(context.Context) ([]ImageInfo, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []ImageInfo{}
	for _, i := range f.imgs {
		out = append(out, ImageInfo{Base: i.Base, CreatedAt: i.CreatedAt})
	}
	slices.SortFunc(out, func(a, b ImageInfo) int { return strings.Compare(a.Base, b.Base) })
	return out, nil
}

func (f *fakeImageStore) Get(_ context.Context, base string) (VerbImage, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	i, ok := f.imgs[base]
	if !ok {
		return VerbImage{}, ErrImageNotFound
	}
	return i, nil
}

func (f *fakeImageStore) Save(_ context.Context, img VerbImage) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.imgs[img.Base] = img
	return nil
}

func (f *fakeImageStore) Delete(_ context.Context, base string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.imgs, base)
	return nil
}

// fakeDrawer draws a small PNG, or fails with err for the words in fail.
type fakeDrawer struct {
	mu   sync.Mutex
	reqs []ai.ImageRequest
	fail map[string]error
}

func (f *fakeDrawer) GenerateImage(_ context.Context, req ai.ImageRequest) (ai.Image, error) {
	f.mu.Lock()
	f.reqs = append(f.reqs, req)
	err := f.fail[req.Word]
	f.mu.Unlock()
	if err != nil {
		return ai.Image{}, err
	}
	return ai.Image{MIME: "image/png", Data: pngBytes()}, nil
}

func pngBytes() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 8, 6))
	img.Set(1, 1, color.RGBA{R: 200, A: 255})
	var b bytes.Buffer
	_ = png.Encode(&b, img)
	return b.Bytes()
}

type fakeBank struct{}

func (fakeBank) Pictures(_ context.Context, words []string) (map[string]BankPicture, error) {
	out := map[string]BankPicture{}
	for _, w := range words {
		if w == "swim" {
			out[w] = BankPicture{MeaningVi: "bơi", CreatedAt: time.Unix(100, 0)}
		}
	}
	return out, nil
}

func (fakeBank) Image(_ context.Context, word string) (string, []byte, error) {
	if word != "swim" {
		return "", nil, ErrNoBankImage
	}
	return "image/png", pngBytes(), nil
}

func newImages(gemini, codex ai.ImageProvider) (*Images, *fakeImageStore) {
	store := newFakeImageStore()
	return NewImages(ImageDeps{
		Store: store, Gemini: gemini, Codex: codex, CodexConcurrency: 2, Bank: fakeBank{},
		Log: slog.New(slog.DiscardHandler), Now: func() time.Time { return time.Unix(1000, 0) },
	}), store
}

// waitBatch waits for the running batch to end.
func waitBatch(t *testing.T, s *Images) BatchStatus {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if st := s.Batch(); !st.Running {
			return st
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("batch did not end")
	return BatchStatus{}
}

func TestBatchNeedsAnEngine(t *testing.T) {
	t.Parallel()
	s, _ := newImages(&fakeDrawer{}, &fakeDrawer{})
	_, err := s.StartBatch(t.Context(), BatchRequest{Items: []DrawItem{{Base: "go"}}})
	var verr *ValidationError
	if !errors.As(err, &verr) || verr.Fields["engine"] == "" {
		t.Fatalf("no engine: %v", err)
	}
	if _, err := newImagesNoCodex().StartBatch(t.Context(), BatchRequest{Items: []DrawItem{{Base: "go"}}, Engine: EngineCodex}); !errors.Is(err, ErrCodexUnavailable) {
		t.Fatalf("codex off: %v", err)
	}
}

func newImagesNoCodex() *Images {
	s, _ := newImages(&fakeDrawer{}, nil)
	return s
}

func TestBatchDrawsTheVerbSense(t *testing.T) {
	t.Parallel()
	drawer := &fakeDrawer{}
	s, store := newImages(nil, drawer)
	st, err := s.StartBatch(t.Context(), BatchRequest{Engine: EngineCodex, Items: []DrawItem{
		{Base: " Ring ", MeaningVi: "reo, rung (chuông)", Sentence: "The phone rang twice."},
		{Base: "swim", MeaningVi: "bơi"},
		{Base: "ring"}, // a repeat is drawn once
	}})
	if err != nil || st.Total != 2 || st.Engine != EngineCodex {
		t.Fatalf("start = %+v, %v", st, err)
	}
	st = waitBatch(t, s)
	if len(st.Saved) != 2 || len(st.Failures) != 0 {
		t.Fatalf("status = %+v", st)
	}
	if _, err := store.Get(t.Context(), "ring"); err != nil {
		t.Fatalf("ring saved: %v", err)
	}
	var ring ai.ImageRequest
	for _, r := range drawer.reqs {
		if r.Word == "to ring" {
			ring = r
		}
	}
	if ring.Sentence != "The phone rang twice." || ring.MeaningVi != "reo, rung (chuông)" {
		t.Fatalf("prompt = %+v", drawer.reqs)
	}
}

func TestBatchStopsWhenTheQuotaRunsOut(t *testing.T) {
	t.Parallel()
	drawer := &fakeDrawer{fail: map[string]error{"to go": ai.ErrQuota, "to do": errors.New("bad answer")}}
	s, _ := newImages(drawer, nil)
	if _, err := s.StartBatch(t.Context(), BatchRequest{Engine: EngineGemini, Items: []DrawItem{{Base: "do"}, {Base: "go"}, {Base: "see"}}}); err != nil {
		t.Fatal(err)
	}
	st := waitBatch(t, s)
	if len(st.Failures) != 2 || !errors.Is(st.StopErr, ai.ErrQuota) || slices.Contains(st.Saved, "see") {
		t.Fatalf("status = %+v", st)
	}
	if !errors.Is(st.Failures[0].Err, ErrDrawFailed) {
		t.Fatalf("first failure = %v", st.Failures[0].Err)
	}
}

func TestUploadFromBankAndDelete(t *testing.T) {
	t.Parallel()
	s, store := newImages(nil, nil)
	if _, err := s.Upload(t.Context(), "go", []byte("not a picture")); !errors.Is(err, ErrBadImage) {
		t.Fatalf("bad upload: %v", err)
	}
	if _, err := s.Upload(t.Context(), "Go!", pngBytes()); err == nil {
		t.Fatal("a bad base is refused")
	}
	if info, err := s.Upload(t.Context(), "go", pngBytes()); err != nil || info.Base != "go" {
		t.Fatalf("upload = %+v, %v", info, err)
	}
	if _, err := s.FromBank(t.Context(), "drink"); !errors.Is(err, ErrNoBankImage) {
		t.Fatalf("no bank picture: %v", err)
	}
	if _, err := s.FromBank(t.Context(), "swim"); err != nil {
		t.Fatalf("from bank: %v", err)
	}
	list, _ := s.List(t.Context())
	if len(list) != 2 {
		t.Fatalf("list = %+v", list)
	}
	if err := s.Delete(t.Context(), "go"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(t.Context(), "go"); !errors.Is(err, ErrImageNotFound) {
		t.Fatalf("deleted: %v", err)
	}
}

// withRole passes every request through as user u1 with that role.
func withRole(role string) httpx.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(httpx.WithPrincipal(r.Context(), httpx.Principal{UserID: "u1", Role: role})))
		})
	}
}

func TestImageRoutes(t *testing.T) {
	t.Parallel()
	s, _ := newImages(&fakeDrawer{}, nil)
	if _, err := s.Upload(t.Context(), "go", pngBytes()); err != nil {
		t.Fatal(err)
	}
	learner := http.NewServeMux()
	NewImageHandler(s, slog.New(slog.DiscardHandler)).Register(learner, withRole("member"))
	admin := http.NewServeMux()
	NewImageHandler(s, slog.New(slog.DiscardHandler)).Register(admin, withRole(httpx.RoleAdmin))

	do := func(mux *http.ServeMux, method, path, body string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		mux.ServeHTTP(rec, req)
		return rec
	}
	rec := do(learner, "GET", "/api/verbs/images", "")
	var list struct {
		Images []imageJSON `json:"images"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&list); err != nil || len(list.Images) != 1 ||
		!strings.HasPrefix(list.Images[0].ImageURL, "/api/verbs/images/go?v=") {
		t.Fatalf("learner list = %d %+v %v", rec.Code, list, err)
	}
	if rec := do(learner, "GET", "/api/verbs/images/go", ""); rec.Code != 200 || rec.Header().Get("Content-Type") != "image/jpeg" {
		t.Fatalf("image = %d %s", rec.Code, rec.Header().Get("Content-Type"))
	}
	if rec := do(learner, "GET", "/api/verbs/images/see", ""); rec.Code != 404 {
		t.Fatalf("missing image = %d", rec.Code)
	}
	if rec := do(learner, "GET", "/api/admin/verb-images", ""); rec.Code != http.StatusForbidden {
		t.Fatalf("learner on admin route = %d", rec.Code)
	}

	rec = do(admin, "GET", "/api/admin/verb-images?bases=swim,drink", "")
	body, _ := io.ReadAll(rec.Body)
	if rec.Code != 200 || !strings.Contains(string(body), `"bank":[{"base":"swim","meaningVi":"bơi"`) {
		t.Fatalf("admin list = %d %s", rec.Code, body)
	}
	if rec := do(admin, "POST", "/api/admin/verb-image-batch", `{"items":[{"base":"see"}]}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("batch without engine = %d", rec.Code)
	}
	if rec := do(admin, "POST", "/api/admin/verb-image-batch", `{"items":[{"base":"see","meaningVi":"nhìn"}],"engine":"gemini"}`); rec.Code != http.StatusAccepted {
		t.Fatalf("batch = %d", rec.Code)
	}
	if st := waitBatch(t, s); len(st.Saved) != 1 {
		t.Fatalf("batch status = %+v", st)
	}
	if rec := do(admin, "DELETE", "/api/admin/verb-images/see", ""); rec.Code != http.StatusNoContent {
		t.Fatalf("delete = %d", rec.Code)
	}
}
