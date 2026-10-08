package lesson

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/luongtran/luna/backend/internal/ai"
	"github.com/luongtran/luna/backend/internal/platform/httpx"
)

// --- JSON shapes of the AI check (F22) ---

type reviewJSON struct {
	CheckedAt  time.Time  `json:"checkedAt"`
	VerifiedAt *time.Time `json:"verifiedAt"`
	Flags      []flagJSON `json:"flags"`
}

type flagJSON struct {
	Area      string `json:"area"`
	Index     int    `json:"index"`
	Kind      string `json:"kind"`
	NoteVi    string `json:"noteVi"`
	Confirmed bool   `json:"confirmed"`
}

func toReviewJSON(r *Review) *reviewJSON {
	if r == nil {
		return nil
	}
	out := &reviewJSON{CheckedAt: r.CheckedAt, Flags: make([]flagJSON, len(r.Flags))}
	if !r.VerifiedAt.IsZero() {
		v := r.VerifiedAt
		out.VerifiedAt = &v
	}
	for i, f := range r.Flags {
		out.Flags[i] = flagJSON{Area: string(f.Area), Index: f.Index, Kind: string(f.Kind), NoteVi: f.NoteVi, Confirmed: f.Confirmed}
	}
	return out
}

func (h *Handler) check(w http.ResponseWriter, r *http.Request) {
	if err := http.NewResponseController(w).SetWriteDeadline(time.Now().Add(generateWriteTimeout)); err != nil &&
		!errors.Is(err, http.ErrNotSupported) {
		h.log.WarnContext(r.Context(), "lesson check: extend write deadline", slog.Any("error", err))
	}
	l, err := h.svc.Check(r.Context(), r.PathValue("id"))
	if err != nil {
		h.writeCheckError(w, r, err)
		return
	}
	h.writeLesson(w, r, http.StatusOK, l)
}

func (h *Handler) confirmFlag(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Area  string `json:"area"`
		Index int    `json:"index"`
	}
	if httpx.DecodeJSON(w, r, &body) != nil {
		return
	}
	l, err := h.svc.ConfirmFlag(r.Context(), r.PathValue("id"), FlagArea(body.Area), body.Index)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeLesson(w, r, http.StatusOK, l)
}

func (h *Handler) verify(w http.ResponseWriter, r *http.Request) {
	l, err := h.svc.Verify(r.Context(), r.PathValue("id"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeLesson(w, r, http.StatusOK, l)
}

// writeCheckError maps the errors of the AI call; the rest are the lesson errors.
func (h *Handler) writeCheckError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ai.ErrNotConfigured):
		httpx.WriteError(w, http.StatusServiceUnavailable, "ai_not_configured", "AI chưa được cấu hình. Liên hệ người vận hành.")
	case errors.Is(err, ai.ErrInvalidKey):
		httpx.WriteError(w, http.StatusServiceUnavailable, "ai_not_configured", "Khoá AI không hợp lệ. Liên hệ người vận hành.")
	case errors.Is(err, ai.ErrQuota):
		httpx.WriteError(w, http.StatusTooManyRequests, "ai_quota", "Đã hết lượt AI, vui lòng thử lại sau.")
	case errors.Is(err, errReviewAI):
		h.log.WarnContext(r.Context(), "lesson check failed", slog.Any("error", err))
		httpx.WriteError(w, http.StatusBadGateway, "ai_failed", "Kiểm tra bằng AI thất bại, vui lòng thử lại.")
	default:
		h.writeError(w, r, err)
	}
}

// writeFlagError handles the errors of confirming and verifying; it reports whether it did.
func writeFlagError(w http.ResponseWriter, err error) bool {
	var flags *FlagsError
	switch {
	case errors.As(err, &flags):
		httpx.WriteJSON(w, http.StatusConflict, map[string]any{
			"error": "flags_unresolved", "count": flags.Count,
			"message": fmt.Sprintf("Còn %d chỗ bị gắn cờ chưa xác nhận.", flags.Count),
		})
	case errors.Is(err, ErrNotChecked):
		httpx.WriteError(w, http.StatusConflict, "not_checked", "Hãy kiểm tra bằng AI trước khi xác nhận.")
	case errors.Is(err, ErrFlagNotFound):
		httpx.WriteError(w, http.StatusNotFound, "not_found", "Không tìm thấy chỗ bị gắn cờ")
	default:
		return false
	}
	return true
}

// --- AI help to fix a flagged item (F22) ---

type fixJSON struct {
	Area      string        `json:"area"`
	Index     int           `json:"index"`
	Text      string        `json:"text,omitempty"`
	MeaningVi string        `json:"meaningVi,omitempty"`
	Question  *questionJSON `json:"question,omitempty"`
	Vi        string        `json:"vi,omitempty"`
	En        string        `json:"en,omitempty"`
	NoteVi    string        `json:"noteVi,omitempty"`
}

func (h *Handler) suggestFix(w http.ResponseWriter, r *http.Request) {
	if err := http.NewResponseController(w).SetWriteDeadline(time.Now().Add(generateWriteTimeout)); err != nil &&
		!errors.Is(err, http.ErrNotSupported) {
		h.log.WarnContext(r.Context(), "lesson fix: extend write deadline", slog.Any("error", err))
	}
	var body struct {
		Area  string `json:"area"`
		Index int    `json:"index"`
	}
	if httpx.DecodeJSON(w, r, &body) != nil {
		return
	}
	s, err := h.svc.SuggestFix(r.Context(), r.PathValue("id"), FlagArea(body.Area), body.Index)
	if err != nil {
		h.writeCheckError(w, r, err)
		return
	}
	out := fixJSON{
		Area: string(s.Area), Index: s.Index, Text: s.Text, MeaningVi: s.MeaningVi, Vi: s.Vi, En: s.En, NoteVi: s.NoteVi,
	}
	if s.Question != nil {
		q := questionJSON(*s.Question)
		out.Question = &q
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]fixJSON{"suggestion": out})
}

func (h *Handler) applyFix(w http.ResponseWriter, r *http.Request) {
	var body fixJSON
	if httpx.DecodeJSON(w, r, &body) != nil {
		return
	}
	in := FixInput{
		Area: FlagArea(body.Area), Index: body.Index, Text: body.Text, MeaningVi: body.MeaningVi, Vi: body.Vi, En: body.En,
	}
	if body.Question != nil {
		q := Question(*body.Question)
		in.Question = &q
	}
	l, err := h.svc.ApplyFix(r.Context(), r.PathValue("id"), in)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeLesson(w, r, http.StatusOK, l)
}
