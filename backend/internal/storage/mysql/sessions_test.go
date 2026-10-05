package mysql

import (
	"errors"
	"testing"
	"time"

	"github.com/luongtran/luna/backend/internal/auth"
)

func TestSessionsCreateFindDelete(t *testing.T) {
	t.Parallel()
	r := NewSessions(testDB(t))
	ctx := t.Context()
	now := time.Now().UTC().Truncate(time.Microsecond)
	s := auth.Session{TokenHash: "tok1", UserID: newID(), ExpiresAt: now.Add(time.Hour), CreatedAt: now}
	if err := r.Create(ctx, s); err != nil {
		t.Fatal(err)
	}
	got, err := r.FindByTokenHash(ctx, "tok1")
	if err != nil {
		t.Fatal(err)
	}
	if got.TokenHash != "tok1" || got.UserID != s.UserID || !got.ExpiresAt.Equal(s.ExpiresAt) || !got.CreatedAt.Equal(now) {
		t.Fatalf("got %+v want %+v", got, s)
	}
	if err := r.Delete(ctx, "tok1"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.FindByTokenHash(ctx, "tok1"); !errors.Is(err, auth.ErrNotFound) {
		t.Fatalf("after delete err = %v", err)
	}
	if err := r.Delete(ctx, "tok1"); err != nil {
		t.Fatalf("deleting a missing session: %v", err)
	}
}

func TestSessionsNotFound(t *testing.T) {
	t.Parallel()
	r := NewSessions(testDB(t))
	if _, err := r.FindByTokenHash(t.Context(), "nope"); !errors.Is(err, auth.ErrNotFound) {
		t.Fatalf("err = %v", err)
	}
}

func TestSessionsRejectsBadUserIDAndDuplicateToken(t *testing.T) {
	t.Parallel()
	r := NewSessions(testDB(t))
	now := time.Now()
	if err := r.Create(t.Context(), auth.Session{TokenHash: "t", UserID: "bad", ExpiresAt: now.Add(time.Hour), CreatedAt: now}); err == nil {
		t.Fatal("accepted a malformed user id")
	}
	s := auth.Session{TokenHash: "t", UserID: newID(), ExpiresAt: now.Add(time.Hour), CreatedAt: now}
	if err := r.Create(t.Context(), s); err != nil {
		t.Fatal(err)
	}
	if err := r.Create(t.Context(), s); err == nil {
		t.Fatal("accepted a duplicate token hash")
	}
}

func TestSessionsExpiredIsMissingAndPurged(t *testing.T) {
	t.Parallel()
	db := testDB(t)
	r := NewSessions(db)
	ctx := t.Context()
	now := time.Now().UTC()
	if err := r.Create(ctx, auth.Session{TokenHash: "old", UserID: newID(), ExpiresAt: now.Add(-time.Minute), CreatedAt: now.Add(-time.Hour)}); err != nil {
		t.Fatal(err)
	}
	if _, err := r.FindByTokenHash(ctx, "old"); !errors.Is(err, auth.ErrNotFound) {
		t.Fatalf("expired session found: %v", err)
	}
	var n int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sessions").Scan(&n); err != nil || n != 1 {
		t.Fatalf("rows before purge = %d, %v", n, err)
	}
	if err := r.Create(ctx, auth.Session{TokenHash: "new", UserID: newID(), ExpiresAt: now.Add(time.Hour), CreatedAt: now}); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sessions").Scan(&n); err != nil || n != 1 {
		t.Fatalf("rows after purge = %d, %v", n, err)
	}
	if _, err := r.FindByTokenHash(ctx, "new"); err != nil {
		t.Fatal(err)
	}
}
