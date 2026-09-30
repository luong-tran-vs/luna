// Package logger builds the structured JSON logger used across the backend.
package logger

import (
	"io"
	"log/slog"
)

// New returns a JSON logger writing to w at the given level (debug, info, warn, error).
// Unknown levels fall back to info; config.Load validates the level beforehand.
func New(w io.Writer, level string) *slog.Logger {
	var lvl slog.Level
	if err := lvl.UnmarshalText([]byte(level)); err != nil {
		lvl = slog.LevelInfo
	}
	return slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{Level: lvl}))
}
