package verbs

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/luongtran/luna/backend/internal/platform/httpx"
)

// Handler serves the verbs API: the learned verbs and their reviews.
type Handler struct {
	svc *Service
	log *slog.Logger
}

// NewHandler returns the HTTP handler of the verbs.
func NewHandler(svc *Service, log *slog.Logger) *Handler { return &Handler{svc: svc, log: log} }

// Register adds the routes to mux. The verbs are open to every account, guests included.
func (h *Handler) Register(mux *http.ServeMux, requireAuth httpx.Middleware) {
	mux.Handle("GET /api/verbs/progress", requireAuth(http.HandlerFunc(h.list)))
	mux.Handle("POST /api/verbs/results", requireAuth(http.HandlerFunc(h.record)))
	mux.Handle("POST /api/verbs/learned", requireAuth(http.HandlerFunc(h.learned)))
}

type progressJSON struct {
	Base      string     `json:"base"`
	LearnedAt *time.Time `json:"learnedAt"`
	Level     int        `json:"level"`
	DueAt     *time.Time `json:"dueAt"`
	Right     int        `json:"right"`
	Wrong     int        `json:"wrong"`
}

func toJSON(list []Progress) []progressJSON {
	out := make([]progressJSON, len(list))
	for i, p := range list {
		out[i] = progressJSON{Base: p.Base, Level: p.Level, Right: p.Right, Wrong: p.Wrong}
		if !p.LearnedAt.IsZero() {
			at := p.LearnedAt
			out[i].LearnedAt = &at
		}
		if !p.DueAt.IsZero() {
			due := p.DueAt
			out[i].DueAt = &due
		}
	}
	return out
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	p, _ := httpx.PrincipalFrom(r.Context())
	list, err := h.svc.List(r.Context(), p.UserID)
	if err != nil {
		h.fail(w, r, "verb progress failed", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"verbs": toJSON(list)})
}

type resultsRequest struct {
	Results []struct {
		Base  string `json:"base"`
		Right bool   `json:"right"`
	} `json:"results"`
}

// record applies a finished review round (POST /api/verbs/results) and returns the verbs it changed.
func (h *Handler) record(w http.ResponseWriter, r *http.Request) {
	p, _ := httpx.PrincipalFrom(r.Context())
	var req resultsRequest
	if httpx.DecodeJSON(w, r, &req) != nil {
		return
	}
	results := make([]Result, len(req.Results))
	for i, res := range req.Results {
		results[i] = Result{Base: res.Base, Right: res.Right}
	}
	list, err := h.svc.Record(r.Context(), p.UserID, results)
	if err != nil {
		h.fail(w, r, "verb results failed", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"verbs": toJSON(list)})
}

func (h *Handler) fail(w http.ResponseWriter, r *http.Request, msg string, err error) {
	var verr *ValidationError
	if errors.As(err, &verr) {
		httpx.WriteFieldErrors(w, verr.Fields)
		return
	}
	h.log.ErrorContext(r.Context(), msg, slog.Any("error", err))
	httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "Có lỗi xảy ra, vui lòng thử lại")
}

type learnedRequest struct {
	Bases   []string `json:"bases"`
	Learned bool     `json:"learned"`
}

// learned marks verbs as learned or no longer learned (POST /api/verbs/learned) and returns them.
func (h *Handler) learned(w http.ResponseWriter, r *http.Request) {
	p, _ := httpx.PrincipalFrom(r.Context())
	var req learnedRequest
	if httpx.DecodeJSON(w, r, &req) != nil {
		return
	}
	list, err := h.svc.SetLearned(r.Context(), p.UserID, req.Bases, req.Learned)
	if err != nil {
		h.fail(w, r, "verb learned failed", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"verbs": toJSON(list)})
}
