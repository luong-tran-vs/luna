package wordbank

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/luongtran/luna/backend/internal/ai"
	"github.com/luongtran/luna/backend/internal/dictionary"
	"github.com/luongtran/luna/backend/internal/lesson"
)

// TopicWords gives the words of every topic (F18), the source of Import.
type TopicWords interface {
	All(ctx context.Context) ([]string, error)
}

// Deps are the collaborators of Service.
type Deps struct {
	Repo Repository
	// Dict fills the IPA and meaning of new words; dictionary.None leaves them empty.
	Dict   dictionary.Resolver
	Topics TopicWords
	// ImageAI draws a picture on request; nil (or ai.DisabledImages) turns drawing off.
	ImageAI ai.ImageProvider
	// FetchImage downloads the picture behind a link.
	FetchImage lesson.ImageFetcher
	Now        func() time.Time
	Log        *slog.Logger
}

// Service manages the word bank (F24). Every method is for admins except Words and Image, which
// the lessons use.
type Service struct {
	repo    Repository
	dict    dictionary.Resolver
	topics  TopicWords
	imageAI ai.ImageProvider
	fetch   lesson.ImageFetcher
	now     func() time.Time
	log     *slog.Logger
}

// NewService returns a Service.
func NewService(d Deps) *Service {
	return &Service{
		repo: d.Repo, dict: d.Dict, topics: d.Topics, imageAI: d.ImageAI, fetch: d.FetchImage, now: d.Now, log: d.Log,
	}
}

// Input is a new word as an admin types it; empty IPA and meaning are filled from the dictionary.
type Input struct {
	Lemma     string
	MeaningVi string
	IPA       string
}

// Details are the fields of a word an admin can edit; nil fields are unchanged.
type Details struct {
	MeaningVi *string
	IPA       *string
}

// Page is one page of the admin list.
type Page struct {
	Words   []Word
	Total   int
	HasMore bool
}

// List returns page (from 1) of the words matching search and missing.
func (s *Service) List(ctx context.Context, search string, missing Missing, page int) (Page, error) {
	fields := map[string]string{}
	search = strings.Join(strings.Fields(search), " ")
	if utf8.RuneCountInString(search) > MaxSearch {
		fields["q"] = fmt.Sprintf("Từ khoá tối đa %d ký tự", MaxSearch)
	}
	switch missing {
	case MissingNone, MissingImage, MissingIPA, MissingMeaning:
	default:
		fields["missing"] = "Bộ lọc không hợp lệ"
	}
	if page < 1 {
		fields["page"] = "Trang không hợp lệ"
	}
	if len(fields) > 0 {
		return Page{}, &ValidationError{Fields: fields}
	}
	q := ListQuery{Search: search, Missing: missing, Limit: PageSize, Skip: (page - 1) * PageSize}
	words, total, err := s.repo.List(ctx, q)
	if err != nil {
		return Page{}, fmt.Errorf("wordbank: list: %w", err)
	}
	return Page{Words: words, Total: total, HasMore: q.Skip+len(words) < total}, nil
}

// Get returns one word, or ErrNotFound.
func (s *Service) Get(ctx context.Context, lemma string) (Word, error) {
	return s.repo.Get(ctx, Normalize(lemma))
}

// Add puts a new word in the bank; ErrExists when it is there already.
func (s *Service) Add(ctx context.Context, in Input) (Word, error) {
	w := Word{
		Lemma: Normalize(in.Lemma), MeaningVi: strings.TrimSpace(in.MeaningVi), IPA: strings.TrimSpace(in.IPA),
	}
	fields := map[string]string{}
	switch n := utf8.RuneCountInString(w.Lemma); {
	case n == 0:
		fields["lemma"] = "Vui lòng nhập từ"
	case n > MaxLemma:
		fields["lemma"] = fmt.Sprintf("Từ tối đa %d ký tự", MaxLemma)
	}
	checkDetails(fields, &w.MeaningVi, &w.IPA)
	if len(fields) > 0 {
		return Word{}, &ValidationError{Fields: fields}
	}
	w, err := s.fill(ctx, w)
	if err != nil {
		return Word{}, err
	}
	w.CreatedAt = s.now()
	w.UpdatedAt = w.CreatedAt
	if err := s.repo.Create(ctx, w); err != nil {
		if errors.Is(err, ErrExists) {
			return Word{}, err
		}
		return Word{}, fmt.Errorf("wordbank: create: %w", err)
	}
	return w, nil
}

func checkDetails(fields map[string]string, meaning, ipa *string) {
	if meaning != nil && utf8.RuneCountInString(*meaning) > MaxMeaning {
		fields["meaningVi"] = fmt.Sprintf("Nghĩa tối đa %d ký tự", MaxMeaning)
	}
	if ipa != nil && utf8.RuneCountInString(*ipa) > MaxIPA {
		fields["ipa"] = fmt.Sprintf("Phiên âm tối đa %d ký tự", MaxIPA)
	}
}

// fill gives a word without IPA or meaning the dictionary's.
func (s *Service) fill(ctx context.Context, w Word) (Word, error) {
	if s.dict == nil {
		return w, nil
	}
	var err error
	if w.IPA == "" {
		if w.IPA, err = dictionary.IPA(ctx, s.dict, w.Lemma); err != nil {
			return Word{}, fmt.Errorf("wordbank: %w", err)
		}
	}
	if w.MeaningVi == "" {
		if w.MeaningVi, err = dictionary.MeaningVi(ctx, s.dict, w.Lemma); err != nil {
			return Word{}, fmt.Errorf("wordbank: %w", err)
		}
		if utf8.RuneCountInString(w.MeaningVi) > MaxMeaning {
			w.MeaningVi = string([]rune(w.MeaningVi)[:MaxMeaning])
		}
	}
	return w, nil
}

// Update edits the meaning or IPA of a word; the word itself never changes.
func (s *Service) Update(ctx context.Context, lemma string, d Details) (Word, error) {
	if d.MeaningVi != nil {
		v := strings.TrimSpace(*d.MeaningVi)
		d.MeaningVi = &v
	}
	if d.IPA != nil {
		v := strings.TrimSpace(*d.IPA)
		d.IPA = &v
	}
	fields := map[string]string{}
	checkDetails(fields, d.MeaningVi, d.IPA)
	if len(fields) > 0 {
		return Word{}, &ValidationError{Fields: fields}
	}
	w, err := s.repo.Get(ctx, Normalize(lemma))
	if err != nil {
		return Word{}, err
	}
	if d.MeaningVi != nil {
		w.MeaningVi = *d.MeaningVi
	}
	if d.IPA != nil {
		w.IPA = *d.IPA
	}
	w.UpdatedAt = s.now()
	if err := s.repo.Update(ctx, w); err != nil {
		return Word{}, err
	}
	return w, nil
}

// Delete removes a word and its picture. Lessons keep their own pictures and meanings.
func (s *Service) Delete(ctx context.Context, lemma string) error {
	return s.repo.Delete(ctx, Normalize(lemma))
}

// Import adds every topic word that is not in the bank yet, with the dictionary's IPA and meaning,
// and returns how many it added.
func (s *Service) Import(ctx context.Context) (int, error) {
	texts, err := s.topics.All(ctx)
	if err != nil {
		return 0, fmt.Errorf("wordbank: topic words: %w", err)
	}
	seen := map[string]bool{}
	var lemmas []string
	for _, t := range texts {
		lemma := Normalize(t)
		if lemma == "" || seen[lemma] || utf8.RuneCountInString(lemma) > MaxLemma {
			continue
		}
		seen[lemma] = true
		lemmas = append(lemmas, lemma)
	}
	have, err := s.repo.Find(ctx, lemmas)
	if err != nil {
		return 0, fmt.Errorf("wordbank: find: %w", err)
	}
	for _, w := range have {
		seen[w.Lemma] = false
	}
	now := s.now()
	var add []Word
	for _, lemma := range lemmas {
		if !seen[lemma] {
			continue
		}
		w, err := s.fill(ctx, Word{Lemma: lemma, CreatedAt: now, UpdatedAt: now})
		if err != nil {
			return 0, err
		}
		add = append(add, w)
	}
	if len(add) == 0 {
		return 0, nil
	}
	n, err := s.repo.InsertMissing(ctx, add)
	if err != nil {
		return 0, fmt.Errorf("wordbank: insert: %w", err)
	}
	return n, nil
}

// UploadImage stores a picture chosen by an admin (JPEG, PNG or GIF, at most
// lesson.MaxUploadBytes), scaled down and kept as JPEG, replacing the word's picture.
func (s *Service) UploadImage(ctx context.Context, lemma string, data []byte) (Word, error) {
	if len(data) > lesson.MaxUploadBytes {
		return Word{}, &ValidationError{Fields: map[string]string{"image": fmt.Sprintf("Ảnh tối đa %d MB", lesson.MaxUploadBytes>>20)}}
	}
	small, err := lesson.ShrinkImage(data)
	if err != nil {
		return Word{}, &ValidationError{Fields: map[string]string{"image": "Không đọc được ảnh. Hãy dùng ảnh JPEG, PNG hoặc GIF."}}
	}
	return s.saveImage(ctx, lemma, small)
}

// ImportImage downloads the picture behind a link and stores it like an uploaded one.
func (s *Service) ImportImage(ctx context.Context, lemma, rawURL string) (Word, error) {
	if s.fetch == nil {
		return Word{}, ErrImagesUnavailable
	}
	if _, err := s.repo.Get(ctx, Normalize(lemma)); err != nil {
		return Word{}, err
	}
	data, err := lesson.FetchImageLink(ctx, s.fetch, rawURL)
	var lerr *lesson.ValidationError
	if errors.As(err, &lerr) {
		if cause := errors.Unwrap(err); cause != nil {
			s.log.InfoContext(ctx, "wordbank: fetch word image", "error", cause)
		}
		return Word{}, &ValidationError{Fields: lerr.Fields}
	}
	if err != nil {
		return Word{}, err
	}
	small, err := lesson.ShrinkImage(data)
	if err != nil {
		return Word{}, &ValidationError{Fields: map[string]string{"url": lesson.NotAnImageMessage}}
	}
	return s.saveImage(ctx, lemma, small)
}

// GenerateImage draws a picture of the word with the AI (one paid request) and stores it,
// replacing the word's picture. style, if given, describes how it should look.
func (s *Service) GenerateImage(ctx context.Context, lemma, style string) (Word, error) {
	style = strings.TrimSpace(style)
	if utf8.RuneCountInString(style) > lesson.MaxImageStyle {
		return Word{}, &ValidationError{Fields: map[string]string{"style": fmt.Sprintf("Mô tả ảnh tối đa %d ký tự", lesson.MaxImageStyle)}}
	}
	if s.imageAI == nil {
		return Word{}, ErrImagesUnavailable
	}
	w, err := s.repo.Get(ctx, Normalize(lemma))
	if err != nil {
		return Word{}, err
	}
	img, err := s.imageAI.GenerateImage(ctx, ai.ImageRequest{Word: w.Lemma, MeaningVi: w.MeaningVi, Style: style})
	switch {
	case errors.Is(err, ai.ErrNotConfigured), errors.Is(err, ai.ErrInvalidKey), errors.Is(err, ai.ErrQuota):
		return Word{}, err
	case err != nil:
		return Word{}, fmt.Errorf("%w: %w", ErrDrawFailed, err)
	}
	small, err := lesson.ShrinkImage(img.Data)
	if err != nil {
		return Word{}, fmt.Errorf("%w: %w", ErrDrawFailed, err)
	}
	return s.saveImage(ctx, w.Lemma, small)
}

func (s *Service) saveImage(ctx context.Context, lemma string, data []byte) (Word, error) {
	lemma = Normalize(lemma)
	if err := s.repo.SaveImage(ctx, Image{Lemma: lemma, MIME: "image/jpeg", Data: data, CreatedAt: s.now()}); err != nil {
		if errors.Is(err, ErrNotFound) {
			return Word{}, err
		}
		return Word{}, fmt.Errorf("wordbank: save image: %w", err)
	}
	return s.repo.Get(ctx, lemma)
}

// DeleteImage removes the picture of a word.
func (s *Service) DeleteImage(ctx context.Context, lemma string) (Word, error) {
	lemma = Normalize(lemma)
	if _, err := s.repo.Get(ctx, lemma); err != nil {
		return Word{}, err
	}
	if err := s.repo.DeleteImage(ctx, lemma); err != nil {
		return Word{}, fmt.Errorf("wordbank: delete image: %w", err)
	}
	return s.repo.Get(ctx, lemma)
}

// Image returns the picture of a word, or ErrImageNotFound.
func (s *Service) Image(ctx context.Context, lemma string) (Image, error) {
	return s.repo.Image(ctx, Normalize(lemma))
}

// Words returns the entries of the bank among lemmas, by lemma (for the lessons).
func (s *Service) Words(ctx context.Context, lemmas []string) (map[string]Word, error) {
	keys := make([]string, 0, len(lemmas))
	for _, l := range lemmas {
		if k := Normalize(l); k != "" {
			keys = append(keys, k)
		}
	}
	out := map[string]Word{}
	if len(keys) == 0 {
		return out, nil
	}
	ws, err := s.repo.Find(ctx, keys)
	if err != nil {
		return nil, fmt.Errorf("wordbank: find: %w", err)
	}
	for _, w := range ws {
		out[w.Lemma] = w
	}
	return out, nil
}
