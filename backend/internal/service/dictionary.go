package service

import (
	"context"
	"log/slog"

	"github.com/luongtran/luna/backend/internal/dictionary"
	"github.com/luongtran/luna/backend/internal/lesson"
)

// closableDictionary is the dictionary used by the Reading step.
type closableDictionary interface {
	lesson.Dictionary
	Close() error
}

type noDictionary struct{ dictionary.None }

func (noDictionary) Close() error { return nil }

// initDictionary opens the offline dictionary; without the file, lookups use annotations only.
func (c *Container) initDictionary(ctx context.Context) {
	path := c.cfg.DictionaryPath
	d, err := dictionary.Open(ctx, path)
	if err != nil {
		c.log.WarnContext(ctx, "dictionary unavailable: run deploy/fetch-dictionary.sh", slog.String("path", path), slog.Any("error", err))
		c.dict = noDictionary{}
		return
	}
	c.log.InfoContext(ctx, "dictionary loaded", slog.String("path", path))
	c.dict = d
}
