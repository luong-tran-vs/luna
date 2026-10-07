package lesson

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/luongtran/luna/backend/internal/ai"
	"github.com/luongtran/luna/backend/internal/job"
	"github.com/luongtran/luna/backend/internal/platform/httpx"
)

type fakeImageStore struct {
	mu       sync.Mutex
	settings map[string]ImageSettings
	images   map[string]WordImage
}

func newFakeImageStore() *fakeImageStore {
	return &fakeImageStore{settings: map[string]ImageSettings{}, images: map[string]WordImage{}}
}

func (f *fakeImageStore) Settings(_ context.Context, id string) (ImageSettings, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.settings[id], nil
}

func (f *fakeImageStore) SaveSettings(_ context.Context, id string, s ImageSettings) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.settings[id] = s
	return nil
}

func (f *fakeImageStore) Lemmas(_ context.Context, id string) ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []string{}
	for _, img := range f.images {
		if img.LessonID == id {
			out = append(out, img.Lemma)
		}
	}
	slices.Sort(out)
	return out, nil
}

func (f *fakeImageStore) Get(_ context.Context, id, lemma string) (WordImage, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	img, ok := f.images[id+"/"+lemma]
	if !ok {
		return WordImage{}, ErrImageNotFound
	}
	return img, nil
}

func (f *fakeImageStore) Save(_ context.Context, img WordImage) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.images[img.LessonID+"/"+img.Lemma] = img
	return nil
}

func (f *fakeImageStore) Delete(_ context.Context, id, lemma string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.images, id+"/"+lemma)
	return nil
}

func (f *fakeImageStore) DeleteForLesson(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.settings, id)
	for k, img := range f.images {
		if img.LessonID == id {
			delete(f.images, k)
		}
	}
	return nil
}

// fakeImageAI draws a 1000×800 PNG; words in fail return err (or failErr for every word).
type fakeImageAI struct {
	mu       sync.Mutex
	requests []ai.ImageRequest
	fail     map[string]bool
	failErr  error
}

func (f *fakeImageAI) GenerateImage(_ context.Context, req ai.ImageRequest) (ai.Image, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.requests = append(f.requests, req)
	if f.failErr != nil {
		return ai.Image{}, f.failErr
	}
	if f.fail[req.Word] {
		return ai.Image{}, errors.New("model refused")
	}
	return ai.Image{MIME: "image/png", Data: testPNG(1000, 800)}, nil
}

func (f *fakeImageAI) words() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, len(f.requests))
	for i, r := range f.requests {
		out[i] = r.Word
	}
	return out
}

func testPNG(w, h int) []byte {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			img.Set(x, y, color.NRGBA{R: uint8(x), G: uint8(y), B: 200, A: 255})
		}
	}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		panic(err)
	}
	return b.Bytes()
}

type imageEnv struct {
	*env
	store *fakeImageStore
	draw  *fakeImageAI
}

func newImageEnv(t *testing.T) *imageEnv {
	t.Helper()
	e := &imageEnv{env: newEnv(t), store: newFakeImageStore(), draw: &fakeImageAI{fail: map[string]bool{}}}
	e.svc.ImageStore, e.svc.ImageAI = e.store, e.draw
	return e
}

// annotated creates a lesson with pictures as given, annotated with two words.
func (e *imageEnv) annotated(t *testing.T, images *ImageInput) Lesson {
	t.Helper()
	l := e.create(t, func(in *Input) { in.Images = images })
	e.ai.result = []ai.Annotation{
		{Text: "went", Lemma: "go", MeaningVi: "đi", SentenceIndex: 0},
		{Text: "gave up", Lemma: "give up", MeaningVi: "bỏ", SentenceIndex: 1},
	}
	if err := e.svc.ProcessAnnotate(t.Context(), jobFor(l, job.TypeAnnotate)); err != nil {
		t.Fatal(err)
	}
	got, _ := e.lessons.Get(t.Context(), l.ID)
	return got
}

func (e *imageEnv) jobTypes() []job.Type {
	var out []job.Type
	for _, j := range e.jobs.all() {
		out = append(out, j.Type)
	}
	return out
}

func TestImagesDrawnAfterAnnotation(t *testing.T) {
	t.Parallel()
	e := newImageEnv(t)
	l := e.annotated(t, &ImageInput{Enabled: true, Style: "  tranh màu nước  "})

	set, _ := e.store.Settings(t.Context(), l.ID)
	if !set.Enabled || set.Style != "tranh màu nước" || set.Status != StatusRunning {
		t.Fatalf("settings = %+v", set)
	}
	if got := e.jobTypes(); !slices.Equal(got, []job.Type{job.TypeAnnotate, job.TypePractice, job.TypeImages}) {
		t.Fatalf("jobs = %v", got)
	}

	if err := e.svc.ProcessImages(t.Context(), jobFor(l, job.TypeImages)); err != nil {
		t.Fatal(err)
	}
	if got := e.draw.words(); !slices.Equal(got, []string{"go", "give up"}) {
		t.Fatalf("drawn = %v", got)
	}
	first := e.draw.requests[0]
	if first.MeaningVi != "đi" || first.Sentence != "We went to the park." || first.Style != "tranh màu nước" {
		t.Fatalf("request = %+v", first)
	}
	img, err := e.store.Get(t.Context(), l.ID, "go")
	if err != nil || img.MIME != "image/jpeg" {
		t.Fatalf("image = %+v, %v", img.MIME, err)
	}
	cfg, err := jpeg.DecodeConfig(bytes.NewReader(img.Data))
	if err != nil || cfg.Width != imageSide || cfg.Height != 288 {
		t.Fatalf("stored picture %dx%d, %v", cfg.Width, cfg.Height, err)
	}
	if set, _ := e.store.Settings(t.Context(), l.ID); set.Status != StatusDone {
		t.Fatalf("status = %s", set.Status)
	}

	// A second run has nothing left to draw.
	if err := e.svc.ProcessImages(t.Context(), jobFor(l, job.TypeImages)); err != nil {
		t.Fatal(err)
	}
	if n := len(e.draw.words()); n != 2 {
		t.Fatalf("drawn again: %d requests", n)
	}
}

func TestImagesOffByDefault(t *testing.T) {
	t.Parallel()
	e := newImageEnv(t)
	l := e.annotated(t, nil)
	if got := e.jobTypes(); slices.Contains(got, job.TypeImages) {
		t.Fatalf("jobs = %v", got)
	}
	if set, _ := e.store.Settings(t.Context(), l.ID); set != (ImageSettings{}) {
		t.Fatalf("settings = %+v", set)
	}
}

func TestImagesPartialFailureRetriesOnlyMissing(t *testing.T) {
	t.Parallel()
	e := newImageEnv(t)
	l := e.annotated(t, &ImageInput{Enabled: true})
	e.draw.fail["give up"] = true

	err := e.svc.ProcessImages(t.Context(), jobFor(l, job.TypeImages))
	if err == nil || job.IsPermanent(err) {
		t.Fatalf("err = %v, want a retryable error", err)
	}
	if lemmas, _ := e.store.Lemmas(t.Context(), l.ID); !slices.Equal(lemmas, []string{"go"}) {
		t.Fatalf("lemmas = %v", lemmas)
	}

	e.draw.fail["give up"] = false
	if err := e.svc.ProcessImages(t.Context(), jobFor(l, job.TypeImages)); err != nil {
		t.Fatal(err)
	}
	if got := e.draw.words(); !slices.Equal(got, []string{"go", "give up", "give up"}) {
		t.Fatalf("drawn = %v", got)
	}
}

func TestImagesErrors(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		err       error
		permanent bool
	}{
		"not configured": {ai.ErrNotConfigured, true},
		"invalid key":    {ai.ErrInvalidKey, true},
		"quota":          {ai.ErrQuota, false},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			e := newImageEnv(t)
			l := e.annotated(t, &ImageInput{Enabled: true})
			e.draw.failErr = tc.err
			err := e.svc.ProcessImages(t.Context(), jobFor(l, job.TypeImages))
			if !errors.Is(err, tc.err) || job.IsPermanent(err) != tc.permanent {
				t.Fatalf("err = %v, permanent = %v", err, job.IsPermanent(err))
			}
			if tc.err == ai.ErrQuota && len(e.draw.words()) != 1 {
				t.Fatalf("kept drawing after the quota ran out: %v", e.draw.words())
			}
		})
	}
}

func TestImagesJobFailed(t *testing.T) {
	t.Parallel()
	e := newImageEnv(t)
	l := e.annotated(t, &ImageInput{Enabled: true})
	e.svc.JobFailed(t.Context(), jobFor(l, job.TypeImages), errors.New("boom"))
	set, _ := e.store.Settings(t.Context(), l.ID)
	if set.Status != StatusFailed || set.Error != "Không sinh được ảnh cho một số từ" || !set.Enabled {
		t.Fatalf("settings = %+v", set)
	}
	// The annotation is untouched.
	if got, _ := e.lessons.Get(t.Context(), l.ID); got.AnnotationStatus != StatusDone {
		t.Fatalf("annotation status = %s", got.AnnotationStatus)
	}
}

func TestSetImages(t *testing.T) {
	t.Parallel()
	e := newImageEnv(t)
	l := e.annotated(t, nil)

	st, err := e.svc.SetImages(t.Context(), l.ID, ImageInput{Enabled: true, Style: "ảnh chụp"})
	if err != nil || !st.Enabled || st.Status != StatusRunning || st.Count != 0 {
		t.Fatalf("state = %+v, %v", st, err)
	}
	if got := e.jobTypes(); got[len(got)-1] != job.TypeImages {
		t.Fatalf("jobs = %v", got)
	}

	st, err = e.svc.SetImages(t.Context(), l.ID, ImageInput{Enabled: false})
	if err != nil || st.Enabled || st.Status != StatusNone {
		t.Fatalf("off: %+v, %v", st, err)
	}

	var verr *ValidationError
	if _, err := e.svc.SetImages(t.Context(), l.ID, ImageInput{Enabled: true, Style: strings.Repeat("x", MaxImageStyle+1)}); !errors.As(err, &verr) {
		t.Fatalf("long style: %v", err)
	}
	if _, err := e.svc.SetImages(t.Context(), "nope", ImageInput{Enabled: true}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown lesson: %v", err)
	}
}

func TestCreateWithImagesWithoutStore(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	_, err := e.svc.Create(t.Context(), Input{
		Title: "Park", Content: sampleContent, TopicID: "topic-b1", Level: "B1", Source: "Tự viết", License: "CC BY",
		Images: &ImageInput{Enabled: true},
	})
	if !errors.Is(err, ErrImagesUnavailable) {
		t.Fatalf("err = %v", err)
	}
}

func TestDeleteRemovesImages(t *testing.T) {
	t.Parallel()
	e := newImageEnv(t)
	l := e.annotated(t, &ImageInput{Enabled: true})
	if err := e.svc.ProcessImages(t.Context(), jobFor(l, job.TypeImages)); err != nil {
		t.Fatal(err)
	}
	if err := e.svc.Delete(t.Context(), l.ID); err != nil {
		t.Fatal(err)
	}
	if lemmas, _ := e.store.Lemmas(t.Context(), l.ID); len(lemmas) != 0 {
		t.Fatalf("lemmas = %v", lemmas)
	}
}

func TestShrinkImageKeepsSmallPictures(t *testing.T) {
	t.Parallel()
	data, err := shrinkImage(testPNG(200, 100))
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := jpeg.DecodeConfig(bytes.NewReader(data))
	if err != nil || cfg.Width != 200 || cfg.Height != 100 {
		t.Fatalf("%dx%d, %v", cfg.Width, cfg.Height, err)
	}
	if _, err := shrinkImage([]byte("not an image")); err == nil {
		t.Fatal("garbage decoded")
	}
}

func TestVocabularyImagesEndpoints(t *testing.T) {
	t.Parallel()
	r, id := newVocabularyEnv(t, StatusDone, []Annotation{{Text: "went", Lemma: "go", MeaningVi: "đi", SentenceIndex: 0}})
	store := newFakeImageStore()
	_ = store.Save(t.Context(), WordImage{LessonID: id, Lemma: "go", MIME: "image/jpeg", Data: []byte("jpeg")})
	mux := newReadingMux(r.WithImages(store))

	rec := get(t, mux, "/api/lessons/"+id+"/vocabulary", "learner")
	if !strings.Contains(rec.Body.String(), `"imageUrl":"/api/lessons/`+id+`/images/go"`) {
		t.Fatalf("vocabulary: %s", rec.Body)
	}
	rec = get(t, mux, "/api/lessons/"+id+"/images/go", "learner")
	if rec.Code != http.StatusOK || rec.Body.String() != "jpeg" || rec.Header().Get("Content-Type") != "image/jpeg" {
		t.Fatalf("image: %d %q %v", rec.Code, rec.Body, rec.Header())
	}
	if rec := get(t, mux, "/api/lessons/"+id+"/images/smoke", "learner"); rec.Code != http.StatusNotFound {
		t.Fatalf("missing image: %d", rec.Code)
	}
	if rec := get(t, mux, "/api/lessons/"+id+"/images/go", ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous: %d", rec.Code)
	}
}

func TestUploadImageResizesAndReplaces(t *testing.T) {
	t.Parallel()
	e := newImageEnv(t)
	l := e.annotated(t, nil)

	// A large PNG is scaled down to imageSide and stored as JPEG.
	if err := e.svc.UploadImage(t.Context(), l.ID, "  Give Up ", testPNG(2000, 1000)); err != nil {
		t.Fatal(err)
	}
	img, err := e.store.Get(t.Context(), l.ID, "give up")
	if err != nil || img.MIME != "image/jpeg" {
		t.Fatalf("image = %s, %v", img.MIME, err)
	}
	cfg, err := jpeg.DecodeConfig(bytes.NewReader(img.Data))
	if err != nil || cfg.Width != imageSide || cfg.Height != 180 {
		t.Fatalf("stored %dx%d, %v", cfg.Width, cfg.Height, err)
	}

	// A second upload replaces it.
	if err := e.svc.UploadImage(t.Context(), l.ID, "give up", testPNG(100, 100)); err != nil {
		t.Fatal(err)
	}
	img, _ = e.store.Get(t.Context(), l.ID, "give up")
	if cfg, _ := jpeg.DecodeConfig(bytes.NewReader(img.Data)); cfg.Width != 100 {
		t.Fatalf("not replaced: %dx%d", cfg.Width, cfg.Height)
	}

	st, err := e.svc.Images(t.Context(), l.ID)
	if err != nil || st.Count != 1 || len(st.Words) != 2 || !st.Words[1].HasImage || st.Words[0].HasImage {
		t.Fatalf("state = %+v, %v", st, err)
	}

	// Pictures on: the AI only draws the word without one.
	if _, err := e.svc.SetImages(t.Context(), l.ID, ImageInput{Enabled: true}); err != nil {
		t.Fatal(err)
	}
	if err := e.svc.ProcessImages(t.Context(), jobFor(l, job.TypeImages)); err != nil {
		t.Fatal(err)
	}
	if got := e.draw.words(); !slices.Equal(got, []string{"go"}) {
		t.Fatalf("drawn = %v", got)
	}

	if err := e.svc.DeleteImage(t.Context(), l.ID, "give up"); err != nil {
		t.Fatal(err)
	}
	if _, err := e.store.Get(t.Context(), l.ID, "give up"); !errors.Is(err, ErrImageNotFound) {
		t.Fatalf("after delete: %v", err)
	}
}

func TestUploadImageRefusals(t *testing.T) {
	t.Parallel()
	e := newImageEnv(t)
	l := e.annotated(t, nil)
	var verr *ValidationError
	if err := e.svc.UploadImage(t.Context(), l.ID, "go", []byte("not a picture")); !errors.As(err, &verr) || verr.Fields["image"] == "" {
		t.Fatalf("garbage: %v", err)
	}
	if err := e.svc.UploadImage(t.Context(), l.ID, "go", make([]byte, MaxUploadBytes+1)); !errors.As(err, &verr) {
		t.Fatalf("too big: %v", err)
	}
	if err := e.svc.UploadImage(t.Context(), l.ID, "banana", testPNG(10, 10)); !errors.Is(err, ErrNotLessonWord) {
		t.Fatalf("other word: %v", err)
	}
	if err := e.svc.UploadImage(t.Context(), "nope", "go", testPNG(10, 10)); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown lesson: %v", err)
	}
	// Before the annotation is done the lesson has no words yet.
	fresh := e.create(t)
	if err := e.svc.UploadImage(t.Context(), fresh.ID, "go", testPNG(10, 10)); !errors.Is(err, ErrNotLessonWord) {
		t.Fatalf("not annotated: %v", err)
	}
}

func TestUploadImageEndpoint(t *testing.T) {
	t.Parallel()
	e := newImageEnv(t)
	l := e.annotated(t, nil)
	mux := http.NewServeMux()
	NewHandler(e.svc, slog.New(slog.DiscardHandler)).Register(mux, httpx.RequireAuth(resolver))

	send := func(method, path string, body []byte) *httptest.ResponseRecorder {
		req := httptest.NewRequestWithContext(t.Context(), method, path, bytes.NewReader(body))
		req.AddCookie(&http.Cookie{Name: httpx.SessionCookieName, Value: "admin"})
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		return rec
	}
	base := "/api/admin/lessons/" + l.ID + "/images/"
	rec := send(http.MethodPut, base+"give%20up", testPNG(600, 600))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"imageUrl":"`+base+`give%20up"`) {
		t.Fatalf("upload: %d %s", rec.Code, rec.Body)
	}
	rec = send(http.MethodGet, base+"give%20up", nil)
	if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "image/jpeg" || rec.Header().Get("Last-Modified") == "" {
		t.Fatalf("get: %d %v", rec.Code, rec.Header())
	}
	if rec := send(http.MethodPut, base+"go", []byte("nope")); rec.Code != http.StatusBadRequest {
		t.Fatalf("bad picture: %d", rec.Code)
	}
	if rec := send(http.MethodPut, base+"banana", testPNG(10, 10)); rec.Code != http.StatusNotFound {
		t.Fatalf("other word: %d", rec.Code)
	}
	if rec := send(http.MethodDelete, base+"give%20up", nil); rec.Code != http.StatusOK || strings.Contains(rec.Body.String(), `give%20up"`) {
		t.Fatalf("delete: %d %s", rec.Code, rec.Body)
	}
}
