package vocab

import (
	"net/http"

	"github.com/luongtran/luna/backend/internal/platform/httpx"
)

type bulkRequest struct {
	LessonID string   `json:"lessonId"`
	Lemmas   []string `json:"lemmas"`
}

type bulkJSON struct {
	Added int        `json:"added"`
	Cards []cardJSON `json:"cards"`
}

func (h *Handler) bulk(w http.ResponseWriter, r *http.Request) {
	p, _ := httpx.PrincipalFrom(r.Context())
	var req bulkRequest
	if httpx.DecodeJSON(w, r, &req) != nil {
		return
	}
	res, err := h.svc.SaveBulk(r.Context(), p.UserID, req.LessonID, req.Lemmas)
	if err != nil {
		h.fail(w, r, "save lesson words failed", err)
		return
	}
	out := bulkJSON{Added: res.Added, Cards: make([]cardJSON, len(res.Cards))}
	for i, c := range res.Cards {
		out.Cards[i] = toJSON(c)
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}
