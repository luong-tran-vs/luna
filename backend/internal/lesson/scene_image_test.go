package lesson

import (
	"bytes"
	"errors"
	"image/jpeg"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/luongtran/luna/backend/internal/ai"
	"github.com/luongtran/luna/backend/internal/platform/httpx"
)

func TestDrawSceneWithGemini(t *testing.T) {
	t.Parallel()
	e := newImageEnv(t)
	l := e.annotated(t)

	st, err := e.svc.DrawScene(t.Context(), l.ID, ImageInput{SceneDescription: "  Two friends in a park  ", Style: "watercolor"}, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(e.draw.requests) != 1 {
		t.Fatalf("requests = %+v, want only the scene", e.draw.requests)
	}
	req := e.draw.requests[0]
	if sc := req.Scene; sc == nil || sc.Title != "Park" || sc.Text != l.Content || sc.Description != "Two friends in a park" || req.Style != "watercolor" {
		t.Fatalf("request = %+v, scene %+v", req, req.Scene)
	}
	img, err := e.store.Get(t.Context(), l.ID, SceneKey)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := jpeg.DecodeConfig(bytes.NewReader(img.Data))
	if err != nil || cfg.Width != sceneSide {
		t.Fatalf("stored scene %dx%d, %v", cfg.Width, cfg.Height, err)
	}
	if !st.HasScene || st.Count != 0 || st.SceneDescription != "Two friends in a park" || st.Style != "watercolor" {
		t.Fatalf("state = %+v", st)
	}

	// Drawing again replaces the picture; the word pictures are never drawn.
	if _, err := e.svc.DrawScene(t.Context(), l.ID, ImageInput{}, EngineGemini); err != nil {
		t.Fatal(err)
	}
	if got := e.draw.words(); !slices.Equal(got, []string{"", ""}) {
		t.Fatalf("drawn = %q", got)
	}
}

func TestDrawSceneWithCodex(t *testing.T) {
	t.Parallel()
	e := newImageEnv(t)
	l := e.annotated(t)
	if e.svc.CodexAvailable() {
		t.Fatal("codex available without a codex drawer")
	}
	if _, err := e.svc.DrawScene(t.Context(), l.ID, ImageInput{}, EngineCodex); !errors.Is(err, ErrCodexUnavailable) {
		t.Fatalf("no codex: %v", err)
	}

	codex := &fakeImageAI{}
	e.svc.CodexImageAI = codex
	if _, err := e.svc.DrawScene(t.Context(), l.ID, ImageInput{}, EngineCodex); err != nil {
		t.Fatal(err)
	}
	if len(codex.requests) != 1 || len(e.draw.requests) != 0 {
		t.Fatalf("codex %d, gemini %d requests", len(codex.requests), len(e.draw.requests))
	}

	codex.failErr = ai.ErrNotConfigured
	if _, err := e.svc.DrawScene(t.Context(), l.ID, ImageInput{}, EngineCodex); !errors.Is(err, ErrCodexNotLoggedIn) {
		t.Fatalf("not logged in: %v", err)
	}
}

func TestDrawSceneErrors(t *testing.T) {
	t.Parallel()
	e := newImageEnv(t)
	l := e.annotated(t)
	var verr *ValidationError
	if _, err := e.svc.DrawScene(t.Context(), l.ID, ImageInput{}, "dall-e"); !errors.As(err, &verr) || verr.Fields["engine"] == "" {
		t.Fatalf("bad engine: %v", err)
	}
	if _, err := e.svc.DrawScene(t.Context(), e.create(t).ID, ImageInput{}, ""); !errors.Is(err, ErrAnnotationNotDone) {
		t.Fatalf("not annotated: %v", err)
	}

	e.draw.failErr = ai.ErrQuota
	if _, err := e.svc.DrawScene(t.Context(), l.ID, ImageInput{SceneDescription: "A park"}, ""); !errors.Is(err, ai.ErrQuota) {
		t.Fatalf("quota: %v", err)
	}
	// The settings were saved before the drawing failed.
	if set, _ := e.store.Settings(t.Context(), l.ID); set.SceneDescription != "A park" {
		t.Fatalf("settings = %+v", set)
	}
	e.draw.failErr = errors.New("model refused")
	if _, err := e.svc.DrawScene(t.Context(), l.ID, ImageInput{}, ""); !errors.Is(err, ErrDrawFailed) {
		t.Fatalf("refused: %v", err)
	}
	if _, err := e.store.Get(t.Context(), l.ID, SceneKey); !errors.Is(err, ErrImageNotFound) {
		t.Fatalf("scene saved after failures: %v", err)
	}
}

func TestSceneImageUploadAndValidation(t *testing.T) {
	t.Parallel()
	e := newImageEnv(t)
	l := e.annotated(t)
	if err := e.svc.UploadImage(t.Context(), l.ID, SceneKey, testPNG(1600, 900)); err != nil {
		t.Fatal(err)
	}
	img, _ := e.store.Get(t.Context(), l.ID, SceneKey)
	if cfg, err := jpeg.DecodeConfig(bytes.NewReader(img.Data)); err != nil || cfg.Width != sceneSide || cfg.Height != 450 {
		t.Fatalf("stored scene %dx%d, %v", cfg.Width, cfg.Height, err)
	}
	if err := e.svc.DeleteImage(t.Context(), l.ID, SceneKey); err != nil {
		t.Fatal(err)
	}
	if st, _ := e.svc.Images(t.Context(), l.ID); st.HasScene {
		t.Fatal("scene kept after delete")
	}

	var verr *ValidationError
	_, err := e.svc.SetImages(t.Context(), l.ID, ImageInput{SceneDescription: strings.Repeat("x", MaxSceneDescription+1)})
	if !errors.As(err, &verr) || verr.Fields["sceneDescription"] == "" {
		t.Fatalf("long description: %v", err)
	}
}

func TestSetSceneDescriptionKeepsSettings(t *testing.T) {
	t.Parallel()
	e := newImageEnv(t)
	l := e.annotated(t)
	if _, err := e.svc.SetImages(t.Context(), l.ID, ImageInput{Style: "watercolor"}); err != nil {
		t.Fatal(err)
	}
	if err := e.svc.SetSceneDescription(t.Context(), l.ID, " A family at home "); err != nil {
		t.Fatal(err)
	}
	set, _ := e.store.Settings(t.Context(), l.ID)
	if set.Style != "watercolor" || set.SceneDescription != "A family at home" {
		t.Fatalf("settings = %+v", set)
	}
}

func TestSceneBriefAvoidsRevealingQuestions(t *testing.T) {
	t.Parallel()
	l := Lesson{Title: "Park", Content: "We went to the park."}
	l.Extras.Questions = []Question{
		{Kind: "gist", Prompt: "What is the story about?"},
		{Kind: "detail", Prompt: "Where did they go?"},
		{Kind: "vocab", Prompt: "What does went mean?"},
		{Prompt: "An old question without a kind"},
	}
	l.Practice = &Practice{Steps: &Steps{ListeningQuestions: []ListenQuestion{
		{Kind: "tfn", Prompt: "They stayed at home."}, {Kind: "gist", Prompt: "What is it about?"},
	}}}
	b := sceneBrief(l, "A park")
	want := []string{"Where did they go?", "An old question without a kind", "They stayed at home."}
	if b.Title != "Park" || b.Text != l.Content || b.Description != "A park" || !slices.Equal(b.Avoid, want) {
		t.Fatalf("brief = %+v", b)
	}
	if b := sceneBrief(Lesson{Practice: &Practice{}}, ""); len(b.Avoid) != 0 {
		t.Fatalf("avoid without questions = %q", b.Avoid)
	}
}

func TestLessonViewSceneImageURL(t *testing.T) {
	t.Parallel()
	r, id := newVocabularyEnv(t, StatusDone, []Annotation{{Text: "went", Lemma: "go", MeaningVi: "đi", SentenceIndex: 0}})
	store := newFakeImageStore()
	mux := newReadingMux(r.WithImages(store))
	if rec := get(t, mux, "/api/lessons/"+id, "learner"); !strings.Contains(rec.Body.String(), `"sceneImageUrl":""`) {
		t.Fatalf("without scene: %d %s", rec.Code, rec.Body)
	}
	_ = store.Save(t.Context(), WordImage{LessonID: id, Lemma: SceneKey, MIME: "image/jpeg", Data: []byte("scene")})
	rec := get(t, mux, "/api/lessons/"+id, "learner")
	if !strings.Contains(rec.Body.String(), `"sceneImageUrl":"/api/lessons/`+id+`/images/_scene"`) {
		t.Fatalf("with scene: %s", rec.Body)
	}
	if rec := get(t, mux, "/api/lessons/"+id+"/images/_scene", "learner"); rec.Body.String() != "scene" {
		t.Fatalf("scene picture: %d %q", rec.Code, rec.Body)
	}
}

func TestGenerateSceneEndpoint(t *testing.T) {
	t.Parallel()
	e := newImageEnv(t)
	l := e.annotated(t)
	mux := http.NewServeMux()
	NewHandler(e.svc, slog.New(slog.DiscardHandler)).Register(mux, httpx.RequireAuth(resolver))
	send := func(method, path, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequestWithContext(t.Context(), method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(&http.Cookie{Name: httpx.SessionCookieName, Value: "admin"})
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		return rec
	}
	base := "/api/admin/lessons/" + l.ID + "/images"
	if rec := send(http.MethodGet, base, ""); !strings.Contains(rec.Body.String(), `"codexImages":false`) {
		t.Fatalf("state: %d %s", rec.Code, rec.Body)
	}
	rec := send(http.MethodPost, base+"/scene/generate", `{"sceneDescription":"A park","engine":"gemini"}`)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"sceneImageUrl":"`+base+`/_scene"`) {
		t.Fatalf("generate: %d %s", rec.Code, rec.Body)
	}
	if rec := send(http.MethodPost, base+"/scene/generate", `{"engine":"codex"}`); rec.Code != http.StatusServiceUnavailable ||
		!strings.Contains(rec.Body.String(), "codex_unavailable") {
		t.Fatalf("codex off: %d %s", rec.Code, rec.Body)
	}
	e.draw.failErr = ai.ErrQuota
	if rec := send(http.MethodPost, base+"/scene/generate", `{}`); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("quota: %d %s", rec.Code, rec.Body)
	}
}
