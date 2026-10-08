package grammar

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/luongtran/luna/backend/internal/ai"
	"github.com/luongtran/luna/backend/internal/platform/httpx"
)

// generateWriteTimeout replaces the server's write timeout for the generate route, which waits up
// to generateTimeout for the AI.
const generateWriteTimeout = 135 * time.Second

// Handler serves the admin grammar lesson API and the learner grammar API.
type Handler struct {
	svc *Service
	log *slog.Logger
}

// NewHandler returns a Handler for svc.
func NewHandler(svc *Service, log *slog.Logger) *Handler {
	return &Handler{svc: svc, log: log}
}

// Register adds the admin routes (behind requireAuth + admin role) and the learner routes
// (behind requireAuth, any role).
func (h *Handler) Register(mux *http.ServeMux, requireAuth httpx.Middleware) {
	admin := func(f http.HandlerFunc) http.Handler { return requireAuth(httpx.RequireAdmin(f)) }
	// Grammar is for members: guests get 403 members_only.
	learner := func(f http.HandlerFunc) http.Handler { return requireAuth(httpx.RequireMember(f)) }

	mux.Handle("GET /api/admin/grammar-lessons", admin(h.adminList))
	mux.Handle("GET /api/admin/grammar-lessons/{pointId}", admin(h.adminGet))
	mux.Handle("POST /api/admin/grammar-lessons/{pointId}/generate", admin(h.generate))
	mux.Handle("PUT /api/admin/grammar-lessons/{pointId}", admin(h.save))
	mux.Handle("POST /api/admin/grammar-lessons/{pointId}/check", admin(h.check))
	mux.Handle("POST /api/admin/grammar-lessons/{pointId}/checks/{exerciseId}/confirm", admin(h.confirmCheck))
	mux.Handle("POST /api/admin/grammar-lessons/{pointId}/verify", admin(h.verify))
	mux.Handle("PUT /api/admin/grammar-lessons/{pointId}/exercises/{exerciseId}", admin(h.replaceExercise))
	mux.Handle("DELETE /api/admin/grammar-lessons/{pointId}/exercises/{exerciseId}", admin(h.deleteExercise))
	mux.Handle("POST /api/admin/grammar-lessons/{pointId}/publish", admin(h.publish))
	mux.Handle("POST /api/admin/grammar-lessons/{pointId}/unpublish", admin(h.unpublish))
	mux.Handle("GET /api/admin/grammar-reports", admin(h.adminReports))
	mux.Handle("POST /api/admin/grammar-reports/resolve", admin(h.resolveReports))

	mux.Handle("GET /api/grammar", learner(h.list))
	mux.Handle("GET /api/grammar/{pointId}", learner(h.get))
	mux.Handle("POST /api/grammar/{pointId}/attempts", learner(h.attempt))
	mux.Handle("POST /api/grammar/{pointId}/reports", learner(h.report))
}

// --- JSON shapes ---

// exerciseJSON is used both ways. AnswerIndex is a pointer so the first option (0) is kept.
type exerciseJSON struct {
	ID            string   `json:"id"`
	Kind          string   `json:"kind"`
	PromptVi      string   `json:"promptVi,omitempty"`
	Text          string   `json:"text,omitempty"`
	Options       []string `json:"options,omitempty"`
	AnswerIndex   *int     `json:"answerIndex,omitempty"`
	Answers       []string `json:"answers,omitempty"`
	Words         []string `json:"words,omitempty"`
	Sentence      string   `json:"sentence,omitempty"`
	ExplanationVi string   `json:"explanationVi,omitempty"`
}

type structureJSON struct {
	Label   string `json:"label"`
	Pattern string `json:"pattern"`
	Example string `json:"example"`
}

type exampleJSON struct {
	En string `json:"en"`
	Vi string `json:"vi"`
}

type mistakeJSON struct {
	Wrong  string `json:"wrong"`
	Right  string `json:"right"`
	NoteVi string `json:"noteVi"`
}

type contentJSON struct {
	Objective   string          `json:"objective"`
	Explanation []string        `json:"explanation"`
	Usage       []string        `json:"usage"`
	Structures  []structureJSON `json:"structures"`
	Examples    []exampleJSON   `json:"examples"`
	Mistakes    []mistakeJSON   `json:"mistakes"`
	Practice    []exerciseJSON  `json:"practice"`
	Mastery     []exerciseJSON  `json:"mastery"`
}

func toContentJSON(c Content) contentJSON {
	out := contentJSON{
		Objective:   c.Objective,
		Explanation: orEmpty(c.Explanation),
		Usage:       orEmpty(c.Usage),
		Structures:  make([]structureJSON, len(c.Structures)),
		Examples:    make([]exampleJSON, len(c.Examples)),
		Mistakes:    make([]mistakeJSON, len(c.Mistakes)),
		Practice:    toExercisesJSON(c.Practice),
		Mastery:     toExercisesJSON(c.Mastery),
	}
	for i, s := range c.Structures {
		out.Structures[i] = structureJSON(s)
	}
	for i, e := range c.Examples {
		out.Examples[i] = exampleJSON(e)
	}
	for i, m := range c.Mistakes {
		out.Mistakes[i] = mistakeJSON(m)
	}
	return out
}

func toExercisesJSON(in []Exercise) []exerciseJSON {
	out := make([]exerciseJSON, len(in))
	for i, e := range in {
		out[i] = exerciseJSON{
			ID: e.ID, Kind: string(e.Kind), PromptVi: e.PromptVi, Text: e.Text, Options: e.Options,
			Answers: e.Answers, Words: e.Words, Sentence: e.Sentence, ExplanationVi: e.ExplanationVi,
		}
		if e.Kind == KindChoice {
			idx := e.AnswerIndex
			out[i].AnswerIndex = &idx
		}
	}
	return out
}

func (c contentJSON) toContent() Content {
	out := Content{
		Objective: c.Objective, Explanation: c.Explanation, Usage: c.Usage,
		Structures: make([]Structure, len(c.Structures)),
		Examples:   make([]Example, len(c.Examples)),
		Mistakes:   make([]Mistake, len(c.Mistakes)),
		Practice:   make([]Exercise, len(c.Practice)),
		Mastery:    make([]Exercise, len(c.Mastery)),
	}
	for i, s := range c.Structures {
		out.Structures[i] = Structure(s)
	}
	for i, e := range c.Examples {
		out.Examples[i] = Example(e)
	}
	for i, m := range c.Mistakes {
		out.Mistakes[i] = Mistake(m)
	}
	for i, e := range c.Practice {
		out.Practice[i] = e.toExercise()
	}
	for i, e := range c.Mastery {
		out.Mastery[i] = e.toExercise()
	}
	return out
}

func orEmpty(in []string) []string {
	if in == nil {
		return []string{}
	}
	return in
}

type summaryJSON struct {
	PointID     string     `json:"pointId"`
	Status      Status     `json:"status"`
	Edited      bool       `json:"edited"`
	Flags       int        `json:"flags"`
	Checked     bool       `json:"checked"`
	Verified    bool       `json:"verified"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	PublishedAt *time.Time `json:"publishedAt"`
}

type checkJSON struct {
	ExerciseID string `json:"exerciseId"`
	Kind       string `json:"kind"`
	NoteVi     string `json:"noteVi"`
	Confirmed  bool   `json:"confirmed"`
}

type reportGroupJSON struct {
	ExerciseID string         `json:"exerciseId"`
	Count      int            `json:"count"`
	Reasons    map[string]int `json:"reasons"`
	Notes      []string       `json:"notes"`
}

type lessonJSON struct {
	PointID     string            `json:"pointId"`
	Status      Status            `json:"status"`
	Edited      bool              `json:"edited"`
	UpdatedAt   time.Time         `json:"updatedAt"`
	PublishedAt *time.Time        `json:"publishedAt"`
	Checks      []checkJSON       `json:"checks"`
	CheckedAt   *time.Time        `json:"checkedAt"`
	VerifiedAt  *time.Time        `json:"verifiedAt"`
	Reports     []reportGroupJSON `json:"reports"`
	Content     contentJSON       `json:"content"`
}

func timePtr(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

func toLessonJSON(l Lesson, reports []ReportGroup) lessonJSON {
	out := lessonJSON{
		PointID: l.PointID, Status: l.Status, Edited: l.Edited, UpdatedAt: l.UpdatedAt,
		PublishedAt: timePtr(l.PublishedAt), CheckedAt: timePtr(l.CheckedAt), VerifiedAt: timePtr(l.VerifiedAt), Content: toContentJSON(l.Content),
		Checks:  make([]checkJSON, len(l.Checks)),
		Reports: make([]reportGroupJSON, len(reports)),
	}
	for i, c := range l.Checks {
		out.Checks[i] = checkJSON{ExerciseID: c.ExerciseID, Kind: string(c.Kind), NoteVi: c.NoteVi, Confirmed: c.Confirmed}
	}
	for i, g := range reports {
		out.Reports[i] = reportGroupJSON{ExerciseID: g.ExerciseID, Count: g.Count, Reasons: reasonsJSON(g.Reasons), Notes: g.Notes}
	}
	return out
}

func reasonsJSON(in map[ReportReason]int) map[string]int {
	out := make(map[string]int, len(in))
	for k, v := range in {
		out[string(k)] = v
	}
	return out
}

// writeLesson writes {"lesson": ...} with the lesson's open reports.
func (h *Handler) writeLesson(w http.ResponseWriter, r *http.Request, status int, l Lesson) {
	reports, err := h.svc.AdminReportsOf(r.Context(), l.PointID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, status, map[string]lessonJSON{"lesson": toLessonJSON(l, reports)})
}

// --- admin handlers ---

func (h *Handler) adminList(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.AdminList(r.Context())
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	out := make([]summaryJSON, len(items))
	for i, s := range items {
		out[i] = summaryJSON{
			PointID: s.PointID, Status: s.Status, Edited: s.Edited, Flags: s.Flags, Checked: s.Checked, Verified: s.Verified,
			UpdatedAt: s.UpdatedAt, PublishedAt: timePtr(s.PublishedAt),
		}
	}
	httpx.WriteJSON(w, http.StatusOK, map[string][]summaryJSON{"lessons": out})
}

func (h *Handler) adminGet(w http.ResponseWriter, r *http.Request) {
	l, err := h.svc.AdminGet(r.Context(), r.PathValue("pointId"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeLesson(w, r, http.StatusOK, l)
}

func (h *Handler) generate(w http.ResponseWriter, r *http.Request) {
	if err := http.NewResponseController(w).SetWriteDeadline(time.Now().Add(generateWriteTimeout)); err != nil &&
		!errors.Is(err, http.ErrNotSupported) {
		h.log.WarnContext(r.Context(), "grammar generate: extend write deadline", slog.Any("error", err))
	}
	var body struct {
		Force bool `json:"force"`
	}
	if httpx.DecodeJSON(w, r, &body) != nil {
		return
	}
	l, created, err := h.svc.Generate(r.Context(), r.PathValue("pointId"), body.Force)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	h.writeLesson(w, r, status, l)
}

// check runs the AI check again; like generate it waits for the AI.
func (h *Handler) check(w http.ResponseWriter, r *http.Request) {
	if err := http.NewResponseController(w).SetWriteDeadline(time.Now().Add(generateWriteTimeout)); err != nil &&
		!errors.Is(err, http.ErrNotSupported) {
		h.log.WarnContext(r.Context(), "grammar check: extend write deadline", slog.Any("error", err))
	}
	l, err := h.svc.Check(r.Context(), r.PathValue("pointId"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeLesson(w, r, http.StatusOK, l)
}

func (h *Handler) save(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Content contentJSON `json:"content"`
	}
	if httpx.DecodeJSON(w, r, &body) != nil {
		return
	}
	l, err := h.svc.Save(r.Context(), r.PathValue("pointId"), body.Content.toContent())
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeLesson(w, r, http.StatusOK, l)
}

// publish takes an optional body {"acknowledgeFlags": true}.
func (h *Handler) publish(w http.ResponseWriter, r *http.Request) {
	var body struct {
		AcknowledgeFlags bool `json:"acknowledgeFlags"`
	}
	if r.ContentLength != 0 && httpx.DecodeJSON(w, r, &body) != nil {
		return
	}
	l, err := h.svc.Publish(r.Context(), r.PathValue("pointId"), body.AcknowledgeFlags)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeLesson(w, r, http.StatusOK, l)
}

func (h *Handler) unpublish(w http.ResponseWriter, r *http.Request) {
	l, err := h.svc.Unpublish(r.Context(), r.PathValue("pointId"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeLesson(w, r, http.StatusOK, l)
}

type adminReportJSON struct {
	PointID    string         `json:"pointId"`
	ExerciseID string         `json:"exerciseId"`
	Count      int            `json:"count"`
	Reasons    map[string]int `json:"reasons"`
	Notes      []string       `json:"notes"`
	LatestAt   time.Time      `json:"latestAt"`
}

func (h *Handler) adminReports(w http.ResponseWriter, r *http.Request) {
	groups, err := h.svc.AdminReports(r.Context())
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	out := make([]adminReportJSON, len(groups))
	for i, g := range groups {
		out[i] = adminReportJSON{
			PointID: g.PointID, ExerciseID: g.ExerciseID, Count: g.Count, Reasons: reasonsJSON(g.Reasons),
			Notes: g.Notes, LatestAt: g.LatestAt,
		}
	}
	httpx.WriteJSON(w, http.StatusOK, map[string][]adminReportJSON{"reports": out})
}

func (h *Handler) resolveReports(w http.ResponseWriter, r *http.Request) {
	var body struct {
		PointID    string `json:"pointId"`
		ExerciseID string `json:"exerciseId"`
	}
	if httpx.DecodeJSON(w, r, &body) != nil {
		return
	}
	n, err := h.svc.ResolveReports(r.Context(), body.PointID, body.ExerciseID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]int{"resolved": n})
}

// --- learner handlers ---

type pointJSON struct {
	ID          string `json:"id"`
	Level       string `json:"level"`
	TitleVi     string `json:"titleVi"`
	TitleEn     string `json:"titleEn"`
	HintVi      string `json:"hintVi"`
	Available   bool   `json:"available"`
	Status      string `json:"status"`
	BestMastery int    `json:"bestMastery"`
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	p, _ := httpx.PrincipalFrom(r.Context())
	ov, err := h.svc.List(r.Context(), p.UserID, r.URL.Query().Get("level"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	points := make([]pointJSON, len(ov.Points))
	for i, ps := range ov.Points {
		points[i] = pointJSON{
			ID: ps.ID, Level: ps.Level, TitleVi: ps.TitleVi, TitleEn: ps.TitleEn, HintVi: ps.HintVi,
			Available: ps.Available, Status: string(ps.Status), BestMastery: ps.BestMastery,
		}
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"points": points, "next": ov.Next})
}

type detailPointJSON struct {
	ID       string   `json:"id"`
	Level    string   `json:"level"`
	TitleVi  string   `json:"titleVi"`
	TitleEn  string   `json:"titleEn"`
	Pattern  string   `json:"pattern"`
	HintVi   string   `json:"hintVi"`
	Examples []string `json:"examples"`
}

type progressJSON struct {
	Status           string   `json:"status"`
	PracticeAttempts int      `json:"practiceAttempts"`
	LastPractice     int      `json:"lastPractice"`
	MasteryAttempts  int      `json:"masteryAttempts"`
	BestMastery      int      `json:"bestMastery"`
	Mastered         bool     `json:"mastered"`
	Weak             []string `json:"weak"`
}

func toProgressJSON(p Progress) progressJSON {
	status := p.Status
	if status == "" {
		status = ProgressNew
	}
	return progressJSON{
		Status: string(status), PracticeAttempts: p.PracticeAttempts, LastPractice: p.LastPractice,
		MasteryAttempts: p.MasteryAttempts, BestMastery: p.BestMastery, Mastered: status == ProgressMastered,
		Weak: orEmpty(p.Weak),
	}
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	p, _ := httpx.PrincipalFrom(r.Context())
	d, err := h.svc.Get(r.Context(), p.UserID, r.PathValue("pointId"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"point": detailPointJSON{
			ID: d.Point.ID, Level: d.Point.Level, TitleVi: d.Point.TitleVi, TitleEn: d.Point.TitleEn,
			Pattern: d.Point.Pattern, HintVi: d.Point.HintVi, Examples: orEmpty(d.Point.Examples),
		},
		"content":     toContentJSON(d.Content),
		"progress":    toProgressJSON(d.Progress),
		"lessonCount": d.LessonCount,
	})
}

func (h *Handler) attempt(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Kind    string   `json:"kind"`
		Correct int      `json:"correct"`
		Total   int      `json:"total"`
		Wrong   []string `json:"wrong"`
	}
	if httpx.DecodeJSON(w, r, &body) != nil {
		return
	}
	p, _ := httpx.PrincipalFrom(r.Context())
	rec, passed, err := h.svc.RecordAttempt(r.Context(), p.UserID, r.PathValue("pointId"), Attempt(body))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"progress": toProgressJSON(rec), "passed": passed})
}

func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	var verr *ValidationError
	var flags *FlagsError
	switch {
	case errors.As(err, &verr):
		httpx.WriteFieldErrors(w, verr.Fields)
	case errors.As(err, &flags):
		httpx.WriteJSON(w, http.StatusConflict, map[string]any{
			"error": "grammar_flags_unresolved", "count": flags.Count,
			"message": fmt.Sprintf("Còn %d câu bị gắn cờ. Hãy xem lại hoặc xác nhận đăng.", flags.Count),
		})
	case errors.Is(err, ErrNotChecked):
		httpx.WriteError(w, http.StatusConflict, "grammar_not_checked", "Hãy kiểm tra bằng AI trước khi xác nhận.")
	case errors.Is(err, ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "not_found", "Không tìm thấy bài ngữ pháp")
	case errors.Is(err, ErrExists):
		httpx.WriteError(w, http.StatusConflict, "grammar_lesson_exists",
			"Điểm này đã có bài đã sửa hoặc đã đăng. Sinh lại sẽ thay toàn bộ nội dung.")
	case errors.Is(err, ai.ErrNotConfigured):
		httpx.WriteError(w, http.StatusServiceUnavailable, "ai_not_configured", "AI chưa được cấu hình. Liên hệ người vận hành.")
	case errors.Is(err, ai.ErrInvalidKey):
		httpx.WriteError(w, http.StatusServiceUnavailable, "ai_not_configured", "Khoá AI không hợp lệ. Liên hệ người vận hành.")
	case errors.Is(err, ai.ErrQuota):
		httpx.WriteError(w, http.StatusTooManyRequests, "ai_quota", "Đã hết lượt AI, vui lòng thử lại sau.")
	case errors.Is(err, ErrUnusable):
		httpx.WriteError(w, http.StatusBadGateway, "ai_failed", "AI trả về nội dung không dùng được, vui lòng thử lại.")
	case errors.Is(err, errAI):
		h.log.WarnContext(r.Context(), "grammar lesson generation failed", slog.Any("error", err))
		httpx.WriteError(w, http.StatusBadGateway, "ai_failed", "Sinh bài thất bại, vui lòng thử lại.")
	default:
		h.log.ErrorContext(r.Context(), "grammar request failed", slog.Any("error", err))
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "Có lỗi xảy ra, vui lòng thử lại")
	}
}

func (h *Handler) report(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ExerciseID string `json:"exerciseId"`
		Reason     string `json:"reason"`
		Note       string `json:"note"`
	}
	if httpx.DecodeJSON(w, r, &body) != nil {
		return
	}
	p, _ := httpx.PrincipalFrom(r.Context())
	in := ReportInput{ExerciseID: body.ExerciseID, Reason: ReportReason(body.Reason), Note: body.Note}
	if err := h.svc.Report(r.Context(), p.UserID, r.PathValue("pointId"), in); err != nil {
		h.writeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// toExercise converts the JSON of one exercise; a choice without answerIndex gets -1, which the
// validation reports.
func (e exerciseJSON) toExercise() Exercise {
	out := Exercise{
		ID: e.ID, Kind: Kind(e.Kind), PromptVi: e.PromptVi, Text: e.Text, Options: e.Options, Answers: e.Answers,
		Words: e.Words, Sentence: e.Sentence, ExplanationVi: e.ExplanationVi,
	}
	if e.AnswerIndex != nil {
		out.AnswerIndex = *e.AnswerIndex
	} else if e.Kind == string(KindChoice) {
		out.AnswerIndex = -1
	}
	return out
}

func (h *Handler) confirmCheck(w http.ResponseWriter, r *http.Request) {
	l, err := h.svc.ConfirmCheck(r.Context(), r.PathValue("pointId"), r.PathValue("exerciseId"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeLesson(w, r, http.StatusOK, l)
}

func (h *Handler) verify(w http.ResponseWriter, r *http.Request) {
	l, err := h.svc.Verify(r.Context(), r.PathValue("pointId"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeLesson(w, r, http.StatusOK, l)
}

func (h *Handler) replaceExercise(w http.ResponseWriter, r *http.Request) {
	var body exerciseJSON
	if httpx.DecodeJSON(w, r, &body) != nil {
		return
	}
	l, err := h.svc.ReplaceExercise(r.Context(), r.PathValue("pointId"), r.PathValue("exerciseId"), body.toExercise())
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeLesson(w, r, http.StatusOK, l)
}

func (h *Handler) deleteExercise(w http.ResponseWriter, r *http.Request) {
	l, err := h.svc.DeleteExercise(r.Context(), r.PathValue("pointId"), r.PathValue("exerciseId"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeLesson(w, r, http.StatusOK, l)
}
