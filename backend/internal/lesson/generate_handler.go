package lesson

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/luongtran/luna/backend/internal/ai"
	"github.com/luongtran/luna/backend/internal/platform/httpx"
)

// generateWriteTimeout replaces the server's write timeout for the generate route, which
// waits up to generateTimeout for the AI.
const generateWriteTimeout = 135 * time.Second

// --- JSON shapes (contracts/generate-api.md) ---

type generateJSON struct {
	Count       int        `json:"count"`
	Words       int        `json:"words"`
	Kind        string     `json:"kind"`
	Idea        string     `json:"idea"`
	TargetWords [][]string `json:"targetWords"`
}

type draftJSON struct {
	Title        string   `json:"title"`
	Content      string   `json:"content"`
	Words        int      `json:"words"`
	TargetWords  []string `json:"targetWords"`
	MissingWords []string `json:"missingWords"`
}

type generateResultJSON struct {
	Drafts      []draftJSON     `json:"drafts"`
	Requested   int             `json:"requested"`
	Dropped     int             `json:"dropped"`
	DropReasons dropReasonsJSON `json:"dropReasons"`
}

type dropReasonsJSON struct {
	DuplicateTitle int `json:"duplicateTitle"`
	Empty          int `json:"empty"`
	TooLong        int `json:"tooLong"`
}

func (h *Handler) generate(w http.ResponseWriter, r *http.Request) {
	if err := http.NewResponseController(w).SetWriteDeadline(time.Now().Add(generateWriteTimeout)); err != nil &&
		!errors.Is(err, http.ErrNotSupported) {
		h.log.WarnContext(r.Context(), "generate: extend write deadline", slog.Any("error", err))
	}
	var in generateJSON
	if httpx.DecodeJSON(w, r, &in) != nil {
		return
	}
	res, err := h.svc.Generate(r.Context(), r.PathValue("id"), GenerateInput(in))
	if err != nil {
		h.writeGenerateError(w, r, err)
		return
	}
	out := generateResultJSON{
		Drafts: make([]draftJSON, len(res.Drafts)), Requested: res.Requested, Dropped: res.Dropped,
		DropReasons: dropReasonsJSON(res.DropReasons),
	}
	for i, d := range res.Drafts {
		out.Drafts[i] = draftJSON{
			Title: d.Title, Content: d.Content, Words: d.Words, TargetWords: d.TargetWords, MissingWords: d.MissingWords,
		}
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) writeGenerateError(w http.ResponseWriter, r *http.Request, err error) {
	var verr *ValidationError
	switch {
	case errors.As(err, &verr):
		httpx.WriteFieldErrors(w, verr.Fields)
	case errors.Is(err, ErrTopicNotFound):
		httpx.WriteError(w, http.StatusNotFound, "not_found", "Không tìm thấy chủ đề")
	case errors.Is(err, ai.ErrNotConfigured):
		httpx.WriteError(w, http.StatusServiceUnavailable, "ai_not_configured", "AI chưa được cấu hình. Liên hệ người vận hành.")
	case errors.Is(err, ai.ErrInvalidKey):
		httpx.WriteError(w, http.StatusServiceUnavailable, "ai_not_configured", "Khoá AI không hợp lệ. Liên hệ người vận hành.")
	case errors.Is(err, ai.ErrQuota):
		httpx.WriteError(w, http.StatusTooManyRequests, "ai_quota", "Đã hết lượt AI, vui lòng thử lại sau.")
	case errors.Is(err, ErrUnusableDraft):
		httpx.WriteError(w, http.StatusBadGateway, "ai_unusable", "AI trả về nội dung không dùng được, vui lòng thử lại.")
	case errors.Is(err, errGenerateAI):
		h.log.WarnContext(r.Context(), "lesson generation failed", slog.Any("error", err))
		httpx.WriteError(w, http.StatusBadGateway, "ai_failed", "Sinh bài thất bại, vui lòng thử lại.")
	default:
		h.writeError(w, r, err)
	}
}
