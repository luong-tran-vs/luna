package grammar

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/luongtran/luna/backend/internal/ai"
	"github.com/luongtran/luna/backend/internal/platform/httpx"
)

// registerPictures adds the routes that add picture exercises to lessons (added 2026-11-06).
func (h *Handler) registerPictures(mux *http.ServeMux, admin func(http.HandlerFunc) http.Handler) {
	mux.Handle("POST /api/admin/grammar-lessons/{pointId}/pictures", admin(h.addPictures))
	mux.Handle("GET /api/admin/grammar-picture-batch", admin(h.pictureBatch))
	mux.Handle("POST /api/admin/grammar-picture-batch", admin(h.startPictureBatch))
	mux.Handle("POST /api/admin/grammar-picture-batch/stop", admin(h.stopPictureBatch))
}

// addPictures serves POST /api/admin/grammar-lessons/{pointId}/pictures: like generate it waits for
// the AI, then answers with the lesson and "added", how many exercises it got.
func (h *Handler) addPictures(w http.ResponseWriter, r *http.Request) {
	if err := http.NewResponseController(w).SetWriteDeadline(time.Now().Add(generateWriteTimeout)); err != nil &&
		!errors.Is(err, http.ErrNotSupported) {
		h.log.WarnContext(r.Context(), "grammar pictures: extend write deadline", slog.Any("error", err))
	}
	l, _, err := h.svc.AddPictures(r.Context(), r.PathValue("pointId"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeLesson(w, r, http.StatusOK, l)
}

type pictureBatchItemJSON struct {
	PointID string `json:"pointId"`
	Title   string `json:"title"`
	Added   int    `json:"added"`
	// Message says why the lesson got no exercise; "" for one that got them.
	Message string `json:"message,omitempty"`
}

// pictureBatchJSON is how the batch adding picture exercises is going; done counts the lessons dealt with.
type pictureBatchJSON struct {
	Running     bool                   `json:"running"`
	Stopping    bool                   `json:"stopping"`
	Stopped     bool                   `json:"stopped"`
	Key         string                 `json:"key"`
	Label       string                 `json:"label"`
	Total       int                    `json:"total"`
	Done        int                    `json:"done"`
	Current     *pictureBatchItemJSON  `json:"current"`
	Saved       []pictureBatchItemJSON `json:"saved"`
	Failures    []pictureBatchItemJSON `json:"failures"`
	StopMessage string                 `json:"stopMessage,omitempty"`
	StartedAt   *time.Time             `json:"startedAt,omitempty"`
	FinishedAt  *time.Time             `json:"finishedAt,omitempty"`
}

func pictureItemsJSON(in []PictureBatchItem) []pictureBatchItemJSON {
	out := make([]pictureBatchItemJSON, len(in))
	for i, it := range in {
		out[i] = pictureBatchItemJSON{PointID: it.PointID, Title: it.Title, Added: it.Added}
		if it.Err != nil {
			out[i].Message = pictureFailMessage(it.Err)
		}
	}
	return out
}

func toPictureBatchJSON(b PictureBatchStatus) pictureBatchJSON {
	out := pictureBatchJSON{
		Running: b.Running, Stopping: b.Stopping, Stopped: b.Stopped, Key: b.Key, Label: b.Label,
		Total: b.Total, Done: b.Done(), Saved: pictureItemsJSON(b.Saved), Failures: pictureItemsJSON(b.Failures),
	}
	if b.Current != nil {
		out.Current = &pictureBatchItemJSON{PointID: b.Current.PointID, Title: b.Current.Title}
	}
	if b.StopErr != nil {
		out.StopMessage = pictureFailMessage(b.StopErr)
	}
	if !b.StartedAt.IsZero() {
		out.StartedAt = &b.StartedAt
	}
	if !b.FinishedAt.IsZero() {
		out.FinishedAt = &b.FinishedAt
	}
	return out
}

// pictureBatch serves GET /api/admin/grammar-picture-batch.
func (h *Handler) pictureBatch(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, toPictureBatchJSON(h.svc.PictureBatch()))
}

// startPictureBatch serves POST /api/admin/grammar-picture-batch with {"pointIds", "key"?, "label"?}:
// 202 once the batch runs in the background; 409 while another one runs.
func (h *Handler) startPictureBatch(w http.ResponseWriter, r *http.Request) {
	var in struct {
		PointIDs []string `json:"pointIds"`
		Key      string   `json:"key"`
		Label    string   `json:"label"`
	}
	if httpx.DecodeJSON(w, r, &in) != nil {
		return
	}
	st, err := h.svc.StartPictureBatch(r.Context(), PictureBatchRequest{PointIDs: in.PointIDs, Key: in.Key, Label: in.Label})
	if errors.Is(err, ErrPictureBatchRunning) {
		httpx.WriteError(w, http.StatusConflict, "batch_running", "Đang có một lượt thêm bài nhìn ảnh chạy. Chờ xong hoặc bấm Dừng.")
		return
	}
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusAccepted, toPictureBatchJSON(st))
}

// stopPictureBatch serves POST /api/admin/grammar-picture-batch/stop.
func (h *Handler) stopPictureBatch(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, toPictureBatchJSON(h.svc.StopPictureBatch()))
}

// pictureFailMessage is the Vietnamese reason a lesson of a batch got no picture exercise.
func pictureFailMessage(err error) string {
	var verr *ValidationError
	switch {
	case errors.As(err, &verr):
		for _, m := range verr.Fields {
			return m
		}
	case errors.Is(err, ai.ErrNotConfigured):
		return "AI chưa được cấu hình. Liên hệ người vận hành."
	case errors.Is(err, ai.ErrInvalidKey):
		return "Khoá AI không hợp lệ. Liên hệ người vận hành."
	case errors.Is(err, ai.ErrQuota):
		return "Đã hết lượt AI, vui lòng thử lại sau."
	case errors.Is(err, ErrUnusable):
		return "AI trả về nội dung không dùng được, vui lòng thử lại."
	case errors.Is(err, ErrNotFound):
		return "Không tìm thấy bài ngữ pháp"
	}
	return "Thêm bài nhìn ảnh thất bại, vui lòng thử lại."
}
