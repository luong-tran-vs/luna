// Package httpx holds HTTP helpers and middleware shared by every domain.
package httpx

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
)

// maxBodyBytes bounds JSON request bodies.
const maxBodyBytes = 16 << 10

// ErrBadRequest means DecodeJSON already wrote an error response; the handler should just return.
var ErrBadRequest = errors.New("httpx: bad request body")

// WriteJSON writes v as a JSON response with the given status code.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	// Headers are already sent, so an encode error cannot be reported to the client.
	_ = json.NewEncoder(w).Encode(v)
}

// DecodeJSON reads a JSON object into dst. It requires Content-Type application/json
// (415 otherwise), limits the body to 16 KiB and rejects unknown fields (400 invalid_body).
// On failure it writes the response and returns ErrBadRequest.
func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	return DecodeJSONLimit(w, r, dst, maxBodyBytes)
}

// DecodeJSONLimit is DecodeJSON with a body of at most limit bytes, for the few larger bodies.
func DecodeJSONLimit(w http.ResponseWriter, r *http.Request, dst any, limit int64) error {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		WriteError(w, http.StatusUnsupportedMediaType, "unsupported_media_type", "Yêu cầu phải là JSON")
		return ErrBadRequest
	}

	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, limit))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_body", "Dữ liệu gửi lên không hợp lệ")
		return ErrBadRequest
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		WriteError(w, http.StatusBadRequest, "invalid_body", "Dữ liệu gửi lên không hợp lệ")
		return ErrBadRequest
	}
	return nil
}

// DecodeJSONDetailed is DecodeJSONLimit for the bodies written by hand or by an agent (the lesson
// imports, added 2026-10-26): a body that cannot be read gets 400 invalid_body with the reason
// (`json: unknown field "x"`) in fields["body"], so the writer can fix it.
func DecodeJSONDetailed(w http.ResponseWriter, r *http.Request, dst any, limit int64) error {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		WriteError(w, http.StatusUnsupportedMediaType, "unsupported_media_type", "Yêu cầu phải là JSON")
		return ErrBadRequest
	}
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, limit))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		WriteJSON(w, http.StatusBadRequest, ErrorBody{
			Error: "invalid_body", Message: "Dữ liệu gửi lên không hợp lệ", Fields: map[string]string{"body": err.Error()},
		})
		return ErrBadRequest
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		WriteJSON(w, http.StatusBadRequest, ErrorBody{
			Error: "invalid_body", Message: "Dữ liệu gửi lên không hợp lệ", Fields: map[string]string{"body": "chỉ được gửi một đối tượng JSON"},
		})
		return ErrBadRequest
	}
	return nil
}
