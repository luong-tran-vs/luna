package topic

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"slices"
	"strconv"
	"time"

	"github.com/luongtran/luna/backend/internal/ai"
	"github.com/luongtran/luna/backend/internal/platform/httpx"
)

// Handler serves the admin topic API and the learner topic list.
type Handler struct {
	svc *Service
	log *slog.Logger
}

// NewHandler returns a Handler for svc.
func NewHandler(svc *Service, log *slog.Logger) *Handler {
	return &Handler{svc: svc, log: log}
}

// Register adds the admin routes (behind requireAuth + admin role) and the learner list
// (behind requireAuth, any role).
func (h *Handler) Register(mux *http.ServeMux, requireAuth httpx.Middleware) {
	admin := func(f http.HandlerFunc) http.Handler { return requireAuth(httpx.RequireAdmin(f)) }

	mux.Handle("GET /api/admin/topics", admin(h.list))
	mux.Handle("POST /api/admin/topics", admin(h.create))
	mux.Handle("PUT /api/admin/topics/{id}", admin(h.update))
	mux.Handle("DELETE /api/admin/topics/{id}", admin(h.delete))
	mux.Handle("GET /api/admin/topics/{id}/roadmap", admin(h.getRoadmap))
	mux.Handle("PUT /api/admin/topics/{id}/roadmap", admin(h.setRoadmap))
	mux.Handle("GET /api/admin/topics/{id}/words", admin(h.getWords))
	mux.Handle("PUT /api/admin/topics/{id}/words", admin(h.setWords))
	mux.Handle("GET /api/admin/topics/{id}/word-plan", admin(h.wordPlan))
	mux.Handle("POST /api/admin/topics/{id}/words/suggest", admin(h.suggestWords))
	mux.Handle("GET /api/topics", requireAuth(http.HandlerFunc(h.public)))
}

// --- JSON shapes (contracts/topics-api.md) ---

type levelJSON struct {
	Level        string `json:"level"`
	LessonCount  int    `json:"lessonCount"`
	RoadmapCount int    `json:"roadmapCount"`
	Remaining    int    `json:"remaining"`
	Warning      bool   `json:"warning"`
}

type topicJSON struct {
	ID            string      `json:"id"`
	Name          string      `json:"name"`
	Description   string      `json:"description"`
	LessonCount   int         `json:"lessonCount"`
	Levels        []levelJSON `json:"levels"`
	WordCount     int         `json:"wordCount"`
	UsedWordCount int         `json:"usedWordCount"`
	CreatedAt     time.Time   `json:"createdAt"`
}

func toJSON(s Summary) topicJSON {
	out := topicJSON{
		ID: s.ID, Name: s.Name, Description: s.Description, LessonCount: s.LessonCount,
		Levels: make([]levelJSON, len(s.Levels)), CreatedAt: s.CreatedAt,
		WordCount: s.WordCount, UsedWordCount: s.UsedWordCount,
	}
	for i, l := range s.Levels {
		out.Levels[i] = levelJSON(l)
	}
	return out
}

// lessonJSON matches the admin lesson summary (contracts/admin-lessons-api.md).
type lessonJSON struct {
	ID               string    `json:"id"`
	Title            string    `json:"title"`
	Level            string    `json:"level"`
	TopicID          string    `json:"topicId"`
	TopicName        string    `json:"topicName"`
	AnnotationStatus string    `json:"annotationStatus"`
	InRoadmap        bool      `json:"inRoadmap"`
	CreatedAt        time.Time `json:"createdAt"`
}

type roadmapJSON struct {
	Topic     topicJSON    `json:"topic"`
	Level     string       `json:"level"`
	Lessons   []lessonJSON `json:"lessons"`
	Remaining int          `json:"remaining"`
	Warning   bool         `json:"warning"`
}

func toRoadmapJSON(r Roadmap) roadmapJSON {
	lv := r.Topic.Level(r.Level)
	out := roadmapJSON{
		Topic: toJSON(r.Topic), Level: r.Level, Lessons: make([]lessonJSON, len(r.Lessons)),
		Remaining: lv.Remaining, Warning: lv.Warning,
	}
	for i, l := range r.Lessons {
		out.Lessons[i] = lessonJSON{
			ID: l.ID, Title: l.Title, Level: l.Level, TopicID: r.Topic.ID, TopicName: r.Topic.Name,
			AnnotationStatus: l.AnnotationStatus, InRoadmap: true, CreatedAt: l.CreatedAt,
		}
	}
	return out
}

type publicJSON struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Level       string `json:"level"`
	Description string `json:"description"`
	LessonCount int    `json:"lessonCount"`
}

type inputJSON struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type inUseBody struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Count   int    `json:"count"`
}

// --- handlers ---

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.List(r.Context())
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	out := make([]topicJSON, len(items))
	for i, s := range items {
		out[i] = toJSON(s)
	}
	httpx.WriteJSON(w, http.StatusOK, map[string][]topicJSON{"topics": out})
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var in inputJSON
	if httpx.DecodeJSON(w, r, &in) != nil {
		return
	}
	s, err := h.svc.Create(r.Context(), Input(in))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, map[string]topicJSON{"topic": toJSON(s)})
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	var in inputJSON
	if httpx.DecodeJSON(w, r, &in) != nil {
		return
	}
	s, err := h.svc.Update(r.Context(), r.PathValue("id"), Input(in))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]topicJSON{"topic": toJSON(s)})
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), r.PathValue("id")); err != nil {
		h.writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) getRoadmap(w http.ResponseWriter, r *http.Request) {
	rm, err := h.svc.Roadmap(r.Context(), r.PathValue("id"), r.URL.Query().Get("level"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toRoadmapJSON(rm))
}

func (h *Handler) setRoadmap(w http.ResponseWriter, r *http.Request) {
	var body struct {
		LessonIDs []string `json:"lessonIds"`
	}
	if httpx.DecodeJSON(w, r, &body) != nil {
		return
	}
	rm, err := h.svc.SetRoadmap(r.Context(), r.PathValue("id"), r.URL.Query().Get("level"), body.LessonIDs)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toRoadmapJSON(rm))
}

func (h *Handler) public(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.Public(r.Context(), r.URL.Query().Get("level"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	out := make([]publicJSON, len(items))
	for i, p := range items {
		out[i] = publicJSON(p)
	}
	httpx.WriteJSON(w, http.StatusOK, map[string][]publicJSON{"topics": out})
}

func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	var verr *ValidationError
	var inUse *InUseError
	switch {
	case errors.As(err, &verr):
		httpx.WriteFieldErrors(w, verr.Fields)
	case errors.As(err, &inUse):
		httpx.WriteJSON(w, http.StatusConflict, inUseBody{
			Error: "topic_in_use", Count: inUse.Count,
			Message: "Chủ đề còn " + strconv.Itoa(inUse.Count) + " bài, hãy chuyển hoặc xoá bài trước",
		})
	case errors.Is(err, ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "not_found", "Không tìm thấy chủ đề")
	default:
		h.log.ErrorContext(r.Context(), "topic request failed", slog.Any("error", err))
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "Có lỗi xảy ra, vui lòng thử lại")
	}
}

type wordJSON struct {
	Text        string `json:"text"`
	Level       string `json:"level"`
	Used        bool   `json:"used"`
	LessonCount int    `json:"lessonCount"`
}

func toWordsJSON(uses []WordUse) []wordJSON {
	out := make([]wordJSON, len(uses))
	for i, u := range uses {
		out[i] = wordJSON(u)
	}
	return out
}

func writeWords(w http.ResponseWriter, uses []WordUse) {
	httpx.WriteJSON(w, http.StatusOK, map[string][]wordJSON{"words": toWordsJSON(uses)})
}

// wordInJSON is a word of PUT .../words: {"text", "level"}, or a plain string (any level).
type wordInJSON Word

func (w *wordInJSON) UnmarshalJSON(b []byte) error {
	var text string
	if json.Unmarshal(b, &text) == nil {
		*w = wordInJSON{Text: text}
		return nil
	}
	var obj struct {
		Text  string `json:"text"`
		Level string `json:"level"`
	}
	if err := json.Unmarshal(b, &obj); err != nil {
		return err
	}
	*w = wordInJSON(obj)
	return nil
}

// getWords lists the topic's words; ?level= keeps the words of exactly that level.
func (h *Handler) getWords(w http.ResponseWriter, r *http.Request) {
	uses, err := h.svc.Words(r.Context(), r.PathValue("id"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	if level := r.URL.Query().Get("level"); level != "" {
		uses = slices.DeleteFunc(uses, func(u WordUse) bool { return u.Level != level })
	}
	writeWords(w, uses)
}

func (h *Handler) setWords(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Words []wordInJSON `json:"words"`
	}
	if httpx.DecodeJSON(w, r, &body) != nil {
		return
	}
	words := make([]Word, len(body.Words))
	for i, wd := range body.Words {
		words[i] = Word(wd)
	}
	uses, err := h.svc.SetWords(r.Context(), r.PathValue("id"), words)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	writeWords(w, uses)
}

func (h *Handler) wordPlan(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	count, countErr := strconv.Atoi(q.Get("count"))
	perLesson, perErr := strconv.Atoi(q.Get("perLesson"))
	if countErr != nil || perErr != nil {
		fields := map[string]string{}
		if countErr != nil {
			fields["count"] = "Số bài không hợp lệ"
		}
		if perErr != nil {
			fields["perLesson"] = "Số từ mỗi bài không hợp lệ"
		}
		httpx.WriteFieldErrors(w, fields)
		return
	}
	plan, err := h.svc.WordPlan(r.Context(), r.PathValue("id"), q.Get("level"), count, perLesson)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, wordPlanJSON(plan))
}

type wordPlanJSON struct {
	Groups   [][]string `json:"groups"`
	Shortage int        `json:"shortage"`
}

type suggestWordsBody struct {
	Count int    `json:"count"`
	Level string `json:"level"`
}

// suggestWords adds AI-suggested words to the topic (F18) and returns the added words and the
// whole list with coverage.
func (h *Handler) suggestWords(w http.ResponseWriter, r *http.Request) {
	var body suggestWordsBody
	if httpx.DecodeJSON(w, r, &body) != nil {
		return
	}
	added, uses, err := h.svc.SuggestWords(r.Context(), r.PathValue("id"), body.Level, body.Count)
	switch {
	case errors.Is(err, ai.ErrNotConfigured):
		httpx.WriteError(w, http.StatusServiceUnavailable, "ai_not_configured", "AI chưa được cấu hình. Liên hệ người vận hành.")
	case errors.Is(err, ai.ErrInvalidKey):
		httpx.WriteError(w, http.StatusServiceUnavailable, "ai_not_configured", "Khoá AI không hợp lệ. Liên hệ người vận hành.")
	case errors.Is(err, ai.ErrQuota):
		httpx.WriteError(w, http.StatusTooManyRequests, "ai_quota", "Đã hết lượt AI, vui lòng thử lại sau.")
	case errors.Is(err, ErrNoSuggestion):
		httpx.WriteError(w, http.StatusBadGateway, "ai_unusable", "AI không gợi ý được từ mới, vui lòng thử lại.")
	case err != nil:
		h.writeError(w, r, err)
	default:
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"added": added, "words": toWordsJSON(uses)})
	}
}
