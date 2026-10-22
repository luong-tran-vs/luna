package curriculum

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/luongtran/luna/backend/internal/ai"
	"github.com/luongtran/luna/backend/internal/cefr"
	"github.com/luongtran/luna/backend/internal/lesson"
	"github.com/luongtran/luna/backend/internal/platform/httpx"
	"github.com/luongtran/luna/backend/internal/topic"
)

// generateWriteTimeout replaces the server's write timeout for a slot's generation, which waits
// for the AI (as the generate route of lessons).
const generateWriteTimeout = 135 * time.Second

// Handler serves the admin's course page.
type Handler struct {
	svc *Service
	log *slog.Logger
}

// NewHandler returns a Handler.
func NewHandler(svc *Service, log *slog.Logger) *Handler { return &Handler{svc: svc, log: log} }

// Register mounts the admin routes.
func (h *Handler) Register(mux *http.ServeMux, requireAuth httpx.Middleware) {
	admin := func(f http.HandlerFunc) http.Handler { return requireAuth(httpx.RequireAdmin(f)) }
	mux.Handle("GET /api/admin/curriculum", admin(h.board))
	mux.Handle("POST /api/admin/curriculum/apply", admin(h.apply))
	mux.Handle("POST /api/admin/curriculum/slots/{key}/generate", admin(h.generate))
	mux.Handle("GET /api/admin/curriculum/checks/{lessonId}", admin(h.checks))
	mux.Handle("POST /api/admin/curriculum/lessons/{lessonId}/remove", admin(h.remove))
}

// --- JSON shapes ---

type grammarJSON struct {
	ID        string `json:"id"`
	TitleVi   string `json:"titleVi"`
	Published bool   `json:"published"`
}

type lessonJSON struct {
	ID               string `json:"id"`
	Title            string `json:"title"`
	Draft            bool   `json:"draft"`
	AnnotationStatus string `json:"annotationStatus"`
}

type checkJSON struct {
	Key    string `json:"key"`
	OK     bool   `json:"ok"`
	Warn   bool   `json:"warn"`
	TextVi string `json:"textVi"`
}

type wordJSON struct {
	Text  string `json:"text"`
	Level string `json:"level"`
}

type checksJSON struct {
	Items []checkJSON `json:"items"`
	Above []wordJSON  `json:"above"`
}

type slotJSON struct {
	Key     string        `json:"key"`
	Kind    Kind          `json:"kind"`
	Title   string        `json:"title"`
	Idea    string        `json:"idea"`
	Grammar []grammarJSON `json:"grammar"`
	Lesson  *lessonJSON   `json:"lesson"`
	Checks  *checksJSON   `json:"checks"`
}

type topicJSON struct {
	Number   int           `json:"number"`
	Name     string        `json:"name"`
	CanDoVi  string        `json:"canDoVi"`
	TopicID  string        `json:"topicId"`
	Position int           `json:"position"`
	Grammar  []grammarJSON `json:"grammar"`
	Slots    []slotJSON    `json:"slots"`
	Others   []lessonJSON  `json:"others"`
}

type boardJSON struct {
	Level     string      `json:"level"`
	Applied   bool        `json:"applied"`
	Main      int         `json:"main"`
	Extra     int         `json:"extra"`
	Published int         `json:"published"`
	Drafts    int         `json:"drafts"`
	Topics    []topicJSON `json:"topics"`
}

func grammarsJSON(gs []GrammarView) []grammarJSON {
	out := make([]grammarJSON, len(gs))
	for i, g := range gs {
		out[i] = grammarJSON(g)
	}
	return out
}

func refJSON(l topic.LessonRef) lessonJSON {
	return lessonJSON{ID: l.ID, Title: l.Title, Draft: l.Draft, AnnotationStatus: l.AnnotationStatus}
}

func toChecksJSON(c Checks) checksJSON {
	out := checksJSON{Items: make([]checkJSON, len(c.Items)), Above: make([]wordJSON, len(c.Above))}
	for i, it := range c.Items {
		out.Items[i] = checkJSON(it)
	}
	for i, w := range c.Above {
		out.Above[i] = wordJSON(cefr.Word(w))
	}
	return out
}

func toSlotJSON(s SlotView) slotJSON {
	out := slotJSON{Key: s.Key, Kind: s.Kind, Title: s.Title, Idea: s.Idea, Grammar: grammarsJSON(s.Grammar)}
	if s.Lesson != nil {
		l := refJSON(*s.Lesson)
		out.Lesson = &l
	}
	if s.Checks != nil {
		c := toChecksJSON(*s.Checks)
		out.Checks = &c
	}
	return out
}

func toBoardJSON(b Board) boardJSON {
	out := boardJSON{
		Level: b.Level, Applied: b.Applied, Main: b.Main, Extra: b.Extra, Published: b.Published, Drafts: b.Drafts,
		Topics: make([]topicJSON, len(b.Topics)),
	}
	for i, t := range b.Topics {
		tj := topicJSON{
			Number: t.Number, Name: t.Name, CanDoVi: t.CanDoVi, TopicID: t.TopicID, Position: t.Position,
			Grammar: grammarsJSON(t.Grammar), Slots: make([]slotJSON, len(t.Slots)), Others: make([]lessonJSON, len(t.Others)),
		}
		for j, s := range t.Slots {
			tj.Slots[j] = toSlotJSON(s)
		}
		for j, l := range t.Others {
			tj.Others[j] = refJSON(l)
		}
		out.Topics[i] = tj
	}
	return out
}

// --- handlers ---

func (h *Handler) board(w http.ResponseWriter, r *http.Request) {
	b, err := h.svc.Board(r.Context(), levelOf(r))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toBoardJSON(b))
}

func (h *Handler) apply(w http.ResponseWriter, r *http.Request) {
	b, err := h.svc.Apply(r.Context(), levelOf(r))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toBoardJSON(b))
}

func (h *Handler) generate(w http.ResponseWriter, r *http.Request) {
	if err := http.NewResponseController(w).SetWriteDeadline(time.Now().Add(generateWriteTimeout)); err != nil &&
		!errors.Is(err, http.ErrNotSupported) {
		h.log.WarnContext(r.Context(), "curriculum: extend write deadline", slog.Any("error", err))
	}
	s, err := h.svc.GenerateSlot(r.Context(), r.PathValue("key"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toSlotJSON(s))
}

func (h *Handler) checks(w http.ResponseWriter, r *http.Request) {
	c, err := h.svc.Checks(r.Context(), r.PathValue("lessonId"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toChecksJSON(c))
}

func (h *Handler) remove(w http.ResponseWriter, r *http.Request) {
	res, err := h.svc.Remove(r.Context(), r.PathValue("lessonId"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]bool{"deleted": res.Deleted})
}

func levelOf(r *http.Request) string {
	if l := r.URL.Query().Get("level"); l != "" {
		return l
	}
	return "A1"
}

func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	var verr *lesson.ValidationError
	switch {
	case errors.Is(err, ErrNoPlan):
		httpx.WriteError(w, http.StatusNotFound, "no_curriculum", "Trình độ này chưa có giáo trình")
	case errors.Is(err, ErrSlotNotFound):
		httpx.WriteError(w, http.StatusNotFound, "not_found", "Không tìm thấy ô bài trong giáo trình")
	case errors.Is(err, ErrNotApplied):
		httpx.WriteError(w, http.StatusConflict, "not_applied", "Chủ đề chưa có trong danh mục, hãy bấm Áp dụng giáo trình trước")
	case errors.Is(err, ErrSlotFilled):
		httpx.WriteError(w, http.StatusConflict, "slot_filled", "Ô này đã có bài")
	case errors.Is(err, ErrNoDraft), errors.Is(err, lesson.ErrUnusableDraft):
		httpx.WriteError(w, http.StatusBadGateway, "ai_unusable", "AI trả về nội dung không dùng được, vui lòng thử lại.")
	case errors.As(err, &verr):
		httpx.WriteFieldErrors(w, verr.Fields)
	case errors.Is(err, lesson.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "not_found", "Không tìm thấy bài học")
	case errors.Is(err, ai.ErrNotConfigured), errors.Is(err, ai.ErrInvalidKey):
		httpx.WriteError(w, http.StatusServiceUnavailable, "ai_not_configured", "AI chưa được cấu hình. Liên hệ người vận hành.")
	case errors.Is(err, ai.ErrQuota):
		httpx.WriteError(w, http.StatusTooManyRequests, "ai_quota", "Đã hết lượt AI, vui lòng thử lại sau.")
	case lesson.IsGenerateAIError(err):
		h.log.WarnContext(r.Context(), "curriculum generation failed", slog.Any("error", err))
		httpx.WriteError(w, http.StatusBadGateway, "ai_failed", "Sinh bài thất bại, vui lòng thử lại.")
	default:
		h.log.ErrorContext(r.Context(), "curriculum request failed", slog.Any("error", err))
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "Có lỗi xảy ra, vui lòng thử lại")
	}
}
