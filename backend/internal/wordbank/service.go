package wordbank

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/luongtran/luna/backend/internal/ai"
	"github.com/luongtran/luna/backend/internal/dictionary"
	"github.com/luongtran/luna/backend/internal/lesson"
)

// TopicWords gives the word lists of the topics (F18): the source of Import, of the topics shown
// for each word, and where Add puts a word into a topic.
type TopicWords interface {
	All(ctx context.Context) ([]TopicList, error)
	// AddWord appends text (for every level) to the topic's list. It returns ErrInTopic when the
	// list holds it already, a ValidationError keyed "lemma" for a word the list does not accept,
	// and keyed "topicId" when there is no such topic.
	AddWord(ctx context.Context, topicID, text string) error
}

// Deps are the collaborators of Service.
type Deps struct {
	Repo Repository
	// Dict fills the IPA and meaning of new words; dictionary.None leaves them empty.
	Dict   dictionary.Resolver
	Topics TopicWords
	// ImageAI draws a picture on request; nil (or ai.DisabledImages) turns drawing off.
	ImageAI ai.ImageProvider
	// MeaningAI fills the missing meanings of a topic on request; nil turns it off.
	MeaningAI ai.MeaningProvider
	// FetchImage downloads the picture behind a link.
	FetchImage lesson.ImageFetcher
	Now        func() time.Time
	Log        *slog.Logger
}

// Service manages the word bank (F24). Every method is for admins except Words and Image, which
// the lessons use.
type Service struct {
	repo      Repository
	dict      dictionary.Resolver
	topics    TopicWords
	imageAI   ai.ImageProvider
	meaningAI ai.MeaningProvider
	fetch     lesson.ImageFetcher
	now       func() time.Time
	log       *slog.Logger
}

// NewService returns a Service.
func NewService(d Deps) *Service {
	return &Service{
		repo: d.Repo, dict: d.Dict, topics: d.Topics, imageAI: d.ImageAI, meaningAI: d.MeaningAI, fetch: d.FetchImage, now: d.Now, log: d.Log,
	}
}

// Input is a new word as an admin types it; empty IPA and meaning are filled from the dictionary.
type Input struct {
	Lemma     string
	MeaningVi string
	IPA       string
	// TopicID, if given, is the topic whose word list also gets the word.
	TopicID string
}

// Added tells what Add did besides returning the word.
type Added struct {
	// InBank is true when the word was in the bank already (only possible with a topic).
	InBank bool
	// Topic is the topic the word was added to, empty without one.
	Topic TopicRef
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

// List returns page (from 1) of the words matching search and missing, each with the topics
// holding it; a topicID keeps only the words of that topic.
func (s *Service) List(ctx context.Context, search string, missing Missing, topicID string, page int) (Page, error) {
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
	topics, err := s.topics.All(ctx)
	if err != nil {
		return Page{}, fmt.Errorf("wordbank: topic words: %w", err)
	}
	q := ListQuery{Search: search, Missing: missing, Limit: PageSize, Skip: (page - 1) * PageSize}
	if topicID != "" {
		t, ok := findTopic(topics, topicID)
		if !ok {
			return Page{}, &ValidationError{Fields: map[string]string{"topicId": msgNoTopic}}
		}
		q.Lemmas = lemmasOf(t)
	}
	words, total, err := s.repo.List(ctx, q)
	if err != nil {
		return Page{}, fmt.Errorf("wordbank: list: %w", err)
	}
	holders := topicsByLemma(topics)
	for i := range words {
		words[i].Topics = holders[words[i].Lemma]
	}
	return Page{Words: words, Total: total, HasMore: q.Skip+len(words) < total}, nil
}

const msgNoTopic = "Chủ đề không tồn tại"

func findTopic(topics []TopicList, id string) (TopicList, bool) {
	for _, t := range topics {
		if t.ID == id {
			return t, true
		}
	}
	return TopicList{}, false
}

// lemmasOf returns the normalized words of a topic, never nil.
func lemmasOf(t TopicList) []string {
	out := make([]string, 0, len(t.Words))
	for _, w := range t.Words {
		if l := Normalize(w); l != "" {
			out = append(out, l)
		}
	}
	return out
}

// topicsByLemma maps each normalized topic word to the topics holding it, by name.
func topicsByLemma(topics []TopicList) map[string][]TopicRef {
	out := map[string][]TopicRef{}
	for _, t := range topics {
		for _, l := range lemmasOf(t) {
			if !slices.Contains(out[l], t.TopicRef) {
				out[l] = append(out[l], t.TopicRef)
			}
		}
	}
	for _, refs := range out {
		slices.SortFunc(refs, func(a, b TopicRef) int { return strings.Compare(a.Name, b.Name) })
	}
	return out
}

// Get returns one word, or ErrNotFound.
func (s *Service) Get(ctx context.Context, lemma string) (Word, error) {
	return s.repo.Get(ctx, Normalize(lemma))
}

// Add puts a new word in the bank; ErrExists when it is there already.
//
// With a topic, the word also goes into the topic's list: ErrInTopic when the list holds it
// already, and a word already in the bank is not an error then (Added.InBank).
func (s *Service) Add(ctx context.Context, in Input) (Word, Added, error) {
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
		return Word{}, Added{}, &ValidationError{Fields: fields}
	}
	topicID := strings.TrimSpace(in.TopicID)
	if topicID == "" {
		w, err := s.create(ctx, w)
		return w, Added{}, err
	}

	topics, err := s.topics.All(ctx)
	if err != nil {
		return Word{}, Added{}, fmt.Errorf("wordbank: topic words: %w", err)
	}
	t, ok := findTopic(topics, topicID)
	if !ok {
		return Word{}, Added{}, &ValidationError{Fields: map[string]string{"topicId": msgNoTopic}}
	}
	added := Added{Topic: t.TopicRef}
	if slices.Contains(lemmasOf(t), w.Lemma) {
		return Word{}, added, ErrInTopic
	}
	if err := s.topics.AddWord(ctx, t.ID, w.Lemma); err != nil {
		return Word{}, added, err
	}
	have, err := s.repo.Get(ctx, w.Lemma)
	switch {
	case err == nil:
		added.InBank = true
		return have, added, nil
	case !errors.Is(err, ErrNotFound):
		return Word{}, added, fmt.Errorf("wordbank: get: %w", err)
	}
	created, err := s.create(ctx, w)
	if errors.Is(err, ErrExists) { // added meanwhile
		added.InBank = true
		created, err = s.repo.Get(ctx, w.Lemma)
	}
	return created, added, err
}

// create fills a new word from the dictionary and stores it; ErrExists when it is there already.
func (s *Service) create(ctx context.Context, w Word) (Word, error) {
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
	topics, err := s.topics.All(ctx)
	if err != nil {
		return 0, fmt.Errorf("wordbank: topic words: %w", err)
	}
	seen := map[string]bool{}
	var lemmas []string
	for _, t := range topics {
		for _, lemma := range lemmasOf(t) {
			if seen[lemma] || utf8.RuneCountInString(lemma) > MaxLemma {
				continue
			}
			seen[lemma] = true
			lemmas = append(lemmas, lemma)
		}
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

// Filled tells what FillMissing did.
type Filled struct {
	// Asked is how many words lacking a meaning or an IPA went to the AI.
	Asked int
	// Meanings and IPAs count the words that got one.
	Meanings int
	IPAs     int
}

// FillMissing asks the AI, in one request, for what the topic's words in the bank lack: the
// Vietnamese meaning, the IPA or both, marked per word so that nothing they have is asked again.
// Only empty fields are saved. A topic without such words makes no request.
func (s *Service) FillMissing(ctx context.Context, topicID string) (Filled, error) {
	topics, err := s.topics.All(ctx)
	if err != nil {
		return Filled{}, fmt.Errorf("wordbank: topic words: %w", err)
	}
	t, ok := findTopic(topics, strings.TrimSpace(topicID))
	if !ok {
		return Filled{}, &ValidationError{Fields: map[string]string{"topicId": msgNoTopic}}
	}
	lemmas := lemmasOf(t)
	if len(lemmas) == 0 {
		return Filled{}, nil
	}
	byLemma := map[string]Word{}
	var order []string
	for _, m := range []Missing{MissingMeaning, MissingIPA} {
		words, _, err := s.repo.List(ctx, ListQuery{Missing: m, Lemmas: lemmas, Limit: len(lemmas)})
		if err != nil {
			return Filled{}, fmt.Errorf("wordbank: list: %w", err)
		}
		for _, w := range words {
			if _, seen := byLemma[w.Lemma]; !seen {
				byLemma[w.Lemma] = w
				order = append(order, w.Lemma)
			}
		}
	}
	if len(order) == 0 {
		return Filled{}, nil
	}
	if s.meaningAI == nil {
		return Filled{}, ai.ErrNotConfigured
	}
	slices.Sort(order)
	need := make([]ai.WordNeed, len(order))
	for i, l := range order {
		w := byLemma[l]
		need[i] = ai.WordNeed{Word: l, Meaning: w.MeaningVi == "", IPA: w.IPA == ""}
	}
	answers, err := s.meaningAI.WordMeanings(ctx, ai.MeaningsRequest{TopicName: t.Name, Words: need})
	switch {
	case errors.Is(err, ai.ErrNotConfigured), errors.Is(err, ai.ErrInvalidKey), errors.Is(err, ai.ErrQuota):
		return Filled{}, err
	case err != nil:
		return Filled{}, fmt.Errorf("%w: %w", ErrMeaningsFailed, err)
	}
	out := Filled{Asked: len(order)}
	now := s.now()
	for _, a := range answers {
		w, ok := byLemma[Normalize(a.Word)]
		if !ok {
			continue
		}
		delete(byLemma, w.Lemma) // a word answered twice is saved once
		gotMeaning, gotIPA := false, false
		if m := strings.TrimSpace(a.MeaningVi); w.MeaningVi == "" && m != "" && utf8.RuneCountInString(m) <= MaxMeaning {
			w.MeaningVi, gotMeaning = m, true
		}
		if ipa := strings.TrimSpace(a.IPA); w.IPA == "" && ipa != "" && utf8.RuneCountInString(ipa) <= MaxIPA {
			w.IPA, gotIPA = ipa, true
		}
		if !gotMeaning && !gotIPA {
			continue
		}
		w.UpdatedAt = now
		if err := s.repo.Update(ctx, w); err != nil {
			if errors.Is(err, ErrNotFound) { // deleted meanwhile
				continue
			}
			return out, fmt.Errorf("wordbank: update: %w", err)
		}
		if gotMeaning {
			out.Meanings++
		}
		if gotIPA {
			out.IPAs++
		}
	}
	if out.Meanings+out.IPAs == 0 {
		return Filled{}, ErrMeaningsFailed
	}
	return out, nil
}
