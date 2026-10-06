// Package service builds the app: every domain service, the adapters between them, the background
// worker and the HTTP routes. It is the one place that knows how the domains fit together, so
// cmd/server only opens the database and calls Init.
//
// Each domain has its own init function in its own file (initAuth in auth.go, initLesson in
// lesson.go...). Init runs them in dependency order; a service only reads the ones before it.
package service

import (
	"context"
	"log/slog"

	"github.com/luongtran/luna/backend/internal/ai"
	"github.com/luongtran/luna/backend/internal/auth"
	"github.com/luongtran/luna/backend/internal/export"
	"github.com/luongtran/luna/backend/internal/grammar"
	"github.com/luongtran/luna/backend/internal/job"
	"github.com/luongtran/luna/backend/internal/lesson"
	"github.com/luongtran/luna/backend/internal/platform/config"
	"github.com/luongtran/luna/backend/internal/platform/httpx"
	"github.com/luongtran/luna/backend/internal/progress"
	"github.com/luongtran/luna/backend/internal/settings"
	"github.com/luongtran/luna/backend/internal/storage"
	"github.com/luongtran/luna/backend/internal/topic"
	"github.com/luongtran/luna/backend/internal/vocab"
	"github.com/luongtran/luna/backend/internal/writing"
)

// Container holds every service and what they share. Build it with Init.
type Container struct {
	cfg   config.Config
	log   *slog.Logger
	store storage.Store

	// Repositories used by more than one service, opened once.
	lessons storage.LessonStore
	jobs    job.Repository

	ai ai.Provider
	// imageAI draws the vocabulary pictures (F23).
	imageAI ai.ImageProvider
	dict    closableDictionary

	auth        *auth.Service
	authHandler *auth.Handler
	requireAuth httpx.Middleware

	settings *settings.Service
	topic    *topic.Service
	// lessonTopics is the topic service as the lesson domain sees it.
	lessonTopics lessonTopicsPort
	lesson       *lesson.Service
	reader       *lesson.Reader
	writing      *writing.Service
	// writeSteps is the study service as the writing domain sees it; it is set by initProgress.
	writeSteps *writingSteps
	worker     *job.Worker
	vocab      *vocab.Service
	grammar    *grammar.Service
	progress   *progress.Service
	study      *progress.StudyService
	export     *export.Service
}

// Init builds every service on store. The store stays open and is closed by the caller, after
// Close and after the worker has stopped. Cancelling ctx stops anything Init started.
func Init(ctx context.Context, cfg config.Config, log *slog.Logger, store storage.Store) (*Container, error) {
	c := &Container{cfg: cfg, log: log, store: store, lessons: store.Lessons(), jobs: store.Jobs()}

	if err := c.initAuth(); err != nil {
		return nil, err
	}
	c.initDictionary(ctx)
	c.initAI()
	c.initSettings()
	c.initTopic()
	c.initLesson()
	c.initWriting()
	c.initWorker()
	c.initReader()
	c.initVocab()
	c.initProgress()
	c.initGrammar()
	c.initExport()

	c.queueMissingPractice(ctx)
	return c, nil
}

// StartWorker runs the background job worker until ctx is cancelled. The returned function waits
// for the worker to stop; call it before the database is closed.
func (c *Container) StartWorker(ctx context.Context) (wait func()) {
	done := make(chan struct{})
	go func() {
		defer close(done)
		c.worker.Run(ctx)
	}()
	return func() { <-done }
}

// Close releases what Init opened besides the database (the offline dictionary).
func (c *Container) Close() error {
	return c.dict.Close()
}
