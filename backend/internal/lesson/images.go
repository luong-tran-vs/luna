package lesson

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/luongtran/luna/backend/internal/ai"
	"github.com/luongtran/luna/backend/internal/imageutil"
	"github.com/luongtran/luna/backend/internal/job"
)

// Pictures of a lesson (F23). Since 2026-11-06 the AI draws only the lesson's scene picture, when
// the admin asks for it, with the engine they choose (Gemini or the Codex CLI of the server, as in
// the word bank). Word pictures are drawn in the word bank only; a lesson can still have its own
// uploaded or linked picture of a word.

// MaxImageStyle is the longest style an admin may give for a picture (F23).
const MaxImageStyle = 500

// imageSide is the longest side of a stored picture, in pixels: enough for the word card on a phone.
const imageSide = 360

// sceneSide is the longest side of a stored scene picture: it spans the width of a phone screen.
const sceneSide = 800

// SceneKey is the key under which the lesson's scene picture is kept among its word pictures
// (added 2026-11-05). No vocabulary word can have it: a lemma never starts with "_".
const SceneKey = "_scene"

// MaxSceneDescription is the longest description of the scene an admin or an import may give.
const MaxSceneDescription = 600

// MaxUploadBytes is the largest picture an admin may upload for a word.
const MaxUploadBytes = 5 << 20

// ErrImageNotFound means the word has no picture.
var ErrImageNotFound = errors.New("lesson: image not found")

// ErrNotLessonWord means the word is not one of the lesson's vocabulary words (or they are not
// annotated yet), so it cannot have a picture.
var ErrNotLessonWord = errors.New("lesson: not a vocabulary word of the lesson")

var (
	// ErrCodexUnavailable means the admin chose Codex but this server does not run it.
	ErrCodexUnavailable = errors.New("lesson: codex images not enabled on this server")
	// ErrCodexNotLoggedIn means codex runs but is not logged in to ChatGPT.
	ErrCodexNotLoggedIn = errors.New("lesson: codex not logged in")
	// ErrDrawFailed means the AI answered without a usable picture.
	ErrDrawFailed = errors.New("lesson: drawing failed")
)

// ImageEngine is what draws the scene picture: Gemini (the default) or the Codex CLI of the server.
type ImageEngine string

const (
	// EngineGemini draws with GEMINI_IMAGE_MODEL.
	EngineGemini ImageEngine = "gemini"
	// EngineCodex draws with the Codex CLI (CODEX_IMAGES=true).
	EngineCodex ImageEngine = "codex"
)

// ImageSettings say how the AI draws the scene picture of a lesson. The zero value uses the
// default style and lets the AI choose the scene from the story.
type ImageSettings struct {
	// SceneDescription is what the scene picture shows, in English; "" lets the AI choose from the story.
	SceneDescription string
	// Style is the admin's description of the picture; "" uses the default style.
	Style     string
	UpdatedAt time.Time
}

// WordImage is the picture of one vocabulary word of a lesson.
type WordImage struct {
	LessonID  string
	Lemma     string
	MIME      string
	Data      []byte
	CreatedAt time.Time
}

// ImageRepository stores the picture settings of lessons and their word pictures (F23).
// Implementations live in internal/storage.
type ImageRepository interface {
	// Settings returns the zero ImageSettings for a lesson that never had any.
	Settings(ctx context.Context, lessonID string) (ImageSettings, error)
	SaveSettings(ctx context.Context, lessonID string, s ImageSettings) error
	// Lemmas lists the words of the lesson that have a picture.
	Lemmas(ctx context.Context, lessonID string) ([]string, error)
	// Get returns ErrImageNotFound when the word has no picture.
	Get(ctx context.Context, lessonID, lemma string) (WordImage, error)
	// Save stores a picture, replacing the word's previous one.
	Save(ctx context.Context, img WordImage) error
	// Delete removes the picture of one word; a word without one is not an error.
	Delete(ctx context.Context, lessonID, lemma string) error
	// DeleteForLesson removes the settings and every picture of a lesson.
	DeleteForLesson(ctx context.Context, lessonID string) error
}

// ImageInput is what an admin chooses for the scene picture of a lesson.
type ImageInput struct {
	SceneDescription string
	Style            string
}

// ImageState is the picture settings of a lesson with how many words have one, and its
// vocabulary words (empty until annotated).
type ImageState struct {
	ImageSettings
	Count int
	Words []ImageWord
	// BankCount is how many words without a picture of their own show the word bank's.
	BankCount int
	// HasScene says the lesson has its scene picture.
	HasScene bool
}

// ImageWord is one vocabulary word of a lesson and whether it has a picture.
type ImageWord struct {
	Lemma     string
	MeaningVi string
	HasImage  bool
	// FromBank says the word has no picture of its own but the word bank has one (F24), which the
	// learner sees (added 2026-11-05).
	FromBank bool
}

func validateImageInput(in ImageInput) (ImageInput, error) {
	in.Style = strings.TrimSpace(in.Style)
	in.SceneDescription = strings.TrimSpace(in.SceneDescription)
	fields := map[string]string{}
	if utf8.RuneCountInString(in.Style) > MaxImageStyle {
		fields["imageStyle"] = fmt.Sprintf("Mô tả ảnh tối đa %d ký tự", MaxImageStyle)
	}
	if utf8.RuneCountInString(in.SceneDescription) > MaxSceneDescription {
		fields["sceneDescription"] = fmt.Sprintf("Mô tả cảnh tối đa %d ký tự", MaxSceneDescription)
	}
	if len(fields) > 0 {
		return in, &ValidationError{Fields: fields}
	}
	return in, nil
}

// pictureKeys splits the keys of a lesson's stored pictures into its word lemmas and whether it
// has the scene picture.
func pictureKeys(keys []string) (lemmas []string, scene bool) {
	lemmas = make([]string, 0, len(keys))
	for _, k := range keys {
		if k == SceneKey {
			scene = true
			continue
		}
		lemmas = append(lemmas, k)
	}
	return lemmas, scene
}

// Images returns the picture settings of a lesson and its vocabulary words with whether each has
// a picture.
func (s *Service) Images(ctx context.Context, id string) (ImageState, error) {
	l, err := s.Lessons.Get(ctx, id)
	if err != nil {
		return ImageState{}, err
	}
	if s.ImageStore == nil {
		return ImageState{Words: []ImageWord{}}, nil
	}
	set, err := s.ImageStore.Settings(ctx, id)
	if err != nil {
		return ImageState{}, fmt.Errorf("lesson: image settings: %w", err)
	}
	keys, err := s.ImageStore.Lemmas(ctx, id)
	if err != nil {
		return ImageState{}, fmt.Errorf("lesson: image lemmas: %w", err)
	}
	lemmas, scene := pictureKeys(keys)
	words := []ImageWord{}
	if l.AnnotationStatus == StatusDone {
		for _, w := range vocabularyWords(l) {
			words = append(words, ImageWord{Lemma: w.Lemma, MeaningVi: w.MeaningVi, HasImage: slices.Contains(lemmas, w.Lemma)})
		}
	}
	bankCount, err := s.markBankImages(ctx, words)
	if err != nil {
		return ImageState{}, err
	}
	return ImageState{ImageSettings: set, Count: len(lemmas), Words: words, BankCount: bankCount, HasScene: scene}, nil
}

// markBankImages marks the words without a picture of their own that have one in the word bank,
// which the Vocabulary step shows instead, and counts them.
func (s *Service) markBankImages(ctx context.Context, words []ImageWord) (int, error) {
	lemmas := make([]string, len(words))
	for i, w := range words {
		lemmas[i] = w.Lemma
	}
	pictured, err := s.bankPictured(ctx, lemmas)
	if err != nil {
		return 0, err
	}
	n := 0
	for i := range words {
		if pictured[words[i].Lemma] && !words[i].HasImage {
			words[i].FromBank = true
			n++
		}
	}
	return n, nil
}

// bankPictured says which of lemmas have a picture in the word bank (none without a bank).
func (s *Service) bankPictured(ctx context.Context, lemmas []string) (map[string]bool, error) {
	out := map[string]bool{}
	if s.Bank == nil || len(lemmas) == 0 {
		return out, nil
	}
	bank, err := s.Bank.Words(ctx, lemmas)
	if err != nil {
		return nil, fmt.Errorf("lesson: word bank: %w", err)
	}
	for lemma, w := range bank {
		if w.HasImage {
			out[lemma] = true
		}
	}
	return out, nil
}

// UploadImage stores a picture chosen by an admin for one vocabulary word (or the scene),
// replacing the previous one. It is scaled down and kept as JPEG.
func (s *Service) UploadImage(ctx context.Context, id, lemma string, data []byte) error {
	if s.ImageStore == nil {
		return ErrImagesUnavailable
	}
	l, err := s.Lessons.Get(ctx, id)
	if err != nil {
		return err
	}
	lemma, err = lessonWord(l, lemma)
	if err != nil {
		return err
	}
	if len(data) > MaxUploadBytes {
		return &ValidationError{Fields: map[string]string{"image": fmt.Sprintf("Ảnh tối đa %d MB", MaxUploadBytes>>20)}}
	}
	small, err := shrinkFor(lemma, data)
	if err != nil {
		return &ValidationError{Fields: map[string]string{"image": "Không đọc được ảnh. Hãy dùng ảnh JPEG, PNG hoặc GIF."}}
	}
	return s.ImageStore.Save(ctx, WordImage{LessonID: l.ID, Lemma: lemma, MIME: "image/jpeg", Data: small, CreatedAt: s.Now()})
}

// DeleteImage removes the picture of one vocabulary word (or the scene). A word then shows the
// word bank's picture, if it has one.
func (s *Service) DeleteImage(ctx context.Context, id, lemma string) error {
	if s.ImageStore == nil {
		return ErrImagesUnavailable
	}
	l, err := s.Lessons.Get(ctx, id)
	if err != nil {
		return err
	}
	if lemma, err = lessonWord(l, lemma); err != nil {
		return err
	}
	return s.ImageStore.Delete(ctx, l.ID, lemma)
}

// AdminImage returns the picture of one word of a lesson, for the admin page: its own, else the
// word bank's.
func (s *Service) AdminImage(ctx context.Context, id, lemma string) (WordImage, error) {
	lemma = normalize(lemma)
	if s.ImageStore != nil {
		img, err := s.ImageStore.Get(ctx, id, lemma)
		if !errors.Is(err, ErrImageNotFound) {
			return img, err
		}
	}
	// A word without its own picture shows the word bank's, as on the learner side.
	if s.Bank == nil || lemma == SceneKey {
		return WordImage{}, ErrImageNotFound
	}
	return s.Bank.Image(ctx, lemma)
}

// lessonWord returns lemma in its stored form if it is one of the lesson's vocabulary words, or
// the scene key: every picture route of a word also serves the scene picture.
func lessonWord(l Lesson, lemma string) (string, error) {
	lemma = normalize(lemma)
	if lemma == SceneKey {
		return lemma, nil
	}
	if l.AnnotationStatus == StatusDone && lemma != "" {
		for _, w := range vocabularyWords(l) {
			if w.Lemma == lemma {
				return lemma, nil
			}
		}
	}
	return "", ErrNotLessonWord
}

// SetImages saves the description and style of a lesson's scene picture, without drawing it.
func (s *Service) SetImages(ctx context.Context, id string, in ImageInput) (ImageState, error) {
	if _, err := s.saveImageSettings(ctx, id, in); err != nil {
		return ImageState{}, err
	}
	return s.Images(ctx, id)
}

// saveImageSettings checks and stores the scene settings of a lesson, and returns the lesson.
func (s *Service) saveImageSettings(ctx context.Context, id string, in ImageInput) (Lesson, error) {
	if s.ImageStore == nil {
		return Lesson{}, ErrImagesUnavailable
	}
	in, err := validateImageInput(in)
	if err != nil {
		return Lesson{}, err
	}
	l, err := s.Lessons.Get(ctx, id)
	if err != nil {
		return Lesson{}, err
	}
	set := ImageSettings{SceneDescription: in.SceneDescription, Style: in.Style, UpdatedAt: s.Now()}
	if err := s.ImageStore.SaveSettings(ctx, l.ID, set); err != nil {
		return Lesson{}, fmt.Errorf("lesson: save image settings: %w", err)
	}
	return l, nil
}

// CodexAvailable tells whether this server can draw with the Codex CLI.
func (s *Service) CodexAvailable() bool { return s.CodexImageAI != nil }

// imageDrawer is the AI that engine names.
func (s *Service) imageDrawer(engine ImageEngine) (ai.ImageProvider, error) {
	switch engine {
	case "", EngineGemini:
		if s.ImageAI == nil {
			return nil, ErrImagesUnavailable
		}
		return s.ImageAI, nil
	case EngineCodex:
		if s.CodexImageAI == nil {
			return nil, ErrCodexUnavailable
		}
		return s.CodexImageAI, nil
	default:
		return nil, &ValidationError{Fields: map[string]string{"engine": "Cách sinh ảnh không hợp lệ"}}
	}
}

// DrawScene saves the scene settings of a lesson, then draws its scene picture now with engine
// (one AI request, answered once the picture is saved), replacing the one it has. The lesson must
// be annotated: the picture must not show the answers to the questions on the story. The settings
// stay saved when the drawing fails.
func (s *Service) DrawScene(ctx context.Context, id string, in ImageInput, engine ImageEngine) (ImageState, error) {
	drawer, err := s.imageDrawer(engine)
	if err != nil {
		return ImageState{}, err
	}
	l, err := s.saveImageSettings(ctx, id, in)
	if err != nil {
		return ImageState{}, err
	}
	if err := s.drawScene(ctx, l, drawer, engine); err != nil {
		return ImageState{}, err
	}
	return s.Images(ctx, l.ID)
}

// drawScene draws the scene picture of lesson l with its saved settings and stores it, replacing
// the one it has. The lesson must be annotated.
func (s *Service) drawScene(ctx context.Context, l Lesson, drawer ai.ImageProvider, engine ImageEngine) error {
	if l.AnnotationStatus != StatusDone {
		return ErrAnnotationNotDone
	}
	set, err := s.ImageStore.Settings(ctx, l.ID)
	if err != nil {
		return fmt.Errorf("lesson: image settings: %w", err)
	}
	img, err := drawer.GenerateImage(ctx, ai.ImageRequest{Style: set.Style, Scene: sceneBrief(l, set.SceneDescription)})
	if engine == EngineCodex && errors.Is(err, ai.ErrNotConfigured) {
		return fmt.Errorf("%w: %w", ErrCodexNotLoggedIn, err)
	}
	switch {
	case errors.Is(err, ai.ErrNotConfigured), errors.Is(err, ai.ErrInvalidKey), errors.Is(err, ai.ErrQuota),
		errors.Is(err, ai.ErrNoImageTool), errors.Is(err, context.DeadlineExceeded):
		return err
	case err != nil:
		return fmt.Errorf("%w: %w", ErrDrawFailed, err)
	}
	data, err := shrinkFor(SceneKey, img.Data)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrDrawFailed, err)
	}
	if err := s.ImageStore.Save(ctx, WordImage{LessonID: l.ID, Lemma: SceneKey, MIME: "image/jpeg", Data: data, CreatedAt: s.Now()}); err != nil {
		return fmt.Errorf("lesson: save scene image: %w", err)
	}
	return nil
}

// HasScenes says which of the lessons have their scene picture (for the course page).
func (s *Service) HasScenes(ctx context.Context, ids []string) (map[string]bool, error) {
	out := map[string]bool{}
	if s.ImageStore == nil {
		return out, nil
	}
	for _, id := range ids {
		keys, err := s.ImageStore.Lemmas(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("lesson: image lemmas: %w", err)
		}
		if _, scene := pictureKeys(keys); scene {
			out[id] = true
		}
	}
	return out, nil
}

// ProcessImages finishes a picture job queued before 2026-11-06, or by an older server sharing the
// queue, without drawing: pictures are no longer drawn in the background.
func (s *Service) ProcessImages(context.Context, job.Job) error { return nil }

// sceneBrief is what the AI needs to draw the scene of a lesson: its story and the questions on it
// whose answers the picture must not show (the detail and true/false questions of Reading and
// Listening; a gist question may well be helped by the picture, a vocabulary one is about a word).
func sceneBrief(l Lesson, description string) *ai.SceneBrief {
	b := &ai.SceneBrief{Title: l.Title, Text: l.Content, Description: description}
	revealing := func(kind string) bool { return kind != "gist" && kind != "vocab" }
	for _, q := range l.Extras.Questions {
		if revealing(q.Kind) {
			b.Avoid = append(b.Avoid, q.Prompt)
		}
	}
	if l.Practice != nil && l.Practice.Steps != nil {
		for _, q := range l.Practice.Steps.ListeningQuestions {
			if revealing(q.Kind) {
				b.Avoid = append(b.Avoid, q.Prompt)
			}
		}
	}
	return b
}

// SetSceneDescription stores the description of a lesson's scene picture, keeping its other
// picture settings; an import gives it (added 2026-11-05).
func (s *Service) SetSceneDescription(ctx context.Context, id, description string) error {
	if s.ImageStore == nil {
		return nil
	}
	set, err := s.ImageStore.Settings(ctx, id)
	if err != nil {
		return fmt.Errorf("lesson: image settings: %w", err)
	}
	set.SceneDescription, set.UpdatedAt = strings.TrimSpace(description), s.Now()
	if err := s.ImageStore.SaveSettings(ctx, id, set); err != nil {
		return fmt.Errorf("lesson: save image settings: %w", err)
	}
	return nil
}

// shrinkFor scales a picture down for its place: the scene spans the screen, a word fills a card.
func shrinkFor(key string, data []byte) ([]byte, error) {
	if key == SceneKey {
		return shrinkImage(data, sceneSide)
	}
	return ShrinkImage(data)
}

// ErrImagesUnavailable means the server has no place to keep word pictures, or no AI to draw them.
var ErrImagesUnavailable = errors.New("lesson: word images are not available")

// vocabularyWords lists the lesson's words as the Vocabulary step shows them: one per base form,
// in the order they first appear, with the sentence they come from (no IPA, no dictionary).
func vocabularyWords(l Lesson) []VocabItem {
	anns := slices.Clone(l.Annotations)
	slices.SortStableFunc(anns, func(a, b Annotation) int { return a.SentenceIndex - b.SentenceIndex })
	seen := map[string]bool{}
	out := []VocabItem{}
	for _, a := range anns {
		lemma := normalize(a.Lemma)
		if lemma == "" || seen[lemma] {
			continue
		}
		seen[lemma] = true
		item := VocabItem{Lemma: lemma, Text: a.Text, MeaningVi: a.MeaningVi, SentenceIndex: a.SentenceIndex, POS: a.POS}
		if a.SentenceIndex >= 0 && a.SentenceIndex < len(l.Sentences) {
			item.Sentence = l.Sentences[a.SentenceIndex].Text
		}
		out = append(out, item)
	}
	return out
}

// ShrinkImage scales a picture down so its longest side is at most imageSide and encodes it as
// JPEG on white: a word card needs no more, and the database keeps tens of kilobytes instead of megabytes.
func ShrinkImage(data []byte) ([]byte, error) {
	return shrinkImage(data, imageSide)
}

func shrinkImage(data []byte, side int) ([]byte, error) { return imageutil.Shrink(data, side) }
