package progress

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/luongtran/luna/backend/internal/platform/httpx"
)

// Handler serves the dictation endpoints. Results always belong to the session user.
type Handler struct {
	svc *Service
	log *slog.Logger
}

// NewHandler returns a Handler.
func NewHandler(svc *Service, log *slog.Logger) *Handler {
	return &Handler{svc: svc, log: log}
}

// Register adds the dictation routes behind requireAuth, then guard (lesson access, L).
func (h *Handler) Register(mux *http.ServeMux, requireAuth, guard httpx.Middleware) {
	mux.Handle("POST /api/lessons/{id}/dictation", requireAuth(guard(http.HandlerFunc(h.record))))
	mux.Handle("GET /api/lessons/{id}/dictation/summary", requireAuth(guard(http.HandlerFunc(h.summary))))
}

type recordRequest struct {
	SentenceIndex int    `json:"sentenceIndex"`
	Typed         string `json:"typed"`
	CorrectWords  int    `json:"correctWords"`
	TotalWords    int    `json:"totalWords"`
}

type resultJSON struct {
	SentenceIndex int       `json:"sentenceIndex"`
	Typed         string    `json:"typed"`
	CorrectWords  int       `json:"correctWords"`
	TotalWords    int       `json:"totalWords"`
	CheckedAt     time.Time `json:"checkedAt"`
}

type summaryJSON struct {
	SentenceCount int          `json:"sentenceCount"`
	CheckedCount  int          `json:"checkedCount"`
	CorrectWords  int          `json:"correctWords"`
	TotalWords    int          `json:"totalWords"`
	Rate          float64      `json:"rate"`
	Completed     bool         `json:"completed"`
	Results       []resultJSON `json:"results"`
}

func toJSON(s Summary) summaryJSON {
	out := summaryJSON{
		SentenceCount: s.SentenceCount, CheckedCount: s.CheckedCount, CorrectWords: s.CorrectWords,
		TotalWords: s.TotalWords, Rate: s.Rate, Completed: s.Completed,
		Results: make([]resultJSON, len(s.Results)),
	}
	for i, r := range s.Results {
		out.Results[i] = resultJSON(r)
	}
	return out
}

func (h *Handler) record(w http.ResponseWriter, r *http.Request) {
	p, _ := httpx.PrincipalFrom(r.Context())
	var req recordRequest
	if httpx.DecodeJSON(w, r, &req) != nil {
		return // unknown fields such as "userId" are rejected here
	}
	sum, err := h.svc.Record(r.Context(), p.UserID, r.PathValue("id"), Input(req))
	h.write(w, r, sum, err)
}

func (h *Handler) summary(w http.ResponseWriter, r *http.Request) {
	p, _ := httpx.PrincipalFrom(r.Context())
	sum, err := h.svc.Summary(r.Context(), p.UserID, r.PathValue("id"))
	h.write(w, r, sum, err)
}

func (h *Handler) write(w http.ResponseWriter, r *http.Request, sum Summary, err error) {
	var verr *ValidationError
	switch {
	case err == nil:
		httpx.WriteJSON(w, http.StatusOK, map[string]summaryJSON{"summary": toJSON(sum)})
	case errors.Is(err, ErrLessonNotFound):
		httpx.WriteError(w, http.StatusNotFound, "not_found", "Không tìm thấy bài học")
	case errors.As(err, &verr):
		httpx.WriteFieldErrors(w, verr.Fields)
	default:
		h.log.ErrorContext(r.Context(), "dictation failed", slog.Any("error", err))
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "Có lỗi xảy ra, vui lòng thử lại")
	}
}
