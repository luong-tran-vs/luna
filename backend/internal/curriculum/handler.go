package curriculum

import (
	"errors"
	"log/slog"
	"net/http"
	"slices"
	"strconv"
	"time"

	"github.com/luongtran/luna/backend/internal/ai"
	"github.com/luongtran/luna/backend/internal/cefr"
	"github.com/luongtran/luna/backend/internal/lesson"
	"github.com/luongtran/luna/backend/internal/platform/httpx"
	"github.com/luongtran/luna/backend/internal/topic"
)

// generateWriteTimeout replaces the server's write timeout for a slot's generation, which waits
// for the AI (as the generate route of lessons).
const generateWriteTimeout = 5*time.Minute + 10*time.Second

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
	mux.Handle("GET /api/admin/curriculum/topics", admin(h.courseTopics))
	mux.Handle("GET /api/admin/curriculum/summary", admin(h.summary))
	mux.Handle("POST /api/admin/curriculum/apply", admin(h.apply))
	mux.Handle("POST /api/admin/curriculum/slots/{key}/generate", admin(h.generate))
	mux.Handle("GET /api/admin/curriculum/checks/{lessonId}", admin(h.checks))
	mux.Handle("POST /api/admin/curriculum/draft", admin(h.draftWithAI))
	mux.Handle("PUT /api/admin/curriculum/draft", admin(h.saveDraft))
	mux.Handle("DELETE /api/admin/curriculum/draft", admin(h.discardDraft))
	mux.Handle("POST /api/admin/curriculum/draft/topics/{number}/write", admin(h.writeTopic))
	mux.Handle("POST /api/admin/curriculum/edit", admin(h.editApproved))
	mux.Handle("POST /api/admin/curriculum/approve", admin(h.approve))
	mux.Handle("GET /api/admin/curriculum/slots/{key}/brief", admin(h.brief))
	mux.Handle("POST /api/admin/curriculum/slots/{key}/import", admin(h.importSlot))
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
	Source           string `json:"source"`
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
	Words    int           `json:"words"`
	Grammar  []grammarJSON `json:"grammar"`
	Slots    []slotJSON    `json:"slots"`
	Others   []lessonJSON  `json:"others"`
}

type boardJSON struct {
	Level       string        `json:"level"`
	Status      Status        `json:"status"`
	HasApproved bool          `json:"hasApproved"`
	Warnings    []string      `json:"warnings"`
	Points      []grammarJSON `json:"points"`
	Applied     bool          `json:"applied"`
	Main        int           `json:"main"`
	Extra       int           `json:"extra"`
	Published   int           `json:"published"`
	Drafts      int           `json:"drafts"`
	Topics      []topicJSON   `json:"topics"`
}

func grammarsJSON(gs []GrammarView) []grammarJSON {
	out := make([]grammarJSON, len(gs))
	for i, g := range gs {
		out[i] = grammarJSON(g)
	}
	return out
}

func refJSON(l topic.LessonRef) lessonJSON {
	return lessonJSON{ID: l.ID, Title: l.Title, Draft: l.Draft, AnnotationStatus: l.AnnotationStatus, Source: l.Source}
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
		Level: b.Level, Status: b.Status, HasApproved: b.HasApproved, Warnings: append([]string{}, b.Warnings...), Points: grammarsJSON(b.Points),
		Applied: b.Applied, Main: b.Main, Extra: b.Extra, Published: b.Published, Drafts: b.Drafts,
		Topics: make([]topicJSON, len(b.Topics)),
	}
	for i, t := range b.Topics {
		tj := topicJSON{
			Number: t.Number, Name: t.Name, CanDoVi: t.CanDoVi, TopicID: t.TopicID, Position: t.Position, Words: t.Words,
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
	if level, ok := h.level(w, r); ok {
		h.writeBoard(w, r, func() (Board, error) { return h.svc.Board(r.Context(), level) })
	}
}

func (h *Handler) apply(w http.ResponseWriter, r *http.Request) {
	if level, ok := h.level(w, r); ok {
		h.writeBoard(w, r, func() (Board, error) { return h.svc.Apply(r.Context(), level) })
	}
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

// remove takes an optional body {"delete": false} that keeps the lesson as a draft (added
// 2026-10-26); without it the lesson is deleted, as before.
func (h *Handler) remove(w http.ResponseWriter, r *http.Request) {
	body := struct {
		Delete *bool `json:"delete"`
	}{}
	if r.ContentLength != 0 && httpx.DecodeJSON(w, r, &body) != nil {
		return
	}
	res, err := h.svc.Remove(r.Context(), r.PathValue("lessonId"), body.Delete == nil || *body.Delete)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]bool{"deleted": res.Deleted})
}

// level reads ?level= (A1 when absent); an unknown level gets 400.
func (h *Handler) level(w http.ResponseWriter, r *http.Request) (string, bool) {
	l := r.URL.Query().Get("level")
	if l == "" {
		return "A1", true
	}
	if !slices.Contains(Levels, l) {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_level", "Trình độ không hợp lệ")
		return "", false
	}
	return l, true
}

func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	var verr *lesson.ValidationError
	var perr *PlanError
	switch {
	case errors.As(err, &perr):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "invalid_curriculum", perr.Message)
	case errors.Is(err, ErrNoDraftPlan):
		httpx.WriteError(w, http.StatusConflict, "no_draft", "Trình độ này chưa có bản nháp giáo trình")
	case errors.Is(err, ErrNoAI):
		httpx.WriteError(w, http.StatusServiceUnavailable, "ai_not_configured", "AI chưa được cấu hình. Liên hệ người vận hành.")
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
	case errors.Is(err, ErrAIFailed):
		h.log.WarnContext(r.Context(), "curriculum draft failed", slog.Any("error", err))
		httpx.WriteError(w, http.StatusBadGateway, "ai_failed", "AI soạn giáo trình thất bại, vui lòng thử lại.")
	case lesson.IsGenerateAIError(err):
		h.log.WarnContext(r.Context(), "curriculum generation failed", slog.Any("error", err))
		httpx.WriteError(w, http.StatusBadGateway, "ai_failed", "Sinh bài thất bại, vui lòng thử lại.")
	default:
		h.log.ErrorContext(r.Context(), "curriculum request failed", slog.Any("error", err))
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "Có lỗi xảy ra, vui lòng thử lại")
	}
}

// --- the draft of a course (added 2026-10-23) ---

// draftWriteTimeout replaces the server's write timeout while the AI drafts a course or a topic.
const draftWriteTimeout = 5*time.Minute + 10*time.Second

// maxPlanBytes bounds a draft sent by the admin: about 10 topics of 8 lessons.
const maxPlanBytes = 256 << 10

type planSlotJSON struct {
	Title   string   `json:"title"`
	Idea    string   `json:"idea"`
	Grammar []string `json:"grammar"`
}

type planTopicJSON struct {
	TopicID string         `json:"topicId"`
	Name    string         `json:"name"`
	CanDoVi string         `json:"canDoVi"`
	Grammar []string       `json:"grammar"`
	Slots   []planSlotJSON `json:"slots"`
}

type planJSON struct {
	Topics []planTopicJSON `json:"topics"`
}

func (p planJSON) plan() Plan {
	out := Plan{Topics: make([]Topic, len(p.Topics))}
	for i, t := range p.Topics {
		top := Topic{TopicID: t.TopicID, Name: t.Name, CanDoVi: t.CanDoVi, Grammar: t.Grammar, Slots: make([]Slot, len(t.Slots))}
		for j, s := range t.Slots {
			top.Slots[j] = Slot{Title: s.Title, Idea: s.Idea, Grammar: s.Grammar}
		}
		out.Topics[i] = top
	}
	return out
}

func (h *Handler) extendDeadline(w http.ResponseWriter, r *http.Request) {
	if err := http.NewResponseController(w).SetWriteDeadline(time.Now().Add(draftWriteTimeout)); err != nil &&
		!errors.Is(err, http.ErrNotSupported) {
		h.log.WarnContext(r.Context(), "curriculum: extend write deadline", slog.Any("error", err))
	}
}

// writeBoard writes the board f returns, or its error.
func (h *Handler) writeBoard(w http.ResponseWriter, r *http.Request, f func() (Board, error)) {
	b, err := f()
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toBoardJSON(b))
}

func (h *Handler) draftWithAI(w http.ResponseWriter, r *http.Request) {
	level, ok := h.level(w, r)
	if !ok {
		return
	}
	h.extendDeadline(w, r)
	h.writeBoard(w, r, func() (Board, error) { return h.svc.DraftWithAI(r.Context(), level) })
}

func (h *Handler) writeTopic(w http.ResponseWriter, r *http.Request) {
	level, ok := h.level(w, r)
	if !ok {
		return
	}
	n, err := strconv.Atoi(r.PathValue("number"))
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "Không tìm thấy chủ đề trong bản nháp")
		return
	}
	h.extendDeadline(w, r)
	h.writeBoard(w, r, func() (Board, error) {
		return h.svc.WriteTopic(r.Context(), level, n, r.URL.Query().Get("mode") == "extra")
	})
}

func (h *Handler) saveDraft(w http.ResponseWriter, r *http.Request) {
	level, ok := h.level(w, r)
	if !ok {
		return
	}
	var in planJSON
	if err := httpx.DecodeJSONLimit(w, r, &in, maxPlanBytes); err != nil {
		return
	}
	h.writeBoard(w, r, func() (Board, error) { return h.svc.SaveDraft(r.Context(), level, in.plan()) })
}

func (h *Handler) discardDraft(w http.ResponseWriter, r *http.Request) {
	if level, ok := h.level(w, r); ok {
		h.writeBoard(w, r, func() (Board, error) { return h.svc.DiscardDraft(r.Context(), level) })
	}
}

func (h *Handler) editApproved(w http.ResponseWriter, r *http.Request) {
	if level, ok := h.level(w, r); ok {
		h.writeBoard(w, r, func() (Board, error) { return h.svc.EditApproved(r.Context(), level) })
	}
}

func (h *Handler) approve(w http.ResponseWriter, r *http.Request) {
	if level, ok := h.level(w, r); ok {
		h.writeBoard(w, r, func() (Board, error) { return h.svc.Approve(r.Context(), level) })
	}
}

type courseTopicJSON struct {
	TopicID string `json:"topicId"`
	Name    string `json:"name"`
	Number  int    `json:"number"`
}

// courseTopics answers GET /api/admin/curriculum/topics: per level with an approved course, its
// topics in the catalogue.
func (h *Handler) courseTopics(w http.ResponseWriter, r *http.Request) {
	byLevel, err := h.svc.CourseTopics(r.Context())
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	out := make(map[string][]courseTopicJSON, len(byLevel))
	for level, list := range byLevel {
		js := make([]courseTopicJSON, len(list))
		for i, t := range list {
			js[i] = courseTopicJSON(t)
		}
		out[level] = js
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"levels": out})
}

type levelSummaryJSON struct {
	Level    string `json:"level"`
	Status   Status `json:"status"`
	HasDraft bool   `json:"hasDraft"`
	Applied  bool   `json:"applied"`
	Empty    int    `json:"empty"`
	Drafts   int    `json:"drafts"`
}

// summary answers GET /api/admin/curriculum/summary: where the course of each level is, for the
// admin home.
func (h *Handler) summary(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.Summary(r.Context())
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	out := make([]levelSummaryJSON, len(list))
	for i, l := range list {
		out[i] = levelSummaryJSON(l)
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"levels": out})
}
