package vocab

import (
	"net/http"

	"github.com/luongtran/luna/backend/internal/platform/httpx"
)

type missesRequest struct {
	LessonID string   `json:"lessonId"`
	Words    []string `json:"words"`
}

type missesJSON struct {
	Added       int `json:"added"`
	Rescheduled int `json:"rescheduled"`
}

func (h *Handler) misses(w http.ResponseWriter, r *http.Request) {
	p, _ := httpx.PrincipalFrom(r.Context())
	var req missesRequest
	if httpx.DecodeJSON(w, r, &req) != nil {
		return
	}
	res, err := h.svc.PracticeMisses(r.Context(), p.UserID, req.LessonID, req.Words)
	if err != nil {
		h.fail(w, r, "save practice misses failed", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, missesJSON(res))
}
