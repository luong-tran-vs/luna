// Package settings stores a learner's settings (F12): theme, daily review limit and timezone.
package settings

import (
	"errors"
	"fmt"
)

// Theme is the light/dark choice; ThemeSystem follows the device.
type Theme string

const (
	ThemeLight  Theme = "light"
	ThemeDark   Theme = "dark"
	ThemeSystem Theme = "system"
)

const (
	DefaultTheme       = ThemeSystem
	DefaultReviewLimit = 30
	MinReviewLimit     = 5
	MaxReviewLimit     = 200
	// DefaultTimezone is used when an account has no loadable timezone (as F1).
	DefaultTimezone = "Asia/Ho_Chi_Minh"
)

// Settings are a learner's settings.
type Settings struct {
	Theme            Theme
	DailyReviewLimit int
	Timezone         string
}

// Patch holds the fields to change; nil means unchanged.
type Patch struct {
	Theme            *Theme
	DailyReviewLimit *int
	Timezone         *string
}

// ErrNotFound is returned for an unknown user.
var ErrNotFound = errors.New("settings: user not found")

// ValidationError lists invalid fields with Vietnamese messages.
type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string { return fmt.Sprintf("settings: invalid input %v", e.Fields) }
