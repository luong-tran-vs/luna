package topicimage

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/luongtran/luna/backend/internal/ai"
	"github.com/luongtran/luna/backend/internal/platform/httpx"
)

// Handler serves the topic pictures: the admin's buttons that draw them and the learners' pages
// that show them.
type Handler struct {
	svc *Service
	log *slog.Logger
}

// NewHandler returns the HTTP handler of the topic pictures.
func NewHandler(svc *Service, log *slog.Logger) *Handler {
	return &Handler{svc: svc, log: log}
}

// Register adds the routes: the pictures to every account (guests included), the rest to admins.
func (h *Handler) Register(mux *http.ServeMux, requireAuth httpx.Middleware) {
	admin := func(f http.Handler) http.Handler { return requireAuth(httpx.RequireAdmin(f)) }
	mux.Handle("GET /api/topic-images", requireAuth(http.HandlerFunc(h.list)))
	mux.Handle("GET /api/topic-images/{id}", requireAuth(http.HandlerFunc(h.image)))

	mux.Handle("GET /api/admin/topic-images", admin(http.HandlerFunc(h.adminList)))
	mux.Handle("DELETE /api/admin/topic-images/{id}", admin(http.HandlerFunc(h.remove)))
	mux.Handle("GET /api/admin/topic-image-batch", admin(http.HandlerFunc(h.batch)))
	mux.Handle("POST /api/admin/topic-image-batch", admin(http.HandlerFunc(h.startBatch)))
	mux.Handle("POST /api/admin/topic-image-batch/stop", admin(http.HandlerFunc(h.stopBatch)))
}

// imageURL is where a topic's picture is served; it changes when the picture does.
func imageURL(id string, at time.Time) string {
	return "/api/topic-images/" + url.PathEscape(id) + "?v=" + strconv.FormatInt(at.UnixMilli(), 10)
}

type imageJSON struct {
	TopicID   string    `json:"topicId"`
	ImageURL  string    `json:"imageUrl"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func imagesJSON(list []Info) []imageJSON {
	out := make([]imageJSON, len(list))
	for i, im := range list {
		out[i] = imageJSON{TopicID: im.TopicID, ImageURL: imageURL(im.TopicID, im.CreatedAt), UpdatedAt: im.CreatedAt}
	}
	return out
}

// list serves GET /api/topic-images: the topics that have a picture, with its URL.
func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.List(r.Context())
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"images": imagesJSON(list)})
}

// adminList serves GET /api/admin/topic-images: as list, and whether this server runs Codex.
func (h *Handler) adminList(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.List(r.Context())
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"images": imagesJSON(list), "codexImages": h.svc.CodexAvailable()})
}

// image serves GET /api/topic-images/{id}: the picture itself.
func (h *Handler) image(w http.ResponseWriter, r *http.Request) {
	img, err := h.svc.Image(r.Context(), r.PathValue("id"))
	if errors.Is(err, ErrImageNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "Chủ đề này chưa có ảnh")
		return
	}
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	// The URL carries the picture's time, so a new picture gets a new URL: cache it for long.
	w.Header().Set("Content-Type", img.MIME)
	w.Header().Set("Cache-Control", "private, max-age=31536000, immutable")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeContent(w, r, "", img.CreatedAt, bytes.NewReader(img.Data))
}

// remove serves DELETE /api/admin/topic-images/{id}.
func (h *Handler) remove(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), r.PathValue("id")); err != nil {
		h.writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type failureJSON struct {
	TopicID string `json:"topicId"`
	Message string `json:"message"`
}

// batchJSON is how the topic picture batch is going. done counts the topics dealt with.
type batchJSON struct {
	Running     bool          `json:"running"`
	Stopping    bool          `json:"stopping"`
	Stopped     bool          `json:"stopped"`
	Engine      Engine        `json:"engine"`
	Total       int           `json:"total"`
	Done        int           `json:"done"`
	Drawing     []string      `json:"drawing"`
	Saved       []string      `json:"saved"`
	Failures    []failureJSON `json:"failures"`
	StopMessage string        `json:"stopMessage,omitempty"`
	CodexImages bool          `json:"codexImages"`
	StartedAt   *time.Time    `json:"startedAt,omitempty"`
	FinishedAt  *time.Time    `json:"finishedAt,omitempty"`
}

func (h *Handler) batchJSON(b BatchStatus) batchJSON {
	out := batchJSON{
		Running: b.Running, Stopping: b.Stopping, Stopped: b.Stopped, Engine: b.Engine, Total: b.Total, Done: b.Done(),
		Drawing: b.Drawing, Saved: b.Saved, Failures: make([]failureJSON, len(b.Failures)),
		CodexImages: h.svc.CodexAvailable(),
	}
	if out.Drawing == nil {
		out.Drawing = []string{}
	}
	if out.Saved == nil {
		out.Saved = []string{}
	}
	for i, f := range b.Failures {
		out.Failures[i] = failureJSON{TopicID: f.TopicID, Message: failMessage(f.Err)}
	}
	if b.StopErr != nil {
		out.StopMessage = failMessage(b.StopErr)
	}
	if !b.StartedAt.IsZero() {
		out.StartedAt = &b.StartedAt
	}
	if !b.FinishedAt.IsZero() {
		out.FinishedAt = &b.FinishedAt
	}
	return out
}

// batch serves GET /api/admin/topic-image-batch: how the running, or last, batch is going.
func (h *Handler) batch(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, h.batchJSON(h.svc.Batch()))
}

// startBatch serves POST /api/admin/topic-image-batch with {"topicIds": [...], "engine":
// "gemini"|"codex", "style"?}: 202 once the batch runs in the background; 409 while another runs.
func (h *Handler) startBatch(w http.ResponseWriter, r *http.Request) {
	var in struct {
		TopicIDs []string `json:"topicIds"`
		Engine   Engine   `json:"engine"`
		Style    string   `json:"style"`
	}
	if httpx.DecodeJSON(w, r, &in) != nil {
		return
	}
	st, err := h.svc.StartBatch(r.Context(), BatchRequest{TopicIDs: in.TopicIDs, Engine: in.Engine, Style: in.Style})
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusAccepted, h.batchJSON(st))
}

// stopBatch serves POST /api/admin/topic-image-batch/stop.
func (h *Handler) stopBatch(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, h.batchJSON(h.svc.StopBatch()))
}

func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	var verr *ValidationError
	switch {
	case errors.As(err, &verr):
		httpx.WriteFieldErrors(w, verr.Fields)
	case errors.Is(err, ErrBatchRunning):
		httpx.WriteError(w, http.StatusConflict, "batch_running", "Đang có một lượt tạo ảnh chủ đề chạy. Chờ xong hoặc bấm Dừng.")
	default:
		if status, code, msg, ok := drawError(err); ok {
			httpx.WriteError(w, status, code, msg)
			return
		}
		h.log.ErrorContext(r.Context(), "topic images request failed", slog.Any("error", err))
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "Có lỗi xảy ra, vui lòng thử lại")
	}
}

// drawError is the HTTP status, code and Vietnamese message of a failed drawing, worded as for the
// other pictures; ok is false for any other error.
func drawError(err error) (status int, code, msg string, ok bool) {
	switch {
	case errors.Is(err, ErrCodexUnavailable):
		return http.StatusServiceUnavailable, "codex_unavailable", "Máy chủ này không bật sinh ảnh bằng Codex CLI (CODEX_IMAGES).", true
	case errors.Is(err, ErrCodexNotLoggedIn):
		return http.StatusServiceUnavailable, "codex_unavailable", "Codex CLI trên máy chủ chưa đăng nhập. Chạy codex login rồi thử lại.", true
	case errors.Is(err, ai.ErrNoImageTool):
		return http.StatusBadGateway, "codex_no_image_tool", "Codex CLI không có công cụ tạo ảnh (image_gen) khi chạy codex exec. Hãy dùng Gemini.", true
	case errors.Is(err, ErrImagesUnavailable), errors.Is(err, ai.ErrNotConfigured):
		return http.StatusServiceUnavailable, "ai_not_configured", "AI chưa được cấu hình. Liên hệ người vận hành.", true
	case errors.Is(err, ai.ErrInvalidKey):
		return http.StatusServiceUnavailable, "ai_not_configured", "Khoá AI không hợp lệ. Liên hệ người vận hành.", true
	case errors.Is(err, ai.ErrQuota):
		return http.StatusTooManyRequests, "ai_quota", "Đã hết lượt AI, vui lòng thử lại sau.", true
	case errors.Is(err, ErrDrawFailed), errors.Is(err, context.DeadlineExceeded):
		return http.StatusBadGateway, "ai_failed", "Sinh ảnh thất bại, vui lòng thử lại.", true
	}
	return 0, "", "", false
}

// failMessage is the Vietnamese reason a topic of a batch got no picture.
func failMessage(err error) string {
	if _, _, msg, ok := drawError(err); ok {
		return msg
	}
	return "Sinh ảnh thất bại, vui lòng thử lại."
}
