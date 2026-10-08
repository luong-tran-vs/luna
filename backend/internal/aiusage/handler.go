package aiusage

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/luongtran/luna/backend/internal/ai"
	"github.com/luongtran/luna/backend/internal/platform/httpx"
)

// Handler serves the admin AI usage page.
type Handler struct {
	svc *Service
	log *slog.Logger
}

// NewHandler returns a Handler for svc.
func NewHandler(svc *Service, log *slog.Logger) *Handler {
	return &Handler{svc: svc, log: log}
}

// Register adds GET /api/admin/ai-usage, behind requireAuth and the admin role.
func (h *Handler) Register(mux *http.ServeMux, requireAuth httpx.Middleware) {
	mux.Handle("GET /api/admin/ai-usage", requireAuth(httpx.RequireAdmin(http.HandlerFunc(h.summary))))
}

type totalsJSON struct {
	Requests     int `json:"requests"`
	Errors       int `json:"errors"`
	Quota        int `json:"quota"`
	PromptTokens int `json:"promptTokens"`
	OutputTokens int `json:"outputTokens"`
	TotalTokens  int `json:"totalTokens"`
}

type modelJSON struct {
	Model string `json:"model"`
	totalsJSON
}

type minuteJSON struct {
	Start time.Time `json:"start"`
	totalsJSON
}

type opJSON struct {
	Op string `json:"op"`
	totalsJSON
}

type callJSON struct {
	At            time.Time `json:"at"`
	Model         string    `json:"model"`
	Op            string    `json:"op"`
	Outcome       Outcome   `json:"outcome"`
	Status        int       `json:"status"`
	PromptTokens  int       `json:"promptTokens"`
	OutputTokens  int       `json:"outputTokens"`
	ThoughtTokens int       `json:"thoughtTokens"`
	TotalTokens   int       `json:"totalTokens"`
	DurationMs    int64     `json:"durationMs"`
}

type summaryJSON struct {
	Now        time.Time    `json:"now"`
	LastMinute []modelJSON  `json:"lastMinute"`
	PeakMinute totalsJSON   `json:"peakMinute"`
	Minutes    []minuteJSON `json:"minutes"`
	DayStart   time.Time    `json:"dayStart"`
	Today      []modelJSON  `json:"today"`
	Week       []opJSON     `json:"week"`
	Recent     []callJSON   `json:"recent"`
}

func totals(t Totals) totalsJSON { return totalsJSON(t) }

func models(ms []ModelTotals) []modelJSON {
	out := make([]modelJSON, len(ms))
	for i, m := range ms {
		out[i] = modelJSON{Model: m.Model, totalsJSON: totals(m.Totals)}
	}
	return out
}

// summary serves GET /api/admin/ai-usage.
func (h *Handler) summary(w http.ResponseWriter, r *http.Request) {
	s, err := h.svc.Summary(r.Context())
	if err != nil {
		h.log.ErrorContext(r.Context(), "ai usage summary failed", slog.Any("error", err))
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "Có lỗi xảy ra, vui lòng thử lại")
		return
	}
	out := summaryJSON{
		Now: s.Now, LastMinute: models(s.LastMinute), PeakMinute: totals(s.PeakMinute), DayStart: s.DayStart,
		Today: models(s.Today), Minutes: make([]minuteJSON, len(s.Minutes)), Week: make([]opJSON, len(s.Week)),
		Recent: make([]callJSON, len(s.Recent)),
	}
	for i, m := range s.Minutes {
		out.Minutes[i] = minuteJSON{Start: m.Start, totalsJSON: totals(m.Totals)}
	}
	for i, o := range s.Week {
		out.Week[i] = opJSON{Op: o.Op, totalsJSON: totals(o.Totals)}
	}
	for i, u := range s.Recent {
		out.Recent[i] = toCall(u)
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func toCall(u ai.Usage) callJSON {
	return callJSON{
		At: u.At, Model: u.Model, Op: u.Op, Outcome: OutcomeOf(u.Status), Status: u.Status,
		PromptTokens: u.PromptTokens, OutputTokens: u.OutputTokens, ThoughtTokens: u.ThoughtTokens,
		TotalTokens: u.TotalTokens, DurationMs: u.Duration.Milliseconds(),
	}
}
