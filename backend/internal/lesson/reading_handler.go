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
	mux.Handle("POST /api/lessons/{id}/answers", route(h.answer))
	mux.Handle("POST /api/lessons/{id}/ask", route(h.ask))
	mux.Handle("GET /api/lessons/{id}/practice", route(h.practice))
}

type readingJSON struct {
	ID          string            `json:"id"`
	Title       string            `json:"title"`
	Level       Level             `json:"level"`
	Topic       string            `json:"topic"`
	Sentences   []sentenceJSON    `json:"sentences"`
	Paragraphs  [][]int           `json:"paragraphs"`
	Lemmas      map[string]string `json:"lemmas"`
	Phrases     []phraseJSON      `json:"phrases"`
	Quiz        *quizJSON         `json:"quiz"`
	GrammarNote *grammarNoteJSON  `json:"grammarNote"`
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
	Note     string        `json:"note"`
}

func (h *ReadingHandler) view(w http.ResponseWriter, r *http.Request) {
	p, _ := httpx.PrincipalFrom(r.Context())
	v, err := h.reader.View(r.Context(), p.UserID, r.PathValue("id"))
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
		out.Sentences[i] = sentenceJSON(s)
	}
	for i, p := range v.Phrases {
		out.Phrases[i] = phraseJSON(p)
	}
	out.Quiz = toQuizJSON(v.Quiz)
	out.GrammarNote = toGrammarNoteJSON(v.GrammarNote)
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
	httpx.WriteJSON(w, http.StatusOK, toLookupJSON(res))
}

func toLookupJSON(res LookupResult) lookupJSON {
	out := lookupJSON{
		Source: res.Source, Text: res.Text, Lemma: res.Lemma, IPA: res.IPA,
		Meanings: make([]meaningJSON, len(res.Meanings)), Note: res.Note,
	}
	for i, m := range res.Meanings {
		out.Meanings[i] = meaningJSON(m)
	}
	return out
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
	var verr *ValidationError
	switch {
	case errors.Is(err, ErrLookupNotFound):
		httpx.WriteError(w, http.StatusNotFound, "not_found", "Chưa có nghĩa")
	case errors.Is(err, ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, lessonMissingCode, "Không tìm thấy bài học")
	case errors.As(err, &verr):
		httpx.WriteFieldErrors(w, verr.Fields)
	case errors.Is(err, ErrQuizChanged):
		httpx.WriteError(w, http.StatusConflict, "quiz_changed", "Câu hỏi vừa được cập nhật, vui lòng tải lại")
	case errors.Is(err, ErrNoQuiz):
		httpx.WriteError(w, http.StatusConflict, "no_quiz", "Bài này chưa có câu hỏi")
	default:
		h.log.ErrorContext(r.Context(), "reading request failed", slog.Any("error", err))
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "Có lỗi xảy ra, vui lòng thử lại")
	}
}
