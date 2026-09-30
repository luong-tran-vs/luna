package export

import (
	"context"
	"time"
)

// Repository reads a learner's stored data (implemented in internal/storage).
type Repository interface {
	// Account returns ErrNotFound when the user does not exist.
	Account(ctx context.Context, userID string) (Account, error)
	// UserDocs returns the user's documents of an Allowed collection, oldest first;
	// ErrCollection for any other collection.
	UserDocs(ctx context.Context, collection, userID string) ([]Doc, error)
	// Lessons returns the lessons among ids that still exist, each with its "_id".
	Lessons(ctx context.Context, ids []string) ([]Doc, error)
}

// Settings gives the learner's settings with defaults (implemented over settings in main).
type Settings interface {
	Values(ctx context.Context, userID string) (map[string]any, error)
	Location(ctx context.Context, userID string) (*time.Location, error)
}
