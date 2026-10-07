package wordbank

import (
	"bytes"
	"errors"
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
const drawTimeout = 90 * time.Second

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
	mux.Handle("POST /api/admin/words", admin(h.add))
	mux.Handle("POST /api/admin/words/import", admin(h.importTopics))
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
}

func toJSON(w Word) wordJSON {
	out := wordJSON{Lemma: w.Lemma, MeaningVi: w.MeaningVi, IPA: w.IPA, UpdatedAt: w.UpdatedAt}
	if w.HasImage() {
		out.ImageURL = "/api/admin/words/" + url.PathEscape(w.Lemma) + "/image?v=" + strconv.FormatInt(w.ImageAt.UnixMilli(), 10)
	}
	return out
}

type pageJSON struct {
	Words   []wordJSON `json:"words"`
	Total   int        `json:"total"`
	HasMore bool       `json:"hasMore"`
}

// list serves GET /api/admin/words?q=&missing=image|ipa|meaning&page=1.
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
	p, err := h.svc.List(r.Context(), q.Get("q"), Missing(q.Get("missing")), page)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	out := pageJSON{Words: make([]wordJSON, len(p.Words)), Total: p.Total, HasMore: p.HasMore}
	for i, wd := range p.Words {
		out.Words[i] = toJSON(wd)
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

type inputJSON struct {
	Lemma     string `json:"lemma"`
	MeaningVi string `json:"meaningVi"`
	IPA       string `json:"ipa"`
}

// add serves POST /api/admin/words: 201 with the word, IPA and meaning filled from the dictionary
// when left empty.
func (h *Handler) add(w http.ResponseWriter, r *http.Request) {
	var in inputJSON
	if httpx.DecodeJSON(w, r, &in) != nil {
		return
	}
	wd, err := h.svc.Add(r.Context(), Input(in))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, toJSON(wd))
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

// generateImage serves POST /api/admin/words/{lemma}/image/generate with {"style"?}: one AI
// request, answered when the picture is saved.
func (h *Handler) generateImage(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Style string `json:"style"`
	}
	if httpx.DecodeJSON(w, r, &in) != nil {
		return
	}
	if err := http.NewResponseController(w).SetWriteDeadline(time.Now().Add(drawTimeout)); err != nil {
		h.log.WarnContext(r.Context(), "word bank: extend write deadline", slog.Any("error", err))
	}
	wd, err := h.svc.GenerateImage(r.Context(), r.PathValue("lemma"), in.Style)
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
	switch {
	case errors.As(err, &verr):
		httpx.WriteFieldErrors(w, verr.Fields)
	case errors.Is(err, ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "not_found", "Không tìm thấy từ này trong kho")
	case errors.Is(err, ErrImageNotFound):
		httpx.WriteError(w, http.StatusNotFound, "not_found", "Từ này chưa có ảnh")
	case errors.Is(err, ErrExists):
		httpx.WriteFieldErrors(w, map[string]string{"lemma": "Từ này đã có trong kho"})
	case errors.Is(err, ErrImagesUnavailable), errors.Is(err, ai.ErrNotConfigured):
		httpx.WriteError(w, http.StatusServiceUnavailable, "ai_not_configured", "AI chưa được cấu hình. Liên hệ người vận hành.")
	case errors.Is(err, ai.ErrInvalidKey):
		httpx.WriteError(w, http.StatusServiceUnavailable, "ai_not_configured", "Khoá AI không hợp lệ. Liên hệ người vận hành.")
	case errors.Is(err, ai.ErrQuota):
		httpx.WriteError(w, http.StatusTooManyRequests, "ai_quota", "Đã hết lượt AI, vui lòng thử lại sau.")
	case errors.Is(err, ErrDrawFailed):
		h.log.WarnContext(r.Context(), "word bank: draw picture", slog.Any("error", err))
		httpx.WriteError(w, http.StatusBadGateway, "ai_failed", "Sinh ảnh thất bại, vui lòng thử lại.")
	default:
		h.log.ErrorContext(r.Context(), "word bank request failed", slog.Any("error", err))
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "Có lỗi xảy ra, vui lòng thử lại")
	}
}
