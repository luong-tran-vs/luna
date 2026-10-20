package service

import (
	"time"

	"github.com/luongtran/luna/backend/internal/verbs"
)

// initVerbs builds the irregular verbs practice (the "Động từ" tab).
func (c *Container) initVerbs() {
	c.verbs = verbs.NewService(c.store.VerbProgress(), c.studyDays, time.Now)
}
