package progress

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/luongtran/luna/backend/internal/platform/httpx"
)

// StudyHandler serves the daily flow API (contracts/study-api.md) and guards lesson content.
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
	mux.Handle("GET /api/today", requireAuth(http.HandlerFunc(h.today)))
	mux.Handle("POST /api/today/steps/{step}/complete", requireAuth(http.HandlerFunc(h.complete)))
	mux.Handle("PUT /api/today/position", requireAuth(http.HandlerFunc(h.position)))
	mux.Handle("GET /api/lessons/mine", requireAuth(http.HandlerFunc(h.mine)))
	mux.Handle("GET /api/dashboard", requireAuth(http.HandlerFunc(h.dashboard)))
	mux.Handle("GET /api/stats", requireAuth(http.HandlerFunc(h.stats)))
}

// Guard lets a learner open only today's lesson and lessons already started; admins pass.
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
	Active         goalJSON `json:"active"`
	EffectiveFrom  string   `json:"effectiveFrom"`
	StartsTomorrow bool     `json:"startsTomorrow"`
}

type lessonRefJSON struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type todayJSON struct {
	Kind          string            `json:"kind"`
	Goal          *goalJSON         `json:"goal"`
	Lesson        *lessonRefJSON    `json:"lesson"`
	Steps         map[string]string `json:"steps"`
	CurrentStep   string            `json:"currentStep"`
	SentenceIndex int               `json:"sentenceIndex"`
	ReviewCount   int               `json:"reviewCount"`
	Streak        int               `json:"streak"`
	GoalCompleted bool              `json:"goalCompleted"`
}

func toTodayJSON(v TodayView) todayJSON {
	out := todayJSON{
		Kind: string(v.Kind), Steps: map[string]string{}, CurrentStep: string(v.CurrentStep),
		SentenceIndex: v.SentenceIndex, ReviewCount: v.ReviewCount, Streak: v.Streak, GoalCompleted: v.GoalCompleted,
	}
	if v.Goal != nil {
		g := toGoalJSON(*v.Goal)
		out.Goal = &g
	}
	if v.Lesson != nil {
		out.Lesson = &lessonRefJSON{ID: v.Lesson.ID, Title: v.Lesson.Title}
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
	Today     *lessonRefJSON  `json:"today"`
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
	}
	if httpx.DecodeJSON(w, r, &req) != nil {
		return
	}
	if req.TopicID == "" {
		httpx.WriteFieldErrors(w, map[string]string{"topicId": "Vui lòng chọn chủ đề"})
		return
	}
	res, err := h.svc.SetGoal(r.Context(), p.UserID, req.TopicID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, setGoalJSON{
		Active: toGoalJSON(res.Active), EffectiveFrom: res.EffectiveFrom, StartsTomorrow: res.StartsTomorrow,
	})
}

func (h *StudyHandler) today(w http.ResponseWriter, r *http.Request) {
	p, _ := httpx.PrincipalFrom(r.Context())
	v, err := h.svc.Today(r.Context(), p.UserID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toTodayJSON(v))
}

func (h *StudyHandler) complete(w http.ResponseWriter, r *http.Request) {
	p, _ := httpx.PrincipalFrom(r.Context())
	v, err := h.svc.CompleteStep(r.Context(), p.UserID, Step(r.PathValue("step")))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toTodayJSON(v))
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
	if err := h.svc.SetPosition(r.Context(), p.UserID, Step(req.Step), req.SentenceIndex); err != nil {
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
	if v.Today != nil {
		out.Today = &lessonRefJSON{ID: v.Today.ID, Title: v.Today.Title}
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
	case errors.Is(err, ErrStepLocked):
		httpx.WriteError(w, http.StatusConflict, "step_locked", "Hoàn thành bước trước")
	case errors.Is(err, ErrWriteIncomplete):
		httpx.WriteError(w, http.StatusConflict, "write_incomplete", "Hãy nộp bài viết")
	case errors.Is(err, ErrReadIncomplete):
		httpx.WriteError(w, http.StatusConflict, "read_incomplete", "Hãy trả lời hết câu hỏi hiểu bài")
	case errors.Is(err, ErrListenIncomplete):
		httpx.WriteError(w, http.StatusConflict, "listen_incomplete", "Hãy kiểm tra hết các câu của bước Nghe")
	case errors.Is(err, ErrNoLesson):
		httpx.WriteError(w, http.StatusConflict, "no_lesson", "Hôm nay không có bài để học")
	case errors.Is(err, ErrNotCurrentStep):
		httpx.WriteError(w, http.StatusConflict, "not_current_step", "Bước này không phải bước hiện tại")
	default:
		h.log.ErrorContext(r.Context(), "study request failed", slog.Any("error", err))
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "Có lỗi xảy ra, vui lòng thử lại")
	}
}
