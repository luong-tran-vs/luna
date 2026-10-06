package lesson

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/url"

	"github.com/luongtran/luna/backend/internal/platform/httpx"
)

type imageStateJSON struct {
	Enabled bool   `json:"enabled"`
	Style   string `json:"style"`
	// Status is "" (off), "running", "done" or "failed".
	Status string `json:"status"`
	Error  string `json:"error"`
	// Count is how many words have a picture.
	Count int `json:"count"`
	// Words are the lesson's vocabulary words, empty until annotated.
	Words []imageWordJSON `json:"words"`
}

type imageWordJSON struct {
	Lemma     string `json:"lemma"`
	MeaningVi string `json:"meaningVi"`
	// ImageURL is the admin URL of the word's picture, "" when it has none.
	ImageURL string `json:"imageUrl"`
}

func toImageStateJSON(lessonID string, s ImageState) imageStateJSON {
	out := imageStateJSON{
		Enabled: s.Enabled, Style: s.Style, Status: string(s.Status), Error: s.Error, Count: s.Count,
		Words: make([]imageWordJSON, len(s.Words)),
	}
	for i, w := range s.Words {
		out.Words[i] = imageWordJSON{Lemma: w.Lemma, MeaningVi: w.MeaningVi}
		if w.HasImage {
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
	httpx.WriteJSON(w, http.StatusOK, toImageStateJSON(id, st))
}

// setImages serves PUT /api/admin/lessons/{id}/images: turns the pictures on (drawing the words
// that have none) or off.
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
	httpx.WriteJSON(w, http.StatusOK, toImageStateJSON(id, st))
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
	httpx.WriteJSON(w, http.StatusOK, toImageStateJSON(id, st))
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
