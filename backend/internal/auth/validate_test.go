package auth

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalizeEmail(t *testing.T) {
	t.Parallel()

	if got := NormalizeEmail("  An@Example.COM \t"); got != "an@example.com" {
		t.Fatalf("NormalizeEmail = %q, want an@example.com", got)
	}
}

func TestValidateCredentials(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		email      string
		password   string
		wantFields []string
	}{
		{name: "valid", email: "an@example.com", password: "matkhau1"},
		{name: "128 runes ok", email: "an@example.com", password: strings.Repeat("đ", 128)},
		{name: "empty both", email: "", password: "", wantFields: []string{"email", "password"}},
		{name: "no domain dot", email: "a@b", password: "matkhau1", wantFields: []string{"email"}},
		{name: "display name", email: "An <an@example.com>", password: "matkhau1", wantFields: []string{"email"}},
		{name: "too long email", email: strings.Repeat("a", 250) + "@x.vn", password: "matkhau1", wantFields: []string{"email"}},
		{name: "7 runes", email: "an@example.com", password: "mậtkhẩu", wantFields: []string{"password"}},
		{name: "129 runes", email: "an@example.com", password: strings.Repeat("đ", 129), wantFields: []string{"password"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateCredentials(tt.email, tt.password)
			if len(tt.wantFields) == 0 {
				if err != nil {
					t.Fatalf("ValidateCredentials error = %v, want nil", err)
				}
				return
			}

			var verr *ValidationError
			if !errors.As(err, &verr) {
				t.Fatalf("error = %v, want *ValidationError", err)
			}
			if len(verr.Fields) != len(tt.wantFields) {
				t.Fatalf("fields = %v, want keys %v", verr.Fields, tt.wantFields)
			}
			for _, f := range tt.wantFields {
				if verr.Fields[f] == "" {
					t.Errorf("missing message for field %q in %v", f, verr.Fields)
				}
			}
		})
	}
}

func TestNormalizeTimezone(t *testing.T) {
	t.Parallel()

	tests := []struct{ in, want string }{
		{"Asia/Ho_Chi_Minh", "Asia/Ho_Chi_Minh"},
		{"Europe/Paris", "Europe/Paris"},
		{"  Asia/Tokyo  ", "Asia/Tokyo"},
		{"", DefaultTimezone},
		{"Local", DefaultTimezone},
		{"Mars/Base", DefaultTimezone},
		{"../etc/passwd", DefaultTimezone},
	}
	for _, tt := range tests {
		if got := NormalizeTimezone(tt.in); got != tt.want {
			t.Errorf("NormalizeTimezone(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
