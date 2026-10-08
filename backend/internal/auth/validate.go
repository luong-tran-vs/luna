package auth

import (
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"
)

// DefaultTimezone is stored when the browser sends no valid IANA timezone.
const DefaultTimezone = "Asia/Ho_Chi_Minh"

const (
	maxEmailLen       = 254
	minPasswordLength = 8
	maxPasswordLength = 128
)

// NormalizeEmail trims spaces and lowercases the address.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// ValidateCredentials checks a normalized email and a password (length counted in characters).
func ValidateCredentials(email, password string) error {
	fields := map[string]string{}
	checkEmail(fields, email)
	checkPassword(fields, password)
	if len(fields) > 0 {
		return &ValidationError{Fields: fields}
	}
	return nil
}

// checkEmail adds the error of a normalized email to fields, if any.
func checkEmail(fields map[string]string, email string) {
	switch {
	case email == "":
		fields["email"] = "Vui lòng nhập email"
	case !validEmail(email):
		fields["email"] = "Email không hợp lệ"
	}
}

// checkPassword adds the error of a password (length counted in characters) to fields, if any.
func checkPassword(fields map[string]string, password string) {
	switch n := utf8.RuneCountInString(password); {
	case n == 0:
		fields["password"] = "Vui lòng nhập mật khẩu"
	case n < minPasswordLength:
		fields["password"] = "Mật khẩu cần ít nhất 8 ký tự"
	case n > maxPasswordLength:
		fields["password"] = "Mật khẩu tối đa 128 ký tự"
	}
}

func validEmail(email string) bool {
	if len(email) > maxEmailLen {
		return false
	}
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email || addr.Name != "" {
		return false
	}
	at := strings.LastIndexByte(email, '@')
	domain := email[at+1:]
	return strings.Contains(domain, ".") && !strings.HasPrefix(domain, ".") && !strings.HasSuffix(domain, ".")
}

// NormalizeTimezone returns tz when it is a loadable IANA name, otherwise DefaultTimezone.
func NormalizeTimezone(tz string) string {
	tz = strings.TrimSpace(tz)
	if tz == "" || tz == "Local" {
		return DefaultTimezone
	}
	if _, err := time.LoadLocation(tz); err != nil {
		return DefaultTimezone
	}
	return tz
}
