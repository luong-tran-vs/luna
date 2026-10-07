package lesson

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/luongtran/luna/backend/internal/ai"
)

const (
	generatePath = "/api/admin/topics/topic-a1/generate"
	generateBody = `{"level":"A1","count":2,"words":100,"kind":"reading","idea":""}`
)

func TestGenerateEndpoint(t *testing.T) {
	t.Parallel()
	a := newAPI(t)
	a.env.ai.drafts = []ai.LessonDraft{{Title: "Sunday Lunch", Content: text(100)}, {Title: "", Content: text(100)}}

	rec := a.do(t, http.MethodPost, generatePath, "admin", generateBody)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d %s", rec.Code, rec.Body)
	}
	body := decodeBody(t, rec)
	reasons, _ := body["dropReasons"].(map[string]any)
	if body["requested"] != float64(2) || body["dropped"] != float64(1) || reasons["empty"] != float64(1) || reasons["duplicateTitle"] != float64(0) {
		t.Errorf("body = %v", body)
	}
	drafts := body["drafts"].([]any)
	if len(drafts) != 1 {
		t.Fatalf("drafts = %v", drafts)
	}
	d := drafts[0].(map[string]any)
	if d["title"] != "Sunday Lunch" || d["content"] != text(100) || d["words"] != float64(100) {
		t.Errorf("draft = %v", d)
	}
}

func TestGenerateEndpointPermissions(t *testing.T) {
	t.Parallel()
	a := newAPI(t)
	if rec := a.do(t, http.MethodPost, generatePath, "learner", generateBody); rec.Code != http.StatusForbidden {
		t.Errorf("learner: %d", rec.Code)
	}
	if rec := a.do(t, http.MethodPost, generatePath, "", generateBody); rec.Code != http.StatusUnauthorized {
		t.Errorf("anonymous: %d", rec.Code)
	}
	if _, calls := a.env.ai.lastRequest(); calls != 0 {
		t.Errorf("AI calls = %d, want 0", calls)
	}
}

func TestGenerateEndpointBadInput(t *testing.T) {
	t.Parallel()
	a := newAPI(t)
	for _, body := range []string{
		`{"level":"A1","count":0,"words":100,"kind":"reading"}`,
		`{"level":"A1","count":11,"words":100,"kind":"reading"}`,
	} {
		rec := a.do(t, http.MethodPost, generatePath, "admin", body)
		fields, _ := decodeBody(t, rec)["fields"].(map[string]any)
		if rec.Code != http.StatusBadRequest || fields["count"] != "Số bài từ 1 đến 10" {
			t.Errorf("%s: %d %s", body, rec.Code, rec.Body)
		}
	}
	if rec := a.do(t, http.MethodPost, generatePath, "admin", `{"level":"A1","count":`); rec.Code != http.StatusBadRequest ||
		decodeBody(t, rec)["error"] != "invalid_body" {
		t.Errorf("broken JSON: %d %s", rec.Code, rec.Body)
	}
	rec := a.do(t, http.MethodPost, "/api/admin/topics/nope/generate", "admin", generateBody)
	if rec.Code != http.StatusNotFound || decodeBody(t, rec)["error"] != "not_found" {
		t.Errorf("unknown topic: %d %s", rec.Code, rec.Body)
	}
}

func TestGenerateEndpointAIErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		err     error
		drafts  []ai.LessonDraft
		block   bool
		status  int
		code    string
		message string
	}{
		{
			name: "not configured", err: ai.ErrNotConfigured, status: http.StatusServiceUnavailable, code: "ai_not_configured",
			message: "AI chưa được cấu hình. Liên hệ người vận hành.",
		},
		{
			name: "invalid key", err: ai.ErrInvalidKey, status: http.StatusServiceUnavailable, code: "ai_not_configured",
			message: "Khoá AI không hợp lệ. Liên hệ người vận hành.",
		},
		{
			name: "quota", err: ai.ErrQuota, status: http.StatusTooManyRequests, code: "ai_quota",
			message: "Đã hết lượt AI, vui lòng thử lại sau.",
		},
		{
			name: "unusable", drafts: []ai.LessonDraft{{Title: "", Content: "no title"}}, status: http.StatusBadGateway,
			code: "ai_unusable", message: "AI trả về nội dung không dùng được, vui lòng thử lại.",
		},
		{
			name: "other", err: errors.New("boom"), status: http.StatusBadGateway, code: "ai_failed",
			message: "Sinh bài thất bại, vui lòng thử lại.",
		},
		{
			name: "deadline", block: true, status: http.StatusBadGateway, code: "ai_failed",
			message: "Sinh bài thất bại, vui lòng thử lại.",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := newAPI(t)
			a.env.svc.GenerateTimeout = 20 * time.Millisecond
			a.env.ai.genErr, a.env.ai.drafts, a.env.ai.genBlock = tt.err, tt.drafts, tt.block
			rec := a.do(t, http.MethodPost, generatePath, "admin", generateBody)
			body := decodeBody(t, rec)
			if rec.Code != tt.status || body["error"] != tt.code || body["message"] != tt.message {
				t.Fatalf("got %d %v, want %d %s %q", rec.Code, body, tt.status, tt.code, tt.message)
			}
		})
	}
}

// Cancelling the request (client gone) is also reported, not a 500.
func TestGenerateEndpointClientGone(t *testing.T) {
	t.Parallel()
	a := newAPI(t)
	a.env.ai.genErr = context.Canceled
	if rec := a.do(t, http.MethodPost, generatePath, "admin", generateBody); rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d", rec.Code)
	}
}
