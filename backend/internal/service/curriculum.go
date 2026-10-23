package service

import (
	"time"

	"github.com/luongtran/luna/backend/internal/ai"
	"github.com/luongtran/luna/backend/internal/curriculum"
)

// initCurriculum builds the course service (added 2026-10-22; drafted by the AI and kept in the
// database since 2026-10-23).
func (c *Container) initCurriculum() {
	c.curriculum = &curriculum.Service{
		Topics: c.topic, Lessons: c.lesson, Grammar: c.grammar, Repo: c.store.Curricula(), Now: time.Now,
	}
	if w, ok := c.ai.(ai.CurriculumWriter); ok {
		c.curriculum.Writer = w
	}
}
