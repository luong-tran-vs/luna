// Command server runs the Luna HTTP backend.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/luongtran/luna/backend/internal/platform/config"
	"github.com/luongtran/luna/backend/internal/platform/logger"
	"github.com/luongtran/luna/backend/internal/service"
	"github.com/luongtran/luna/backend/internal/storage/factory"
)

const (
	shutdownTimeout    = 10 * time.Second
	disconnectTimeout  = 5 * time.Second
	indexRetryInterval = 10 * time.Second
)

func main() {
	// Local development: read backend/.env when run from the backend directory.
	// Real environment variables (shell, system service) always take precedence.
	if _, err := config.LoadDotEnv(".env"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	cfg, err := config.Load(os.Getenv)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	log := logger.New(os.Stdout, cfg.LogLevel)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, cfg, log); err != nil {
		log.Error("server stopped with error", slog.Any("error", err))
		stop()
		os.Exit(1) //nolint:gocritic // stop() is called explicitly above
	}
}

// run opens the database, builds the services, serves HTTP until ctx is cancelled, then shuts down
// gracefully. Deferred calls run in reverse: the worker stops, the services close, then the database.
func run(ctx context.Context, cfg config.Config, log *slog.Logger) error {
	store, err := factory.Open(cfg)
	if err != nil {
		return err
	}
	defer func() {
		dctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), disconnectTimeout)
		defer cancel()
		if err := store.Close(dctx); err != nil {
			log.WarnContext(dctx, "database close failed", slog.String("driver", cfg.DBDriver), slog.Any("error", err))
		}
	}()
	log.InfoContext(ctx, "database selected", slog.String("driver", cfg.DBDriver))
	store.Prepare(ctx, indexRetryInterval, log)

	svc, err := service.Init(ctx, cfg, log, store)
	if err != nil {
		return err
	}
	defer func() { _ = svc.Close() }()

	waitWorker := svc.StartWorker(ctx)
	defer waitWorker()

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           svc.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serveErr := make(chan error, 1)
	go func() {
		log.InfoContext(ctx, "server listening", slog.String("addr", cfg.HTTPAddr))
		serveErr <- srv.ListenAndServe()
	}()

	select {
	case err := <-serveErr:
		return fmt.Errorf("listen: %w", err)
	case <-ctx.Done():
	}

	log.InfoContext(ctx, "shutting down", slog.String("timeout", shutdownTimeout.String()))
	sctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(sctx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	if err := <-serveErr; err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("listen: %w", err)
	}
	log.InfoContext(ctx, "server stopped")
	return nil
}
