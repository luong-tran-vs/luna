package mysql

import (
	"errors"
	"testing"
	"time"

	"github.com/luongtran/luna/backend/internal/auth"
	"github.com/luongtran/luna/backend/internal/settings"
)

func settingsUser(t *testing.T, r *Users, tz string) string {
	t.Helper()
	u, err := r.Create(t.Context(), auth.User{Email: "s@x.vn", Role: auth.RoleLearner, Timezone: tz, CreatedAt: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	return u.ID
}

func TestSettingsGetDefaultsFallBackToRegistrationTimezone(t *testing.T) {
	t.Parallel()
	db := testDB(t)
	id := settingsUser(t, NewUsers(db), "Asia/Tokyo")
	got, err := NewSettings(db).Get(t.Context(), id)
	if err != nil {
		t.Fatal(err)
	}
	if got != (settings.Settings{Timezone: "Asia/Tokyo"}) {
		t.Fatalf("got %+v", got)
	}
}

func TestSettingsUpdatePartial(t *testing.T) {
	t.Parallel()
	db := testDB(t)
	id := settingsUser(t, NewUsers(db), "Asia/Tokyo")
	r := NewSettings(db)
	ctx := t.Context()

	dark, limit, tz := settings.ThemeDark, 50, "Asia/Ho_Chi_Minh"
	got, err := r.Update(ctx, id, settings.Patch{Theme: &dark})
	if err != nil || got != (settings.Settings{Theme: settings.ThemeDark, Timezone: "Asia/Tokyo"}) {
		t.Fatalf("theme only: %+v, %v", got, err)
	}
	got, err = r.Update(ctx, id, settings.Patch{DailyReviewLimit: &limit, Timezone: &tz})
	if err != nil || got != (settings.Settings{Theme: settings.ThemeDark, DailyReviewLimit: 50, Timezone: tz}) {
		t.Fatalf("limit+tz: %+v, %v", got, err)
	}
	got, err = r.Update(ctx, id, settings.Patch{}) // empty patch: unchanged
	if err != nil || got != (settings.Settings{Theme: settings.ThemeDark, DailyReviewLimit: 50, Timezone: tz}) {
		t.Fatalf("empty patch: %+v, %v", got, err)
	}
	// Writing the value already stored still succeeds (matched row counts).
	if got, err = r.Update(ctx, id, settings.Patch{Theme: &dark}); err != nil || got.Theme != settings.ThemeDark {
		t.Fatalf("same value: %+v, %v", got, err)
	}
	if again, err := r.Get(ctx, id); err != nil || again != got {
		t.Fatalf("Get after update: %+v, %v", again, err)
	}
}

func TestSettingsUnknownUser(t *testing.T) {
	t.Parallel()
	r := NewSettings(testDB(t))
	theme := settings.ThemeLight
	for _, id := range []string{"", "bad", "0123456789abcdef01234567"} {
		if _, err := r.Get(t.Context(), id); !errors.Is(err, settings.ErrNotFound) {
			t.Fatalf("Get(%q) err = %v", id, err)
		}
		if _, err := r.Update(t.Context(), id, settings.Patch{Theme: &theme}); !errors.Is(err, settings.ErrNotFound) {
			t.Fatalf("Update(%q) err = %v", id, err)
		}
		if _, err := r.Update(t.Context(), id, settings.Patch{}); !errors.Is(err, settings.ErrNotFound) {
			t.Fatalf("empty Update(%q) err = %v", id, err)
		}
	}
}
