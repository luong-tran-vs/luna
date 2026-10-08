package service

import (
	"net/http"
	"time"

	"github.com/luongtran/luna/backend/internal/aiusage"
	"github.com/luongtran/luna/backend/internal/export"
	"github.com/luongtran/luna/backend/internal/grammar"
	"github.com/luongtran/luna/backend/internal/health"
	"github.com/luongtran/luna/backend/internal/lesson"
	"github.com/luongtran/luna/backend/internal/platform/httpx"
	"github.com/luongtran/luna/backend/internal/progress"
	"github.com/luongtran/luna/backend/internal/settings"
	"github.com/luongtran/luna/backend/internal/topic"
	"github.com/luongtran/luna/backend/internal/vocab"
	"github.com/luongtran/luna/backend/internal/wordbank"
	"github.com/luongtran/luna/backend/internal/writing"
)

// healthPingTimeout bounds the database ping behind GET /api/health.
const healthPingTimeout = 2 * time.Second

// Handler returns the whole HTTP API: every route of every service, behind the request-id,
// access-log and panic-recovery middleware.
func (c *Container) Handler() http.Handler {
	mux := http.NewServeMux()
	c.registerRoutes(mux)
	return httpx.Chain(mux, httpx.RequestID, httpx.Logger(c.log), httpx.Recover(c.log))
}

func (c *Container) registerRoutes(mux *http.ServeMux) {
	auth, log := c.requireAuth, c.log

	mux.Handle("GET /api/health", health.NewHandler(c.store, healthPingTimeout, log))
	mux.HandleFunc("POST /api/auth/register", c.authHandler.Register)
	mux.HandleFunc("POST /api/auth/login", c.authHandler.Login)
	mux.HandleFunc("POST /api/auth/logout", c.authHandler.Logout)
	mux.Handle("GET /api/auth/me", auth(http.HandlerFunc(c.authHandler.Me)))
	c.authHandler.RegisterAdmin(mux, auth)

	settings.NewHandler(c.settings, log).Register(mux, auth)
	export.NewHandler(c.export, log).Register(mux, auth)
	lesson.NewHandler(c.lesson, log).Register(mux, auth)
	topic.NewHandler(c.topic, log).Register(mux, auth)
	vocab.NewHandler(c.vocab, log).Register(mux, auth)
	wordbank.NewHandler(c.wordBank, log).Register(mux, auth)
	aiusage.NewHandler(c.aiUsage, log).Register(mux, auth)
	grammar.NewHandler(c.grammar, log).Register(mux, auth)

	studyHandler := progress.NewStudyHandler(c.study, log)
	studyHandler.Register(mux, auth)
	// Lesson content is guarded: learners open only the lesson being studied and lessons already started (L).
	lesson.NewReadingHandler(c.reader, log).Register(mux, auth, studyHandler.Guard)
	progress.NewHandler(c.progress, log).Register(mux, auth, studyHandler.Guard)
	writing.NewHandler(c.writing, log).Register(mux, auth, studyHandler.Guard)
}
