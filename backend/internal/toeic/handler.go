package toeic

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/luongtran/luna/backend/internal/ai"
	"github.com/luongtran/luna/backend/internal/platform/httpx"
)

// importLimit bounds an import file.
const importLimit = 2 << 20

// Handler serves the TOEIC API.
type Handler struct {
	svc *Service
	log *slog.Logger
}

// NewHandler returns the HTTP handler of the TOEIC practice.
func NewHandler(svc *Service, log *slog.Logger) *Handler { return &Handler{svc: svc, log: log} }

// Register adds the routes to mux. Guests use the shared routes and are limited in the service
// (research R10); the notebook is for members; the content is for admins.
func (h *Handler) Register(mux *http.ServeMux, requireAuth httpx.Middleware) {
	auth := func(f http.HandlerFunc) http.Handler { return requireAuth(f) }
	member := func(f http.HandlerFunc) http.Handler { return requireAuth(httpx.RequireMember(f)) }
	admin := func(f http.HandlerFunc) http.Handler { return requireAuth(httpx.RequireAdmin(f)) }

	mux.Handle("GET /api/toeic", auth(h.library))
	mux.Handle("GET /api/toeic/tests/{key}", auth(h.testDetail))
	mux.Handle("POST /api/toeic/attempts", auth(h.start))
	mux.Handle("GET /api/toeic/attempts/{id}", auth(h.open))
	mux.Handle("PUT /api/toeic/attempts/{id}", auth(h.save))
	mux.Handle("POST /api/toeic/attempts/{id}/submit", auth(h.submit))
	mux.Handle("GET /api/toeic/attempts/{id}/result", auth(h.result))
	mux.Handle("GET /api/toeic/attempts/{id}/questions/{qid}", auth(h.question))
	mux.Handle("GET /api/toeic/images/{groupKey}/{slot}", auth(h.image))
	mux.Handle("POST /api/toeic/vocab", auth(h.vocab))

	mux.Handle("GET /api/toeic/mistakes", member(h.mistakes))
	mux.Handle("GET /api/toeic/mistakes/round", member(h.mistakeRound))
	mux.Handle("POST /api/toeic/mistakes/{qid}/check", member(h.checkMistake))
	mux.Handle("POST /api/toeic/mistakes/finish", member(h.finishMistakes))

	mux.Handle("POST /api/admin/toeic/import", admin(h.importGroups))
	mux.Handle("GET /api/admin/toeic/groups", admin(h.listGroups))
	mux.Handle("POST /api/admin/toeic/groups/status", admin(h.groupsStatus))
	mux.Handle("POST /api/admin/toeic/groups/delete", admin(h.deleteGroups))
	mux.Handle("POST /api/admin/toeic/groups/draft", admin(h.draftGroups))
	mux.Handle("GET /api/admin/toeic/groups/{key}", admin(h.getGroup))
	mux.Handle("PATCH /api/admin/toeic/groups/{key}", admin(h.groupStatus))
	mux.Handle("GET /api/admin/toeic/groups/{key}/images", admin(h.groupImages))
	mux.Handle("POST /api/admin/toeic/images/{groupKey}/{slot}/generate", admin(h.drawImage))
	mux.Handle("PUT /api/admin/toeic/images/{groupKey}/{slot}", admin(h.uploadImage))
	mux.Handle("DELETE /api/admin/toeic/images/{groupKey}/{slot}", admin(h.deleteImage))
	mux.Handle("POST /api/admin/toeic/images/batch", admin(h.startPhotoBatch))
	mux.Handle("GET /api/admin/toeic/images/batch", admin(h.photoBatch))
	mux.Handle("GET /api/admin/toeic/images/missing", admin(h.missingPictures))
	mux.Handle("POST /api/admin/toeic/images/batch/stop", admin(h.stopPhotoBatch))
	mux.Handle("GET /api/admin/toeic/tests", admin(h.listTests))
	mux.Handle("POST /api/admin/toeic/tests", admin(h.createTest))
	mux.Handle("POST /api/admin/toeic/tests/compose", admin(h.composeTest))
	mux.Handle("GET /api/admin/toeic/tests/{key}", admin(h.getTest))
	mux.Handle("PUT /api/admin/toeic/tests/{key}", admin(h.updateTest))
	mux.Handle("PATCH /api/admin/toeic/tests/{key}", admin(h.testStatus))
}

func principal(r *http.Request) httpx.Principal {
	p, _ := httpx.PrincipalFrom(r.Context())
	return p
}

func isGuest(r *http.Request) bool { return principal(r).Role == httpx.RoleGuest }

// --- learner routes ---

func (h *Handler) library(w http.ResponseWriter, r *http.Request) {
	p := principal(r)
	v, err := h.svc.Library(r.Context(), p.UserID, p.Role == httpx.RoleGuest)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, v)
}

func (h *Handler) testDetail(w http.ResponseWriter, r *http.Request) {
	v, err := h.svc.TestDetail(r.Context(), principal(r).UserID, isGuest(r), r.PathValue("key"))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, v)
}

func (h *Handler) start(w http.ResponseWriter, r *http.Request) {
	var req StartReq
	if httpx.DecodeJSON(w, r, &req) != nil {
		return
	}
	v, err := h.svc.Start(r.Context(), principal(r).UserID, isGuest(r), req)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, v)
}

func (h *Handler) open(w http.ResponseWriter, r *http.Request) {
	v, err := h.svc.Open(r.Context(), principal(r).UserID, r.PathValue("id"))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, v)
}

// progressLimit bounds a progress body (200 answers and flags).
const progressLimit = 64 << 10

func (h *Handler) save(w http.ResponseWriter, r *http.Request) {
	var p Progress
	if httpx.DecodeJSONLimit(w, r, &p, progressLimit) != nil {
		return
	}
	if err := h.svc.SaveProgress(r.Context(), principal(r).UserID, r.PathValue("id"), p); err != nil {
		h.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) submit(w http.ResponseWriter, r *http.Request) {
	var last *Progress
	if r.ContentLength != 0 && r.Header.Get("Content-Type") != "" {
		var p Progress
		if httpx.DecodeJSONLimit(w, r, &p, progressLimit) != nil {
			return
		}
		last = &p
	}
	v, err := h.svc.Submit(r.Context(), principal(r).UserID, r.PathValue("id"), last)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, v)
}

func (h *Handler) result(w http.ResponseWriter, r *http.Request) {
	v, err := h.svc.Result(r.Context(), principal(r).UserID, r.PathValue("id"))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, v)
}

func (h *Handler) question(w http.ResponseWriter, r *http.Request) {
	v, err := h.svc.QuestionDetail(r.Context(), principal(r).UserID, r.PathValue("id"), r.PathValue("qid"))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, v)
}

func (h *Handler) image(w http.ResponseWriter, r *http.Request) {
	img, err := h.svc.Image(r.Context(), r.PathValue("groupKey"), r.PathValue("slot"), principal(r).Role == httpx.RoleAdmin)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	w.Header().Set("Content-Type", img.MIME)
	w.Header().Set("Cache-Control", "private, max-age=3600")
	w.Header().Set("ETag", `"`+strconv.FormatInt(img.CreatedAt.UnixNano(), 36)+`"`)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeContent(w, r, "", img.CreatedAt, bytes.NewReader(img.Data))
}

func (h *Handler) vocab(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Word      string `json:"word"`
		IPA       string `json:"ipa"`
		MeaningVi string `json:"meaningVi"`
		Sentence  string `json:"sentence"`
	}
	if httpx.DecodeJSON(w, r, &req) != nil {
		return
	}
	err := h.svc.SaveWord(r.Context(), principal(r).UserID, CardInput{
		Word: strings.TrimSpace(req.Word), IPA: strings.TrimSpace(req.IPA), MeaningVi: strings.TrimSpace(req.MeaningVi),
		Sentence: strings.TrimSpace(req.Sentence),
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// partParam reads ?part=, 0 when missing; ok is false (and the error written) when it is not a number.
func partParam(w http.ResponseWriter, r *http.Request) (Part, bool) {
	s := r.URL.Query().Get("part")
	if s == "" {
		return 0, true
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		httpx.WriteFieldErrors(w, map[string]string{"part": "Part phải là số"})
		return 0, false
	}
	return Part(n), true
}

func (h *Handler) mistakes(w http.ResponseWriter, r *http.Request) {
	part, ok := partParam(w, r)
	if !ok {
		return
	}
	v, err := h.svc.Mistakes(r.Context(), principal(r).UserID, part)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	counts := map[string]int{}
	for p, n := range v.Counts {
		counts[strconv.Itoa(int(p))] = n
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": v.Items, "counts": counts, "need": v.Need})
}

func (h *Handler) mistakeRound(w http.ResponseWriter, r *http.Request) {
	part, ok := partParam(w, r)
	if !ok {
		return
	}
	count, _ := strconv.Atoi(r.URL.Query().Get("count"))
	items, err := h.svc.MistakeRound(r.Context(), principal(r).UserID, part, count)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) checkMistake(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Choice *int `json:"choice"`
	}
	if httpx.DecodeJSON(w, r, &req) != nil {
		return
	}
	if req.Choice == nil {
		httpx.WriteFieldErrors(w, map[string]string{"choice": "Cần phương án"})
		return
	}
	v, err := h.svc.CheckMistake(r.Context(), principal(r).UserID, r.PathValue("qid"), *req.Choice)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, v)
}

func (h *Handler) finishMistakes(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.FinishMistakes(r.Context(), principal(r).UserID); err != nil {
		h.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- admin routes ---

func (h *Handler) importGroups(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Groups  []Group `json:"groups"`
		Replace bool    `json:"replace"`
		DryRun  bool    `json:"dryRun"`
	}
	if httpx.DecodeJSONDetailed(w, r, &req, importLimit) != nil {
		return
	}
	for i := range req.Groups {
		// The status and version are the server's; a file cannot set them.
		req.Groups[i].Status, req.Groups[i].Version = "", 0
	}
	dryRun := req.DryRun || r.URL.Query().Get("dryRun") == "true"
	rep, err := h.svc.Import(r.Context(), req.Groups, req.Replace, dryRun)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, rep)
}

func (h *Handler) listGroups(w http.ResponseWriter, r *http.Request) {
	part, ok := partParam(w, r)
	if !ok {
		return
	}
	rows, err := h.svc.ListGroups(r.Context(), GroupFilter{Part: part, Status: Status(r.URL.Query().Get("status"))})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"groups": rows})
}

// groupJSON is a whole group for the admin preview.
type groupJSON struct {
	Group
	UpdatedAt time.Time `json:"updatedAt"`
	ImageURL  *string   `json:"imageUrl"`
}

func (h *Handler) getGroup(w http.ResponseWriter, r *http.Request) {
	g, err := h.svc.GetGroup(r.Context(), r.PathValue("key"))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	out := groupJSON{Group: g, UpdatedAt: g.UpdatedAt}
	if slots, err := h.svc.d.Images.Has(r.Context(), []string{g.Key}); err == nil {
		out.ImageURL = paperGroupOf(g, slots[g.Key]).ImageURL
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"group": out})
}

func (h *Handler) groupStatus(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Status Status `json:"status"`
	}
	if httpx.DecodeJSON(w, r, &req) != nil {
		return
	}
	if err := h.svc.SetGroupStatus(r.Context(), r.PathValue("key"), req.Status); err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"key": r.PathValue("key"), "status": req.Status})
}

func (h *Handler) groupsStatus(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Keys   []string `json:"keys"`
		Status Status   `json:"status"`
	}
	if httpx.DecodeJSONLimit(w, r, &req, progressLimit) != nil {
		return
	}
	n, err := h.svc.SetGroupsStatus(r.Context(), req.Keys, req.Status)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"changed": n})
}

// testJSON is a test for the admin pages.
type testJSON struct {
	Key       string     `json:"key"`
	Title     string     `json:"title"`
	Kind      TestKind   `json:"kind"`
	Order     int        `json:"order"`
	Groups    []string   `json:"groups"`
	Status    Status     `json:"status"`
	UpdatedAt time.Time  `json:"updatedAt"`
	Check     TestCheck  `json:"check"`
	GroupRows []GroupRow `json:"groupRows,omitempty"`
}

func toTestJSON(ta TestAdmin, withGroups bool) testJSON {
	t := ta.Test
	out := testJSON{Key: t.Key, Title: t.Title, Kind: t.Kind, Order: t.Order, Groups: t.Groups, Status: t.Status,
		UpdatedAt: t.UpdatedAt, Check: ta.Check}
	if out.Groups == nil {
		out.Groups = []string{}
	}
	if withGroups {
		out.GroupRows = []GroupRow{}
		for _, g := range ta.Groups {
			out.GroupRows = append(out.GroupRows, GroupRow{Key: g.Key, Part: g.Part, Level: g.Level, Questions: len(g.Questions),
				Tags: groupTags(g), Status: g.Status, Source: g.Source, Version: g.Version, NeedsPhoto: g.Part == 1,
				Tests: []string{}, Title: groupTitle(g)})
		}
	}
	return out
}

func (h *Handler) listTests(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.ListTestsAdmin(r.Context())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	out := make([]testJSON, len(list))
	for i, ta := range list {
		out[i] = toTestJSON(ta, false)
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"tests": out})
}

func (h *Handler) getTest(w http.ResponseWriter, r *http.Request) {
	ta, err := h.svc.TestAdmin(r.Context(), r.PathValue("key"))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"test": toTestJSON(ta, true)})
}

func (h *Handler) createTest(w http.ResponseWriter, r *http.Request) {
	var in TestInput
	if httpx.DecodeJSONDetailed(w, r, &in, progressLimit) != nil {
		return
	}
	ta, err := h.svc.CreateTest(r.Context(), in)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, map[string]any{"test": toTestJSON(ta, true)})
}

// --- pictures (admin) ---

func (h *Handler) codexAvailable() bool {
	return h.svc.d.Drawer != nil && h.svc.d.Drawer.CodexAvailable()
}

func (h *Handler) groupImages(w http.ResponseWriter, r *http.Request) {
	slots, err := h.svc.GroupImages(r.Context(), r.PathValue("key"))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"slots": slots, "codex": h.codexAvailable()})
}

func (h *Handler) drawImage(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Engine string `json:"engine"`
	}
	if httpx.DecodeJSON(w, r, &req) != nil {
		return
	}
	// A picture takes up to a few minutes: give the response the time it needs.
	if err := http.NewResponseController(w).SetWriteDeadline(time.Now().Add(drawTimeout + time.Minute)); err != nil &&
		!errors.Is(err, http.ErrNotSupported) {
		h.log.WarnContext(r.Context(), "toeic: extend write deadline", slog.Any("error", err))
	}
	if _, err := h.svc.DrawImage(r.Context(), r.PathValue("groupKey"), r.PathValue("slot"), req.Engine); err != nil {
		h.fail(w, r, err)
		return
	}
	h.groupImagesOf(w, r, r.PathValue("groupKey"))
}

func (h *Handler) uploadImage(w http.ResponseWriter, r *http.Request) {
	data, err := io.ReadAll(http.MaxBytesReader(w, r.Body, MaxUploadBytes))
	var tooBig *http.MaxBytesError
	if errors.As(err, &tooBig) {
		httpx.WriteFieldErrors(w, map[string]string{"image": "Ảnh tối đa 5 MB"})
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "Không đọc được ảnh")
		return
	}
	if _, err := h.svc.UploadImage(r.Context(), r.PathValue("groupKey"), r.PathValue("slot"), data); err != nil {
		h.fail(w, r, err)
		return
	}
	h.groupImagesOf(w, r, r.PathValue("groupKey"))
}

func (h *Handler) deleteImage(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.DeleteImage(r.Context(), r.PathValue("groupKey"), r.PathValue("slot")); err != nil {
		h.fail(w, r, err)
		return
	}
	h.groupImagesOf(w, r, r.PathValue("groupKey"))
}

// groupImagesOf answers with the picture slots of a group after a change.
func (h *Handler) groupImagesOf(w http.ResponseWriter, r *http.Request, key string) {
	slots, err := h.svc.GroupImages(r.Context(), key)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"slots": slots, "codex": h.codexAvailable()})
}

func (h *Handler) startPhotoBatch(w http.ResponseWriter, r *http.Request) {
	var req BatchInput
	if httpx.DecodeJSON(w, r, &req) != nil {
		return
	}
	st, err := h.svc.StartPhotoBatch(r.Context(), req, h.log)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusAccepted, map[string]any{"batch": st})
}

func (h *Handler) photoBatch(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"batch": h.svc.PhotoBatch(), "codex": h.codexAvailable()})
}

// missingPictures answers how many pictures a batch of each picture part (1, 6, 7) would draw.
func (h *Handler) missingPictures(w http.ResponseWriter, r *http.Request) {
	counts, err := h.svc.MissingPictures(r.Context())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	out := map[string]int{}
	for p, n := range counts {
		out[strconv.Itoa(int(p))] = n
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"missing": out, "codex": h.codexAvailable()})
}

func (h *Handler) stopPhotoBatch(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"batch": h.svc.StopPhotoBatch()})
}

// draftGroups asks the AI for new groups of one part and keeps the valid ones as drafts.
func (h *Handler) draftGroups(w http.ResponseWriter, r *http.Request) {
	var in DraftInput
	if httpx.DecodeJSONDetailed(w, r, &in, progressLimit) != nil {
		return
	}
	// Long parts (conversations, texts) take the AI up to a few minutes: give the response that time.
	if err := http.NewResponseController(w).SetWriteDeadline(time.Now().Add(draftTimeout + time.Minute)); err != nil &&
		!errors.Is(err, http.ErrNotSupported) {
		h.log.WarnContext(r.Context(), "toeic: extend write deadline", slog.Any("error", err))
	}
	rep, err := h.svc.DraftGroups(r.Context(), in)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, rep)
}

// composeTest makes a test from the published groups ("Ghép đề tự động"); with dryRun it only
// returns the plan. A part that cannot be filled answers 422 with the plan unless partial is set.
func (h *Handler) composeTest(w http.ResponseWriter, r *http.Request) {
	var in ComposeInput
	if httpx.DecodeJSONDetailed(w, r, &in, progressLimit) != nil {
		return
	}
	res, err := h.svc.ComposeTest(r.Context(), in)
	var verr *ValidationError
	if errors.As(err, &verr) && len(res.Plan.Parts) > 0 {
		httpx.WriteJSON(w, http.StatusUnprocessableEntity, map[string]any{
			"error": "validation_failed", "message": "Kho chưa đủ cụm đã đăng", "fields": verr.Fields, "plan": res.Plan,
		})
		return
	}
	if err != nil {
		h.fail(w, r, err)
		return
	}
	out := map[string]any{"plan": res.Plan}
	status := http.StatusOK
	if res.Test != nil {
		out["test"] = toTestJSON(*res.Test, true)
		status = http.StatusCreated
	}
	httpx.WriteJSON(w, status, out)
}

func (h *Handler) updateTest(w http.ResponseWriter, r *http.Request) {
	var in TestInput
	if httpx.DecodeJSONDetailed(w, r, &in, progressLimit) != nil {
		return
	}
	ta, err := h.svc.UpdateTest(r.Context(), r.PathValue("key"), in)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"test": toTestJSON(ta, true)})
}

func (h *Handler) testStatus(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Status Status `json:"status"`
	}
	if httpx.DecodeJSON(w, r, &req) != nil {
		return
	}
	ta, err := h.svc.SetTestStatus(r.Context(), r.PathValue("key"), req.Status)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"test": toTestJSON(ta, true)})
}

// fail writes the error of a request.
func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	var verr *ValidationError
	var inProgress *ErrAttemptInProgress
	var notReady *ErrTestNotReady
	if status, code, msg := drawErrorOf(err); status != 0 && !errors.Is(err, errDraftAI) {
		if status >= http.StatusInternalServerError {
			h.log.WarnContext(r.Context(), "toeic picture failed", slog.Any("error", err))
		}
		httpx.WriteError(w, status, code, msg)
		return
	}
	switch {
	case errors.As(err, &verr):
		httpx.WriteFieldErrors(w, verr.Fields)
	case errors.Is(err, ErrMembersOnly):
		httpx.WriteError(w, http.StatusForbidden, "members_only", "Phần này chỉ dành cho thành viên")
	case errors.Is(err, ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "not_found", "Không tìm thấy")
	case errors.Is(err, ErrExists):
		httpx.WriteError(w, http.StatusConflict, "exists", "Mã này đã có")
	case errors.As(err, &inProgress):
		httpx.WriteJSON(w, http.StatusConflict, map[string]any{
			"error": "attempt_in_progress", "message": "Bạn đang có bài làm dở", "attemptId": inProgress.ID,
		})
	case errors.Is(err, ErrAttemptSubmitted):
		httpx.WriteError(w, http.StatusConflict, "attempt_submitted", "Bài này đã nộp")
	case errors.Is(err, ErrNotEnough):
		httpx.WriteError(w, http.StatusConflict, "not_enough_questions", "Chưa đủ câu hỏi đã đăng")
	case errors.Is(err, errDraftAI) && errors.Is(err, ai.ErrNotConfigured):
		httpx.WriteError(w, http.StatusServiceUnavailable, "ai_not_configured", "Chưa cài đặt AI (khoá API)")
	case errors.Is(err, errDraftAI) && errors.Is(err, ai.ErrQuota):
		httpx.WriteError(w, http.StatusTooManyRequests, "ai_quota", "AI hết lượt dùng, thử lại sau")
	case errors.Is(err, errDraftAI):
		h.log.WarnContext(r.Context(), "toeic draft failed", slog.Any("error", err))
		httpx.WriteError(w, http.StatusBadGateway, "ai_failed", "AI soạn thất bại, vui lòng thử lại")
	case errors.As(err, &notReady):
		httpx.WriteJSON(w, http.StatusConflict, map[string]any{
			"error": "test_not_ready", "message": "Đề chưa đủ câu hoặc ảnh", "check": notReady.Check,
		})
	default:
		h.log.ErrorContext(r.Context(), "toeic request failed", slog.Any("error", err))
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "Có lỗi xảy ra, vui lòng thử lại")
	}
}

// deleteGroups removes groups from the bank (one or many); the ones in use are kept and reported.
func (h *Handler) deleteGroups(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Keys []string `json:"keys"`
	}
	if httpx.DecodeJSONLimit(w, r, &req, progressLimit) != nil {
		return
	}
	rep, err := h.svc.DeleteGroups(r.Context(), req.Keys)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, rep)
}
