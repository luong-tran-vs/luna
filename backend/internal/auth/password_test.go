package auth

import (
	"strings"
	"testing"
)

func TestHashPassword(t *testing.T) {
	t.Parallel()

	h1, err := HashPassword("matkhau123")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	h2, err := HashPassword("matkhau123")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}

	if !strings.HasPrefix(h1, "$argon2id$v=19$m=19456,t=2,p=1$") {
		t.Errorf("hash %q has unexpected prefix", h1)
	}
	if h1 == h2 {
		t.Error("two hashes of the same password are equal; salt missing")
	}
	if strings.Contains(h1, "matkhau123") {
		t.Error("hash contains the plain password")
	}
}

func TestVerifyPassword(t *testing.T) {
	t.Parallel()

	long := strings.Repeat("mật khẩu ", 14) + "đủ" // 128 runes, spaces and diacritics
	tests := []struct {
		name     string
		password string
		attempt  string
		want     bool
	}{
		{name: "correct", password: "matkhau123", attempt: "matkhau123", want: true},
		{name: "wrong", password: "matkhau123", attempt: "matkhau124", want: false},
		{name: "case sensitive", password: "MatKhau123", attempt: "matkhau123", want: false},
		{name: "long unicode not truncated", password: long, attempt: long, want: true},
		{name: "long unicode last char differs", password: long, attempt: long[:len(long)-1] + "x", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			encoded, err := HashPassword(tt.password)
			if err != nil {
				t.Fatalf("HashPassword: %v", err)
			}
			got, err := VerifyPassword(tt.attempt, encoded)
			if err != nil {
				t.Fatalf("VerifyPassword: %v", err)
			}
			if got != tt.want {
				t.Fatalf("VerifyPassword = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVerifyPasswordRejectsMalformedHash(t *testing.T) {
	t.Parallel()

	for _, encoded := range []string{
		"",
		"plain",
		"$2a$10$abcdefghijklmnopqrstuv", // bcrypt
		"$argon2i$v=19$m=19456,t=2,p=1$AAAA$AAAA", // wrong variant
		"$argon2id$v=19$m=x,t=2,p=1$AAAA$AAAA",    // bad params
		"$argon2id$v=19$m=19456,t=2,p=1$!!!$AAAA", // bad base64
	} {
		if _, err := VerifyPassword("matkhau123", encoded); err == nil {
			t.Errorf("VerifyPassword(%q) error = nil, want error", encoded)
		}
	}
}
