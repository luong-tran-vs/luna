package ai

import (
	"strings"
	"testing"
)

func TestImagePrompt(t *testing.T) {
	tests := []struct {
		name    string
		req     ImageRequest
		want    []string
		notWant []string
	}{
		{
			name:    "several senses: only the first",
			req:     ImageRequest{Word: "house", MeaningVi: "ngôi nhà; nhà ở; gia đình"},
			want:    []string{`"house"`, `"ngôi nhà; nhà ở; gia đình"`, `Show only its first sense, "ngôi nhà".`, "photorealistic photograph", "35mm lens", "Avoid: illustration"},
			notWant: []string{"semi-flat", "chibi", "pastel"},
		},
		{
			name:    "one sense: no sense line",
			req:     ImageRequest{Word: "apple", MeaningVi: "quả táo, trái táo"},
			notWant: []string{"first sense"},
		},
		{
			name:    "a sentence picks the sense",
			req:     ImageRequest{Word: "house", MeaningVi: "ngôi nhà; nhà ở", Sentence: "My house is big."},
			want:    []string{`"My house is big."`, "Show only this sense"},
			notWant: []string{"first sense"},
		},
		{
			name: "the admin's style is an extra direction, the picture stays a photo",
			req:  ImageRequest{Word: "run", Style: "warm evening light. "},
			want: []string{"photorealistic photograph", "Extra direction from the course author", ": warm evening light.\n", "person, an animal or an object", "Do not show any readable letters"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := ImagePrompt(tt.req)
			for _, w := range tt.want {
				if !strings.Contains(p, w) {
					t.Errorf("prompt misses %q:\n%s", w, p)
				}
			}
			for _, w := range tt.notWant {
				if strings.Contains(p, w) {
					t.Errorf("prompt has %q:\n%s", w, p)
				}
			}
		})
	}
}

func TestScenePrompt(t *testing.T) {
	brief := &SceneBrief{
		Title: "Nhà mới của chị Thảo",
		Text:  "Today Linh is at her sister's new house. Bao's bed is blue.",
		Avoid: []string{"What colour is Bao's bed?", " "},
	}
	p := ImagePrompt(ImageRequest{Word: "ignored", Scene: brief})
	for _, w := range []string{`titled "Nhà mới của chị Thảo"`, `The story: "Today Linh`, "- What colour is Bao's bed?\n",
		"does not answer any of them", "photorealistic photograph", "Avoid: illustration", "16:9", "no speech bubbles"} {
		if !strings.Contains(p, w) {
			t.Errorf("prompt misses %q:\n%s", w, p)
		}
	}
	if strings.Contains(p, "ignored") || strings.Contains(p, "- \n") {
		t.Errorf("prompt has the word or an empty question:\n%s", p)
	}
	if got := ImageAspect(ImageRequest{Scene: brief}); got != "16:9" {
		t.Errorf("scene aspect = %q", got)
	}
	if got := ImageAspect(ImageRequest{Word: "apple"}); got != "4:3" {
		t.Errorf("word aspect = %q", got)
	}

	brief.Description = "Linh and Thao sit on a new blue sofa in the living room"
	p = ImagePrompt(ImageRequest{Scene: brief, Style: "a rainy day"})
	for _, w := range []string{`Show this scene: "Linh and Thao sit`, "The story, for context only:", "Extra direction from the course author", ": a rainy day.\n"} {
		if !strings.Contains(p, w) {
			t.Errorf("prompt misses %q:\n%s", w, p)
		}
	}
	if strings.Contains(p, "Choose the one moment") {
		t.Errorf("a described scene still asks the model to choose:\n%s", p)
	}
}

func TestPromptBackground(t *testing.T) {
	word := ImagePrompt(ImageRequest{Word: "apple"})
	if !strings.Contains(word, "seamless studio backdrop") || strings.Contains(word, "pastel") {
		t.Errorf("word prompt background:\n%s", word)
	}
	scene := ImagePrompt(ImageRequest{Scene: &SceneBrief{Text: "A story."}})
	if !strings.Contains(scene, "Background: the full real setting of the scene") || strings.Contains(scene, "studio backdrop") {
		t.Errorf("scene prompt background:\n%s", scene)
	}
	if strings.Contains(scene, "Extra direction") {
		t.Errorf("no style, no extra direction:\n%s", scene)
	}
}

func TestDocPicturePromptShowsTheSituationOfAnEmail(t *testing.T) {
	t.Parallel()
	ad := docPicturePrompt(DocPictureBrief{Kind: "ad", Title: "Summer sale", Text: "Half price on all chairs."})
	mail := docPicturePrompt(DocPictureBrief{Kind: "email", Title: "My order", Text: "Where is my chair?"})
	if !strings.Contains(ad, "main product") || strings.Contains(ad, "situation") {
		t.Fatalf("ad prompt = %s", ad)
	}
	if !strings.Contains(mail, "situation the text is about") || !strings.Contains(mail, "Do not show the document") {
		t.Fatalf("email prompt = %s", mail)
	}
}

func TestGrammarPicturesArePhotos(t *testing.T) {
	for _, ex := range []ExampleBrief{
		{En: "She is reading a book.", Vi: "Cô ấy đang đọc sách.", Point: "Present continuous"},
		{En: "A cat under a table.", Point: "Prepositions of place", Exercise: true},
	} {
		got := ImagePrompt(ImageRequest{Example: &ex, Style: "watercolour"})
		for _, want := range []string{"photorealistic photograph", "35mm lens", "Avoid: illustration", `"` + ex.En + `"`, "4:3"} {
			if !strings.Contains(got, want) {
				t.Errorf("prompt for %q lacks %q:\n%s", ex.En, want, got)
			}
		}
		for _, bad := range []string{"chibi", "semi-flat", "watercolour"} {
			if strings.Contains(got, bad) {
				t.Errorf("prompt for %q still has %q", ex.En, bad)
			}
		}
	}
}
