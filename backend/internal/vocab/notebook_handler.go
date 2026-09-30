package vocab

import (
	"net/http"
	"strconv"

	"github.com/luongtran/luna/backend/internal/platform/httpx"
)

type dayCardJSON struct {
	cardJSON
	Day string `json:"day"`
}

type pageJSON struct {
	Cards     []dayCardJSON `json:"cards"`
	HasMore   bool          `json:"hasMore"`
	Today     string        `json:"today"`
	Yesterday string        `json:"yesterday"`
}

type lessonCountJSON struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Count int    `json:"count"`
}

type lessonsJSON struct {
	Lessons     []lessonCountJSON `json:"lessons"`
	ManualCount int               `json:"manualCount"`
}

type updateRequest struct {
	MeaningVi       *string `json:"meaningVi"`
	IPA             *string `json:"ipa"`
	ContextSentence *string `json:"contextSentence"`
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	p, _ := httpx.PrincipalFrom(r.Context())
	query := r.URL.Query()
	page := 1
	if v := query.Get("page"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			httpx.WriteFieldErrors(w, map[string]string{"page": "Trang không hợp lệ"})
			return
		}
		page = n
	}
	res, err := h.svc.List(r.Context(), p.UserID, ListQuery{Q: query.Get("q"), LessonID: query.Get("lessonId"), Page: page})
	if err != nil {
		h.fail(w, r, "list cards failed", err)
		return
	}
	out := pageJSON{Cards: make([]dayCardJSON, len(res.Cards)), HasMore: res.HasMore, Today: res.Today, Yesterday: res.Yesterday}
	for i, c := range res.Cards {
		out.Cards[i] = dayCardJSON{cardJSON: toJSON(c.Card), Day: c.Day}
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) lessons(w http.ResponseWriter, r *http.Request) {
	p, _ := httpx.PrincipalFrom(r.Context())
	lessons, manual, err := h.svc.Lessons(r.Context(), p.UserID)
	if err != nil {
		h.fail(w, r, "list card lessons failed", err)
		return
	}
	out := lessonsJSON{Lessons: make([]lessonCountJSON, len(lessons)), ManualCount: manual}
	for i, l := range lessons {
		out.Lessons[i] = lessonCountJSON(l)
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	p, _ := httpx.PrincipalFrom(r.Context())
	var req updateRequest
	if httpx.DecodeJSON(w, r, &req) != nil {
		return // the word, lemma and schedule fields are rejected here
	}
	c, err := h.svc.Update(r.Context(), p.UserID, r.PathValue("id"), Details(req))
	if err != nil {
		h.fail(w, r, "update card failed", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]cardJSON{"card": toJSON(c)})
}

func (h *Handler) remove(w http.ResponseWriter, r *http.Request) {
	p, _ := httpx.PrincipalFrom(r.Context())
	if err := h.svc.Delete(r.Context(), p.UserID, r.PathValue("id")); err != nil {
		h.fail(w, r, "delete card failed", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
