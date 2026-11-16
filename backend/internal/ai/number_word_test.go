package ai

import (
	"strings"
	"testing"
)

func TestNumberWord(t *testing.T) {
	t.Parallel()
	for word, want := range map[string]string{
		"Seven": "7", "zero": "0", "Nineteen": "19", "Twenty-one": "21", "ninety nine": "99",
		"One hundred": "100", "a hundred": "100", "One thousand": "1,000", "two hundred and five": "205",
		"three thousand four hundred": "3,400", "a million": "1,000,000", "7": "7", "1000": "1,000",
		"first": "1st", "Third": "3rd", "fourth": "4th", "twelfth": "12th", "twenty-first": "21st",
		"twenty-second": "22nd", "hundredth": "100th", "eleventh": "11th", "100th": "100th", "2nd": "2nd",
	} {
		if got, ok := NumberWord(word); !ok || got != want {
			t.Errorf("NumberWord(%q) = %q, %v; want %q", word, got, ok, want)
		}
	}
	for _, word := range []string{"apple", "second", "a", "and", "one day", "seventh heaven", "", "zeroth", "teeth", "month"} {
		if got, ok := NumberWord(word); ok {
			t.Errorf("NumberWord(%q) = %q, want no number", word, got)
		}
	}
}

func TestImagePromptDrawsTheNumeralOfANumberWord(t *testing.T) {
	t.Parallel()
	p := ImagePrompt(ImageRequest{Word: "Seven", MeaningVi: "số bảy"})
	if !strings.Contains(p, `Show the numeral "7"`) || !strings.Contains(p, "Do not replace the numeral with a group of objects") ||
		strings.Contains(p, "letters, words or numbers") {
		t.Fatalf("prompt = %s", p)
	}
	if p := ImagePrompt(ImageRequest{Word: "apple"}); !strings.Contains(p, "letters, words or numbers") {
		t.Fatalf("apple prompt = %s", p)
	}
}

func TestImagePromptOfATopic(t *testing.T) {
	t.Parallel()
	p := ImagePrompt(ImageRequest{Word: "ignored", Topic: &TopicBrief{Name: "Gia đình", Words: []string{"mother", "father"}}})
	if !strings.Contains(p, `represents the topic "Gia đình"`) || !strings.Contains(p, "mother, father") ||
		!strings.Contains(p, "16:9") || strings.Contains(p, "ignored") {
		t.Fatalf("prompt = %s", p)
	}
}
