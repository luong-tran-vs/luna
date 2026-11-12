package lesson

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/luongtran/luna/backend/internal/platform/httpx"
)

// Removing scene pictures (added 2026-11-12): before drawing them again, the admin takes away the
// scene pictures of a topic's lessons, or of a whole level, at once. The lessons keep their scene
// description and style, which the next drawing uses.

// MaxSceneRemove is the most lessons one removal takes.
const MaxSceneRemove = 500

// RemoveScenes removes the scene picture of the lessons among ids that have one and tells how many
// it removed. It refuses while a scene batch runs, which could save one back right after.
func (s *Service) RemoveScenes(ctx context.Context, ids []string) (int, error) {
	if s.ImageStore == nil {
		return 0, ErrImagesUnavailable
	}
	ids = uniqueIDs(ids)
	if len(ids) == 0 {
		return 0, &ValidationError{Fields: map[string]string{"lessonIds": "Chưa chọn bài nào"}}
	}
	if len(ids) > MaxSceneRemove {
		return 0, &ValidationError{Fields: map[string]string{"lessonIds": fmt.Sprintf("Mỗi lượt gỡ tối đa %d bài", MaxSceneRemove)}}
	}
	if s.SceneBatch().Running {
		return 0, ErrSceneBatchRunning
	}
	has, err := s.HasScenes(ctx, ids)
	if err != nil {
		return 0, err
	}
	removed := 0
	for _, id := range ids {
		if !has[id] {
			continue
		}
		if err := s.ImageStore.Delete(ctx, id, SceneKey); err != nil {
			return removed, fmt.Errorf("lesson: remove scene: %w", err)
		}
		removed++
	}
	return removed, nil
}

// removeScenes serves POST /api/admin/lessons/scene-images/remove with {"lessonIds"}: {"removed"},
// how many scene pictures went; 409 while a scene batch runs.
func (h *Handler) removeScenes(w http.ResponseWriter, r *http.Request) {
	var in struct {
		LessonIDs []string `json:"lessonIds"`
	}
	if httpx.DecodeJSON(w, r, &in) != nil {
		return
	}
	n, err := h.svc.RemoveScenes(r.Context(), in.LessonIDs)
	if errors.Is(err, ErrSceneBatchRunning) {
		httpx.WriteError(w, http.StatusConflict, "batch_running", "Đang có một lượt tạo ảnh cảnh chạy. Chờ xong hoặc bấm Dừng rồi gỡ.")
		return
	}
	if err != nil {
		if status, code, msg, ok := drawError(err); ok {
			httpx.WriteError(w, status, code, msg)
			return
		}
		h.writeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]int{"removed": n})
}
