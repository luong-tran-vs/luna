package talk

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/luongtran/luna/backend/internal/ai"
	"github.com/luongtran/luna/backend/internal/platform/httpx"
)

// aiWriteTimeout replaces the server's write timeout for the routes that wait for the AI (a slow
// Gemini answer takes longer than the usual 15 seconds).
const aiWriteTimeout = 5*time.Minute + 10*time.Second

// Handler serves the talk API under /api/talk.
type Handler struct {
	svc *Service
	log *slog.Logger
}

// NewHandler returns the talk HTTP handler.
func NewHandler(svc *Service, log *slog.Logger) *Handler {
	return &Handler{svc: svc, log: log}
}

// Register mounts the routes, every one for a signed-in user.
func (h *Handler) Register(mux *http.ServeMux, requireAuth httpx.Middleware) {
	route := func(pattern string, f http.HandlerFunc) { mux.Handle(pattern, requireAuth(f)) }
	route("GET /api/talk/topics", h.topics)
	route("GET /api/talk/sessions", h.list)
	route("POST /api/talk/sessions", h.start)
	route("GET /api/talk/sessions/{id}", h.get)
	route("POST /api/talk/sessions/{id}/turns", h.say)
	route("POST /api/talk/sessions/{id}/end", h.end)
}

type topicJSON struct {
	ID       string     `json:"id"`
	Name     string     `json:"name"`
	Level    string     `json:"level"`
	Lessons  int        `json:"lessons"`
	Sessions int        `json:"sessions"`
	LastAt   *time.Time `json:"lastAt,omitempty"`
}

type turnJSON struct {
	Role     Role      `json:"role"`
	Text     string    `json:"text"`
	TextVi   string    `json:"textVi,omitempty"`
	Spoken   bool      `json:"spoken,omitempty"`
	Reaction string    `json:"reaction,omitempty"`
	At       time.Time `json:"at"`
}

type goodJSON struct {
	Said   string `json:"said"`
	NoteVi string `json:"noteVi"`
}

type hintJSON struct {
	Text string `json:"text"`
	Vi   string `json:"vi"`
}

type fixJSON struct {
	Said   string `json:"said"`
	Better string `json:"better"`
	NoteVi string `json:"noteVi"`
}

type reviewJSON struct {
	OverallVi   string     `json:"overallVi"`
	Good        []goodJSON `json:"good"`
	Fixes       []fixJSON  `json:"fixes"`
	WordsUsed   []string   `json:"wordsUsed"`
	WordsUnused []string   `json:"wordsUnused"`
}

type sessionJSON struct {
	ID         string      `json:"id"`
	TopicID    string      `json:"topicId"`
	TopicName  string      `json:"topicName"`
	Level      string      `json:"level"`
	ScenarioVi string      `json:"scenarioVi"`
	Turns      []turnJSON  `json:"turns"`
	Hints      []hintJSON  `json:"hints"`
	Status     Status      `json:"status"`
	Review     *reviewJSON `json:"review,omitempty"`
	MaxTurns   int         `json:"maxTurns"`
	CreatedAt  time.Time   `json:"createdAt"`
	EndedAt    *time.Time  `json:"endedAt,omitempty"`
}

// summaryJSON is a session in the history: no turns, only how many the learner said.
type summaryJSON struct {
	ID           string    `json:"id"`
	TopicID      string    `json:"topicId"`
	TopicName    string    `json:"topicName"`
	ScenarioVi   string    `json:"scenarioVi"`
	Status       Status    `json:"status"`
	LearnerTurns int       `json:"learnerTurns"`
	Fixes        int       `json:"fixes"`
	CreatedAt    time.Time `json:"createdAt"`
}

func timePtr(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

func toSessionJSON(s Session) sessionJSON {
	out := sessionJSON{
		ID: s.ID, TopicID: s.TopicID, TopicName: s.TopicName, Level: s.Level, ScenarioVi: s.ScenarioVi,
		Turns: make([]turnJSON, len(s.Turns)), Hints: make([]hintJSON, len(s.Hints)), Status: s.Status,
		MaxTurns: MaxTurns, CreatedAt: s.CreatedAt, EndedAt: timePtr(s.EndedAt),
	}
	for i, t := range s.Turns {
		out.Turns[i] = turnJSON(t)
	}
	for i, h := range s.Hints {
		out.Hints[i] = hintJSON(h)
	}
	if r := s.Review; r != nil {
		rj := reviewJSON{
			OverallVi: r.OverallVi, Good: make([]goodJSON, len(r.Good)), Fixes: make([]fixJSON, len(r.Fixes)),
			WordsUsed: nonNil(r.WordsUsed), WordsUnused: nonNil(r.WordsUnused),
		}
		for i, g := range r.Good {
			rj.Good[i] = goodJSON(g)
		}
		for i, f := range r.Fixes {
			rj.Fixes[i] = fixJSON(f)
		}
		out.Review = &rj
	}
	return out
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func (h *Handler) topics(w http.ResponseWriter, r *http.Request) {
	p, _ := httpx.PrincipalFrom(r.Context())
	topics, err := h.svc.Topics(r.Context(), p.UserID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	out := make([]topicJSON, len(topics))
	for i, t := range topics {
		out[i] = topicJSON{ID: t.ID, Name: t.Name, Level: t.Level, Lessons: t.Lessons, Sessions: t.Sessions, LastAt: timePtr(t.LastAt)}
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"topics": out, "maxTurns": MaxTurns})
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	p, _ := httpx.PrincipalFrom(r.Context())
	sessions, err := h.svc.List(r.Context(), p.UserID, r.URL.Query().Get("topicId"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	out := make([]summaryJSON, len(sessions))
	for i, s := range sessions {
		fixes := 0
		if s.Review != nil {
			fixes = len(s.Review.Fixes)
		}
		out[i] = summaryJSON{
			ID: s.ID, TopicID: s.TopicID, TopicName: s.TopicName, ScenarioVi: s.ScenarioVi, Status: s.Status,
			LearnerTurns: s.LearnerTurns(), Fixes: fixes, CreatedAt: s.CreatedAt,
		}
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"sessions": out})
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	p, _ := httpx.PrincipalFrom(r.Context())
	s, err := h.svc.Get(r.Context(), p.UserID, r.PathValue("id"))
	h.writeSession(w, r, s, err)
}

func (h *Handler) start(w http.ResponseWriter, r *http.Request) {
	h.extendDeadline(w, r)
	var in struct {
		TopicID string `json:"topicId"`
	}
	if httpx.DecodeJSON(w, r, &in) != nil {
		return
	}
	p, _ := httpx.PrincipalFrom(r.Context())
	s, err := h.svc.Start(r.Context(), p.UserID, in.TopicID)
	h.writeSession(w, r, s, err)
}

func (h *Handler) say(w http.ResponseWriter, r *http.Request) {
	h.extendDeadline(w, r)
	var in struct {
		Text   string `json:"text"`
		Spoken bool   `json:"spoken"`
	}
	if httpx.DecodeJSON(w, r, &in) != nil {
		return
	}
	p, _ := httpx.PrincipalFrom(r.Context())
	s, err := h.svc.Say(r.Context(), p.UserID, r.PathValue("id"), in.Text, in.Spoken)
	h.writeSession(w, r, s, err)
}

func (h *Handler) end(w http.ResponseWriter, r *http.Request) {
	h.extendDeadline(w, r)
	p, _ := httpx.PrincipalFrom(r.Context())
	s, err := h.svc.End(r.Context(), p.UserID, r.PathValue("id"))
	h.writeSession(w, r, s, err)
}

func (h *Handler) extendDeadline(w http.ResponseWriter, r *http.Request) {
	if err := http.NewResponseController(w).SetWriteDeadline(time.Now().Add(aiWriteTimeout)); err != nil &&
		!errors.Is(err, http.ErrNotSupported) {
		h.log.WarnContext(r.Context(), "talk: extend write deadline", slog.Any("error", err))
	}
}

func (h *Handler) writeSession(w http.ResponseWriter, r *http.Request, s Session, err error) {
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"session": toSessionJSON(s)})
}

func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	var ve *ValidationError
	switch {
	case errors.As(err, &ve):
		httpx.WriteFieldErrors(w, ve.Fields)
	case errors.Is(err, ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "not_found", "Không tìm thấy buổi trò chuyện")
	case errors.Is(err, ErrTopicNotStudied):
		httpx.WriteError(w, http.StatusConflict, "topic_not_studied", "Bạn chưa học bài nào của chủ đề này")
	case errors.Is(err, ErrEnded):
		httpx.WriteError(w, http.StatusConflict, "ended", "Buổi trò chuyện đã kết thúc")
	case errors.Is(err, ErrTurnLimit):
		httpx.WriteError(w, http.StatusConflict, "turn_limit",
			fmt.Sprintf("Mỗi buổi tối đa %d lượt. Bấm Kết thúc để xem nhận xét.", MaxTurns))
	case errors.Is(err, ai.ErrNotConfigured):
		httpx.WriteError(w, http.StatusServiceUnavailable, "ai_not_configured", "AI chưa được cấu hình. Liên hệ người vận hành.")
	case errors.Is(err, ai.ErrInvalidKey):
		httpx.WriteError(w, http.StatusServiceUnavailable, "ai_not_configured", "Khoá AI không hợp lệ. Liên hệ người vận hành.")
	case errors.Is(err, ai.ErrQuota):
		httpx.WriteError(w, http.StatusTooManyRequests, "ai_quota", "Đã hết lượt AI, vui lòng thử lại sau.")
	case IsAIError(err):
		h.log.WarnContext(r.Context(), "talk AI failed", slog.Any("error", err))
		httpx.WriteError(w, http.StatusBadGateway, "ai_failed", "AI chưa trả lời được, vui lòng thử lại.")
	default:
		h.log.ErrorContext(r.Context(), "talk request failed", slog.Any("error", err))
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "Có lỗi xảy ra, vui lòng thử lại")
	}
}
