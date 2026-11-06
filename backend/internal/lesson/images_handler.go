package lesson

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/luongtran/luna/backend/internal/ai"
	"github.com/luongtran/luna/backend/internal/platform/httpx"
)

type imageStateJSON struct {
	// Style and SceneDescription are how the AI draws the scene picture; SceneImageURL is the admin
	// URL of the picture, "" when it has none.
	Style            string `json:"style"`
	SceneDescription string `json:"sceneDescription"`
	SceneImageURL    string `json:"sceneImageUrl"`
	// CodexImages tells the page it may offer drawing with the Codex CLI (added 2026-11-06).
	CodexImages bool `json:"codexImages"`
	// Count is how many words have a picture of their own, BankCount how many show the word bank's.
	Count     int `json:"count"`
	BankCount int `json:"bankCount"`
	// Words are the lesson's vocabulary words, empty until annotated.
	Words []imageWordJSON `json:"words"`
}

type imageWordJSON struct {
	Lemma     string `json:"lemma"`
	MeaningVi string `json:"meaningVi"`
	// ImageURL is the admin URL of the word's picture, "" when it has none.
	ImageURL string `json:"imageUrl"`
	// FromBank says the picture is the word bank's (F24): the word has none of its own.
	FromBank bool `json:"fromBank"`
}

type imageInputJSON struct {
	SceneDescription string `json:"sceneDescription"`
	Style            string `json:"style"`
}

func (h *Handler) toImageStateJSON(lessonID string, s ImageState) imageStateJSON {
	out := imageStateJSON{
		Style: s.Style, SceneDescription: s.SceneDescription, CodexImages: h.svc.CodexAvailable(),
		Count: s.Count, BankCount: s.BankCount, Words: make([]imageWordJSON, len(s.Words)),
	}
	if s.HasScene {
		out.SceneImageURL = adminImageURL(lessonID, SceneKey)
	}
	for i, w := range s.Words {
		out.Words[i] = imageWordJSON{Lemma: w.Lemma, MeaningVi: w.MeaningVi, FromBank: w.FromBank}
		if w.HasImage || w.FromBank {
			out.Words[i].ImageURL = adminImageURL(lessonID, w.Lemma)
		}
	}
	return out
}

// images serves GET /api/admin/lessons/{id}/images: the picture settings of a lesson and its
// words (F23).
func (h *Handler) images(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	st, err := h.svc.Images(r.Context(), id)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, h.toImageStateJSON(id, st))
}

// setImages serves PUT /api/admin/lessons/{id}/images with {"sceneDescription", "style"}: saves
// how the scene picture is drawn, without drawing it.
func (h *Handler) setImages(w http.ResponseWriter, r *http.Request) {
	var in imageInputJSON
	if httpx.DecodeJSON(w, r, &in) != nil {
		return
	}
	id := r.PathValue("id")
	st, err := h.svc.SetImages(r.Context(), id, ImageInput(in))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, h.toImageStateJSON(id, st))
}

// generateScene serves POST /api/admin/lessons/{id}/images/scene/generate with {"sceneDescription",
// "style", "engine"}: saves the settings, then draws the scene picture with engine ("gemini", the
// default, or "codex") and answers once it is saved, replacing the old one (added 2026-11-06).
func (h *Handler) generateScene(w http.ResponseWriter, r *http.Request) {
	var in struct {
		imageInputJSON
		Engine ImageEngine `json:"engine"`
	}
	if httpx.DecodeJSON(w, r, &in) != nil {
		return
	}
	id := r.PathValue("id")
	// The server writes within 15 seconds; a picture takes longer (Codex about a minute).
	if err := http.NewResponseController(w).SetWriteDeadline(time.Now().Add(sceneBatchTimeout + 30*time.Second)); err != nil &&
		!errors.Is(err, http.ErrNotSupported) {
		h.log.WarnContext(r.Context(), "lesson: extend write deadline", slog.Any("error", err))
	}
	st, err := h.svc.DrawScene(r.Context(), id, ImageInput(in.imageInputJSON), in.Engine)
	if err != nil {
		if status, code, msg, ok := drawError(err); ok {
			httpx.WriteError(w, status, code, msg)
			return
		}
		h.writeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, h.toImageStateJSON(id, st))
}

// drawError is the HTTP status, code and Vietnamese message of a failed drawing, with the word
// bank's wording; ok is false for an error writeError handles.
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

// uploadImage serves PUT /api/admin/lessons/{id}/images/{lemma}: the body is the picture itself
// (JPEG, PNG or GIF), at most MaxUploadBytes. It answers with the lesson's picture state.
func (h *Handler) uploadImage(w http.ResponseWriter, r *http.Request) {
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
	id := r.PathValue("id")
	if err := h.svc.UploadImage(r.Context(), id, r.PathValue("lemma"), data); err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeImageState(w, r, id)
}

// importImage serves POST /api/admin/lessons/{id}/images/{lemma}/import with {"url": "..."}: the
// server downloads the picture behind the link and stores it like an uploaded one.
func (h *Handler) importImage(w http.ResponseWriter, r *http.Request) {
	var in struct {
		URL string `json:"url"`
	}
	if httpx.DecodeJSON(w, r, &in) != nil {
		return
	}
	id := r.PathValue("id")
	if err := h.svc.ImportImage(r.Context(), id, r.PathValue("lemma"), in.URL); err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeImageState(w, r, id)
}

// deleteImage serves DELETE /api/admin/lessons/{id}/images/{lemma}.
func (h *Handler) deleteImage(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.svc.DeleteImage(r.Context(), id, r.PathValue("lemma")); err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeImageState(w, r, id)
}

// adminImage serves GET /api/admin/lessons/{id}/images/{lemma}: one word's picture, for the admin
// page (the learner route is guarded by the study flow).
func (h *Handler) adminImage(w http.ResponseWriter, r *http.Request) {
	img, err := h.svc.AdminImage(r.Context(), r.PathValue("id"), r.PathValue("lemma"))
	if errors.Is(err, ErrImageNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "Từ này chưa có ảnh")
		return
	}
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	serveImage(w, r, img)
}

func (h *Handler) writeImageState(w http.ResponseWriter, r *http.Request, id string) {
	st, err := h.svc.Images(r.Context(), id)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, h.toImageStateJSON(id, st))
}

// image serves GET /api/lessons/{id}/images/{lemma}: the picture of one word.
func (h *ReadingHandler) image(w http.ResponseWriter, r *http.Request) {
	img, err := h.reader.Image(r.Context(), r.PathValue("id"), r.PathValue("lemma"))
	if errors.Is(err, ErrImageNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "Từ này chưa có ảnh")
		return
	}
	if err != nil {
		h.writeError(w, r, err, "not_found")
		return
	}
	serveImage(w, r, img)
}

// serveImage writes a picture. An admin can replace a picture under the same URL, so the browser
// keeps it but asks again each time (If-Modified-Since answered with 304 while unchanged).
func serveImage(w http.ResponseWriter, r *http.Request, img WordImage) {
	w.Header().Set("Content-Type", img.MIME)
	w.Header().Set("Cache-Control", "private, no-cache")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeContent(w, r, "", img.CreatedAt, bytes.NewReader(img.Data))
}

// imageURL is where learners get the picture of a word.
func imageURL(lessonID, lemma string) string {
	return "/api/lessons/" + url.PathEscape(lessonID) + "/images/" + url.PathEscape(lemma)
}

// adminImageURL is where admins get the picture of a word.
func adminImageURL(lessonID, lemma string) string {
	return "/api/admin/lessons/" + url.PathEscape(lessonID) + "/images/" + url.PathEscape(lemma)
}

type sceneBatchLessonJSON struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	// Message says why the lesson got no picture; "" for one that got it.
	Message string `json:"message,omitempty"`
}

// sceneBatchJSON is how the scene picture batch is going (added 2026-11-06). done counts the
// lessons dealt with (saved or failed); skipped ones were never drawn.
type sceneBatchJSON struct {
	Running     bool                   `json:"running"`
	Stopping    bool                   `json:"stopping"`
	Stopped     bool                   `json:"stopped"`
	Engine      ImageEngine            `json:"engine"`
	Key         string                 `json:"key"`
	Label       string                 `json:"label"`
	Total       int                    `json:"total"`
	Done        int                    `json:"done"`
	Drawing     *sceneBatchLessonJSON  `json:"drawing"`
	Saved       []sceneBatchLessonJSON `json:"saved"`
	Failures    []sceneBatchLessonJSON `json:"failures"`
	Skipped     []sceneBatchLessonJSON `json:"skipped"`
	StopMessage string                 `json:"stopMessage,omitempty"`
	// CodexImages tells the page it may offer drawing with the Codex CLI.
	CodexImages bool       `json:"codexImages"`
	StartedAt   *time.Time `json:"startedAt,omitempty"`
	FinishedAt  *time.Time `json:"finishedAt,omitempty"`
}

func batchLessons(in []SceneBatchLesson) []sceneBatchLessonJSON {
	out := make([]sceneBatchLessonJSON, len(in))
	for i, l := range in {
		out[i] = sceneBatchLessonJSON{ID: l.ID, Title: l.Title}
		if l.Err != nil {
			out[i].Message = batchMessage(l.Err)
		}
	}
	return out
}

// batchMessage is the Vietnamese reason a lesson of a batch got no picture.
func batchMessage(err error) string {
	if errors.Is(err, ErrAnnotationNotDone) {
		return "Bài chưa chú thích xong"
	}
	if _, _, msg, ok := drawError(err); ok {
		return msg
	}
	return "Sinh ảnh thất bại, vui lòng thử lại."
}

func (h *Handler) sceneBatchJSON(b SceneBatchStatus) sceneBatchJSON {
	out := sceneBatchJSON{
		Running: b.Running, Stopping: b.Stopping, Stopped: b.Stopped, Engine: b.Engine, Key: b.Key, Label: b.Label,
		Total: b.Total, Done: b.Done(), Saved: batchLessons(b.Saved), Failures: batchLessons(b.Failures),
		Skipped: batchLessons(b.Skipped), CodexImages: h.svc.CodexAvailable(),
	}
	if b.Drawing != nil {
		out.Drawing = &sceneBatchLessonJSON{ID: b.Drawing.ID, Title: b.Drawing.Title}
	}
	if b.StopErr != nil {
		out.StopMessage = batchMessage(b.StopErr)
	}
	if !b.StartedAt.IsZero() {
		out.StartedAt = &b.StartedAt
	}
	if !b.FinishedAt.IsZero() {
		out.FinishedAt = &b.FinishedAt
	}
	return out
}

// sceneBatch serves GET /api/admin/lessons/scene-batch: how the running, or last, batch is going.
func (h *Handler) sceneBatch(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, h.sceneBatchJSON(h.svc.SceneBatch()))
}

// startSceneBatch serves POST /api/admin/lessons/scene-batch with {"lessonIds", "replace"?, "engine"?, "key"?,
// "label"?}: 202 once the batch runs in the background; 409 while another one runs.
func (h *Handler) startSceneBatch(w http.ResponseWriter, r *http.Request) {
	var in struct {
		LessonIDs []string    `json:"lessonIds"`
		Replace   bool        `json:"replace"`
		Engine    ImageEngine `json:"engine"`
		Key       string      `json:"key"`
		Label     string      `json:"label"`
	}
	if httpx.DecodeJSON(w, r, &in) != nil {
		return
	}
	st, err := h.svc.StartSceneBatch(r.Context(), SceneBatchRequest{LessonIDs: in.LessonIDs, Replace: in.Replace, Engine: in.Engine, Key: in.Key, Label: in.Label})
	if errors.Is(err, ErrSceneBatchRunning) {
		httpx.WriteError(w, http.StatusConflict, "batch_running", "Đang có một lượt tạo ảnh cảnh chạy. Chờ xong hoặc bấm Dừng.")
		return
	}
	if err != nil {
		if status, code, msg, ok := drawError(err); ok {
			httpx.WriteError(w, status, code, msg)
			return
		}
		h.writeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusAccepted, h.sceneBatchJSON(st))
}

// stopSceneBatch serves POST /api/admin/lessons/scene-batch/stop.
func (h *Handler) stopSceneBatch(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, h.sceneBatchJSON(h.svc.StopSceneBatch()))
}
