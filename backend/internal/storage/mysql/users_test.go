package mysql

import (
	"errors"
	"testing"
	"time"

	"github.com/luongtran/luna/backend/internal/auth"
)

func TestUsersCreateAndFind(t *testing.T) {
	t.Parallel()
	db := testDB(t)
	r := NewUsers(db)
	ctx := t.Context()

	created := time.Date(2026, 9, 29, 8, 0, 0, 123456000, time.UTC)
	u, err := r.Create(ctx, auth.User{Email: "a@x.vn", PasswordHash: "hash", Role: auth.RoleAdmin, Timezone: "Asia/Tokyo", CreatedAt: created})
	if err != nil {
		t.Fatal(err)
	}
	if len(u.ID) != 24 || u.Email != "a@x.vn" || u.Role != auth.RoleAdmin {
		t.Fatalf("created = %+v", u)
	}

	for name, find := range map[string]func() (auth.User, error){
		"email": func() (auth.User, error) { return r.FindByEmail(ctx, "a@x.vn") },
		"id":    func() (auth.User, error) { return r.FindByID(ctx, u.ID) },
	} {
		got, err := find()
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if got.ID != u.ID || got.PasswordHash != "hash" || got.Role != auth.RoleAdmin || got.Timezone != "Asia/Tokyo" || !got.CreatedAt.Equal(created) {
			t.Fatalf("%s: got %+v", name, got)
		}
	}
}

func TestUsersNotFound(t *testing.T) {
	t.Parallel()
	r := NewUsers(testDB(t))
	if _, err := r.FindByEmail(t.Context(), "nobody@x.vn"); !errors.Is(err, auth.ErrNotFound) {
		t.Fatalf("FindByEmail err = %v", err)
	}
	for _, id := range []string{"", "zzz", "0123456789abcdef01234567"} {
		if _, err := r.FindByID(t.Context(), id); !errors.Is(err, auth.ErrNotFound) {
			t.Fatalf("FindByID(%q) err = %v", id, err)
		}
	}
}

func TestUsersDuplicateEmail(t *testing.T) {
	t.Parallel()
	r := NewUsers(testDB(t))
	u := auth.User{Email: "a@x.vn", PasswordHash: "h", Role: auth.RoleMember, CreatedAt: time.Now()}
	if _, err := r.Create(t.Context(), u); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Create(t.Context(), u); !errors.Is(err, auth.ErrEmailTaken) {
		t.Fatalf("second Create err = %v", err)
	}
	if n, _ := r.Count(t.Context()); n != 1 {
		t.Fatalf("count = %d", n)
	}
}

func TestUsersCount(t *testing.T) {
	t.Parallel()
	r := NewUsers(testDB(t))
	if n, err := r.Count(t.Context()); err != nil || n != 0 {
		t.Fatalf("empty count = %d, %v", n, err)
	}
	for _, e := range []string{"a@x.vn", "b@x.vn", "c@x.vn"} {
		if _, err := r.Create(t.Context(), auth.User{Email: e, Role: auth.RoleMember, CreatedAt: time.Now()}); err != nil {
			t.Fatal(err)
		}
	}
	if n, err := r.Count(t.Context()); err != nil || n != 3 {
		t.Fatalf("count = %d, %v", n, err)
	}
}

func TestUsersUpdateAndDelete(t *testing.T) {
	t.Parallel()
	db := testDB(t)
	r, sessions := NewUsers(db), NewSessions(db)
	u, err := r.Create(t.Context(), auth.User{Email: "a@x.vn", PasswordHash: "h", Role: auth.RoleGuest, CreatedAt: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	other, _ := r.Create(t.Context(), auth.User{Email: "b@x.vn", PasswordHash: "h", Role: auth.RoleGuest, CreatedAt: time.Now()})

	if err := r.Update(t.Context(), u.ID, auth.AccountChange{Email: "c@x.vn", Role: auth.RoleMember}); err != nil {
		t.Fatal(err)
	}
	got, _ := r.FindByID(t.Context(), u.ID)
	if got.Email != "c@x.vn" || got.Role != auth.RoleMember || got.PasswordHash != "h" {
		t.Fatalf("after update = %+v", got)
	}
	// Unchanged values are not "not found".
	if err := r.Update(t.Context(), u.ID, auth.AccountChange{Email: "c@x.vn", Role: auth.RoleMember}); err != nil {
		t.Fatalf("same values: %v", err)
	}
	if err := r.Update(t.Context(), u.ID, auth.AccountChange{Email: "b@x.vn", Role: auth.RoleMember}); !errors.Is(err, auth.ErrEmailTaken) {
		t.Fatalf("taken email: %v", err)
	}
	if err := r.Update(t.Context(), "missing", auth.AccountChange{Email: "z@x.vn", Role: auth.RoleMember}); !errors.Is(err, auth.ErrNotFound) {
		t.Fatalf("missing: %v", err)
	}

	if err := sessions.Create(t.Context(), auth.Session{TokenHash: "t1", UserID: u.ID, ExpiresAt: time.Now().Add(time.Hour), CreatedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	if err := r.Delete(t.Context(), u.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := r.FindByID(t.Context(), u.ID); !errors.Is(err, auth.ErrNotFound) {
		t.Fatalf("still there: %v", err)
	}
	if _, err := sessions.FindByTokenHash(t.Context(), "t1"); !errors.Is(err, auth.ErrNotFound) {
		t.Fatalf("session kept: %v", err)
	}
	if err := r.Delete(t.Context(), u.ID); !errors.Is(err, auth.ErrNotFound) {
		t.Fatalf("delete again: %v", err)
	}
	if _, err := r.FindByID(t.Context(), other.ID); err != nil {
		t.Fatalf("other account: %v", err)
	}
}
