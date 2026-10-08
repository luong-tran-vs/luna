package writing

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/luongtran/luna/backend/internal/platform/httpx"
)

// Handler serves the writing endpoints (contracts/writing-api.md).
type Handler struct {
	svc *Service
	log *slog.Logger
}

// NewHandler returns a Handler.
func NewHandler(svc *Service, log *slog.Logger) *Handler {
	return &Handler{svc: svc, log: log}
}

// Register adds the lesson routes behind requireAuth and guard (L: the learner may open the
// lesson) and the learner's writing routes behind requireAuth.
func (h *Handler) Register(mux *http.ServeMux, requireAuth, guard httpx.Middleware) {
	lesson := func(f http.HandlerFunc) http.Handler { return requireAuth(guard(f)) }
	mux.Handle("GET /api/lessons/{id}/writing", lesson(h.get))
	mux.Handle("PUT /api/lessons/{id}/writing", lesson(h.saveDraft))
	mux.Handle("POST /api/lessons/{id}/writing/submit", lesson(h.submit))

	own := func(f http.HandlerFunc) http.Handler { return requireAuth(f) }
	mux.Handle("GET /api/writings", own(h.list))
	mux.Handle("GET /api/writings/unseen-count", own(h.unseen))
	mux.Handle("GET /api/writings/{id}", own(h.detail))
	mux.Handle("POST /api/writings/{id}/seen", own(h.seen))
	mux.Handle("POST /api/writings/{id}/regrade", own(h.regrade))
	mux.Handle("POST /api/writings/{id}/resubmit", own(h.resubmit))
}

type summaryJSON struct {
	ID          string      `json:"id"`
	LessonID    string      `json:"lessonId"`
	LessonTitle string      `json:"lessonTitle"`
	SubmittedAt time.Time   `json:"submittedAt"`
	GradeStatus GradeStatus `json:"gradeStatus"`
	Average     *float64    `json:"average"`
	Seen        bool        `json:"seen"`
}

type latestJSON struct {
	ID     string      `json:"id"`
	Status GradeStatus `json:"status"`
}

type unseenJSON struct {
	Unseen  int         `json:"unseen"`
	Pending int         `json:"pending"`
	Latest  *latestJSON `json:"latest"`
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	p, _ := httpx.PrincipalFrom(r.Context())
	list, err := h.svc.List(r.Context(), p.UserID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	out := make([]summaryJSON, len(list))
	for i, s := range list {
		out[i] = summaryJSON(s)
	}
	httpx.WriteJSON(w, http.StatusOK, map[string][]summaryJSON{"writings": out})
}

func (h *Handler) unseen(w http.ResponseWriter, r *http.Request) {
	p, _ := httpx.PrincipalFrom(r.Context())
	c, err := h.svc.Unseen(r.Context(), p.UserID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	out := unseenJSON{Unseen: c.Unseen, Pending: c.Pending}
	if c.Latest != nil {
		out.Latest = &latestJSON{ID: c.Latest.ID, Status: c.Latest.Status}
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) detail(w http.ResponseWriter, r *http.Request) {
	p, _ := httpx.PrincipalFrom(r.Context())
	wr, err := h.svc.Detail(r.Context(), p.UserID, r.PathValue("id"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]writingJSON{"writing": toWritingJSON(wr)})
}

func (h *Handler) seen(w http.ResponseWriter, r *http.Request) {
	p, _ := httpx.PrincipalFrom(r.Context())
	if err := h.svc.MarkSeen(r.Context(), p.UserID, r.PathValue("id")); err != nil {
		h.writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// resubmit sends the edited writing to grading again (one of its MaxGradings gradings).
func (h *Handler) resubmit(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Text string `json:"text"`
	}
	if httpx.DecodeJSON(w, r, &body) != nil {
		return
	}
	p, _ := httpx.PrincipalFrom(r.Context())
	wr, err := h.svc.Resubmit(r.Context(), p.UserID, r.PathValue("id"), body.Text)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusAccepted, map[string]writingJSON{"writing": toWritingJSON(wr)})
}

func (h *Handler) regrade(w http.ResponseWriter, r *http.Request) {
	p, _ := httpx.PrincipalFrom(r.Context())
	wr, err := h.svc.Regrade(r.Context(), p.UserID, r.PathValue("id"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusAccepted, map[string]writingJSON{"writing": toWritingJSON(wr)})
}

// --- JSON shapes ---

type criterionJSON struct {
	Name      string `json:"name"`
	Score     int    `json:"score"`
	CommentVi string `json:"commentVi"`
}

type gradeJSON struct {
	Status        GradeStatus     `json:"status"`
	Error         string          `json:"error"`
	Criteria      []criterionJSON `json:"criteria"`
	Average       *float64        `json:"average"`
	OverallVi     string          `json:"overallVi"`
	CorrectedText string          `json:"correctedText"`
	GradedAt      *time.Time      `json:"gradedAt"`
}

type writingJSON struct {
	ID          string     `json:"id"`
	LessonID    string     `json:"lessonId"`
	LessonTitle string     `json:"lessonTitle"`
	Prompt      string     `json:"prompt"`
	Text        string     `json:"text"`
	Status      Status     `json:"status"`
	SubmittedAt *time.Time `json:"submittedAt"`
	Grade       *gradeJSON `json:"grade"`
	// Gradings: how many of its MaxGradings gradings the writing used (submit, resubmit, regrade).
	Gradings gradingsJSON `json:"gradings"`
}

type gradingsJSON struct {
	Used int `json:"used"`
	Max  int `json:"max"`
}

func timeOrNil(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

func toWritingJSON(w Writing) writingJSON {
	out := writingJSON{
		ID: w.ID, LessonID: w.LessonID, LessonTitle: w.LessonTitle, Prompt: w.Prompt, Text: w.Text,
		Status: w.Status, SubmittedAt: timeOrNil(w.SubmittedAt),
	}
	if w.Status == StatusSubmitted {
		out.Gradings = gradingsJSON{Used: GradingsUsed(w), Max: MaxGradings}
	} else {
		out.Gradings = gradingsJSON{Max: MaxGradings}
	}
	if g := w.Grade; g != nil {
		out.Grade = &gradeJSON{
			Status: g.Status, Error: g.Error, Criteria: make([]criterionJSON, len(g.Criteria)), Average: Average(g),
			OverallVi: g.OverallVi, CorrectedText: g.CorrectedText, GradedAt: timeOrNil(g.GradedAt),
		}
		for i, c := range g.Criteria {
			out.Grade.Criteria[i] = criterionJSON(c)
		}
	}
	return out
}

type lessonWritingJSON struct {
	Prompt   string       `json:"prompt"`
	Level    string       `json:"level"`
	CanWrite bool         `json:"canWrite"`
	Writing  *writingJSON `json:"writing"`
}

type textJSON struct {
	Text string `json:"text"`
}

// --- lesson routes ---

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	p, _ := httpx.PrincipalFrom(r.Context())
	v, err := h.svc.Get(r.Context(), p.UserID, r.PathValue("id"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	out := lessonWritingJSON{Prompt: v.Prompt, Level: v.Level, CanWrite: v.CanWrite}
	if v.Writing != nil {
		j := toWritingJSON(*v.Writing)
		out.Writing = &j
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) saveDraft(w http.ResponseWriter, r *http.Request) {
	h.withText(w, r, h.svc.SaveDraft)
}

func (h *Handler) submit(w http.ResponseWriter, r *http.Request) {
	h.withText(w, r, h.svc.Submit)
}

func (h *Handler) withText(w http.ResponseWriter, r *http.Request,
	f func(ctx context.Context, userID, lessonID, text string) (Writing, error),
) {
	var in textJSON
	if httpx.DecodeJSON(w, r, &in) != nil {
		return
	}
	p, _ := httpx.PrincipalFrom(r.Context())
	wr, err := f(r.Context(), p.UserID, r.PathValue("id"), in.Text)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]writingJSON{"writing": toWritingJSON(wr)})
}

func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	var verr *ValidationError
	switch {
	case errors.As(err, &verr):
		httpx.WriteFieldErrors(w, verr.Fields)
	case errors.Is(err, ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "not_found", "Không tìm thấy bài viết")
	case errors.Is(err, ErrSubmitted):
		httpx.WriteError(w, http.StatusConflict, "already_submitted", "Bài viết đã được nộp")
	case errors.Is(err, ErrLocked):
		httpx.WriteError(w, http.StatusConflict, "write_locked", "Hãy học tới bước Viết của bài đang học")
	case errors.Is(err, ErrNotFailed):
		httpx.WriteError(w, http.StatusConflict, "not_failed", "Chỉ chấm lại được bài chấm lỗi")
	case errors.Is(err, ErrNoGradings):
		httpx.WriteError(w, http.StatusConflict, "no_gradings", fmt.Sprintf("Mỗi bài viết chỉ được chấm %d lần", MaxGradings))
	case errors.Is(err, ErrGrading):
		httpx.WriteError(w, http.StatusConflict, "grading", "Bài đang được chấm, vui lòng chờ")
	default:
		h.log.ErrorContext(r.Context(), "writing request failed", slog.Any("error", err))
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "Có lỗi xảy ra, vui lòng thử lại")
	}
}
