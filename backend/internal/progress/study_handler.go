package progress

import (
	"errors"
	"log/slog"
	"net/http"
	"slices"
	"time"

	"github.com/luongtran/luna/backend/internal/platform/httpx"
)

// goalLevels are the CEFR levels a goal can be set at.
var goalLevels = []string{"A1", "A2", "B1", "B2", "C1", "C2"}

// StudyHandler serves the study flow API (contracts/study-api.md) and guards lesson content.
type StudyHandler struct {
	svc *StudyService
	log *slog.Logger
}

// NewStudyHandler returns a StudyHandler.
func NewStudyHandler(svc *StudyService, log *slog.Logger) *StudyHandler {
	return &StudyHandler{svc: svc, log: log}
}

// Register adds the routes behind requireAuth.
func (h *StudyHandler) Register(mux *http.ServeMux, requireAuth httpx.Middleware) {
	mux.Handle("GET /api/goals", requireAuth(http.HandlerFunc(h.goals)))
	mux.Handle("POST /api/goals", requireAuth(http.HandlerFunc(h.setGoal)))
	mux.Handle("GET /api/lessons/mine", requireAuth(http.HandlerFunc(h.mine)))
	mux.Handle("GET /api/lessons/{id}/study", requireAuth(h.Guard(http.HandlerFunc(h.study))))
	mux.Handle("POST /api/lessons/{id}/steps/{step}/complete", requireAuth(http.HandlerFunc(h.complete)))
	mux.Handle("POST /api/lessons/{id}/steps/write/skip", requireAuth(http.HandlerFunc(h.skipWrite)))
	mux.Handle("PUT /api/lessons/{id}/position", requireAuth(http.HandlerFunc(h.position)))
	mux.Handle("GET /api/dashboard", requireAuth(http.HandlerFunc(h.dashboard)))
	mux.Handle("GET /api/stats", requireAuth(http.HandlerFunc(h.stats)))
}

// Guard lets a learner open only the lesson being studied and lessons already started; admins pass.
// It must run after RequireAuth on routes with an {id} lesson path value.
func (h *StudyHandler) Guard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p, _ := httpx.PrincipalFrom(r.Context())
		ok, err := h.svc.CanOpen(r.Context(), p.UserID, p.Role == "admin", r.PathValue("id"))
		if err != nil {
			h.fail(w, r, err)
			return
		}
		if !ok {
			httpx.WriteError(w, http.StatusForbidden, "lesson_locked", "Bài này sẽ mở khi tới lượt")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// --- JSON shapes ---

type goalJSON struct {
	TopicID          string `json:"topicId"`
	TopicName        string `json:"topicName"`
	Level            string `json:"level"`
	CompletedLessons int    `json:"completedLessons"`
	TotalLessons     int    `json:"totalLessons"`
	Status           string `json:"status"`
	EffectiveFrom    string `json:"effectiveFrom"`
}

func toGoalJSON(g GoalView) goalJSON {
	return goalJSON{
		TopicID: g.TopicID, TopicName: g.TopicName, Level: g.Level, CompletedLessons: g.CompletedLessons,
		TotalLessons: g.TotalLessons, Status: string(g.Status), EffectiveFrom: g.EffectiveFrom,
	}
}

type goalsJSON struct {
	Active *goalJSON  `json:"active"`
	Others []goalJSON `json:"others"`
}

type setGoalJSON struct {
	Active goalJSON `json:"active"`
}

type lessonRefJSON struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	// MembersOnly marks an upcoming lesson a guest cannot study.
	MembersOnly bool `json:"membersOnly,omitempty"`
}

type studyJSON struct {
	Status        string            `json:"status"`
	Steps         map[string]string `json:"steps"`
	CurrentStep   string            `json:"currentStep"`
	SentenceIndex int               `json:"sentenceIndex"`
	Next          *lessonRefJSON    `json:"next"`
	MembersOnly   bool              `json:"membersOnly,omitempty"`
	Goal          *goalJSON         `json:"goal"`
	GoalCompleted bool              `json:"goalCompleted"`
	Streak        int               `json:"streak"`
}

func toStudyJSON(v LessonStudyView) studyJSON {
	out := studyJSON{
		Status: string(v.Status), Steps: map[string]string{}, CurrentStep: string(v.CurrentStep),
		SentenceIndex: v.SentenceIndex, Streak: v.Streak, GoalCompleted: v.GoalCompleted, MembersOnly: v.MembersOnly,
	}
	if v.Goal != nil {
		g := toGoalJSON(*v.Goal)
		out.Goal = &g
	}
	if v.Next != nil {
		out.Next = &lessonRefJSON{ID: v.Next.ID, Title: v.Next.Title}
	}
	for s, st := range v.Steps {
		out.Steps[string(s)] = string(st)
	}
	return out
}

type completedJSON struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	TopicName   string    `json:"topicName"`
	CompletedAt time.Time `json:"completedAt"`
}

type mineJSON struct {
	Current   *lessonRefJSON  `json:"current"`
	Completed []completedJSON `json:"completed"`
	Upcoming  []lessonRefJSON `json:"upcoming"`
}

// --- handlers ---

func (h *StudyHandler) goals(w http.ResponseWriter, r *http.Request) {
	p, _ := httpx.PrincipalFrom(r.Context())
	v, err := h.svc.Goals(r.Context(), p.UserID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	out := goalsJSON{Others: make([]goalJSON, len(v.Others))}
	if v.Active != nil {
		g := toGoalJSON(*v.Active)
		out.Active = &g
	}
	for i, g := range v.Others {
		out.Others[i] = toGoalJSON(g)
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *StudyHandler) setGoal(w http.ResponseWriter, r *http.Request) {
	p, _ := httpx.PrincipalFrom(r.Context())
	var req struct {
		TopicID string `json:"topicId"`
		Level   string `json:"level"`
	}
	if httpx.DecodeJSON(w, r, &req) != nil {
		return
	}
	fields := map[string]string{}
	if req.TopicID == "" {
		fields["topicId"] = "Vui lòng chọn chủ đề"
	}
	if !slices.Contains(goalLevels, req.Level) {
		fields["level"] = "Vui lòng chọn trình độ"
	}
	if len(fields) > 0 {
		httpx.WriteFieldErrors(w, fields)
		return
	}
	res, err := h.svc.SetGoal(r.Context(), p.UserID, req.TopicID, req.Level)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, setGoalJSON{Active: toGoalJSON(res)})
}

func (h *StudyHandler) study(w http.ResponseWriter, r *http.Request) {
	p, _ := httpx.PrincipalFrom(r.Context())
	v, err := h.svc.LessonStudy(r.Context(), p.UserID, r.PathValue("id"))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toStudyJSON(v))
}

func (h *StudyHandler) complete(w http.ResponseWriter, r *http.Request) {
	p, _ := httpx.PrincipalFrom(r.Context())
	v, err := h.svc.CompleteStep(r.Context(), p.UserID, r.PathValue("id"), Step(r.PathValue("step")))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toStudyJSON(v))
}

// skipWrite finishes the lesson without writing (the Write step is optional).
func (h *StudyHandler) skipWrite(w http.ResponseWriter, r *http.Request) {
	p, _ := httpx.PrincipalFrom(r.Context())
	v, err := h.svc.SkipWrite(r.Context(), p.UserID, r.PathValue("id"))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toStudyJSON(v))
}

func (h *StudyHandler) position(w http.ResponseWriter, r *http.Request) {
	p, _ := httpx.PrincipalFrom(r.Context())
	var req struct {
		Step          string `json:"step"`
		SentenceIndex int    `json:"sentenceIndex"`
	}
	if httpx.DecodeJSON(w, r, &req) != nil {
		return
	}
	if err := h.svc.SetPosition(r.Context(), p.UserID, r.PathValue("id"), Step(req.Step), req.SentenceIndex); err != nil {
		h.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *StudyHandler) mine(w http.ResponseWriter, r *http.Request) {
	p, _ := httpx.PrincipalFrom(r.Context())
	v, err := h.svc.MyLessons(r.Context(), p.UserID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	out := mineJSON{Completed: make([]completedJSON, len(v.Completed)), Upcoming: make([]lessonRefJSON, len(v.Upcoming))}
	if v.Current != nil {
		out.Current = &lessonRefJSON{ID: v.Current.ID, Title: v.Current.Title}
	}
	for i, c := range v.Completed {
		out.Completed[i] = completedJSON{ID: c.ID, Title: c.Title, TopicName: c.TopicName, CompletedAt: c.CompletedAt}
	}
	for i, u := range v.Upcoming {
		out.Upcoming[i] = lessonRefJSON(u)
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *StudyHandler) fail(w http.ResponseWriter, r *http.Request, err error) {
	var verr *ValidationError
	switch {
	case errors.As(err, &verr):
		httpx.WriteFieldErrors(w, verr.Fields)
	case errors.Is(err, ErrTopicNotFound):
		httpx.WriteError(w, http.StatusNotFound, "not_found", "Không tìm thấy chủ đề")
	case errors.Is(err, ErrWriteIncomplete):
		httpx.WriteError(w, http.StatusConflict, "write_incomplete", "Hãy nộp bài viết")
	case errors.Is(err, ErrReadIncomplete):
		httpx.WriteError(w, http.StatusConflict, "read_incomplete", "Hãy trả lời hết câu hỏi hiểu bài")
	case errors.Is(err, ErrListenIncomplete):
		httpx.WriteError(w, http.StatusConflict, "listen_incomplete", "Hãy kiểm tra hết các câu của bước Nghe")
	case errors.Is(err, ErrNotCurrentLesson):
		httpx.WriteError(w, http.StatusConflict, "not_current_lesson", "Bài này không phải bài đang học")
	case errors.Is(err, ErrNotCurrentStep):
		httpx.WriteError(w, http.StatusConflict, "not_current_step", "Bước này đã hoàn thành")
	default:
		h.log.ErrorContext(r.Context(), "study request failed", slog.Any("error", err))
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "Có lỗi xảy ra, vui lòng thử lại")
	}
}
