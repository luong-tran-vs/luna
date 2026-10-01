package lesson

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/luongtran/luna/backend/internal/ai"
	"github.com/luongtran/luna/backend/internal/platform/httpx"
)

type askInputJSON struct {
	Text          string `json:"text"`
	SentenceIndex int    `json:"sentenceIndex"`
}

type askJSON struct {
	Result lookupJSON `json:"result"`
	Cached bool       `json:"cached"`
}

// ask explains a word or phrase of one sentence with the AI (F9).
func (h *ReadingHandler) ask(w http.ResponseWriter, r *http.Request) {
	var in askInputJSON
	if httpx.DecodeJSON(w, r, &in) != nil {
		return
	}
	res, cached, err := h.reader.Ask(r.Context(), r.PathValue("id"), in.Text, in.SentenceIndex)
	switch {
	case err == nil:
		httpx.WriteJSON(w, http.StatusOK, askJSON{Result: toLookupJSON(res), Cached: cached})
	case errors.Is(err, ai.ErrNotConfigured):
		httpx.WriteError(w, http.StatusServiceUnavailable, "ai_not_configured", "AI chưa được cấu hình. Liên hệ người vận hành.")
	case errors.Is(err, ai.ErrInvalidKey):
		httpx.WriteError(w, http.StatusServiceUnavailable, "ai_not_configured", "Khoá AI không hợp lệ. Liên hệ người vận hành.")
	case errors.Is(err, ai.ErrQuota):
		httpx.WriteError(w, http.StatusTooManyRequests, "ai_quota", "Đã hết lượt AI, vui lòng thử lại sau.")
	case errors.Is(err, errAskAI), errors.Is(err, ErrUnusableExplanation):
		h.log.WarnContext(r.Context(), "ask AI failed", slog.Any("error", err))
		httpx.WriteError(w, http.StatusBadGateway, "ai_failed", "Không hỏi được AI, vui lòng thử lại.")
	default:
		h.writeError(w, r, err, "not_found")
	}
}
