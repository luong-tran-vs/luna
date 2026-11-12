package wordbank

import (
	"cmp"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/luongtran/luna/backend/internal/platform/httpx"
)

// Pictures of the words learners review (added 2026-11-12): the review cards show the word bank
// picture of their word, and the admin draws, in one batch, the pictures the saved words lack. A
// saved word not in the bank yet is added first, with the meaning and IPA of a learner's card, so
// that the AI knows which sense to draw.

// CardWord is a word learners saved: one per lemma, with how many of them have a card of it.
type CardWord struct {
	Lemma     string
	MeaningVi string
	IPA       string
	Learners  int
}

// CardWords gives the words of every learner's cards, the most saved first (implemented over the
// cards in internal/service).
type CardWords interface {
	CardWords(ctx context.Context) ([]CardWord, error)
}

// CardWordsShown is how many words lacking a picture the admin page lists.
const CardWordsShown = 50

// CardWordMissing is a saved word without a picture; InBank is false when the bank lacks it too.
type CardWordMissing struct {
	CardWord
	InBank bool
}

// CardWordsView is what the admin sees of the saved words: how many have a picture, and those
// without, the most saved first.
type CardWordsView struct {
	Total     int
	WithImage int
	Missing   []CardWordMissing
}

// ImageURL is the address learners load the picture of a word from; v changes with the picture.
func ImageURL(lemma string, at time.Time) string {
	return "/api/words/" + url.PathEscape(lemma) + "/image?v=" + strconv.FormatInt(at.Unix(), 10)
}

// ImageURLs gives the address of the picture of each of lemmas that has one, by lemma as given.
func (s *Service) ImageURLs(ctx context.Context, lemmas []string) (map[string]string, error) {
	words, err := s.Words(ctx, lemmas)
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	for _, l := range lemmas {
		if w, ok := words[Normalize(l)]; ok && w.HasImage() {
			out[l] = ImageURL(w.Lemma, w.ImageAt)
		}
	}
	return out, nil
}

// CardWords tells which saved words have a picture in the bank.
func (s *Service) CardWords(ctx context.Context) (CardWordsView, error) {
	if s.cards == nil {
		return CardWordsView{}, nil
	}
	saved, err := s.cards.CardWords(ctx)
	if err != nil {
		return CardWordsView{}, fmt.Errorf("wordbank: card words: %w", err)
	}
	// The cards keep the lemma as typed: words differing in case or spaces are one word here.
	var words []CardWord
	at := map[string]int{}
	for _, c := range saved {
		k := Normalize(c.Lemma)
		if k == "" {
			continue
		}
		if i, ok := at[k]; ok {
			words[i].Learners += c.Learners
			continue
		}
		c.Lemma = k
		at[k] = len(words)
		words = append(words, c)
	}
	lemmas := make([]string, len(words))
	for i, w := range words {
		lemmas[i] = w.Lemma
	}
	bank, err := s.Words(ctx, lemmas)
	if err != nil {
		return CardWordsView{}, err
	}
	view := CardWordsView{Total: len(words)}
	for _, w := range words {
		b, inBank := bank[w.Lemma]
		if inBank && b.HasImage() {
			view.WithImage++
			continue
		}
		view.Missing = append(view.Missing, CardWordMissing{CardWord: w, InBank: inBank})
	}
	// Merging may have moved a word up.
	sortByLearners(view.Missing)
	return view, nil
}

// StartCardWordsBatch draws the pictures the saved words lack, the most saved first, MaxBatch at
// most; the words not in the bank are added to it first.
func (s *Service) StartCardWordsBatch(ctx context.Context, engine Engine, style string) (BatchStatus, error) {
	if _, err := checkStyle(style); err != nil {
		return BatchStatus{}, err
	}
	if _, _, err := s.imageDrawer(engine); err != nil {
		return BatchStatus{}, err
	}
	if s.Batch().Running {
		return BatchStatus{}, ErrBatchRunning
	}
	view, err := s.CardWords(ctx)
	if err != nil {
		return BatchStatus{}, err
	}
	if len(view.Missing) == 0 {
		return BatchStatus{}, &ValidationError{Fields: map[string]string{"lemmas": "Mọi từ trong thẻ ôn đã có ảnh"}}
	}
	missing := view.Missing[:min(len(view.Missing), MaxBatch)]
	now := s.now()
	var add []Word
	lemmas := make([]string, len(missing))
	for i, m := range missing {
		lemmas[i] = m.Lemma
		if !m.InBank {
			add = append(add, Word{Lemma: m.Lemma, MeaningVi: m.MeaningVi, IPA: m.IPA, CreatedAt: now, UpdatedAt: now})
		}
	}
	if len(add) > 0 {
		if _, err := s.repo.InsertMissing(ctx, add); err != nil {
			return BatchStatus{}, fmt.Errorf("wordbank: add card words: %w", err)
		}
	}
	return s.StartBatch(ctx, BatchRequest{Lemmas: lemmas, Engine: engine, Style: style, CardWords: true})
}

func sortByLearners(ws []CardWordMissing) {
	slices.SortFunc(ws, func(a, b CardWordMissing) int {
		if a.Learners != b.Learners {
			return cmp.Compare(b.Learners, a.Learners)
		}
		return strings.Compare(a.Lemma, b.Lemma)
	})
}

type cardWordJSON struct {
	Lemma     string `json:"lemma"`
	MeaningVi string `json:"meaningVi"`
	Learners  int    `json:"learners"`
	InBank    bool   `json:"inBank"`
}

type cardWordsJSON struct {
	Total     int `json:"total"`
	WithImage int `json:"withImage"`
	// MissingCount counts every word lacking a picture; Missing lists the first CardWordsShown.
	MissingCount int            `json:"missingCount"`
	Missing      []cardWordJSON `json:"missing"`
	// CodexImages says this server can draw with Codex.
	CodexImages bool `json:"codexImages"`
}

// cardWords serves GET /api/admin/words/card-words: the words learners saved, how many have a
// picture, and the most saved of those without.
func (h *Handler) cardWords(w http.ResponseWriter, r *http.Request) {
	view, err := h.svc.CardWords(r.Context())
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	out := cardWordsJSON{Total: view.Total, WithImage: view.WithImage, MissingCount: len(view.Missing), Missing: []cardWordJSON{}, CodexImages: h.svc.CodexAvailable()}
	for _, m := range view.Missing[:min(len(view.Missing), CardWordsShown)] {
		out.Missing = append(out.Missing, cardWordJSON{Lemma: m.Lemma, MeaningVi: m.MeaningVi, Learners: m.Learners, InBank: m.InBank})
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

// startCardWordsBatch serves POST /api/admin/words/card-words/image-batch with {"engine", "style"?}:
// 202 once the batch draws the pictures the saved words lack; 409 while another batch runs.
func (h *Handler) startCardWordsBatch(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Engine Engine `json:"engine"`
		Style  string `json:"style"`
	}
	if httpx.DecodeJSON(w, r, &in) != nil {
		return
	}
	st, err := h.svc.StartCardWordsBatch(r.Context(), in.Engine, in.Style)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusAccepted, batchToJSON(st))
}
