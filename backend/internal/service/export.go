package service

import (
	"context"
	"fmt"
	"time"

	"github.com/luongtran/luna/backend/internal/export"
	"github.com/luongtran/luna/backend/internal/settings"
)

// initExport builds the data export a learner downloads (F13).
func (c *Container) initExport() {
	c.export = export.NewService(c.store.Export(), exportSettings{c.settings}, time.Now)
}

// exportSettings adapts settings.Service to export.Settings.
type exportSettings struct {
	svc *settings.Service
}

func (e exportSettings) Values(ctx context.Context, userID string) (map[string]any, error) {
	s, err := e.svc.Get(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get settings: %w", err)
	}
	return map[string]any{"theme": string(s.Theme), "dailyReviewLimit": s.DailyReviewLimit, "timezone": s.Timezone}, nil
}

func (e exportSettings) Location(ctx context.Context, userID string) (*time.Location, error) {
	return e.svc.Location(ctx, userID)
}
