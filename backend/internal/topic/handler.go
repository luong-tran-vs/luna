package topic

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

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
	mux.Handle("GET /api/topics", requireAuth(http.HandlerFunc(h.public)))
}

// --- JSON shapes (contracts/topics-api.md) ---

type topicJSON struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Level         string    `json:"level"`
	Description   string    `json:"description"`
	LessonCount   int       `json:"lessonCount"`
	RoadmapCount  int       `json:"roadmapCount"`
	Remaining     int       `json:"remaining"`
	Warning       bool      `json:"warning"`
	WordCount     int       `json:"wordCount"`
	UsedWordCount int       `json:"usedWordCount"`
	CreatedAt     time.Time `json:"createdAt"`
}

func toJSON(s Summary) topicJSON {
	return topicJSON{
		ID: s.ID, Name: s.Name, Level: s.Level, Description: s.Description, LessonCount: s.LessonCount,
		RoadmapCount: len(s.LessonIDs), Remaining: s.Remaining, Warning: s.Warning, CreatedAt: s.CreatedAt,
		WordCount: s.WordCount, UsedWordCount: s.UsedWordCount,
	}
}

// lessonJSON matches the admin lesson summary (contracts/admin-lessons-api.md).
type lessonJSON struct {
	ID               string    `json:"id"`
	Title            string    `json:"title"`
	Level            string    `json:"level"`
	TopicID          string    `json:"topicId"`
	TopicName        string    `json:"topicName"`
	AudioStatus      string    `json:"audioStatus"`
	AnnotationStatus string    `json:"annotationStatus"`
	InRoadmap        bool      `json:"inRoadmap"`
	CreatedAt        time.Time `json:"createdAt"`
}

type roadmapJSON struct {
	Topic     topicJSON    `json:"topic"`
	Lessons   []lessonJSON `json:"lessons"`
	Remaining int          `json:"remaining"`
	Warning   bool         `json:"warning"`
}

func toRoadmapJSON(r Roadmap) roadmapJSON {
	out := roadmapJSON{
		Topic: toJSON(r.Topic), Lessons: make([]lessonJSON, len(r.Lessons)),
		Remaining: r.Topic.Remaining, Warning: r.Topic.Warning,
	}
	for i, l := range r.Lessons {
		out.Lessons[i] = lessonJSON{
			ID: l.ID, Title: l.Title, Level: l.Level, TopicID: r.Topic.ID, TopicName: r.Topic.Name,
			AudioStatus: l.AudioStatus, AnnotationStatus: l.AnnotationStatus, InRoadmap: true, CreatedAt: l.CreatedAt,
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
	Level       string `json:"level"`
	Description string `json:"description"`
}

type inUseBody struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Count   int    `json:"count"`
}

// --- handlers ---

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.List(r.Context(), r.URL.Query().Get("level"))
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
	rm, err := h.svc.Roadmap(r.Context(), r.PathValue("id"))
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
	rm, err := h.svc.SetRoadmap(r.Context(), r.PathValue("id"), body.LessonIDs)
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
	Used        bool   `json:"used"`
	LessonCount int    `json:"lessonCount"`
}

func writeWords(w http.ResponseWriter, uses []WordUse) {
	out := make([]wordJSON, len(uses))
	for i, u := range uses {
		out[i] = wordJSON(u)
	}
	httpx.WriteJSON(w, http.StatusOK, map[string][]wordJSON{"words": out})
}

func (h *Handler) getWords(w http.ResponseWriter, r *http.Request) {
	uses, err := h.svc.Words(r.Context(), r.PathValue("id"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	writeWords(w, uses)
}

func (h *Handler) setWords(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Words []string `json:"words"`
	}
	if httpx.DecodeJSON(w, r, &body) != nil {
		return
	}
	uses, err := h.svc.SetWords(r.Context(), r.PathValue("id"), body.Words)
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
	groups, err := h.svc.WordPlan(r.Context(), r.PathValue("id"), count, perLesson)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string][][]string{"groups": groups})
}
