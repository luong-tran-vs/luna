package grammar

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/url"
	"time"

	"github.com/luongtran/luna/backend/internal/ai"
	"github.com/luongtran/luna/backend/internal/platform/httpx"
)

// registerImages adds the routes of the example pictures (added 2026-11-06).
func (h *Handler) registerImages(mux *http.ServeMux, admin, learner func(http.HandlerFunc) http.Handler) {
	mux.Handle("GET /api/admin/grammar-lessons/{pointId}/images/{key}", admin(h.adminImage))
	mux.Handle("DELETE /api/admin/grammar-lessons/{pointId}/images/{key}", admin(h.deleteImage))
	mux.Handle("GET /api/admin/grammar-image-batch", admin(h.imageBatch))
	mux.Handle("POST /api/admin/grammar-image-batch", admin(h.startImageBatch))
	mux.Handle("POST /api/admin/grammar-image-batch/stop", admin(h.stopImageBatch))
	mux.Handle("POST /api/admin/grammar-images/remove", admin(h.removeImages))
	mux.Handle("GET /api/grammar/{pointId}/images/{key}", learner(h.image))
}

// exampleImageURLs is the URL of each example's picture of content, in order, "" for one without.
func (h *Handler) exampleImageURLs(r *http.Request, pointID string, c Content, admin bool) ([]string, error) {
	keys, err := h.svc.ExampleImageKeys(r.Context(), pointID, c)
	if err != nil {
		return nil, err
	}
	base := "/api/grammar/"
	if admin {
		base = "/api/admin/grammar-lessons/"
	}
	out := make([]string, len(keys))
	for i, k := range keys {
		if k != "" {
			out[i] = base + url.PathEscape(pointID) + "/images/" + url.PathEscape(k)
		}
	}
	return out, nil
}

func (h *Handler) adminImage(w http.ResponseWriter, r *http.Request) {
	h.serveImage(w, r, true)
}

// image serves GET /api/grammar/{pointId}/images/{key}: the picture of an example of a published lesson.
func (h *Handler) image(w http.ResponseWriter, r *http.Request) {
	h.serveImage(w, r, false)
}

func (h *Handler) serveImage(w http.ResponseWriter, r *http.Request, admin bool) {
	img, err := h.svc.Image(r.Context(), r.PathValue("pointId"), r.PathValue("key"), admin)
	if errors.Is(err, ErrImageNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "Câu ví dụ này chưa có ảnh")
		return
	}
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	// A picture can be drawn again under the same URL: the browser keeps it but asks again each time.
	w.Header().Set("Content-Type", img.MIME)
	w.Header().Set("Cache-Control", "private, no-cache")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeContent(w, r, "", img.CreatedAt, bytes.NewReader(img.Data))
}

// deleteImage serves DELETE /api/admin/grammar-lessons/{pointId}/images/{key}; it answers with the lesson.
func (h *Handler) deleteImage(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("pointId")
	if err := h.svc.DeleteImage(r.Context(), id, r.PathValue("key")); err != nil {
		h.writeImageError(w, r, err)
		return
	}
	l, err := h.svc.AdminGet(r.Context(), id)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeLesson(w, r, http.StatusOK, l)
}

type imageBatchItemJSON struct {
	PointID string `json:"pointId"`
	Key     string `json:"key"`
	En      string `json:"en"`
	// Message says why the example got no picture; "" for one that got it.
	Message string `json:"message,omitempty"`
}

// imageBatchJSON is how the example picture batch is going. done counts the pictures dealt with.
type imageBatchJSON struct {
	Running  bool        `json:"running"`
	Stopping bool        `json:"stopping"`
	Stopped  bool        `json:"stopped"`
	Engine   ImageEngine `json:"engine"`
	Key      string      `json:"key"`
	Label    string      `json:"label"`
	Total    int         `json:"total"`
	Points   int         `json:"points"`
	Done     int         `json:"done"`
	// Drawing are the examples being drawn now, [] for none (a list since 2026-11-11).
	Drawing     []imageBatchItemJSON `json:"drawing"`
	Saved       []imageBatchItemJSON `json:"saved"`
	Failures    []imageBatchItemJSON `json:"failures"`
	StopMessage string               `json:"stopMessage,omitempty"`
	CodexImages bool                 `json:"codexImages"`
	StartedAt   *time.Time           `json:"startedAt,omitempty"`
	FinishedAt  *time.Time           `json:"finishedAt,omitempty"`
}

func batchItemsJSON(in []ImageBatchItem) []imageBatchItemJSON {
	out := make([]imageBatchItemJSON, len(in))
	for i, it := range in {
		out[i] = imageBatchItemJSON{PointID: it.PointID, Key: it.Key, En: it.En}
		if it.Err != nil {
			out[i].Message = failMessage(it.Err)
		}
	}
	return out
}

func (h *Handler) imageBatchJSON(b ImageBatchStatus) imageBatchJSON {
	out := imageBatchJSON{
		Running: b.Running, Stopping: b.Stopping, Stopped: b.Stopped, Engine: b.Engine, Key: b.Key, Label: b.Label,
		Total: b.Total, Points: b.Points, Done: b.Done(), Drawing: batchItemsJSON(b.Drawing), Saved: batchItemsJSON(b.Saved),
		Failures: batchItemsJSON(b.Failures), CodexImages: h.svc.CodexAvailable(),
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

// imageBatch serves GET /api/admin/grammar-image-batch: how the running, or last, batch is going.
func (h *Handler) imageBatch(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, h.imageBatchJSON(h.svc.ImageBatch()))
}

// startImageBatch serves POST /api/admin/grammar-image-batch with {"pointIds", "keys"?, "replace"?,
// "engine"?, "key"?, "label"?}: 202 once the batch runs in the background; 409 while another one runs.
func (h *Handler) startImageBatch(w http.ResponseWriter, r *http.Request) {
	var in struct {
		PointIDs []string    `json:"pointIds"`
		Keys     []string    `json:"keys"`
		Replace  bool        `json:"replace"`
		Engine   ImageEngine `json:"engine"`
		Key      string      `json:"key"`
		Label    string      `json:"label"`
	}
	if httpx.DecodeJSON(w, r, &in) != nil {
		return
	}
	st, err := h.svc.StartImageBatch(r.Context(), ImageBatchRequest{
		PointIDs: in.PointIDs, Keys: in.Keys, Replace: in.Replace, Engine: in.Engine, Key: in.Key, Label: in.Label,
	})
	if err != nil {
		h.writeImageError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusAccepted, h.imageBatchJSON(st))
}

// stopImageBatch serves POST /api/admin/grammar-image-batch/stop.
func (h *Handler) stopImageBatch(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, h.imageBatchJSON(h.svc.StopImageBatch()))
}

func (h *Handler) writeImageError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, ErrImageBatchRunning) {
		httpx.WriteError(w, http.StatusConflict, "batch_running", "Đang có một lượt tạo ảnh ví dụ chạy. Chờ xong hoặc bấm Dừng.")
		return
	}
	if status, code, msg, ok := drawError(err); ok {
		httpx.WriteError(w, status, code, msg)
		return
	}
	h.writeError(w, r, err)
}

// drawError is the HTTP status, code and Vietnamese message of a failed drawing, worded as in the
// word bank; ok is false for an error writeError handles.
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

// failMessage is the Vietnamese reason an example of a batch got no picture.
func failMessage(err error) string {
	if _, _, msg, ok := drawError(err); ok {
		return msg
	}
	return "Sinh ảnh thất bại, vui lòng thử lại."
}

// imageURL is where the picture key of point is served, to admins or to learners.
func imageURL(pointID, key string, admin bool) string {
	base := "/api/grammar/"
	if admin {
		base = "/api/admin/grammar-lessons/"
	}
	return base + url.PathEscape(pointID) + "/images/" + url.PathEscape(key)
}

// setPictureURLs gives each picture exercise of c the URL of its picture (keys by exercise id).
func setPictureURLs(c *contentJSON, pointID string, keys map[string]string, admin bool) {
	for _, list := range [][]exerciseJSON{c.Practice, c.Mastery} {
		for i := range list {
			if k := keys[list[i].ID]; k != "" {
				list[i].ImageURL = imageURL(pointID, k, admin)
			}
		}
	}
}

// learnerContentJSON is the content of a learner's lesson with the pictures of its picture exercises.
func learnerContentJSON(d Detail) contentJSON {
	c := toContentJSON(d.Content)
	setPictureURLs(&c, d.Point.ID, d.Pictures, false)
	return c
}
