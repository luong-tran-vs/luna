package lesson

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/luongtran/luna/backend/internal/job"
	"github.com/luongtran/luna/backend/internal/platform/httpx"
)

// Handler serves the admin lesson API.
type Handler struct {
	svc *Service
	log *slog.Logger
}

// NewHandler returns a Handler for svc.
func NewHandler(svc *Service, log *slog.Logger) *Handler {
	return &Handler{svc: svc, log: log}
}

// Register adds the admin routes (behind requireAuth + admin role) to mux.
func (h *Handler) Register(mux *http.ServeMux, requireAuth httpx.Middleware) {
	admin := func(f http.HandlerFunc) http.Handler { return requireAuth(httpx.RequireAdmin(f)) }

	mux.Handle("GET /api/admin/lessons", admin(h.list))
	mux.Handle("POST /api/admin/lessons", admin(h.create))
	mux.Handle("GET /api/admin/lessons/{id}", admin(h.get))
	mux.Handle("PUT /api/admin/lessons/{id}", admin(h.update))
	mux.Handle("DELETE /api/admin/lessons/{id}", admin(h.delete))
	mux.Handle("PUT /api/admin/lessons/{id}/published", admin(h.setPublished))
	mux.Handle("PUT /api/admin/lessons/{id}/annotations", admin(h.updateAnnotations))
	mux.Handle("POST /api/admin/lessons/{id}/retry", admin(h.retry))
	mux.Handle("PUT /api/admin/lessons/{id}/extras", admin(h.updateExtras))
	mux.Handle("PUT /api/admin/lessons/{id}/practice/translations", admin(h.updateTranslations))
	mux.Handle("POST /api/admin/lessons/{id}/practice/regenerate", admin(h.regeneratePractice))
	mux.Handle("GET /api/admin/lessons/{id}/images", admin(h.images))
	mux.Handle("PUT /api/admin/lessons/{id}/images", admin(h.setImages))
	mux.Handle("GET /api/admin/lessons/{id}/images/{lemma}", admin(h.adminImage))
	mux.Handle("PUT /api/admin/lessons/{id}/images/{lemma}", admin(h.uploadImage))
	mux.Handle("DELETE /api/admin/lessons/{id}/images/{lemma}", admin(h.deleteImage))
	mux.Handle("POST /api/admin/lessons/{id}/images/{lemma}/import", admin(h.importImage))
	mux.Handle("POST /api/admin/lessons/{id}/check", admin(h.check))
	mux.Handle("POST /api/admin/lessons/{id}/check/confirm", admin(h.confirmFlag))
	mux.Handle("POST /api/admin/lessons/{id}/check/verify", admin(h.verify))
	mux.Handle("POST /api/admin/lessons/{id}/check/suggest", admin(h.suggestFix))
	mux.Handle("POST /api/admin/lessons/{id}/check/apply", admin(h.applyFix))
	mux.Handle("POST /api/admin/topics/{id}/generate", admin(h.generate))
	mux.Handle("GET /api/admin/grammar", admin(h.grammarPoints))
}

type grammarPointJSON struct {
	ID          string   `json:"id"`
	Level       string   `json:"level"`
	TitleVi     string   `json:"titleVi"`
	TitleEn     string   `json:"titleEn"`
	Pattern     string   `json:"pattern"`
	HintVi      string   `json:"hintVi"`
	Examples    []string `json:"examples"`
	LessonCount int      `json:"lessonCount"`
}

func (h *Handler) grammarPoints(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	pts, err := h.svc.GrammarPoints(r.Context(), q.Get("level"), q.Get("topicId"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	out := make([]grammarPointJSON, len(pts))
	for i, p := range pts {
		ex := p.Examples
		if ex == nil {
			ex = []string{}
		}
		out[i] = grammarPointJSON{
			ID: p.ID, Level: p.Level, TitleVi: p.TitleVi, TitleEn: p.TitleEn, Pattern: p.Pattern,
			HintVi: p.HintVi, Examples: ex, LessonCount: p.LessonCount,
		}
	}
	httpx.WriteJSON(w, http.StatusOK, map[string][]grammarPointJSON{"points": out})
}

// --- JSON shapes (contracts/admin-lessons-api.md) ---

type summaryJSON struct {
	ID               string    `json:"id"`
	Title            string    `json:"title"`
	Level            Level     `json:"level"`
	TopicID          string    `json:"topicId"`
	TopicName        string    `json:"topicName"`
	AnnotationStatus Status    `json:"annotationStatus"`
	InRoadmap        bool      `json:"inRoadmap"`
	Flags            int       `json:"flags"`
	Checked          bool      `json:"checked"`
	Verified         bool      `json:"verified"`
	Draft            bool      `json:"draft"`
	CreatedAt        time.Time `json:"createdAt"`
}

type sentenceJSON struct {
	Index int    `json:"index"`
	Text  string `json:"text"`
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
	AnnotationError string           `json:"annotationError"`
	Sentences       []sentenceJSON   `json:"sentences"`
	Annotations     []annotationJSON `json:"annotations"`
	// F15
	Questions           []questionJSON   `json:"questions"`
	GrammarNote         *grammarNoteJSON `json:"grammarNote"`
	WritingPrompt       string           `json:"writingPrompt"`
	ExtrasEditedByAdmin bool             `json:"extrasEditedByAdmin"`
	QuizVersion         int              `json:"quizVersion"`
	GrammarPointID      string           `json:"grammarPointId"`
	GrammarPointTitle   string           `json:"grammarPointTitle"`
	// F17
	PracticeStatus string             `json:"practiceStatus"`
	PracticeError  string             `json:"practiceError"`
	Practice       *adminPracticeJSON `json:"practice"`
	// F22
	Review *reviewJSON `json:"review"`
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
	flags, checked, verified := SummaryOf(l.Review)
	out := lessonJSON{
		summaryJSON: summaryJSON{
			ID: l.ID, Title: l.Title, Level: l.Level, TopicID: l.TopicID, TopicName: topicName,
			AnnotationStatus: l.AnnotationStatus, InRoadmap: inRoadmap, CreatedAt: l.CreatedAt,
			Flags: flags, Checked: checked, Verified: verified, Draft: l.Draft,
		},
		Content: l.Content, Source: l.Source, License: l.License, Revision: l.Revision,
		AnnotationError: l.AnnotationError,
		Sentences:       make([]sentenceJSON, len(l.Sentences)),
		Annotations:     make([]annotationJSON, len(l.Annotations)),
		Questions:       toQuestionsJSON(l.Extras.Questions), GrammarNote: toGrammarNoteJSON(l.Extras.GrammarNote),
		WritingPrompt: l.Extras.WritingPrompt, ExtrasEditedByAdmin: l.ExtrasEditedByAdmin, QuizVersion: l.QuizVersion,
		PracticeStatus: practiceStatusJSON(l.PracticeStatus), PracticeError: l.PracticeError,
		Practice:       toAdminPracticeJSON(l.Practice),
		GrammarPointID: l.GrammarPointID, GrammarPointTitle: GrammarTitle(l.GrammarPointID),
		Review: toReviewJSON(l.Review),
	}
	for i, s := range l.Sentences {
		out.Sentences[i] = sentenceJSON(s)
	}
	for i, a := range l.Annotations {
		out.Annotations[i] = annotationJSON(a)
	}
	return map[string]lessonJSON{"lesson": out}
}

type inputJSON struct {
	Title           string  `json:"title"`
	Content         string  `json:"content"`
	TopicID         string  `json:"topicId"`
	Level           Level   `json:"level"`
	Source          string  `json:"source"`
	License         string  `json:"license"`
	GrammarPointID  *string `json:"grammarPointId"`
	AppendToRoadmap bool    `json:"appendToRoadmap"`
	// Images turns on the word pictures of a new lesson (F23); omitted leaves them off.
	Images *imageInputJSON `json:"images"`
	// TargetWords are the topic words of a generated draft (F18); omitted means none.
	TargetWords []string `json:"targetWords"`
	// Draft saves a new lesson hidden from learners until it is published.
	Draft bool `json:"draft"`
}

type imageInputJSON struct {
	Enabled bool   `json:"enabled"`
	Style   string `json:"style"`
}

func (in inputJSON) toInput() Input {
	out := Input{
		Title: in.Title, Content: in.Content, TopicID: in.TopicID, Level: in.Level, Source: in.Source, License: in.License,
		AppendToRoadmap: in.AppendToRoadmap, KeepGrammarPoint: in.GrammarPointID == nil,
		TargetWords: in.TargetWords, Draft: in.Draft,
	}
	if in.GrammarPointID != nil {
		out.GrammarPointID = *in.GrammarPointID
	}
	if in.Images != nil {
		images := ImageInput(*in.Images)
		out.Images = &images
	}
	return out
}

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

func (h *Handler) setPublished(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Published bool `json:"published"`
	}
	if httpx.DecodeJSON(w, r, &body) != nil {
		return
	}
	l, err := h.svc.SetPublished(r.Context(), r.PathValue("id"), body.Published)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeLesson(w, r, http.StatusOK, l)
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
	if t != job.TypeAnnotate {
		httpx.WriteFieldErrors(w, map[string]string{"job": "Chỉ chạy lại được annotate"})
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
	case errors.Is(err, ErrFixTarget):
		httpx.WriteError(w, http.StatusNotFound, "not_found", "Không tìm thấy chỗ cần sửa trong bài")
	case errors.Is(err, ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "not_found", "Không tìm thấy bài học")
	case errors.Is(err, ErrInRoadmap):
		httpx.WriteError(w, http.StatusConflict, "lesson_in_roadmap", "Gỡ bài khỏi lộ trình trước khi xoá")
	case errors.Is(err, ErrNotFailed):
		httpx.WriteError(w, http.StatusConflict, "not_failed", "Chỉ chạy lại được việc đang lỗi")
	case errors.Is(err, ErrAnnotationRunning):
		httpx.WriteError(w, http.StatusConflict, "annotation_running", "Chú thích đang được tạo, vui lòng chờ")
	case errors.Is(err, ErrPracticeRunning):
		httpx.WriteError(w, http.StatusConflict, "practice_running", "Phần luyện tập đang được tạo, vui lòng chờ")
	case errors.Is(err, ErrNoPractice):
		httpx.WriteError(w, http.StatusConflict, "no_practice", "Bài chưa có phần luyện tập")
	case errors.Is(err, ErrPracticeChanged):
		httpx.WriteError(w, http.StatusConflict, "practice_changed", "Phần luyện tập vừa thay đổi, hãy tải lại")
	case writeFlagError(w, err):
	case errors.Is(err, ErrAnnotationNotDone):
		httpx.WriteError(w, http.StatusConflict, "annotation_not_done", "Cần chú thích xong trước")
	case errors.Is(err, ErrNotLessonWord):
		httpx.WriteError(w, http.StatusNotFound, "not_lesson_word", "Từ này không thuộc từ vựng của bài")
	case errors.Is(err, ErrImagesUnavailable):
		httpx.WriteError(w, http.StatusServiceUnavailable, "images_unavailable", "Máy chủ chưa hỗ trợ ảnh từ vựng")
	default:
		h.log.ErrorContext(r.Context(), "lesson request failed", slog.Any("error", err))
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "Có lỗi xảy ra, vui lòng thử lại")
	}
}
