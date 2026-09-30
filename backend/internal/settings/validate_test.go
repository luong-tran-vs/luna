package settings

import (
	"errors"
	"testing"
)

func ptr[T any](v T) *T { return &v }

func TestValidatePatch(t *testing.T) {
	t.Parallel()
	ok := []Patch{
		{},
		{Theme: ptr(ThemeLight)},
		{Theme: ptr(ThemeDark)},
		{Theme: ptr(ThemeSystem)},
		{DailyReviewLimit: ptr(5)},
		{DailyReviewLimit: ptr(200)},
		{Timezone: ptr("Asia/Ho_Chi_Minh")},
		{Timezone: ptr("Europe/London")},
		{Timezone: ptr("UTC")},
	}
	for _, p := range ok {
		if err := ValidatePatch(p); err != nil {
			t.Errorf("%+v: %v", p, err)
		}
	}

	bad := map[string]struct {
		p     Patch
		field string
		msg   string
	}{
		"theme blue":       {Patch{Theme: ptr(Theme("blue"))}, "theme", "Chế độ giao diện không hợp lệ"},
		"theme empty":      {Patch{Theme: ptr(Theme(""))}, "theme", "Chế độ giao diện không hợp lệ"},
		"limit 4":          {Patch{DailyReviewLimit: ptr(4)}, "dailyReviewLimit", "Số thẻ từ 5 đến 200"},
		"limit 201":        {Patch{DailyReviewLimit: ptr(201)}, "dailyReviewLimit", "Số thẻ từ 5 đến 200"},
		"limit 0":          {Patch{DailyReviewLimit: ptr(0)}, "dailyReviewLimit", "Số thẻ từ 5 đến 200"},
		"limit negative":   {Patch{DailyReviewLimit: ptr(-1)}, "dailyReviewLimit", "Số thẻ từ 5 đến 200"},
		"timezone empty":   {Patch{Timezone: ptr("")}, "timezone", "Múi giờ không hợp lệ"},
		"timezone Local":   {Patch{Timezone: ptr("Local")}, "timezone", "Múi giờ không hợp lệ"},
		"timezone unknown": {Patch{Timezone: ptr("Mars/Olympus")}, "timezone", "Múi giờ không hợp lệ"},
		"timezone path":    {Patch{Timezone: ptr("../etc")}, "timezone", "Múi giờ không hợp lệ"},
	}
	for name, tc := range bad {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var verr *ValidationError
			if err := ValidatePatch(tc.p); !errors.As(err, &verr) || verr.Fields[tc.field] != tc.msg || len(verr.Fields) != 1 {
				t.Fatalf("err = %v", err)
			}
		})
	}

	var verr *ValidationError
	err := ValidatePatch(Patch{Theme: ptr(Theme("x")), DailyReviewLimit: ptr(1), Timezone: ptr("x")})
	if !errors.As(err, &verr) || len(verr.Fields) != 3 {
		t.Fatalf("all invalid: %v", err)
	}
}
