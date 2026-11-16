package games

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/luongtran/luna/backend/internal/ai"
	"github.com/luongtran/luna/backend/internal/platform/httpx"
)

// generateWriteTimeout replaces the server's write timeout for the AI route.
const generateWriteTimeout = spotGenerateTimeout + 10*time.Second

// Handler serves the games API.
type Handler struct {
	svc *Service
	log *slog.Logger
}

// NewHandler returns the HTTP handler of the games.
func NewHandler(svc *Service, log *slog.Logger) *Handler { return &Handler{svc: svc, log: log} }

// Register adds the routes to mux. Every account plays, guests too, except "Bắn lỗi"; the extra
// sentences are for admins.
func (h *Handler) Register(mux *http.ServeMux, requireAuth httpx.Middleware) {
	auth := func(f http.HandlerFunc) http.Handler { return requireAuth(f) }
	member := func(f http.HandlerFunc) http.Handler { return requireAuth(httpx.RequireMember(f)) }
	admin := func(f http.HandlerFunc) http.Handler { return requireAuth(httpx.RequireAdmin(f)) }

	mux.Handle("GET /api/games", auth(h.list))
	mux.Handle("GET /api/games/picture/round", auth(h.pictureRound))
	mux.Handle("GET /api/games/letters/round", auth(h.lettersRound))
	mux.Handle("GET /api/games/memory/round", auth(h.memoryRound))
	mux.Handle("GET /api/games/grammar/round", member(h.grammarRound))
	mux.Handle("POST /api/games/{game}/results", auth(h.results))
	mux.Handle("GET /api/games/daily", auth(h.daily))
	mux.Handle("POST /api/games/daily/result", auth(h.dailyResult))
	mux.Handle("POST /api/games/daily/practice", auth(h.dailyPractice))

	mux.Handle("GET /api/admin/games/spot-counts", admin(h.spotCounts))
	mux.Handle("GET /api/admin/games/spot-items", admin(h.spotItems))
	mux.Handle("POST /api/admin/games/spot-items/generate", admin(h.generateSpots))
	mux.Handle("PUT /api/admin/games/spot-items/{id}", admin(h.updateSpot))
	mux.Handle("POST /api/admin/games/spot-items/{id}/approve", admin(h.approveSpot))
	mux.Handle("DELETE /api/admin/games/spot-items/{id}", admin(h.deleteSpot))
}

func principal(r *http.Request) httpx.Principal {
	p, _ := httpx.PrincipalFrom(r.Context())
	return p
}

// --- JSON shapes ---

type gameJSON struct {
	ID     Game       `json:"id"`
	Best   int        `json:"best"`
	BestAt *time.Time `json:"bestAt"`
	Locked bool       `json:"locked"`
}

type dailyStatusJSON struct {
	Day    string `json:"day"`
	Done   bool   `json:"done"`
	Streak int    `json:"streak"`
}

type pictureJSON struct {
	Lemma     string   `json:"lemma"`
	ImageURL  string   `json:"imageUrl"`
	MeaningVi string   `json:"meaningVi"`
	Options   []string `json:"options"`
	Answer    int      `json:"answer"`
}

func toPictureJSON(q PictureQuestion) pictureJSON {
	return pictureJSON{Lemma: q.Lemma, ImageURL: q.ImageURL, MeaningVi: q.MeaningVi, Options: q.Options, Answer: q.Answer}
}

type wordJSON struct {
	Lemma     string `json:"lemma"`
	MeaningVi string `json:"meaningVi"`
	ImageURL  string `json:"imageUrl"`
}

type spanJSON struct {
	From int `json:"from"`
	To   int `json:"to"`
}

type spotJSON struct {
	Tokens  []string  `json:"tokens"`
	Wrong   *spanJSON `json:"wrong"`
	Options []string  `json:"options"`
	Answer  int       `json:"answer"`
	RuleVi  string    `json:"ruleVi"`
	TopicVi string    `json:"topicVi"`
	PointID string    `json:"pointId"`
	Source  string    `json:"source"`
}

func toSpotJSON(q SpotQuestion) spotJSON {
	out := spotJSON{Tokens: q.Tokens, Options: q.Options, Answer: q.Answer, RuleVi: q.RuleVi, TopicVi: q.TopicVi, PointID: q.PointID, Source: q.Source}
	if out.Options == nil {
		out.Options = []string{}
	}
	if q.Wrong != nil {
		out.Wrong = &spanJSON{From: q.Wrong.From, To: q.Wrong.To}
	}
	return out
}

type streakJSON struct {
	Current int `json:"current"`
	Longest int `json:"longest"`
}

type compareJSON struct {
	Yesterday *int     `json:"yesterday"`
	Avg7      *float64 `json:"avg7"`
	Best      int      `json:"best"`
}

type answerJSON struct {
	Kind  string `json:"kind"`
	Right bool   `json:"right"`
}

type dailyResultJSON struct {
	Score      int          `json:"score"`
	Answers    []answerJSON `json:"answers"`
	FinishedAt time.Time    `json:"finishedAt"`
}

func toDailyResultJSON(r DailyResult) dailyResultJSON {
	out := dailyResultJSON{Score: r.Score, FinishedAt: r.FinishedAt, Answers: make([]answerJSON, len(r.Answers))}
	for i, a := range r.Answers {
		out.Answers[i] = answerJSON(a)
	}
	return out
}

type dailyQuestionJSON struct {
	Kind    string       `json:"kind"`
	Picture *pictureJSON `json:"picture,omitempty"`
	Verb    *struct {
		Base string `json:"base"`
	} `json:"verb,omitempty"`
	Letters *wordJSON `json:"letters,omitempty"`
	Grammar *spotJSON `json:"grammar,omitempty"`
}

type verbMissJSON struct {
	Base  string `json:"base"`
	Form  string `json:"form"`
	Typed string `json:"typed"`
}

func fromVerbMisses(list []verbMissJSON) []VerbMiss {
	out := make([]VerbMiss, len(list))
	for i, v := range list {
		out[i] = VerbMiss(v)
	}
	return out
}

type spotItemJSON struct {
	ID          string    `json:"id"`
	PointID     string    `json:"pointId"`
	Level       string    `json:"level"`
	Sentence    string    `json:"sentence"`
	Wrong       string    `json:"wrong"`
	Right       string    `json:"right"`
	Distractors []string  `json:"distractors"`
	RuleVi      string    `json:"ruleVi"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func toSpotItemJSON(it SpotItem) spotItemJSON {
	d := it.Distractors
	if d == nil {
		d = []string{}
	}
	return spotItemJSON{
		ID: it.ID, PointID: it.PointID, Level: it.Level, Sentence: it.Sentence, Wrong: it.Wrong, Right: it.Right,
		Distractors: d, RuleVi: it.RuleVi, Status: string(it.Status), CreatedAt: it.CreatedAt, UpdatedAt: it.UpdatedAt,
	}
}

func toSpotItemsJSON(list []SpotItem) []spotItemJSON {
	out := make([]spotItemJSON, len(list))
	for i, it := range list {
		out[i] = toSpotItemJSON(it)
	}
	return out
}

// --- learner routes ---

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	p := principal(r)
	v, err := h.svc.List(r.Context(), p.UserID, p.Role == httpx.RoleGuest)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	games := make([]gameJSON, len(v.Games))
	for i, g := range v.Games {
		games[i] = gameJSON{ID: g.ID, Best: g.Best, Locked: g.Locked}
		if !g.BestAt.IsZero() {
			at := g.BestAt
			games[i].BestAt = &at
		}
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"games": games,
		"daily": dailyStatusJSON{Day: v.Daily.Day, Done: v.Daily.Done, Streak: v.Daily.Streak},
	})
}

func (h *Handler) pictureRound(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.PictureRound(r.Context(), principal(r).UserID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	out := make([]pictureJSON, len(list))
	for i, q := range list {
		out[i] = toPictureJSON(q)
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"questions": out})
}

func (h *Handler) lettersRound(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.LettersRound(r.Context(), principal(r).UserID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	out := make([]wordJSON, len(list))
	for i, w := range list {
		out[i] = wordJSON(w)
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"words": out})
}

func (h *Handler) memoryRound(w http.ResponseWriter, r *http.Request) {
	p := principal(r)
	list, err := h.svc.MemoryRound(r.Context(), p.UserID, p.Role == httpx.RoleAdmin, r.URL.Query().Get("lessonId"))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	out := make([]wordJSON, len(list))
	for i, m := range list {
		out[i] = wordJSON{Lemma: m.Lemma, MeaningVi: m.MeaningVi, ImageURL: m.ImageURL}
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"pairs": out})
}

func (h *Handler) grammarRound(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.GrammarRound(r.Context(), principal(r).UserID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	out := make([]spotJSON, len(list))
	for i, q := range list {
		out[i] = toSpotJSON(q)
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"questions": out})
}

type resultRequest struct {
	Score      int            `json:"score"`
	Completed  bool           `json:"completed"`
	WrongWords []string       `json:"wrongWords"`
	WrongVerbs []verbMissJSON `json:"wrongVerbs"`
}

func (h *Handler) results(w http.ResponseWriter, r *http.Request) {
	p := principal(r)
	var req resultRequest
	if httpx.DecodeJSON(w, r, &req) != nil {
		return
	}
	reply, err := h.svc.Finish(r.Context(), p.UserID, p.Role == httpx.RoleGuest, Game(r.PathValue("game")), Result{
		Score: req.Score, Completed: req.Completed, WrongWords: req.WrongWords, WrongVerbs: fromVerbMisses(req.WrongVerbs),
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"best": reply.Best, "newBest": reply.NewBest, "rescheduled": reply.Rescheduled})
}

func (h *Handler) daily(w http.ResponseWriter, r *http.Request) {
	p := principal(r)
	v, err := h.svc.Daily(r.Context(), p.UserID, p.Role == httpx.RoleGuest)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	questions := make([]dailyQuestionJSON, len(v.Questions))
	for i, q := range v.Questions {
		out := dailyQuestionJSON{Kind: q.Kind}
		switch {
		case q.Picture != nil:
			pj := toPictureJSON(*q.Picture)
			out.Picture = &pj
		case q.Verb != nil:
			out.Verb = &struct {
				Base string `json:"base"`
			}{Base: q.Verb.Base}
		case q.Letters != nil:
			out.Letters = &wordJSON{Lemma: q.Letters.Lemma, MeaningVi: q.Letters.MeaningVi, ImageURL: q.Letters.ImageURL}
		case q.Grammar != nil:
			sj := toSpotJSON(*q.Grammar)
			out.Grammar = &sj
		}
		questions[i] = out
	}
	var result *dailyResultJSON
	if v.Result != nil {
		rj := toDailyResultJSON(*v.Result)
		result = &rj
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"day": v.Day, "seed": v.Seed, "done": v.Done,
		"streak":    streakJSON(v.Streak),
		"week":      v.Week,
		"result":    result,
		"compare":   compareJSON(v.Compare),
		"questions": questions,
	})
}

type dailyResultRequest struct {
	Day        string         `json:"day"`
	Score      int            `json:"score"`
	Answers    []answerJSON   `json:"answers"`
	WrongWords []string       `json:"wrongWords"`
	WrongVerbs []verbMissJSON `json:"wrongVerbs"`
}

func (h *Handler) dailyResult(w http.ResponseWriter, r *http.Request) {
	var req dailyResultRequest
	if httpx.DecodeJSON(w, r, &req) != nil {
		return
	}
	answers := make([]DailyAnswer, len(req.Answers))
	for i, a := range req.Answers {
		answers[i] = DailyAnswer(a)
	}
	reply, err := h.svc.SaveDaily(r.Context(), principal(r).UserID, DailyBody{
		Day: req.Day, Score: req.Score, Answers: answers, WrongWords: req.WrongWords, WrongVerbs: fromVerbMisses(req.WrongVerbs),
	})
	var done *AlreadyDoneError
	if errors.As(err, &done) {
		httpx.WriteJSON(w, http.StatusConflict, map[string]any{
			"error": "already_done", "message": "Hôm nay bạn đã làm thử thách rồi", "result": toDailyResultJSON(done.Result),
		})
		return
	}
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"streak": streakJSON(reply.Streak), "compare": compareJSON(reply.Compare), "rescheduled": reply.Rescheduled,
	})
}

type practiceRequest struct {
	WrongWords []string       `json:"wrongWords"`
	WrongVerbs []verbMissJSON `json:"wrongVerbs"`
}

func (h *Handler) dailyPractice(w http.ResponseWriter, r *http.Request) {
	var req practiceRequest
	if httpx.DecodeJSON(w, r, &req) != nil {
		return
	}
	n, err := h.svc.DailyPractice(r.Context(), principal(r).UserID, req.WrongWords, fromVerbMisses(req.WrongVerbs))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"rescheduled": n})
}

// --- admin routes ---

type spotCountJSON struct {
	PointID string `json:"pointId"`
	Level   string `json:"level"`
	TitleVi string `json:"titleVi"`
	Choice  int    `json:"choice"`
	Fix     int    `json:"fix"`
	Spot    int    `json:"spot"`
	Draft   int    `json:"draft"`
	Total   int    `json:"total"`
}

func (h *Handler) spotCounts(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.SpotCounts(r.Context())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	out := make([]spotCountJSON, len(list))
	for i, c := range list {
		out[i] = spotCountJSON(c)
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"points": out})
}

func (h *Handler) spotItems(w http.ResponseWriter, r *http.Request) {
	pointID := r.URL.Query().Get("pointId")
	if pointID == "" {
		httpx.WriteFieldErrors(w, map[string]string{"pointId": "Cần điểm ngữ pháp"})
		return
	}
	list, err := h.svc.SpotItems(r.Context(), pointID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": toSpotItemsJSON(list)})
}

func (h *Handler) generateSpots(w http.ResponseWriter, r *http.Request) {
	if err := http.NewResponseController(w).SetWriteDeadline(time.Now().Add(generateWriteTimeout)); err != nil &&
		!errors.Is(err, http.ErrNotSupported) {
		h.log.WarnContext(r.Context(), "games generate: extend write deadline", slog.Any("error", err))
	}
	var req struct {
		PointID string `json:"pointId"`
	}
	if httpx.DecodeJSON(w, r, &req) != nil {
		return
	}
	if req.PointID == "" {
		httpx.WriteFieldErrors(w, map[string]string{"pointId": "Cần điểm ngữ pháp"})
		return
	}
	res, err := h.svc.GenerateSpots(r.Context(), req.PointID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"created": toSpotItemsJSON(res.Created), "dropped": res.Dropped})
}

func (h *Handler) updateSpot(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Sentence    string   `json:"sentence"`
		Wrong       string   `json:"wrong"`
		Right       string   `json:"right"`
		Distractors []string `json:"distractors"`
		RuleVi      string   `json:"ruleVi"`
	}
	if httpx.DecodeJSON(w, r, &req) != nil {
		return
	}
	it, err := h.svc.UpdateSpot(r.Context(), r.PathValue("id"), SpotEdit(req))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"item": toSpotItemJSON(it)})
}

func (h *Handler) approveSpot(w http.ResponseWriter, r *http.Request) {
	it, err := h.svc.ApproveSpot(r.Context(), r.PathValue("id"))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"item": toSpotItemJSON(it)})
}

func (h *Handler) deleteSpot(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.DeleteSpot(r.Context(), r.PathValue("id")); err != nil {
		h.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// fail writes the error of a request.
func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	var verr *ValidationError
	var short *NotEnoughError
	switch {
	case errors.As(err, &verr):
		httpx.WriteFieldErrors(w, verr.Fields)
	case errors.Is(err, ErrMembersOnly):
		httpx.WriteError(w, http.StatusForbidden, "members_only", "Phần này chỉ dành cho thành viên")
	case errors.Is(err, ErrLessonLocked):
		httpx.WriteError(w, http.StatusForbidden, "lesson_locked", "Bài này sẽ mở khi tới lượt")
	case errors.As(err, &short):
		msg := "Học thêm vài bài để mở trò này"
		if short.Code == "not_enough_sentences" {
			msg = "Chưa đủ câu ngữ pháp cho trò này"
		}
		httpx.WriteError(w, http.StatusConflict, short.Code, msg)
	case errors.Is(err, ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "not_found", "Không tìm thấy")
	case errors.Is(err, ai.ErrNotConfigured):
		httpx.WriteError(w, http.StatusServiceUnavailable, "ai_not_configured", "AI chưa được cấu hình. Liên hệ người vận hành.")
	case errors.Is(err, ai.ErrInvalidKey):
		httpx.WriteError(w, http.StatusServiceUnavailable, "ai_not_configured", "Khoá AI không hợp lệ. Liên hệ người vận hành.")
	case errors.Is(err, ai.ErrQuota):
		httpx.WriteError(w, http.StatusTooManyRequests, "ai_quota", "Đã hết lượt AI, vui lòng thử lại sau.")
	case IsAIError(err):
		h.log.WarnContext(r.Context(), "games spot generation failed", slog.Any("error", err))
		httpx.WriteError(w, http.StatusBadGateway, "ai_failed", "Soạn câu thất bại, vui lòng thử lại.")
	default:
		h.log.ErrorContext(r.Context(), "games request failed", slog.Any("error", err))
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "Có lỗi xảy ra, vui lòng thử lại")
	}
}
