package toeic

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/luongtran/luna/backend/internal/ai"
)

// pngOf is a w×h PNG.
func pngOf(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for x := range w {
		img.Set(x, 0, color.RGBA{R: 200, A: 255})
	}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

type fakeDrawer struct {
	mu    sync.Mutex
	pic   []byte
	err   error
	codex bool
	reqs  []ai.ImageRequest
	// gate, when set, holds every drawing until it is closed.
	gate chan struct{}
}

func (f *fakeDrawer) Draw(_ context.Context, engine string, req ai.ImageRequest) (ai.Image, error) {
	if f.gate != nil {
		<-f.gate
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if engine == EngineCodex && !f.codex {
		return ai.Image{}, ErrCodexUnavailable
	}
	f.reqs = append(f.reqs, req)
	return ai.Image{MIME: "image/png", Data: f.pic}, f.err
}

func (f *fakeDrawer) CodexAvailable() bool { return f.codex }

func (f *fakeDrawer) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.reqs)
}

func TestDrawPhotoSendsOnlyTheRightStatement(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	ctx := t.Context()
	g := p1("p1-a")
	g.Questions[0].Answer = 2
	e.seed(g)
	d := &fakeDrawer{pic: pngOf(t, 1600, 1200)}
	e.svc.d.Drawer = d

	var verr *ValidationError
	if _, err := e.svc.DrawImage(ctx, "p1-a", SlotPhoto, ""); !errors.As(err, &verr) || verr.Fields["engine"] == "" {
		t.Fatalf("no engine = %v", err)
	}
	if _, err := e.svc.DrawImage(ctx, "p1-a", SlotPhoto, EngineCodex); !errors.Is(err, ErrCodexUnavailable) {
		t.Fatalf("codex = %v", err)
	}
	img, err := e.svc.DrawImage(ctx, "p1-a", SlotPhoto, EngineGemini)
	if err != nil || img.Origin != EngineGemini {
		t.Fatalf("draw = %+v, %v", img, err)
	}
	got := d.reqs[0].Photo
	if got == nil || got.Scene != "A man stacks boxes." || got.Statement != "charlie" {
		t.Fatalf("brief = %+v", got)
	}
	prompt := ai.ImagePrompt(d.reqs[0])
	for _, wrong := range []string{"alpha", "bravo", "delta"} {
		if strings.Contains(prompt, wrong) {
			t.Fatalf("prompt has wrong statement %q", wrong)
		}
	}
	stored, _ := e.images.Get(ctx, "p1-a", SlotPhoto)
	cfg, _, err := image.DecodeConfig(bytes.NewReader(stored.Data))
	if err != nil || cfg.Width > imageSide || cfg.Height > imageSide {
		t.Fatalf("stored %dx%d, %v", cfg.Width, cfg.Height, err)
	}
	if _, err := e.svc.DrawImage(ctx, "p1-a", "doc-1", EngineGemini); !errors.As(err, &verr) {
		t.Fatalf("bad slot = %v", err)
	}
}

func TestDocPictureAndSlots(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	ctx := t.Context()
	e.seed(p7("p7-a", 2))
	d := &fakeDrawer{pic: pngOf(t, 40, 30)}
	e.svc.d.Drawer = d
	slots, err := e.svc.GroupImages(ctx, "p7-a")
	if err != nil || len(slots) != 1 || slots[0].Slot != "doc-1" || slots[0].Has || !strings.Contains(slots[0].Label, "Garden notice") {
		t.Fatalf("slots = %+v, %v", slots, err)
	}
	if _, err := e.svc.DrawImage(ctx, "p7-a", "doc-1", EngineGemini); err != nil {
		t.Fatal(err)
	}
	if dp := d.reqs[0].DocPicture; dp == nil || dp.Title != "Garden notice" || !strings.Contains(dp.Text, "opens at 9") {
		t.Fatalf("doc brief = %+v", dp)
	}
	if ai.ImageOp(d.reqs[0]) != "toeic_photo" {
		t.Fatal("usage op")
	}
	// The learner's paper shows the text's picture once it has one.
	pg := paperGroupOf(e.groups.recs["p7-a"], []string{"doc-1"})
	if pg.Docs[0].ImageURL == nil {
		t.Fatal("paper doc picture missing")
	}
}

func TestUploadImage(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	ctx := t.Context()
	e.seed(p1("p1-a"))
	var verr *ValidationError
	if _, err := e.svc.UploadImage(ctx, "p1-a", SlotPhoto, []byte("hello, not a picture")); !errors.As(err, &verr) {
		t.Fatalf("not an image = %v", err)
	}
	if _, err := e.svc.UploadImage(ctx, "p1-a", SlotPhoto, make([]byte, MaxUploadBytes+1)); !errors.As(err, &verr) {
		t.Fatalf("too big = %v", err)
	}
	img, err := e.svc.UploadImage(ctx, "p1-a", SlotPhoto, pngOf(t, 1000, 500))
	if err != nil || img.Origin != "upload" {
		t.Fatalf("upload = %+v, %v", img, err)
	}
	if err := e.svc.DeleteImage(ctx, "p1-a", SlotPhoto); err != nil {
		t.Fatal(err)
	}
	if _, err := e.images.Get(ctx, "p1-a", SlotPhoto); !errors.Is(err, ErrNotFound) {
		t.Fatalf("deleted = %v", err)
	}
}

func TestPhotoBatchDrawsOnlyMissingPhotos(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	ctx := t.Context()
	test := Test{Key: "t1", Title: "T", Kind: KindFull, Order: 1}
	for _, k := range []string{"p1-a", "p1-b", "p1-c"} {
		e.seed(p1(k))
		test.Groups = append(test.Groups, k)
	}
	e.tests.recs["t1"] = test
	_ = e.images.Save(ctx, Image{GroupKey: "p1-b", Slot: SlotPhoto, MIME: "image/png", Data: pngOf(t, 4, 4)})
	d := &fakeDrawer{pic: pngOf(t, 40, 30), gate: make(chan struct{})}
	e.svc.d.Drawer = d

	st, err := e.svc.StartPhotoBatch(ctx, BatchInput{TestKey: "t1", Engine: EngineGemini}, nil)
	if err != nil || !st.Running || st.Total != 2 {
		t.Fatalf("start = %+v, %v", st, err)
	}
	if _, err := e.svc.StartPhotoBatch(ctx, BatchInput{TestKey: "t1", Engine: EngineGemini}, nil); !errors.Is(err, ErrBatchRunning) {
		t.Fatalf("second = %v", err)
	}
	close(d.gate)
	deadline := time.Now().Add(5 * time.Second)
	for e.svc.PhotoBatch().Running && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if st := e.svc.PhotoBatch(); st.Running || len(st.Saved) != 2 || d.count() != 2 {
		t.Fatalf("done = %+v, drawn %d", st, d.count())
	}
	ta, _ := e.svc.TestAdmin(ctx, "t1")
	if len(ta.Check.MissingPhotos) != 0 {
		t.Fatalf("missing = %+v", ta.Check.MissingPhotos)
	}
}

func TestPhotoBatchStopsOnQuota(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	ctx := t.Context()
	e.seed(p1("p1-a"), p1("p1-b"))
	d := &fakeDrawer{err: ai.ErrQuota}
	e.svc.d.Drawer = d
	if _, err := e.svc.StartPhotoBatch(ctx, BatchInput{Part: 1, Engine: EngineGemini}, nil); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for e.svc.PhotoBatch().Running && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	st := e.svc.PhotoBatch()
	if d.count() != 1 || len(st.Failed) != 1 || !strings.Contains(st.StopReason, "hết lượt") {
		t.Fatalf("quota = %+v, drawn %d", st, d.count())
	}
}

func TestImageEndpoints(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.seed(p1("p1-a"))
	e.svc.d.Drawer = &fakeDrawer{pic: pngOf(t, 40, 30)}
	mux := e.mux()
	rec := call(t, mux, http.MethodPost, "/api/admin/toeic/images/p1-a/photo/generate", "admin", `{"engine":"codex"}`)
	if rec.Code != http.StatusServiceUnavailable || !strings.Contains(rec.Body.String(), "codex_unavailable") {
		t.Fatalf("codex = %d %s", rec.Code, rec.Body)
	}
	rec = call(t, mux, http.MethodPost, "/api/admin/toeic/images/p1-a/photo/generate", "admin", `{"engine":"gemini"}`)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"has":true`) {
		t.Fatalf("draw = %d %s", rec.Code, rec.Body)
	}
	rec = call(t, mux, http.MethodGet, "/api/admin/toeic/images/batch", "admin", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"codex":false`) {
		t.Fatalf("batch = %d %s", rec.Code, rec.Body)
	}
	rec = call(t, mux, http.MethodDelete, "/api/admin/toeic/images/p1-a/photo", "admin", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"has":false`) {
		t.Fatalf("delete = %d %s", rec.Code, rec.Body)
	}
	rec = call(t, mux, http.MethodPost, "/api/admin/toeic/images/p1-a/photo/generate", "member", `{"engine":"gemini"}`)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("member = %d", rec.Code)
	}
}

func TestPictureBatchOfAPart(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	ctx := t.Context()
	ad := p7("p7-ad", 2)
	ad.Docs = []Doc{
		{Kind: "ad", Title: "Summer sale", Body: []Block{{Kind: "p", Text: "Half price on all chairs."}}},
		{Kind: "email", Title: "My order", Body: []Block{{Kind: "p", Text: "Where is my chair?"}}},
		{Kind: "web", Title: "Store page", Body: []Block{{Kind: "p", Text: "Open daily."}}},
	}
	mail := p7("p7-mail", 2)
	mail.Docs[0].Kind = "email"
	hidden := p7("p7-hidden", 2)
	hidden.Docs[0].Kind = "notice"
	e.seed(ad, mail, hidden, p1("p1-a"), p1("p1-b"))
	_ = e.groups.SetStatus(ctx, "p7-hidden", StatusHidden)
	_ = e.images.Save(ctx, Image{GroupKey: "p7-ad", Slot: "doc-3", MIME: "image/png", Data: pngOf(t, 4, 4)})

	counts, err := e.svc.MissingPictures(ctx)
	// Part 7: every text without a picture (the ad and the email beside it, the email of p7-mail); the
	// web page has one and hidden groups are left out.
	if err != nil || counts[1][450] != 2 || counts[6][450] != 0 || counts[7][750] != 3 || counts[7][450] != 0 {
		t.Fatalf("missing = %v, %v", counts, err)
	}
	d := &fakeDrawer{pic: pngOf(t, 40, 30)}
	e.svc.d.Drawer = d
	var verr *ValidationError
	if _, err := e.svc.StartPhotoBatch(ctx, BatchInput{Part: 5, Engine: EngineGemini}, nil); !errors.As(err, &verr) {
		t.Fatalf("part 5 = %v", err)
	}
	st, err := e.svc.StartPhotoBatch(ctx, BatchInput{Part: 7, Engine: EngineGemini}, nil)
	if err != nil || st.Part != 7 || st.Total != 3 {
		t.Fatalf("start = %+v, %v", st, err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for e.svc.PhotoBatch().Running && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if st := e.svc.PhotoBatch(); len(st.Saved) != 3 || !slices.Contains(st.Saved, "p7-ad/doc-2") ||
		!slices.Contains(st.Saved, "p7-mail/doc-1") || d.reqs[0].DocPicture == nil {
		t.Fatalf("done = %+v", st)
	}
	rec := call(t, e.mux(), http.MethodGet, "/api/admin/toeic/images/missing", "admin", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"7":0`) || !strings.Contains(rec.Body.String(), `"1":2`) ||
		!strings.Contains(rec.Body.String(), `"byLevel":{`) || !strings.Contains(rec.Body.String(), `"450":2`) {
		t.Fatalf("missing endpoint = %d %s", rec.Code, rec.Body)
	}
}

func TestPictureBatchOfSomeLevels(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	ctx := t.Context()
	high := p1("p1-900")
	high.Level = 900
	mid := p1("p1-600")
	mid.Level = 600
	e.seed(p1("p1-450"), mid, high)
	d := &fakeDrawer{pic: pngOf(t, 40, 30)}
	e.svc.d.Drawer = d
	var verr *ValidationError
	if _, err := e.svc.StartPhotoBatch(ctx, BatchInput{Part: 1, Levels: []int{500}, Engine: EngineGemini}, nil); !errors.As(err, &verr) {
		t.Fatalf("level 500 = %v", err)
	}
	st, err := e.svc.StartPhotoBatch(ctx, BatchInput{Part: 1, Levels: []int{900, 450, 900}, Engine: EngineGemini}, nil)
	if err != nil || st.Total != 2 || !slices.Equal(st.Levels, []int{450, 900}) {
		t.Fatalf("start = %+v, %v", st, err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for e.svc.PhotoBatch().Running && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if st := e.svc.PhotoBatch(); len(st.Saved) != 2 || slices.Contains(st.Saved, "p1-600/photo") {
		t.Fatalf("done = %+v", st)
	}
	counts, err := e.svc.MissingPictures(ctx)
	if err != nil || counts[1][600] != 1 || counts[1][450] != 0 || counts[1][900] != 0 {
		t.Fatalf("missing after = %v, %v", counts, err)
	}
}

func TestPictureBatchOfSomeGroups(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	ctx := t.Context()
	ad := p7("p7-new", 2)
	ad.Docs[0].Kind = "ad"
	e.seed(p1("p1-new-1"), p1("p1-new-2"), p1("p1-old"), ad)
	_ = e.images.Save(ctx, Image{GroupKey: "p1-new-2", Slot: SlotPhoto, MIME: "image/png", Data: pngOf(t, 4, 4)})
	d := &fakeDrawer{pic: pngOf(t, 40, 30)}
	e.svc.d.Drawer = d

	st, err := e.svc.StartPhotoBatch(ctx, BatchInput{Keys: []string{"p1-new-1", "p1-new-2", "p7-new", "nope"}, Engine: EngineGemini}, nil)
	// p1-new-2 has its photo, "nope" does not exist, p1-old is not asked for.
	if err != nil || st.Total != 2 || st.Part != 0 || len(st.Keys) != 4 {
		t.Fatalf("start = %+v, %v", st, err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for e.svc.PhotoBatch().Running && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if st := e.svc.PhotoBatch(); len(st.Saved) != 2 || st.Saved[0] != "p1-new-1/photo" || st.Saved[1] != "p7-new/doc-1" {
		t.Fatalf("done = %+v", st)
	}
	if _, err := e.images.Get(ctx, "p1-old", SlotPhoto); !errors.Is(err, ErrNotFound) {
		t.Fatal("drew a group that was not asked for")
	}
	var verr *ValidationError
	many := make([]string, maxBatchKeys+1)
	for i := range many {
		many[i] = fmt.Sprintf("k%d", i)
	}
	if _, err := e.svc.StartPhotoBatch(ctx, BatchInput{Keys: many, Engine: EngineGemini}, nil); !errors.As(err, &verr) {
		t.Fatalf("too many keys = %v", err)
	}
}
