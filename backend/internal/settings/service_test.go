package settings

import (
	"errors"
	"testing"
)

func TestGetDefaults(t *testing.T) {
	t.Parallel()
	repo := newFakeRepo("u1", "u2")
	repo.set("u2", Settings{Timezone: "Europe/London"}) // registered in London, never saved settings
	svc := NewService(repo)

	s, err := svc.Get(t.Context(), "u1")
	if err != nil || s != (Settings{Theme: ThemeSystem, DailyReviewLimit: 30, Timezone: DefaultTimezone}) {
		t.Fatalf("u1 = %+v, %v", s, err)
	}
	if s, _ := svc.Get(t.Context(), "u2"); s.Timezone != "Europe/London" || s.DailyReviewLimit != 30 {
		t.Fatalf("u2 = %+v", s)
	}
	if _, err := svc.Get(t.Context(), "nobody"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown user: %v", err)
	}
}

func TestUpdate(t *testing.T) {
	t.Parallel()
	repo := newFakeRepo("u1", "u2")
	svc := NewService(repo)

	s, err := svc.Update(t.Context(), "u1", Patch{Theme: ptr(ThemeDark)})
	if err != nil || s != (Settings{Theme: ThemeDark, DailyReviewLimit: 30, Timezone: DefaultTimezone}) {
		t.Fatalf("theme = %+v, %v", s, err)
	}
	s, err = svc.Update(t.Context(), "u1", Patch{DailyReviewLimit: ptr(20), Timezone: ptr("Europe/London")})
	if err != nil || s != (Settings{Theme: ThemeDark, DailyReviewLimit: 20, Timezone: "Europe/London"}) {
		t.Fatalf("limit + timezone = %+v, %v", s, err)
	}

	// One invalid field: nothing is written.
	var verr *ValidationError
	if _, err := svc.Update(t.Context(), "u1", Patch{Theme: ptr(ThemeLight), DailyReviewLimit: ptr(500)}); !errors.As(err, &verr) {
		t.Fatalf("invalid: %v", err)
	}
	if s, _ := svc.Get(t.Context(), "u1"); s.Theme != ThemeDark || s.DailyReviewLimit != 20 {
		t.Fatalf("after invalid = %+v", s)
	}

	// An empty patch returns the current settings.
	if s, err := svc.Update(t.Context(), "u1", Patch{}); err != nil || s.Timezone != "Europe/London" {
		t.Fatalf("empty = %+v, %v", s, err)
	}
	// Another user is untouched.
	if s, _ := svc.Get(t.Context(), "u2"); s.Theme != ThemeSystem || s.DailyReviewLimit != 30 {
		t.Fatalf("u2 = %+v", s)
	}
	if _, err := svc.Update(t.Context(), "nobody", Patch{Theme: ptr(ThemeDark)}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown user: %v", err)
	}
}

func TestLocationAndReviewLimit(t *testing.T) {
	t.Parallel()
	repo := newFakeRepo("u1", "u2", "u3")
	repo.set("u2", Settings{Timezone: "America/New_York", DailyReviewLimit: 12})
	repo.set("u3", Settings{Timezone: "Broken/Zone"})
	svc := NewService(repo)

	for user, want := range map[string]string{"u1": DefaultTimezone, "u2": "America/New_York", "u3": DefaultTimezone} {
		loc, err := svc.Location(t.Context(), user)
		if err != nil || loc.String() != want {
			t.Errorf("%s location = %v, %v; want %s", user, loc, err, want)
		}
	}
	if n, err := svc.ReviewLimit(t.Context(), "u1"); err != nil || n != 30 {
		t.Errorf("u1 limit = %d, %v", n, err)
	}
	if n, _ := svc.ReviewLimit(t.Context(), "u2"); n != 12 {
		t.Errorf("u2 limit = %d", n)
	}
	if _, err := svc.Location(t.Context(), "nobody"); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown user: %v", err)
	}
}
