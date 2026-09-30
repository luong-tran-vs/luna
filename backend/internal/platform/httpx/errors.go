package httpx

import "net/http"

// ErrorBody is the JSON shape of every API error.
type ErrorBody struct {
	Error   string            `json:"error"`
	Message string            `json:"message,omitempty"`
	Fields  map[string]string `json:"fields,omitempty"`
}

// WriteError writes {"error": code, "message": message}.
func WriteError(w http.ResponseWriter, status int, code, message string) {
	WriteJSON(w, status, ErrorBody{Error: code, Message: message})
}

// WriteFieldErrors writes a 400 validation error listing invalid fields.
func WriteFieldErrors(w http.ResponseWriter, fields map[string]string) {
	WriteJSON(w, http.StatusBadRequest, ErrorBody{
		Error:   "validation_failed",
		Message: "Thông tin chưa hợp lệ",
		Fields:  fields,
	})
}
