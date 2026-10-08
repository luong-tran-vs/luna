package wordbank

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/png"
	"io"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/luongtran/luna/backend/internal/ai"
	"github.com/luongtran/luna/backend/internal/dictionary"
)

// fakeRepo keeps the bank in memory.
type fakeRepo struct {
	mu     sync.Mutex
	words  map[string]Word
	images map[string]Image
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{words: map[string]Word{}, images: map[string]Image{}}
}

func (f *fakeRepo) List(_ context.Context, q ListQuery) ([]Word, int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var all []Word
	for _, w := range f.words {
		s := strings.ToLower(q.Search)
		if s != "" && !strings.Contains(w.Lemma, s) && !strings.Contains(strings.ToLower(w.MeaningVi), s) {
			continue
		}
		switch {
		case q.Missing == MissingImage && w.HasImage(),
			q.Missing == MissingIPA && w.IPA != "",
			q.Missing == MissingMeaning && w.MeaningVi != "":
			continue
		}
		if q.Lemmas != nil && !slices.Contains(q.Lemmas, w.Lemma) {
			continue
		}
		all = append(all, w)
	}
	slices.SortFunc(all, func(a, b Word) int { return strings.Compare(a.Lemma, b.Lemma) })
	end := min(q.Skip+q.Limit, len(all))
	if q.Skip >= len(all) {
		return nil, len(all), nil
	}
	return all[q.Skip:end], len(all), nil
}

func (f *fakeRepo) Get(_ context.Context, lemma string) (Word, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	w, ok := f.words[lemma]
	if !ok {
		return Word{}, ErrNotFound
	}
	return w, nil
}

func (f *fakeRepo) Find(_ context.Context, lemmas []string) ([]Word, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []Word
	for _, l := range lemmas {
		if w, ok := f.words[l]; ok {
			out = append(out, w)
		}
	}
	return out, nil
}

func (f *fakeRepo) Create(_ context.Context, w Word) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.words[w.Lemma]; ok {
		return ErrExists
	}
	f.words[w.Lemma] = w
	return nil
}

func (f *fakeRepo) InsertMissing(_ context.Context, ws []Word) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, w := range ws {
		if _, ok := f.words[w.Lemma]; !ok {
			f.words[w.Lemma] = w
			n++
		}
	}
	return n, nil
}

func (f *fakeRepo) Update(_ context.Context, w Word) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	old, ok := f.words[w.Lemma]
	if !ok {
		return ErrNotFound
	}
	old.MeaningVi, old.IPA, old.UpdatedAt = w.MeaningVi, w.IPA, w.UpdatedAt
	f.words[w.Lemma] = old
	return nil
}

func (f *fakeRepo) Delete(_ context.Context, lemma string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.words[lemma]; !ok {
		return ErrNotFound
	}
	delete(f.words, lemma)
	delete(f.images, lemma)
	return nil
}

func (f *fakeRepo) Image(_ context.Context, lemma string) (Image, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	img, ok := f.images[lemma]
	if !ok {
		return Image{}, ErrImageNotFound
	}
	return img, nil
}

func (f *fakeRepo) SaveImage(_ context.Context, img Image) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	w, ok := f.words[img.Lemma]
	if !ok {
		return ErrNotFound
	}
	f.images[img.Lemma] = img
	w.ImageAt = img.CreatedAt
	f.words[img.Lemma] = w
	return nil
}

func (f *fakeRepo) DeleteImage(_ context.Context, lemma string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.images, lemma)
	if w, ok := f.words[lemma]; ok {
		w.ImageAt = time.Time{}
		f.words[lemma] = w
	}
	return nil
}

// fakeDict knows a few words by their exact form.
type fakeDict map[string]dictionary.Entry

func (f fakeDict) Resolve(_ context.Context, word string) (dictionary.Entry, bool, error) {
	e, ok := f[word]
	return e, ok, nil
}

// fakeTopics keeps topic word lists in memory.
type fakeTopics struct {
	mu    sync.Mutex
	lists []TopicList
}

func (f *fakeTopics) All(context.Context) ([]TopicList, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]TopicList, len(f.lists))
	for i, t := range f.lists {
		out[i] = TopicList{TopicRef: t.TopicRef, Words: slices.Clone(t.Words)}
	}
	return out, nil
}

func (f *fakeTopics) AddWord(_ context.Context, topicID, text string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i, t := range f.lists {
		if t.ID != topicID {
			continue
		}
		if slices.ContainsFunc(t.Words, func(w string) bool { return strings.EqualFold(w, text) }) {
			return ErrInTopic
		}
		f.lists[i].Words = append(f.lists[i].Words, text)
		return nil
	}
	return &ValidationError{Fields: map[string]string{"topicId": "Chủ đề không tồn tại"}}
}

// fakeImageAI draws a small PNG, or fails with err.
type fakeImageAI struct {
	mu   sync.Mutex
	err  error
	reqs []ai.ImageRequest
}

func (f *fakeImageAI) GenerateImage(_ context.Context, req ai.ImageRequest) (ai.Image, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reqs = append(f.reqs, req)
	if f.err != nil {
		return ai.Image{}, f.err
	}
	return ai.Image{MIME: "image/png", Data: testPNG(800, 400)}, nil
}

func testPNG(w, h int) []byte {
	var buf bytes.Buffer
	_ = png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, w, h)))
	return buf.Bytes()
}

type testEnv struct {
	svc    *Service
	repo   *fakeRepo
	topics *fakeTopics
	draw   *fakeImageAI
	means  *fakeMeaningAI
	fetch  map[string][]byte
	now    time.Time
}

func newEnv() *testEnv {
	e := &testEnv{repo: newFakeRepo(), topics: &fakeTopics{lists: []TopicList{
		{TopicRef: TopicRef{ID: "t1", Name: "Nhà cửa"}, Words: []string{"House", "table", "house", ""}},
		{TopicRef: TopicRef{ID: "t2", Name: "Chào hỏi"}, Words: []string{"good  morning", "House"}},
	}}, draw: &fakeImageAI{}, means: &fakeMeaningAI{}, fetch: map[string][]byte{}, now: time.Date(2026, 10, 7, 9, 0, 0, 0, time.UTC)}
	e.svc = NewService(Deps{
		Repo: e.repo,
		Dict: fakeDict{
			"house":   {Word: "house", IPA: "/haʊs/", Meanings: []dictionary.Meaning{{Text: "ngôi nhà"}, {Text: "nhà ở"}, {Text: "gia đình"}}},
			"good":    {Word: "good", IPA: "/ɡʊd/", Meanings: []dictionary.Meaning{{Text: "tốt"}}},
			"morning": {Word: "morning", IPA: "/ˈmɔːnɪŋ/", Meanings: []dictionary.Meaning{{Text: "buổi sáng"}}},
		},
		Topics:    e.topics,
		ImageAI:   e.draw,
		MeaningAI: e.means,
		FetchImage: func(_ context.Context, url string) ([]byte, error) {
			if data, ok := e.fetch[url]; ok {
				return data, nil
			}
			return nil, io.ErrUnexpectedEOF
		},
		Now: func() time.Time { return e.now },
		Log: slog.New(slog.DiscardHandler),
	})
	return e
}

var errFake = errors.New("fake failure")

// fakeMeaningAI answers with answers, or fails with err.
type fakeMeaningAI struct {
	mu      sync.Mutex
	err     error
	answers []ai.WordMeaning
	reqs    []ai.MeaningsRequest
}

func (f *fakeMeaningAI) WordMeanings(_ context.Context, req ai.MeaningsRequest) ([]ai.WordMeaning, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reqs = append(f.reqs, req)
	return f.answers, f.err
}
