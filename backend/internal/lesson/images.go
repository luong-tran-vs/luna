package lesson

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	_ "image/gif" // decodes uploaded GIF pictures
	"image/jpeg"
	_ "image/png" // decodes PNG pictures, drawn or uploaded
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/luongtran/luna/backend/internal/ai"
	"github.com/luongtran/luna/backend/internal/job"
)

// MaxImageStyle is the longest style an admin may give for the pictures of a lesson (F23).
const MaxImageStyle = 500

// maxImagesPerJob bounds the pictures one job draws, so a long lesson cannot run up the bill.
const maxImagesPerJob = 30

// imageSide is the longest side of a stored picture, in pixels: enough for the word card on a phone.
const imageSide = 360

// jpegQuality is the JPEG quality of a stored picture (1–100): small files, no visible loss at card size.
const jpegQuality = 70

// MaxUploadBytes is the largest picture an admin may upload for a word.
const MaxUploadBytes = 5 << 20

// ErrImageNotFound means the word has no picture.
var ErrImageNotFound = errors.New("lesson: image not found")

// ErrNotLessonWord means the word is not one of the lesson's vocabulary words (or they are not
// annotated yet), so it cannot have a picture.
var ErrNotLessonWord = errors.New("lesson: not a vocabulary word of the lesson")

// ImageSettings say whether the vocabulary words of a lesson get a picture, in which style, and how
// the drawing went (F23). The zero value is a lesson without pictures.
type ImageSettings struct {
	Enabled bool
	// Style is the admin's description of the pictures; "" uses the default style.
	Style  string
	Status Status
	// Error is a short Vietnamese reason when Status is StatusFailed.
	Error     string
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

// ImageInput is what an admin chooses for the pictures of a lesson.
type ImageInput struct {
	Enabled bool
	Style   string
}

// ImageState is the picture settings of a lesson with how many words have one, and its
// vocabulary words (empty until annotated).
type ImageState struct {
	ImageSettings
	Count int
	Words []ImageWord
}

// ImageWord is one vocabulary word of a lesson and whether it has a picture.
type ImageWord struct {
	Lemma     string
	MeaningVi string
	HasImage  bool
}

func validateImageInput(in ImageInput) (ImageInput, error) {
	in.Style = strings.TrimSpace(in.Style)
	if utf8.RuneCountInString(in.Style) > MaxImageStyle {
		return in, &ValidationError{Fields: map[string]string{
			"imageStyle": fmt.Sprintf("Mô tả ảnh tối đa %d ký tự", MaxImageStyle),
		}}
	}
	return in, nil
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
	lemmas, err := s.ImageStore.Lemmas(ctx, id)
	if err != nil {
		return ImageState{}, fmt.Errorf("lesson: image lemmas: %w", err)
	}
	words := []ImageWord{}
	if l.AnnotationStatus == StatusDone {
		for _, w := range vocabularyWords(l) {
			words = append(words, ImageWord{Lemma: w.Lemma, MeaningVi: w.MeaningVi, HasImage: slices.Contains(lemmas, w.Lemma)})
		}
	}
	return ImageState{ImageSettings: set, Count: len(lemmas), Words: words}, nil
}

// UploadImage stores a picture chosen by an admin for one vocabulary word, replacing the word's
// picture (drawn or uploaded). Like a drawn one it is scaled down and kept as JPEG. The AI never
// draws over it: it only draws words without a picture.
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
	small, err := ShrinkImage(data)
	if err != nil {
		return &ValidationError{Fields: map[string]string{"image": "Không đọc được ảnh. Hãy dùng ảnh JPEG, PNG hoặc GIF."}}
	}
	return s.ImageStore.Save(ctx, WordImage{LessonID: l.ID, Lemma: lemma, MIME: "image/jpeg", Data: small, CreatedAt: s.Now()})
}

// DeleteImage removes the picture of one vocabulary word. If pictures are on, the AI draws the
// word again the next time the drawing runs.
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

// AdminImage returns the picture of one word of a lesson, for the admin page.
func (s *Service) AdminImage(ctx context.Context, id, lemma string) (WordImage, error) {
	if s.ImageStore == nil {
		return WordImage{}, ErrImageNotFound
	}
	return s.ImageStore.Get(ctx, id, normalize(lemma))
}

// lessonWord returns lemma in its stored form if it is one of the lesson's vocabulary words.
func lessonWord(l Lesson, lemma string) (string, error) {
	lemma = normalize(lemma)
	if l.AnnotationStatus == StatusDone && lemma != "" {
		for _, w := range vocabularyWords(l) {
			if w.Lemma == lemma {
				return lemma, nil
			}
		}
	}
	return "", ErrNotLessonWord
}

// SetImages turns the pictures of a lesson on or off. Turning them on (again) draws the words that
// have no picture yet, so it also retries a failed drawing; words keep the pictures they have.
func (s *Service) SetImages(ctx context.Context, id string, in ImageInput) (ImageState, error) {
	if s.ImageStore == nil {
		return ImageState{}, ErrImagesUnavailable
	}
	in, err := validateImageInput(in)
	if err != nil {
		return ImageState{}, err
	}
	l, err := s.Lessons.Get(ctx, id)
	if err != nil {
		return ImageState{}, err
	}
	if err := s.applyImages(ctx, l, in); err != nil {
		return ImageState{}, err
	}
	return s.Images(ctx, id)
}

// applyImages saves the settings and, when on and the words are known, queues the drawing. When the
// annotation is still running, ProcessAnnotate queues it once the words are there.
func (s *Service) applyImages(ctx context.Context, l Lesson, in ImageInput) error {
	set := ImageSettings{Enabled: in.Enabled, Style: in.Style, UpdatedAt: s.Now()}
	if in.Enabled {
		set.Status = StatusRunning
	}
	if err := s.ImageStore.SaveSettings(ctx, l.ID, set); err != nil {
		return fmt.Errorf("lesson: save image settings: %w", err)
	}
	if in.Enabled && l.AnnotationStatus == StatusDone {
		return s.enqueue(ctx, l.ID, l.Revision, job.TypeImages)
	}
	return nil
}

// queueImages queues the drawing of a lesson's pictures if they are on (after its words changed).
func (s *Service) queueImages(ctx context.Context, id string, revision int) error {
	if s.ImageStore == nil {
		return nil
	}
	set, err := s.ImageStore.Settings(ctx, id)
	if err != nil {
		return fmt.Errorf("lesson: image settings: %w", err)
	}
	if !set.Enabled {
		return nil
	}
	set.Status, set.Error, set.UpdatedAt = StatusRunning, "", s.Now()
	if err := s.ImageStore.SaveSettings(ctx, id, set); err != nil {
		return fmt.Errorf("lesson: save image settings: %w", err)
	}
	return s.enqueue(ctx, id, revision, job.TypeImages)
}

// ProcessImages draws a picture for each vocabulary word that has none, one AI request per word,
// at most maxImagesPerJob. Pictures are saved as they come, so a retry only draws the words still
// missing. Any word left without a picture fails the job, which the worker retries.
func (s *Service) ProcessImages(ctx context.Context, j job.Job) error {
	l, ok, err := s.current(ctx, j)
	if !ok {
		return err
	}
	if s.ImageStore == nil || s.ImageAI == nil {
		return job.Permanent(ErrImagesUnavailable)
	}
	set, err := s.ImageStore.Settings(ctx, l.ID)
	if err != nil {
		return fmt.Errorf("lesson: image settings: %w", err)
	}
	if !set.Enabled {
		return nil
	}
	if l.AnnotationStatus != StatusDone {
		return job.Permanent(ErrAnnotationNotDone)
	}
	have, err := s.ImageStore.Lemmas(ctx, l.ID)
	if err != nil {
		return fmt.Errorf("lesson: image lemmas: %w", err)
	}
	missing := 0
	drawn := 0
	for _, w := range vocabularyWords(l) {
		if slices.Contains(have, w.Lemma) {
			continue
		}
		if drawn >= maxImagesPerJob {
			break
		}
		drawn++
		if err := s.drawImage(ctx, l.ID, w, set.Style); err != nil {
			if errors.Is(err, ai.ErrNotConfigured) || errors.Is(err, ai.ErrInvalidKey) {
				return job.Permanent(err)
			}
			if errors.Is(err, ai.ErrQuota) {
				return err // the next words would fail too: retry later
			}
			s.Log.WarnContext(ctx, "lesson: draw word image", "lesson_id", l.ID, "error", err)
			missing++
		}
	}
	if missing > 0 {
		return fmt.Errorf("lesson: %d word images failed", missing)
	}
	set.Status, set.Error, set.UpdatedAt = StatusDone, "", s.Now()
	if err := s.ImageStore.SaveSettings(ctx, l.ID, set); err != nil {
		return fmt.Errorf("lesson: save image settings: %w", err)
	}
	return nil
}

func (s *Service) drawImage(ctx context.Context, lessonID string, w VocabItem, style string) error {
	img, err := s.ImageAI.GenerateImage(ctx, ai.ImageRequest{
		Word: w.Lemma, MeaningVi: w.MeaningVi, Sentence: w.Sentence, Style: style,
	})
	if err != nil {
		return err
	}
	data, err := ShrinkImage(img.Data)
	if err != nil {
		return err
	}
	return s.ImageStore.Save(ctx, WordImage{
		LessonID: lessonID, Lemma: w.Lemma, MIME: "image/jpeg", Data: data, CreatedAt: s.Now(),
	})
}

// imagesFailed records a drawing that gave up; the pictures already drawn stay.
func (s *Service) imagesFailed(ctx context.Context, j job.Job, err error) {
	if s.ImageStore == nil {
		return
	}
	set, serr := s.ImageStore.Settings(ctx, j.LessonID)
	if serr == nil {
		set.Status, set.Error, set.UpdatedAt = StatusFailed, failureMessage(j.Type, err), s.Now()
		serr = s.ImageStore.SaveSettings(ctx, j.LessonID, set)
	}
	if serr != nil {
		s.Log.ErrorContext(ctx, "record image job failure", "lesson_id", j.LessonID, "error", serr)
	}
}

// ErrImagesUnavailable means the server has no place to keep word pictures.
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
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("lesson: decode image: %w", err)
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w == 0 || h == 0 {
		return nil, errors.New("lesson: empty image")
	}
	scale := min(1, float64(imageSide)/float64(max(w, h)))
	dw, dh := max(1, int(float64(w)*scale)), max(1, int(float64(h)*scale))
	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))
	// Box filter: each target pixel is the mean of the source pixels it covers.
	for y := range dh {
		y0, y1 := b.Min.Y+y*h/dh, b.Min.Y+max((y+1)*h/dh, y*h/dh+1)
		for x := range dw {
			x0, x1 := b.Min.X+x*w/dw, b.Min.X+max((x+1)*w/dw, x*w/dw+1)
			var r, g, bl, a, n uint64
			for sy := y0; sy < y1; sy++ {
				for sx := x0; sx < x1; sx++ {
					cr, cg, cb, ca := src.At(sx, sy).RGBA()
					r, g, bl, a, n = r+uint64(cr), g+uint64(cg), bl+uint64(cb), a+uint64(ca), n+1
				}
			}
			// Premultiplied colors over a white background: JPEG has no transparency.
			white := 0xffff - a/n
			dst.Set(x, y, color.RGBA64{R: uint16(r/n + white), G: uint16(g/n + white), B: uint16(bl/n + white), A: 0xffff})
		}
	}
	var out bytes.Buffer
	if err := jpeg.Encode(&out, dst, &jpeg.Options{Quality: jpegQuality}); err != nil {
		return nil, fmt.Errorf("lesson: encode image: %w", err)
	}
	return out.Bytes(), nil
}
