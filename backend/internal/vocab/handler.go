package vocab

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/luongtran/luna/backend/internal/platform/httpx"
)

// Handler serves /api/vocab/*. The notebook owner is always the session user.
type Handler struct {
	svc *Service
	log *slog.Logger
}

// NewHandler returns a Handler.
func NewHandler(svc *Service, log *slog.Logger) *Handler {
	return &Handler{svc: svc, log: log}
}

// Register adds the vocab routes behind requireAuth.
func (h *Handler) Register(mux *http.ServeMux, requireAuth httpx.Middleware) {
	mux.Handle("POST /api/vocab/cards", requireAuth(http.HandlerFunc(h.save)))
	mux.Handle("GET /api/vocab/words", requireAuth(http.HandlerFunc(h.words)))
	mux.Handle("POST /api/vocab/cards/bulk", requireAuth(http.HandlerFunc(h.bulk)))
	mux.Handle("POST /api/vocab/practice-misses", requireAuth(http.HandlerFunc(h.misses)))
	mux.Handle("GET /api/vocab/cards", requireAuth(http.HandlerFunc(h.list)))
	mux.Handle("GET /api/vocab/lessons", requireAuth(http.HandlerFunc(h.lessons)))
	mux.Handle("PATCH /api/vocab/cards/{id}", requireAuth(http.HandlerFunc(h.update)))
	mux.Handle("DELETE /api/vocab/cards/{id}", requireAuth(http.HandlerFunc(h.remove)))
	mux.Handle("GET /api/vocab/review/due", requireAuth(http.HandlerFunc(h.due)))
	mux.Handle("POST /api/vocab/cards/{id}/review", requireAuth(http.HandlerFunc(h.review)))
}

type cardJSON struct {
	ID              string    `json:"id"`
	Text            string    `json:"text"`
	Lemma           string    `json:"lemma"`
	IPA             string    `json:"ipa"`
	MeaningVi       string    `json:"meaningVi"`
	ContextSentence string    `json:"contextSentence"`
	LessonID        *string   `json:"lessonId"`
	Source          Source    `json:"source"`
	CreatedAt       time.Time `json:"createdAt"`
	Due             time.Time `json:"due"`
	Reps            uint64    `json:"reps"`
	State           string    `json:"state"`
}

var stateNames = map[State]string{StateNew: "new", StateLearning: "learning", StateReview: "review", StateRelearning: "relearning"}

// toJSON renders c; the service fills the effective schedule of cards saved before F5.
func toJSON(c Card) cardJSON {
	out := cardJSON{
		ID: c.ID, Text: c.Text, Lemma: c.Lemma, IPA: c.IPA, MeaningVi: c.MeaningVi,
		ContextSentence: c.ContextSentence, Source: c.Source, CreatedAt: c.CreatedAt,
		Due: c.Schedule.Due.UTC(), Reps: c.Schedule.Reps, State: stateNames[c.Schedule.State],
	}
	if c.LessonID != "" {
		out.LessonID = &c.LessonID
	}
	return out
}

type saveRequest struct {
	Text            string `json:"text"`
	Lemma           string `json:"lemma"`
	IPA             string `json:"ipa"`
	MeaningVi       string `json:"meaningVi"`
	ContextSentence string `json:"contextSentence"`
	LessonID        string `json:"lessonId"`
	Source          string `json:"source"`
}

type existsBody struct {
	Error   string   `json:"error"`
	Message string   `json:"message"`
	Card    cardJSON `json:"card"`
}

type wordJSON struct {
	Lemma string `json:"lemma"`
	Text  string `json:"text"`
}

func (h *Handler) save(w http.ResponseWriter, r *http.Request) {
	p, _ := httpx.PrincipalFrom(r.Context())
	var req saveRequest
	if httpx.DecodeJSON(w, r, &req) != nil {
		return // unknown fields such as "userId" are rejected here
	}
	c, err := h.svc.Save(r.Context(), p.UserID, Input(req))

	var exists *ExistsError
	var verr *ValidationError
	switch {
	case err == nil:
		httpx.WriteJSON(w, http.StatusCreated, map[string]cardJSON{"card": toJSON(c)})
	case errors.As(err, &exists):
		httpx.WriteJSON(w, http.StatusConflict, existsBody{Error: "card_exists", Message: "Từ này đã có trong sổ", Card: toJSON(exists.Card)})
	case errors.As(err, &verr):
		httpx.WriteFieldErrors(w, verr.Fields)
	default:
		h.log.ErrorContext(r.Context(), "save card failed", slog.Any("error", err))
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "Có lỗi xảy ra, vui lòng thử lại")
	}
}

func (h *Handler) words(w http.ResponseWriter, r *http.Request) {
	p, _ := httpx.PrincipalFrom(r.Context())
	refs, err := h.svc.Words(r.Context(), p.UserID)
	if err != nil {
		h.log.ErrorContext(r.Context(), "list words failed", slog.Any("error", err))
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "Có lỗi xảy ra, vui lòng thử lại")
		return
	}
	out := make([]wordJSON, len(refs))
	for i, ref := range refs {
		out[i] = wordJSON(ref)
	}
	httpx.WriteJSON(w, http.StatusOK, map[string][]wordJSON{"words": out})
}
