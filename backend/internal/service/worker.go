package service

import (
	"context"
	"time"

	"github.com/luongtran/luna/backend/internal/job"
)

// initWorker builds the background worker that runs the annotation, practice, word picture and grading jobs
// (constitution III). StartWorker runs it.
func (c *Container) initWorker() {
	c.worker = job.NewWorker(c.jobs, map[job.Type]job.Handler{
		job.TypeAnnotate: c.lesson.ProcessAnnotate,
		job.TypePractice: c.lesson.ProcessPractice,
		job.TypeImages:   c.lesson.ProcessImages,
		job.TypeGrade:    c.writing.ProcessGrade,
	}, func(ctx context.Context, j job.Job, err error) {
		// Grade jobs belong to a writing, the others to a lesson revision.
		if j.Type == job.TypeGrade {
			c.writing.JobFailed(ctx, j, err)
			return
		}
		c.lesson.JobFailed(ctx, j, err)
	}, time.Now, c.log)
}
