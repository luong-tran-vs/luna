package grammar

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/luongtran/luna/backend/internal/platform/httpx"
)

// Removing example pictures (added 2026-11-12): before drawing them again, the admin takes away
// every picture of a grammar lesson, or of every lesson of a level, at once: those of the examples
// and of the "Nhìn ảnh chọn từ" exercises, which learners do not see again until drawn.

// MaxImageRemove is the most points one removal takes.
const MaxImageRemove = 500

// ImagesRemoved tells how many pictures a removal takes away (or took), from how many lessons.
type ImagesRemoved struct {
	Pictures int
	Points   int
}

// RemoveImages removes every picture of the points among pointIDs; with dryRun it only counts
// them. It refuses while an example picture batch runs, which could save one back right after.
func (s *Service) RemoveImages(ctx context.Context, pointIDs []string, dryRun bool) (ImagesRemoved, error) {
	if s.images.Store == nil {
		return ImagesRemoved{}, ErrImagesUnavailable
	}
	seen := map[string]bool{}
	var ids []string
	for _, id := range pointIDs {
		if id = strings.TrimSpace(id); id != "" && !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return ImagesRemoved{}, &ValidationError{Fields: map[string]string{"pointIds": "Chưa chọn bài nào"}}
	}
	if len(ids) > MaxImageRemove {
		return ImagesRemoved{}, &ValidationError{Fields: map[string]string{"pointIds": fmt.Sprintf("Mỗi lượt gỡ tối đa %d bài", MaxImageRemove)}}
	}
	if !dryRun && s.ImageBatch().Running {
		return ImagesRemoved{}, ErrImageBatchRunning
	}
	keys, err := s.images.Store.KeysOf(ctx, ids)
	if err != nil {
		return ImagesRemoved{}, fmt.Errorf("grammar: image keys: %w", err)
	}
	var out ImagesRemoved
	for _, id := range ids {
		n := len(keys[id])
		if n == 0 {
			continue
		}
		if !dryRun {
			if err := s.images.Store.DeleteForPoint(ctx, id); err != nil {
				return out, fmt.Errorf("grammar: remove images: %w", err)
			}
		}
		out.Pictures += n
		out.Points++
	}
	return out, nil
}

// removeImages serves POST /api/admin/grammar-images/remove with {"pointIds", "dryRun"?}:
// {"pictures", "points"}, what went (or would go); 409 while an example picture batch runs.
func (h *Handler) removeImages(w http.ResponseWriter, r *http.Request) {
	var in struct {
		PointIDs []string `json:"pointIds"`
		DryRun   bool     `json:"dryRun"`
	}
	if httpx.DecodeJSON(w, r, &in) != nil {
		return
	}
	out, err := h.svc.RemoveImages(r.Context(), in.PointIDs, in.DryRun)
	if err != nil {
		h.writeImageError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]int{"pictures": out.Pictures, "points": out.Points})
}
