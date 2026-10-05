package service

import "github.com/luongtran/luna/backend/internal/settings"

// initSettings builds the learner settings service (theme, review limit, timezone). Several other
// services read the learner's timezone through it.
func (c *Container) initSettings() {
	c.settings = settings.NewService(c.store.Settings())
}
