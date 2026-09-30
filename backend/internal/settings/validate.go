package settings

import (
	"strings"
	"time"
)

// ValidTimezone reports whether tz is an IANA name the server can load ("Local" and "" are not).
func ValidTimezone(tz string) bool {
	if tz == "" || tz == "Local" || strings.Contains(tz, "..") {
		return false
	}
	_, err := time.LoadLocation(tz)
	return err == nil
}

// ValidatePatch checks the fields present in p; a *ValidationError lists every invalid one.
func ValidatePatch(p Patch) error {
	fields := map[string]string{}
	if p.Theme != nil {
		switch *p.Theme {
		case ThemeLight, ThemeDark, ThemeSystem:
		default:
			fields["theme"] = "Chế độ giao diện không hợp lệ"
		}
	}
	if p.DailyReviewLimit != nil && (*p.DailyReviewLimit < MinReviewLimit || *p.DailyReviewLimit > MaxReviewLimit) {
		fields["dailyReviewLimit"] = "Số thẻ từ 5 đến 200"
	}
	if p.Timezone != nil && !ValidTimezone(*p.Timezone) {
		fields["timezone"] = "Múi giờ không hợp lệ"
	}
	if len(fields) > 0 {
		return &ValidationError{Fields: fields}
	}
	return nil
}
