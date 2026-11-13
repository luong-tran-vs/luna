package verbs

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/luongtran/luna/backend/internal/ai"
	"github.com/luongtran/luna/backend/internal/platform/httpx"
)

// ImageHandler serves the verb pictures: the admin page that draws them (added 2026-11-10) and the
// learners' "Động từ" tab that shows them.
type ImageHandler struct {
	svc *Images
	log *slog.Logger
}

// NewImageHandler returns the HTTP handler of the verb pictures.
func NewImageHandler(svc *Images, log *slog.Logger) *ImageHandler {
	return &ImageHandler{svc: svc, log: log}
}

// Register adds the routes: the pictures to every account (guests included, like the verbs), the
// rest to admins.
func (h *ImageHandler) Register(mux *http.ServeMux, requireAuth httpx.Middleware) {
	admin := func(f http.Handler) http.Handler { return requireAuth(httpx.RequireAdmin(f)) }
	mux.Handle("GET /api/verbs/images", requireAuth(http.HandlerFunc(h.learnerList)))
	mux.Handle("GET /api/verbs/images/{base}", requireAuth(http.HandlerFunc(h.image)))

	mux.Handle("GET /api/admin/verb-images", admin(http.HandlerFunc(h.adminList)))
	mux.Handle("PUT /api/admin/verb-images/{base}", admin(http.HandlerFunc(h.upload)))
	mux.Handle("DELETE /api/admin/verb-images/{base}", admin(http.HandlerFunc(h.remove)))
	mux.Handle("POST /api/admin/verb-images/{base}/from-bank", admin(http.HandlerFunc(h.fromBank)))
	mux.Handle("GET /api/admin/verb-image-batch", admin(http.HandlerFunc(h.batch)))
	mux.Handle("POST /api/admin/verb-image-batch", admin(http.HandlerFunc(h.startBatch)))
	mux.Handle("POST /api/admin/verb-image-batch/stop", admin(http.HandlerFunc(h.stopBatch)))
}

// imageURL is where a verb's picture is served; it changes when the picture does.
func imageURL(base string, at time.Time) string {
	return "/api/verbs/images/" + url.PathEscape(base) + "?v=" + strconv.FormatInt(at.UnixMilli(), 10)
}

type imageJSON struct {
	Base      string    `json:"base"`
	ImageURL  string    `json:"imageUrl"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func toImageJSON(i ImageInfo) imageJSON {
	return imageJSON{Base: i.Base, ImageURL: imageURL(i.Base, i.CreatedAt), UpdatedAt: i.CreatedAt}
}

func imagesJSON(list []ImageInfo) []imageJSON {
	out := make([]imageJSON, len(list))
	for i, im := range list {
		out[i] = toImageJSON(im)
	}
	return out
}

// learnerList serves GET /api/verbs/images: the verbs that have a picture, with its URL.
func (h *ImageHandler) learnerList(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.List(r.Context())
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"images": imagesJSON(list)})
}

// image serves GET /api/verbs/images/{base}: the picture itself.
func (h *ImageHandler) image(w http.ResponseWriter, r *http.Request) {
	img, err := h.svc.Image(r.Context(), r.PathValue("base"))
	if errors.Is(err, ErrImageNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "Động từ này chưa có ảnh")
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

type bankJSON struct {
	Base      string `json:"base"`
	MeaningVi string `json:"meaningVi"`
	ImageURL  string `json:"imageUrl"`
}

// adminList serves GET /api/admin/verb-images?bases=go,went…: the verbs that have a picture, and,
// among bases, those whose word bank entry has one the admin may reuse.
func (h *ImageHandler) adminList(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.List(r.Context())
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	var bases []string
	for b := range strings.SplitSeq(r.URL.Query().Get("bases"), ",") {
		if b = strings.TrimSpace(b); b != "" {
			bases = append(bases, b)
		}
	}
	if len(bases) > MaxListBases {
		bases = bases[:MaxListBases]
	}
	bank, err := h.svc.BankPictures(r.Context(), bases)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	outBank := make([]bankJSON, 0, len(bank))
	for _, b := range bases {
		if p, ok := bank[b]; ok {
			outBank = append(outBank, bankJSON{
				Base: b, MeaningVi: p.MeaningVi,
				ImageURL: "/api/admin/words/" + url.PathEscape(b) + "/image?v=" + strconv.FormatInt(p.CreatedAt.UnixMilli(), 10),
			})
		}
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"images": imagesJSON(list), "bank": outBank, "codexImages": h.svc.CodexAvailable(),
	})
}

// upload serves PUT /api/admin/verb-images/{base} with the picture as the body (JPEG, PNG or GIF).
func (h *ImageHandler) upload(w http.ResponseWriter, r *http.Request) {
	data, err := io.ReadAll(http.MaxBytesReader(w, r.Body, MaxUploadBytes))
	var tooBig *http.MaxBytesError
	if errors.As(err, &tooBig) {
		httpx.WriteFieldErrors(w, map[string]string{"image": "Ảnh tối đa 5 MB"})
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "bad_request", "Không đọc được ảnh gửi lên")
		return
	}
	info, err := h.svc.Upload(r.Context(), r.PathValue("base"), data)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toImageJSON(info))
}

// fromBank serves POST /api/admin/verb-images/{base}/from-bank: the word bank's picture becomes the verb's.
func (h *ImageHandler) fromBank(w http.ResponseWriter, r *http.Request) {
	info, err := h.svc.FromBank(r.Context(), r.PathValue("base"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toImageJSON(info))
}

// remove serves DELETE /api/admin/verb-images/{base}.
func (h *ImageHandler) remove(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), r.PathValue("base")); err != nil {
		h.writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type failureJSON struct {
	Base    string `json:"base"`
	Message string `json:"message"`
}

// batchJSON is how the verb picture batch is going. done counts the verbs dealt with.
type batchJSON struct {
	Running     bool          `json:"running"`
	Stopping    bool          `json:"stopping"`
	Stopped     bool          `json:"stopped"`
	Engine      ImageEngine   `json:"engine"`
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

func (h *ImageHandler) batchJSON(b BatchStatus) batchJSON {
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
		out.Failures[i] = failureJSON{Base: f.Base, Message: failMessage(f.Err)}
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

// batch serves GET /api/admin/verb-image-batch: how the running, or last, batch is going.
func (h *ImageHandler) batch(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, h.batchJSON(h.svc.Batch()))
}

// maxBatchBodyBytes bounds the body of a verb picture batch.
const maxBatchBodyBytes = 256 << 10

// startBatch serves POST /api/admin/verb-image-batch with {"items": [{"base", "meaningVi",
// "sentence"}], "engine": "gemini"|"codex", "style"?}: 202 once the batch runs in the background;
// 409 while another one runs. The engine is required.
func (h *ImageHandler) startBatch(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Items []struct {
			Base      string `json:"base"`
			MeaningVi string `json:"meaningVi"`
			Sentence  string `json:"sentence"`
		} `json:"items"`
		Engine ImageEngine `json:"engine"`
		Style  string      `json:"style"`
	}
	// The body carries each verb's meaning and sentence: 200 of them is well past the 16 KB of
	// DecodeJSON (fixed 2026-11-13, the "missing" batch got 400 invalid_body).
	if httpx.DecodeJSONLimit(w, r, &in, maxBatchBodyBytes) != nil {
		return
	}
	items := make([]DrawItem, len(in.Items))
	for i, it := range in.Items {
		items[i] = DrawItem{Base: it.Base, MeaningVi: it.MeaningVi, Sentence: it.Sentence}
	}
	st, err := h.svc.StartBatch(r.Context(), BatchRequest{Items: items, Engine: in.Engine, Style: in.Style})
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusAccepted, h.batchJSON(st))
}

// stopBatch serves POST /api/admin/verb-image-batch/stop.
func (h *ImageHandler) stopBatch(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, h.batchJSON(h.svc.StopBatch()))
}

func (h *ImageHandler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	var verr *ValidationError
	switch {
	case errors.As(err, &verr):
		httpx.WriteFieldErrors(w, verr.Fields)
	case errors.Is(err, ErrBatchRunning):
		httpx.WriteError(w, http.StatusConflict, "batch_running", "Đang có một lượt tạo ảnh động từ chạy. Chờ xong hoặc bấm Dừng.")
	case errors.Is(err, ErrBadImage):
		httpx.WriteFieldErrors(w, map[string]string{"image": "Không đọc được ảnh. Hãy dùng JPEG, PNG hoặc GIF."})
	case errors.Is(err, ErrNoBankImage):
		httpx.WriteError(w, http.StatusNotFound, "no_bank_image", "Kho từ vựng chưa có ảnh cho từ này")
	default:
		if status, code, msg, ok := drawError(err); ok {
			httpx.WriteError(w, status, code, msg)
			return
		}
		h.log.ErrorContext(r.Context(), "verb images request failed", slog.Any("error", err))
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "Có lỗi xảy ra, vui lòng thử lại")
	}
}

// drawError is the HTTP status, code and Vietnamese message of a failed drawing, worded as in the
// word bank; ok is false for any other error.
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

// failMessage is the Vietnamese reason a verb of a batch got no picture.
func failMessage(err error) string {
	if _, _, msg, ok := drawError(err); ok {
		return msg
	}
	return "Sinh ảnh thất bại, vui lòng thử lại."
}
