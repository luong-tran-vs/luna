package lesson

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/luongtran/luna/backend/internal/job"
	"github.com/luongtran/luna/backend/internal/platform/httpx"
)

// Handler serves the admin lesson API and lesson audio.
type Handler struct {
	svc *Service
	log *slog.Logger
}

// NewHandler returns a Handler for svc.
func NewHandler(svc *Service, log *slog.Logger) *Handler {
	return &Handler{svc: svc, log: log}
}

// Register adds the admin routes (behind requireAuth + admin role) and the audio route
// (behind requireAuth, any role) to mux. audioDir is where mp3 files live.
func (h *Handler) Register(mux *http.ServeMux, requireAuth httpx.Middleware, audioDir string) {
	admin := func(f http.HandlerFunc) http.Handler { return requireAuth(httpx.RequireAdmin(f)) }

	mux.Handle("GET /api/admin/lessons", admin(h.list))
	mux.Handle("POST /api/admin/lessons", admin(h.create))
	mux.Handle("GET /api/admin/lessons/{id}", admin(h.get))
	mux.Handle("PUT /api/admin/lessons/{id}", admin(h.update))
	mux.Handle("DELETE /api/admin/lessons/{id}", admin(h.delete))
	mux.Handle("PUT /api/admin/lessons/{id}/annotations", admin(h.updateAnnotations))
	mux.Handle("POST /api/admin/lessons/{id}/retry", admin(h.retry))
	mux.Handle("PUT /api/admin/lessons/{id}/extras", admin(h.updateExtras))
	mux.Handle("POST /api/admin/topics/{id}/generate", admin(h.generate))
	mux.Handle("GET /api/audio/{lessonId}/{revision}/{index}", requireAuth(AudioHandler(audioDir)))
}

// --- JSON shapes (contracts/admin-lessons-api.md) ---

type summaryJSON struct {
	ID               string    `json:"id"`
	Title            string    `json:"title"`
	Level            Level     `json:"level"`
	TopicID          string    `json:"topicId"`
	TopicName        string    `json:"topicName"`
	AudioStatus      Status    `json:"audioStatus"`
	AnnotationStatus Status    `json:"annotationStatus"`
	InRoadmap        bool      `json:"inRoadmap"`
	CreatedAt        time.Time `json:"createdAt"`
}

type sentenceJSON struct {
	Index    int     `json:"index"`
	Text     string  `json:"text"`
	AudioURL *string `json:"audioUrl"`
}

type annotationJSON struct {
	Text          string `json:"text"`
	Lemma         string `json:"lemma"`
	MeaningVi     string `json:"meaningVi"`
	SentenceIndex int    `json:"sentenceIndex"`
	EditedByAdmin bool   `json:"editedByAdmin"`
}

type lessonJSON struct {
	summaryJSON
	Content         string           `json:"content"`
	Source          string           `json:"source"`
	License         string           `json:"license"`
	Revision        int              `json:"revision"`
	AudioError      string           `json:"audioError"`
	AnnotationError string           `json:"annotationError"`
	Sentences       []sentenceJSON   `json:"sentences"`
	Annotations     []annotationJSON `json:"annotations"`
	// F15
	Questions           []questionJSON   `json:"questions"`
	GrammarNote         *grammarNoteJSON `json:"grammarNote"`
	WritingPrompt       string           `json:"writingPrompt"`
	ExtrasEditedByAdmin bool             `json:"extrasEditedByAdmin"`
	QuizVersion         int              `json:"quizVersion"`
}

type questionJSON struct {
	Prompt        string   `json:"prompt"`
	Options       []string `json:"options"`
	AnswerIndex   int      `json:"answerIndex"`
	ExplanationVi string   `json:"explanationVi"`
}

type grammarNoteJSON struct {
	Title    string   `json:"title"`
	BodyVi   string   `json:"bodyVi"`
	Examples []string `json:"examples"`
}

func toQuestionsJSON(qs []Question) []questionJSON {
	out := make([]questionJSON, len(qs))
	for i, q := range qs {
		out[i] = questionJSON(q)
	}
	return out
}

func toGrammarNoteJSON(n *GrammarNote) *grammarNoteJSON {
	if n == nil {
		return nil
	}
	j := grammarNoteJSON(*n)
	return &j
}

func toSummaries(items []Summary) []summaryJSON {
	out := make([]summaryJSON, len(items))
	for i, s := range items {
		out[i] = summaryJSON(s)
	}
	return out
}

func toLessonJSON(l Lesson, topicName string, inRoadmap bool) map[string]lessonJSON {
	out := lessonJSON{
		summaryJSON: summaryJSON{
			ID: l.ID, Title: l.Title, Level: l.Level, TopicID: l.TopicID, TopicName: topicName,
			AudioStatus: l.AudioStatus, AnnotationStatus: l.AnnotationStatus, InRoadmap: inRoadmap, CreatedAt: l.CreatedAt,
		},
		Content: l.Content, Source: l.Source, License: l.License, Revision: l.Revision,
		AudioError: l.AudioError, AnnotationError: l.AnnotationError,
		Sentences:   make([]sentenceJSON, len(l.Sentences)),
		Annotations: make([]annotationJSON, len(l.Annotations)),
		Questions:   toQuestionsJSON(l.Extras.Questions), GrammarNote: toGrammarNoteJSON(l.Extras.GrammarNote),
		WritingPrompt: l.Extras.WritingPrompt, ExtrasEditedByAdmin: l.ExtrasEditedByAdmin, QuizVersion: l.QuizVersion,
	}
	for i, s := range l.Sentences {
		out.Sentences[i] = sentenceJSON{Index: s.Index, Text: s.Text}
		if s.AudioPath != "" {
			out.Sentences[i].AudioURL = &s.AudioPath
		}
	}
	for i, a := range l.Annotations {
		out.Annotations[i] = annotationJSON(a)
	}
	return map[string]lessonJSON{"lesson": out}
}

type inputJSON struct {
	Title           string `json:"title"`
	Content         string `json:"content"`
	TopicID         string `json:"topicId"`
	Source          string `json:"source"`
	License         string `json:"license"`
	AppendToRoadmap bool   `json:"appendToRoadmap"`
}

func (in inputJSON) toInput() Input { return Input(in) }

// --- handlers ---

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	items, err := h.svc.List(r.Context(), Filter{Level: Level(q.Get("level")), TopicID: q.Get("topicId")})
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string][]summaryJSON{"lessons": toSummaries(items)})
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var in inputJSON
	if httpx.DecodeJSON(w, r, &in) != nil {
		return
	}
	l, err := h.svc.Create(r.Context(), in.toInput())
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeLesson(w, r, http.StatusCreated, l)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	l, err := h.svc.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeLesson(w, r, http.StatusOK, l)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	var in inputJSON
	if httpx.DecodeJSON(w, r, &in) != nil {
		return
	}
	l, err := h.svc.Update(r.Context(), r.PathValue("id"), in.toInput())
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeLesson(w, r, http.StatusOK, l)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), r.PathValue("id")); err != nil {
		h.writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) updateAnnotations(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Annotations []struct {
			Text      string `json:"text"`
			Lemma     string `json:"lemma"`
			MeaningVi string `json:"meaningVi"`
		} `json:"annotations"`
	}
	if httpx.DecodeJSON(w, r, &body) != nil {
		return
	}
	items := make([]AnnotationInput, len(body.Annotations))
	for i, a := range body.Annotations {
		items[i] = AnnotationInput(a)
	}
	l, err := h.svc.UpdateAnnotations(r.Context(), r.PathValue("id"), items)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeLesson(w, r, http.StatusOK, l)
}

func (h *Handler) retry(w http.ResponseWriter, r *http.Request) {
	t := job.Type(r.URL.Query().Get("job"))
	if t != job.TypeTTS && t != job.TypeAnnotate {
		httpx.WriteFieldErrors(w, map[string]string{"job": "Chỉ chạy lại được tts hoặc annotate"})
		return
	}
	l, err := h.svc.Retry(r.Context(), r.PathValue("id"), t)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeLesson(w, r, http.StatusAccepted, l)
}

// writeLesson includes the topic name and whether the lesson is in its topic's roadmap.
func (h *Handler) writeLesson(w http.ResponseWriter, r *http.Request, status int, l Lesson) {
	inRoadmap, err := h.svc.InRoadmap(r.Context(), l.ID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	name, err := h.svc.TopicName(r.Context(), l.TopicID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, status, toLessonJSON(l, name, inRoadmap))
}

func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	var verr *ValidationError
	switch {
	case errors.As(err, &verr):
		httpx.WriteFieldErrors(w, verr.Fields)
	case errors.Is(err, ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "not_found", "Không tìm thấy bài học")
	case errors.Is(err, ErrInRoadmap):
		httpx.WriteError(w, http.StatusConflict, "lesson_in_roadmap", "Gỡ bài khỏi lộ trình trước khi xoá")
	case errors.Is(err, ErrNotFailed):
		httpx.WriteError(w, http.StatusConflict, "not_failed", "Chỉ chạy lại được việc đang lỗi")
	case errors.Is(err, ErrAnnotationRunning):
		httpx.WriteError(w, http.StatusConflict, "annotation_running", "Chú thích đang được tạo, vui lòng chờ")
	default:
		h.log.ErrorContext(r.Context(), "lesson request failed", slog.Any("error", err))
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "Có lỗi xảy ra, vui lòng thử lại")
	}
}
