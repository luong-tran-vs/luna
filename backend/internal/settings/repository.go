package settings

import "context"

// Repository stores settings with the account. Implementations live in internal/storage.
type Repository interface {
	// Get returns the stored fields, zero values for missing ones (Timezone falls back to the
	// timezone saved at registration). ErrNotFound for an unknown user.
	Get(ctx context.Context, userID string) (Settings, error)
	// Update sets the non-nil fields of p and returns the stored settings.
	Update(ctx context.Context, userID string, p Patch) (Settings, error)
}
