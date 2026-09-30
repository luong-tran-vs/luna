package httpx_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/luongtran/luna/backend/internal/platform/httpx"
)

func TestDecodeJSON(t *testing.T) {
	t.Parallel()

	type payload struct {
		Email string `json:"email"`
	}

	tests := []struct {
		name        string
		contentType string
		body        string
		wantErr     bool
		wantStatus  int
		wantCode    string
	}{
		{name: "valid", contentType: "application/json", body: `{"email":"a@b.vn"}`},
		{name: "charset param ok", contentType: "application/json; charset=utf-8", body: `{"email":"a@b.vn"}`},
		{name: "wrong content type", contentType: "text/plain", body: `{"email":"a@b.vn"}`, wantErr: true, wantStatus: 415, wantCode: "unsupported_media_type"},
		{name: "missing content type", body: `{}`, wantErr: true, wantStatus: 415, wantCode: "unsupported_media_type"},
		{name: "unknown field", contentType: "application/json", body: `{"email":"a","admin":true}`, wantErr: true, wantStatus: 400, wantCode: "invalid_body"},
		{name: "malformed", contentType: "application/json", body: `{"email":`, wantErr: true, wantStatus: 400, wantCode: "invalid_body"},
		{name: "trailing data", contentType: "application/json", body: `{"email":"a"}{}`, wantErr: true, wantStatus: 400, wantCode: "invalid_body"},
		{name: "too large", contentType: "application/json", body: `{"email":"` + strings.Repeat("a", 20<<10) + `"}`, wantErr: true, wantStatus: 400, wantCode: "invalid_body"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", strings.NewReader(tt.body))
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}
			rec := httptest.NewRecorder()

			var dst payload
			err := httpx.DecodeJSON(rec, req, &dst)

			if !tt.wantErr {
				if err != nil {
					t.Fatalf("DecodeJSON error = %v", err)
				}
				if dst.Email != "a@b.vn" {
					t.Fatalf("decoded %+v", dst)
				}
				return
			}
			if !errors.Is(err, httpx.ErrBadRequest) {
				t.Fatalf("error = %v, want ErrBadRequest", err)
			}
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if !strings.Contains(rec.Body.String(), `"error":"`+tt.wantCode+`"`) {
				t.Fatalf("body = %s, want code %s", rec.Body.String(), tt.wantCode)
			}
		})
	}
}
