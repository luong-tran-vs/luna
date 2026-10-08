package lesson

import (
	"errors"
	"fmt"
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
	Result lookupJSON    `json:"result"`
	Cached bool          `json:"cached"`
	Quota  askQuotaJSON `json:"quota"`
}

// askQuotaJSON is the learner's quota of AI asks in a lesson; limit 0 means no limit (admins).
type askQuotaJSON struct {
	Limit int      `json:"limit"`
	Asked []string `json:"asked"`
}

func toAskQuotaJSON(q AskQuota) askQuotaJSON {
	asked := q.Asked
	if asked == nil {
		asked = []string{}
	}
	return askQuotaJSON{Limit: q.Limit, Asked: asked}
}

// ask explains a word or phrase of one sentence with the AI (F9).
func (h *ReadingHandler) ask(w http.ResponseWriter, r *http.Request) {
	var in askInputJSON
	if httpx.DecodeJSON(w, r, &in) != nil {
		return
	}
	p, _ := httpx.PrincipalFrom(r.Context())
	res, cached, quota, err := h.reader.AskAs(r.Context(), p.UserID, r.PathValue("id"), in.Text, in.SentenceIndex, p.Role == httpx.RoleAdmin)
	switch {
	case err == nil:
		httpx.WriteJSON(w, http.StatusOK, askJSON{Result: toLookupJSON(res), Cached: cached, Quota: toAskQuotaJSON(quota)})
	case errors.Is(err, ErrAskLimit):
		httpx.WriteJSON(w, http.StatusTooManyRequests, map[string]any{
			"error": "ask_limit", "quota": toAskQuotaJSON(quota),
			"message": fmt.Sprintf("Mỗi bài chỉ được hỏi AI %d từ. Bạn vẫn hỏi lại được các từ đã hỏi.", AskLimit),
		})
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
