package vocab

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/luongtran/luna/backend/internal/platform/httpx"
)

const defaultDueLimit = 50

type intervalsJSON struct {
	Again int64 `json:"again"`
	Hard  int64 `json:"hard"`
	Good  int64 `json:"good"`
	Easy  int64 `json:"easy"`
}

type dueCardJSON struct {
	cardJSON
	Intervals intervalsJSON `json:"intervals"`
}

func toDueJSON(c DueCard) dueCardJSON {
	sec := func(d time.Duration) int64 { return int64(d.Round(time.Second) / time.Second) }
	return dueCardJSON{cardJSON: toJSON(c.Card), Intervals: intervalsJSON{
		Again: sec(c.Intervals.Again), Hard: sec(c.Intervals.Hard), Good: sec(c.Intervals.Good), Easy: sec(c.Intervals.Easy),
	}}
}

type dueListJSON struct {
	Cards   []dueCardJSON `json:"cards"`
	Total   int           `json:"total"`
	NextDue *time.Time    `json:"nextDue"`
}

type reviewRequest struct {
	Rating  int    `json:"rating"`
	Mode    string `json:"mode"`
	Reps    uint64 `json:"reps"`
	Context string `json:"context"`
}

type conflictBody struct {
	Error   string      `json:"error"`
	Message string      `json:"message"`
	Card    dueCardJSON `json:"card"`
}

func (h *Handler) due(w http.ResponseWriter, r *http.Request) {
	p, _ := httpx.PrincipalFrom(r.Context())
	limit := defaultDueLimit
	if v := r.URL.Query().Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			httpx.WriteFieldErrors(w, map[string]string{"limit": "Số thẻ cần từ 1 đến 200"})
			return
		}
		limit = n
	}
	list, err := h.svc.Due(r.Context(), p.UserID, limit)
	if err != nil {
		h.fail(w, r, "list due cards failed", err)
		return
	}
	out := dueListJSON{Cards: make([]dueCardJSON, len(list.Cards)), Total: list.Total}
	for i, c := range list.Cards {
		out.Cards[i] = toDueJSON(c)
	}
	if !list.NextDue.IsZero() {
		out.NextDue = &list.NextDue
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) review(w http.ResponseWriter, r *http.Request) {
	p, _ := httpx.PrincipalFrom(r.Context())
	var req reviewRequest
	if httpx.DecodeJSON(w, r, &req) != nil {
		return
	}
	c, err := h.svc.Review(r.Context(), p.UserID, r.PathValue("id"), ReviewInput(req))
	var conflict *ConflictError
	if errors.As(err, &conflict) {
		httpx.WriteJSON(w, http.StatusConflict, conflictBody{
			Error: "review_conflict", Message: "Thẻ này đã được đánh giá", Card: toDueJSON(conflict.Card),
		})
		return
	}
	if err != nil {
		h.fail(w, r, "review card failed", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]dueCardJSON{"card": toDueJSON(c)})
}

// fail writes the response for an error of the service.
func (h *Handler) fail(w http.ResponseWriter, r *http.Request, msg string, err error) {
	var verr *ValidationError
	switch {
	case errors.As(err, &verr):
		httpx.WriteFieldErrors(w, verr.Fields)
	case errors.Is(err, ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "not_found", "Không tìm thấy thẻ")
	case errors.Is(err, ErrLessonNotFound):
		httpx.WriteError(w, http.StatusNotFound, "not_found", "Không tìm thấy bài học")
	default:
		h.log.ErrorContext(r.Context(), msg, slog.Any("error", err))
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "Có lỗi xảy ra, vui lòng thử lại")
	}
}
