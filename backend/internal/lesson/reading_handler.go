package lesson

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/luongtran/luna/backend/internal/platform/httpx"
)

const (
	maxLookupChars = 100
	maxLookupWords = 6
)

// ReadingHandler serves the learner endpoints of the Reading step.
type ReadingHandler struct {
	reader *Reader
	log    *slog.Logger
}

// NewReadingHandler returns a ReadingHandler.
func NewReadingHandler(reader *Reader, log *slog.Logger) *ReadingHandler {
	return &ReadingHandler{reader: reader, log: log}
}

// Register adds the reading routes behind requireAuth, then guard (which decides whether the
// user may open the lesson {id}: L locks upcoming lessons for learners).
func (h *ReadingHandler) Register(mux *http.ServeMux, requireAuth, guard httpx.Middleware) {
	route := func(f http.HandlerFunc) http.Handler { return requireAuth(guard(f)) }
	mux.Handle("GET /api/lessons/{id}", route(h.view))
	mux.Handle("GET /api/lessons/{id}/lookup", route(h.lookup))
	mux.Handle("GET /api/lessons/{id}/vocabulary", route(h.vocabulary))
}

type readingJSON struct {
	ID         string            `json:"id"`
	Title      string            `json:"title"`
	Level      Level             `json:"level"`
	Topic      string            `json:"topic"`
	Sentences  []sentenceJSON    `json:"sentences"`
	Paragraphs [][]int           `json:"paragraphs"`
	Lemmas     map[string]string `json:"lemmas"`
	Phrases    []phraseJSON      `json:"phrases"`
}

type phraseJSON struct {
	Text  string `json:"text"`
	Lemma string `json:"lemma"`
}

type meaningJSON struct {
	POS  string `json:"pos"`
	Text string `json:"text"`
}

type lookupJSON struct {
	Source   string        `json:"source"`
	Text     string        `json:"text"`
	Lemma    string        `json:"lemma"`
	IPA      string        `json:"ipa"`
	Meanings []meaningJSON `json:"meanings"`
}

func (h *ReadingHandler) view(w http.ResponseWriter, r *http.Request) {
	v, err := h.reader.View(r.Context(), r.PathValue("id"))
	if err != nil {
		h.writeError(w, r, err, "not_found")
		return
	}
	out := readingJSON{
		ID: v.ID, Title: v.Title, Level: v.Level, Topic: v.Topic,
		Sentences:  make([]sentenceJSON, len(v.Sentences)),
		Paragraphs: v.Paragraphs, Lemmas: v.Lemmas,
		Phrases: make([]phraseJSON, len(v.Phrases)),
	}
	for i, s := range v.Sentences {
		out.Sentences[i] = sentenceJSON{Index: s.Index, Text: s.Text}
		if s.AudioPath != "" {
			out.Sentences[i].AudioURL = &s.AudioPath
		}
	}
	for i, p := range v.Phrases {
		out.Phrases[i] = phraseJSON(p)
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]readingJSON{"lesson": out})
}

func (h *ReadingHandler) lookup(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	fields := map[string]string{}
	switch {
	case q == "":
		fields["q"] = "Chưa chọn từ để tra"
	case utf8.RuneCountInString(q) > maxLookupChars:
		fields["q"] = "Tối đa 100 ký tự"
	case len(strings.Fields(q)) > maxLookupWords:
		fields["q"] = "Chọn tối đa 6 từ"
	}
	sentence := -1
	if s := r.URL.Query().Get("sentence"); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil || n < 0 {
			fields["sentence"] = "Câu không hợp lệ"
		}
		sentence = n
	}
	if len(fields) > 0 {
		httpx.WriteFieldErrors(w, fields)
		return
	}

	res, err := h.reader.Lookup(r.Context(), r.PathValue("id"), q, sentence)
	if err != nil {
		h.writeError(w, r, err, "lesson_not_found")
		return
	}
	out := lookupJSON{Source: res.Source, Text: res.Text, Lemma: res.Lemma, IPA: res.IPA, Meanings: make([]meaningJSON, len(res.Meanings))}
	for i, m := range res.Meanings {
		out.Meanings[i] = meaningJSON(m)
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

type vocabItemJSON struct {
	Lemma         string `json:"lemma"`
	Text          string `json:"text"`
	MeaningVi     string `json:"meaningVi"`
	IPA           string `json:"ipa"`
	SentenceIndex int    `json:"sentenceIndex"`
	Sentence      string `json:"sentence"`
}

type vocabularyJSON struct {
	Available bool            `json:"available"`
	Items     []vocabItemJSON `json:"items"`
}

func (h *ReadingHandler) vocabulary(w http.ResponseWriter, r *http.Request) {
	items, available, err := h.reader.Vocabulary(r.Context(), r.PathValue("id"))
	if err != nil {
		h.writeError(w, r, err, "not_found")
		return
	}
	out := vocabularyJSON{Available: available, Items: make([]vocabItemJSON, len(items))}
	for i, it := range items {
		out.Items[i] = vocabItemJSON(it)
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

// writeError maps lesson errors; lessonMissingCode distinguishes a missing lesson from a
// word without meaning on the lookup endpoint.
func (h *ReadingHandler) writeError(w http.ResponseWriter, r *http.Request, err error, lessonMissingCode string) {
	switch {
	case errors.Is(err, ErrLookupNotFound):
		httpx.WriteError(w, http.StatusNotFound, "not_found", "Chưa có nghĩa")
	case errors.Is(err, ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, lessonMissingCode, "Không tìm thấy bài học")
	default:
		h.log.ErrorContext(r.Context(), "reading request failed", slog.Any("error", err))
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "Có lỗi xảy ra, vui lòng thử lại")
	}
}
