package lesson

import (
	"errors"
	"slices"
	"testing"
)

func TestCleanAccepted(t *testing.T) {
	en := "Yesterday I visited an old friend."
	distractors := []string{"visit", "new", "was"}
	got := cleanAccepted([]string{
		"I visited an old friend yesterday.", // another order, punctuation and case differ: kept
		"i visited an old friend YESTERDAY",  // the same as the one before: dropped
		"yesterday, I visited an old friend", // the same as en: dropped
		"I visited a new friend yesterday.",  // "a" is not a tile: dropped
		"I visit an old friend yesterday.",   // a distractor in place of a word: kept
		"I visited an old old friend.",       // a tile used twice: dropped
		"   ",
	}, en, distractors)
	want := []string{"I visited an old friend yesterday.", "I visit an old friend yesterday."}
	if !slices.Equal(got, want) {
		t.Fatalf("cleanAccepted = %q, want %q", got, want)
	}
}

func TestValidateTranslationsAccepted(t *testing.T) {
	ok, err := ValidateTranslations([]TranslationInput{{
		Vi: "Hôm qua tôi đến thăm một người bạn cũ.", En: "Yesterday I visited an old friend.",
		Distractors: []string{"visit"}, Accepted: []string{"I visited an old friend yesterday.", ""},
	}})
	if err != nil {
		t.Fatalf("ValidateTranslations: %v", err)
	}
	if want := []string{"I visited an old friend yesterday."}; !slices.Equal(ok[0].Accepted, want) {
		t.Fatalf("accepted = %q, want %q", ok[0].Accepted, want)
	}

	_, err = ValidateTranslations([]TranslationInput{{
		Vi: "Hôm qua tôi đến thăm một người bạn cũ.", En: "Yesterday I visited an old friend.",
		Accepted: []string{"I saw an old friend yesterday."},
	}})
	var ve *ValidationError
	if !errors.As(err, &ve) || ve.Fields["translations.0.accepted"] == "" {
		t.Fatalf("want an accepted field error, got %v", err)
	}
}
