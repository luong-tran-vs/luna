package wordbank

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/luongtran/luna/backend/internal/ai"
	"github.com/luongtran/luna/backend/internal/lesson"
	"github.com/luongtran/luna/backend/internal/platform/httpx"
)

// drawTimeout is how long drawing one picture may keep the request open: longer than the server's
// write timeout, which suits ordinary requests.
const drawTimeout = 5*time.Minute + 10*time.Second

// meaningsTimeout is how long filling the meanings and IPA of a topic (up to 300 words in one AI
// request) may keep the request open.
const meaningsTimeout = 5*time.Minute + 10*time.Second

// Handler serves the admin word bank API (F24).
type Handler struct {
	svc *Service
	log *slog.Logger
}

// NewHandler returns a Handler for svc.
func NewHandler(svc *Service, log *slog.Logger) *Handler {
	return &Handler{svc: svc, log: log}
}

// Register adds the admin routes, behind requireAuth and the admin role.
func (h *Handler) Register(mux *http.ServeMux, requireAuth httpx.Middleware) {
	admin := func(f http.HandlerFunc) http.Handler { return requireAuth(httpx.RequireAdmin(f)) }

	mux.Handle("GET /api/admin/words", admin(h.list))
	mux.Handle("GET /api/admin/words/missing-by-topic", admin(h.missingByTopic))
	mux.Handle("POST /api/admin/words", admin(h.add))
	mux.Handle("POST /api/admin/words/import", admin(h.importTopics))
	mux.Handle("POST /api/admin/words/fill-missing", admin(h.fillMissing))
	mux.Handle("POST /api/admin/words/ipa-suggestions/accept", admin(h.acceptIPASuggestions))
	mux.Handle("GET /api/admin/words/image-batch", admin(h.batch))
	mux.Handle("POST /api/admin/words/image-batch", admin(h.startBatch))
	mux.Handle("POST /api/admin/words/image-batch/stop", admin(h.stopBatch))
	mux.Handle("PATCH /api/admin/words/{lemma}", admin(h.update))
	mux.Handle("DELETE /api/admin/words/{lemma}", admin(h.delete))
	mux.Handle("GET /api/admin/words/{lemma}/image", admin(h.image))
	mux.Handle("PUT /api/admin/words/{lemma}/image", admin(h.uploadImage))
	mux.Handle("DELETE /api/admin/words/{lemma}/image", admin(h.deleteImage))
	mux.Handle("POST /api/admin/words/{lemma}/image/import", admin(h.importImage))
	mux.Handle("POST /api/admin/words/{lemma}/image/generate", admin(h.generateImage))
}

type wordJSON struct {
	Lemma     string `json:"lemma"`
	MeaningVi string `json:"meaningVi"`
	IPA       string `json:"ipa"`
	// ImageURL is "" when the word has no picture; it changes when the picture does.
	ImageURL  string    `json:"imageUrl"`
	UpdatedAt time.Time `json:"updatedAt"`
	// Topics are the topics whose word list holds the word, by name.
	Topics []topicJSON `json:"topics"`
	// IPASuggestion is the standard IPA offered for a stored one that needs a look.
	IPASuggestion string `json:"ipaSuggestion,omitempty"`
}

type topicJSON struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func toJSON(w Word) wordJSON {
	out := wordJSON{
		Lemma: w.Lemma, MeaningVi: w.MeaningVi, IPA: w.IPA, UpdatedAt: w.UpdatedAt, Topics: make([]topicJSON, len(w.Topics)),
		IPASuggestion: w.IPASuggestion,
	}
	for i, t := range w.Topics {
		out.Topics[i] = topicJSON(t)
	}
	if w.HasImage() {
		out.ImageURL = "/api/admin/words/" + url.PathEscape(w.Lemma) + "/image?v=" + strconv.FormatInt(w.ImageAt.UnixMilli(), 10)
	}
	return out
}

type pageJSON struct {
	Words   []wordJSON `json:"words"`
	Total   int        `json:"total"`
	HasMore bool       `json:"hasMore"`
	// CodexImages tells the page it may offer drawing with the Codex CLI (added 2026-11-04).
	CodexImages bool `json:"codexImages"`
}

// list serves GET /api/admin/words?q=&missing=image|ipa|meaning|ipaCheck&topicId=&page=1.
func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page := 1
	if p := q.Get("page"); p != "" {
		n, err := strconv.Atoi(p)
		if err != nil {
			httpx.WriteFieldErrors(w, map[string]string{"page": "Trang không hợp lệ"})
			return
		}
		page = n
	}
	p, err := h.svc.List(r.Context(), q.Get("q"), Missing(q.Get("missing")), q.Get("topicId"), page)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	out := pageJSON{Words: make([]wordJSON, len(p.Words)), Total: p.Total, HasMore: p.HasMore, CodexImages: h.svc.CodexAvailable()}
	for i, wd := range p.Words {
		out.Words[i] = toJSON(wd)
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

// acceptIPASuggestions serves POST /api/admin/words/ipa-suggestions/accept: every IPA that has a
// suggestion takes it, answered with {"updated": n}.
func (h *Handler) acceptIPASuggestions(w http.ResponseWriter, r *http.Request) {
	n, err := h.svc.AcceptIPASuggestions(r.Context())
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]int{"updated": n})
}

type topicMissingJSON struct {
	TopicID string `json:"topicId"`
	Image   int    `json:"image"`
	IPA     int    `json:"ipa"`
	Meaning int    `json:"meaning"`
}

// missingByTopic serves GET /api/admin/words/missing-by-topic: per topic, how many of its bank
// words lack a picture, an IPA and a meaning; topics lacking nothing are left out.
func (h *Handler) missingByTopic(w http.ResponseWriter, r *http.Request) {
	counts, err := h.svc.MissingByTopic(r.Context())
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	out := make([]topicMissingJSON, len(counts))
	for i, c := range counts {
		out[i] = topicMissingJSON(c)
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"topics": out})
}

type inputJSON struct {
	Lemma     string `json:"lemma"`
	MeaningVi string `json:"meaningVi"`
	IPA       string `json:"ipa"`
	TopicID   string `json:"topicId"`
}

// addedJSON is the word Add stored, with what it did: "inBank" when the word was in the bank
// already, "topic" the topic it was added to (null without one).
type addedJSON struct {
	wordJSON
	InBank bool       `json:"inBank"`
	Topic  *topicJSON `json:"topic"`
}

// add serves POST /api/admin/words with {"lemma", "meaningVi"?, "ipa"?, "topicId"?}: 201 with the
// word, IPA and meaning filled from the dictionary when left empty; 200 when the word was in the
// bank already and only went into the topic.
func (h *Handler) add(w http.ResponseWriter, r *http.Request) {
	var in inputJSON
	if httpx.DecodeJSON(w, r, &in) != nil {
		return
	}
	wd, added, err := h.svc.Add(r.Context(), Input(in))
	if errors.Is(err, ErrInTopic) {
		httpx.WriteFieldErrors(w, map[string]string{"lemma": fmt.Sprintf("Từ này đã có trong chủ đề \"%s\"", added.Topic.Name)})
		return
	}
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	out := addedJSON{wordJSON: toJSON(wd), InBank: added.InBank}
	if added.Topic.ID != "" {
		t := topicJSON(added.Topic)
		out.Topic = &t
	}
	status := http.StatusCreated
	if added.InBank {
		status = http.StatusOK
	}
	httpx.WriteJSON(w, status, out)
}

// importTopics serves POST /api/admin/words/import: adds the topic words missing from the bank.
func (h *Handler) importTopics(w http.ResponseWriter, r *http.Request) {
	n, err := h.svc.Import(r.Context())
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]int{"added": n})
}

// fillMissing serves POST /api/admin/words/fill-missing with {"topicId"}: one AI request for what
// the topic's words lack (meaning, IPA), answered with {"asked", "meanings", "ipas"}.
func (h *Handler) fillMissing(w http.ResponseWriter, r *http.Request) {
	var in struct {
		TopicID string `json:"topicId"`
	}
	if httpx.DecodeJSON(w, r, &in) != nil {
		return
	}
	if err := http.NewResponseController(w).SetWriteDeadline(time.Now().Add(meaningsTimeout)); err != nil {
		h.log.WarnContext(r.Context(), "word bank: extend write deadline", slog.Any("error", err))
	}
	f, err := h.svc.FillMissing(r.Context(), in.TopicID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]int{"asked": f.Asked, "meanings": f.Meanings, "ipas": f.IPAs})
}

type detailsJSON struct {
	MeaningVi *string `json:"meaningVi"`
	IPA       *string `json:"ipa"`
}

// update serves PATCH /api/admin/words/{lemma} with {"meaningVi"?, "ipa"?}.
func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	var in detailsJSON
	if httpx.DecodeJSON(w, r, &in) != nil {
		return
	}
	wd, err := h.svc.Update(r.Context(), r.PathValue("lemma"), Details(in))
	h.writeWord(w, r, wd, err)
}

// delete serves DELETE /api/admin/words/{lemma}: 204.
func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), r.PathValue("lemma")); err != nil {
		h.writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// image serves GET /api/admin/words/{lemma}/image. The URL carries the picture's time, so the
// browser may keep it.
func (h *Handler) image(w http.ResponseWriter, r *http.Request) {
	img, err := h.svc.Image(r.Context(), r.PathValue("lemma"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	ServeImage(w, r, img)
}

// ServeImage writes a picture of the bank. The same lemma may get another picture, so the browser
// asks again each time (If-Modified-Since answered with 304 while unchanged).
func ServeImage(w http.ResponseWriter, r *http.Request, img Image) {
	w.Header().Set("Content-Type", img.MIME)
	w.Header().Set("Cache-Control", "private, no-cache")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeContent(w, r, "", img.CreatedAt, bytes.NewReader(img.Data))
}

// uploadImage serves PUT /api/admin/words/{lemma}/image: the body is the picture (JPEG, PNG or GIF).
func (h *Handler) uploadImage(w http.ResponseWriter, r *http.Request) {
	data, err := io.ReadAll(http.MaxBytesReader(w, r.Body, lesson.MaxUploadBytes))
	var tooBig *http.MaxBytesError
	if errors.As(err, &tooBig) {
		httpx.WriteFieldErrors(w, map[string]string{"image": "Ảnh tối đa 5 MB"})
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "bad_request", "Không đọc được ảnh gửi lên")
		return
	}
	wd, err := h.svc.UploadImage(r.Context(), r.PathValue("lemma"), data)
	h.writeWord(w, r, wd, err)
}

// importImage serves POST /api/admin/words/{lemma}/image/import with {"url": "..."}.
func (h *Handler) importImage(w http.ResponseWriter, r *http.Request) {
	var in struct {
		URL string `json:"url"`
	}
	if httpx.DecodeJSON(w, r, &in) != nil {
		return
	}
	wd, err := h.svc.ImportImage(r.Context(), r.PathValue("lemma"), in.URL)
	h.writeWord(w, r, wd, err)
}

// generateImage serves POST /api/admin/words/{lemma}/image/generate with {"style"?, "engine"?}:
// one AI request, answered when the picture is saved. engine is "gemini" (default) or "codex".
func (h *Handler) generateImage(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Style  string `json:"style"`
		Engine Engine `json:"engine"`
	}
	if httpx.DecodeJSON(w, r, &in) != nil {
		return
	}
	if err := http.NewResponseController(w).SetWriteDeadline(time.Now().Add(drawTimeout)); err != nil {
		h.log.WarnContext(r.Context(), "word bank: extend write deadline", slog.Any("error", err))
	}
	wd, err := h.svc.GenerateImage(r.Context(), r.PathValue("lemma"), in.Style, in.Engine)
	h.writeWord(w, r, wd, err)
}

// deleteImage serves DELETE /api/admin/words/{lemma}/image, answering with the word.
func (h *Handler) deleteImage(w http.ResponseWriter, r *http.Request) {
	wd, err := h.svc.DeleteImage(r.Context(), r.PathValue("lemma"))
	h.writeWord(w, r, wd, err)
}

func (h *Handler) writeWord(w http.ResponseWriter, r *http.Request, wd Word, err error) {
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toJSON(wd))
}

func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	var verr *ValidationError
	if errors.As(err, &verr) {
		httpx.WriteFieldErrors(w, verr.Fields)
		return
	}
	if errors.Is(err, ErrExists) {
		httpx.WriteFieldErrors(w, map[string]string{"lemma": "Từ này đã có trong kho"})
		return
	}
	status, code, msg := describe(err)
	switch {
	case errors.Is(err, ErrDrawFailed):
		h.log.WarnContext(r.Context(), "word bank: draw picture", slog.Any("error", err))
	case errors.Is(err, ErrMeaningsFailed):
		h.log.WarnContext(r.Context(), "word bank: fill missing", slog.Any("error", err))
	case status == http.StatusInternalServerError:
		h.log.ErrorContext(r.Context(), "word bank request failed", slog.Any("error", err))
	}
	httpx.WriteError(w, status, code, msg)
}

// describe is the HTTP status, code and Vietnamese message of an error other than a ValidationError;
// the picture batch shows the same messages for its words.
func describe(err error) (int, string, string) {
	switch {
	case errors.Is(err, ErrNotFound):
		return http.StatusNotFound, "not_found", "Không tìm thấy từ này trong kho"
	case errors.Is(err, ErrImageNotFound):
		return http.StatusNotFound, "not_found", "Từ này chưa có ảnh"
	case errors.Is(err, ErrBatchRunning):
		return http.StatusConflict, "batch_running", "Đang có một lượt sinh ảnh chạy. Chờ xong hoặc bấm Dừng."
	case errors.Is(err, ErrCodexUnavailable):
		return http.StatusServiceUnavailable, "codex_unavailable", "Máy chủ này không bật sinh ảnh bằng Codex CLI (CODEX_IMAGES)."
	case errors.Is(err, ErrCodexNotLoggedIn):
		return http.StatusServiceUnavailable, "codex_unavailable", "Codex CLI trên máy chủ chưa đăng nhập. Chạy codex login rồi thử lại."
	case errors.Is(err, ai.ErrNoImageTool):
		return http.StatusBadGateway, "codex_no_image_tool", "Codex CLI không có công cụ tạo ảnh (image_gen) khi chạy codex exec. Hãy dùng Gemini."
	case errors.Is(err, ErrImagesUnavailable), errors.Is(err, ai.ErrNotConfigured):
		return http.StatusServiceUnavailable, "ai_not_configured", "AI chưa được cấu hình. Liên hệ người vận hành."
	case errors.Is(err, ai.ErrInvalidKey):
		return http.StatusServiceUnavailable, "ai_not_configured", "Khoá AI không hợp lệ. Liên hệ người vận hành."
	case errors.Is(err, ai.ErrQuota):
		return http.StatusTooManyRequests, "ai_quota", "Đã hết lượt AI, vui lòng thử lại sau."
	case errors.Is(err, ErrDrawFailed), errors.Is(err, context.DeadlineExceeded):
		return http.StatusBadGateway, "ai_failed", "Sinh ảnh thất bại, vui lòng thử lại."
	case errors.Is(err, ErrMeaningsFailed):
		return http.StatusBadGateway, "ai_failed", "AI chưa điền được nghĩa và phiên âm, vui lòng thử lại."
	default:
		return http.StatusInternalServerError, "internal_error", "Có lỗi xảy ra, vui lòng thử lại"
	}
}

type batchFailureJSON struct {
	Lemma   string `json:"lemma"`
	Message string `json:"message"`
}

// batchJSON is how the picture batch is going. done counts the words dealt with (saved or failed);
// saved holds the words that got their picture, so the page can show it at once.
type batchJSON struct {
	Running     bool               `json:"running"`
	Stopping    bool               `json:"stopping"`
	Stopped     bool               `json:"stopped"`
	Engine      Engine             `json:"engine"`
	Ticked      bool               `json:"ticked"`
	TopicName   string             `json:"topicName,omitempty"`
	Search      string             `json:"search,omitempty"`
	Topics      []string           `json:"topics"`
	Total       int                `json:"total"`
	Done        int                `json:"done"`
	Drawing     []string           `json:"drawing"`
	Saved       []wordJSON         `json:"saved"`
	Failures    []batchFailureJSON `json:"failures"`
	StopMessage string             `json:"stopMessage,omitempty"`
	StartedAt   *time.Time         `json:"startedAt,omitempty"`
	FinishedAt  *time.Time         `json:"finishedAt,omitempty"`
}

func batchToJSON(b BatchStatus) batchJSON {
	out := batchJSON{
		Running: b.Running, Stopping: b.Stopping, Stopped: b.Stopped, Engine: b.Engine, Total: b.Total, Done: b.Done(),
		Ticked: b.Ticked, TopicName: b.TopicName, Search: b.Search, Topics: b.Topics,
		Drawing: b.Drawing, Saved: make([]wordJSON, len(b.Saved)), Failures: make([]batchFailureJSON, len(b.Failures)),
	}
	if out.Drawing == nil {
		out.Drawing = []string{}
	}
	if out.Topics == nil {
		out.Topics = []string{}
	}
	for i, w := range b.Saved {
		out.Saved[i] = toJSON(w)
	}
	for i, f := range b.Failures {
		_, _, msg := describe(f.Err)
		out.Failures[i] = batchFailureJSON{Lemma: f.Lemma, Message: msg}
	}
	if b.StopErr != nil {
		_, _, out.StopMessage = describe(b.StopErr)
	}
	if !b.StartedAt.IsZero() {
		out.StartedAt = &b.StartedAt
	}
	if !b.FinishedAt.IsZero() {
		out.FinishedAt = &b.FinishedAt
	}
	return out
}

// batch serves GET /api/admin/words/image-batch: how the running, or last, picture batch is going.
func (h *Handler) batch(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, batchToJSON(h.svc.Batch()))
}

// startBatch serves POST /api/admin/words/image-batch with {"lemmas"?, "q"?, "topicId"?, "engine"?,
// "style"?}: 202 once the batch runs in the background; 409 while another one runs.
func (h *Handler) startBatch(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Lemmas  []string `json:"lemmas"`
		Q       string   `json:"q"`
		TopicID string   `json:"topicId"`
		Engine  Engine   `json:"engine"`
		Style   string   `json:"style"`
	}
	if httpx.DecodeJSON(w, r, &in) != nil {
		return
	}
	st, err := h.svc.StartBatch(r.Context(), BatchRequest{
		Lemmas: in.Lemmas, Search: in.Q, TopicID: in.TopicID, Engine: in.Engine, Style: in.Style,
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusAccepted, batchToJSON(st))
}

// stopBatch serves POST /api/admin/words/image-batch/stop: the words being drawn finish, no new one starts.
func (h *Handler) stopBatch(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, batchToJSON(h.svc.StopBatch()))
}
