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
			name: "several senses: only the first",
			req:  ImageRequest{Word: "house", MeaningVi: "ngôi nhà; nhà ở; gia đình"},
			want: []string{`"house"`, `"ngôi nhà; nhà ở; gia đình"`, `Show only its first sense, "ngôi nhà".`, "semi-flat illustration", "chibi"},
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
			name:    "the admin's style replaces the default",
			req:     ImageRequest{Word: "run", Style: "watercolor"},
			want:    []string{"Style: watercolor.", "person, an animal or an object", "Do not draw any letters"},
			notWant: []string{"semi-flat"},
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
		"does not answer any of them", "semi-flat illustration", "chibi", "16:9", "no speech bubbles"} {
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
	p = ImagePrompt(ImageRequest{Scene: brief, Style: "watercolor"})
	for _, w := range []string{`Draw this scene: "Linh and Thao sit`, "The story, for context only:", "Style: watercolor."} {
		if !strings.Contains(p, w) {
			t.Errorf("prompt misses %q:\n%s", w, p)
		}
	}
	if strings.Contains(p, "Choose the one moment") {
		t.Errorf("a described scene still asks the model to choose:\n%s", p)
	}
}

func TestPromptBackgroundAndStyle(t *testing.T) {
	word := ImagePrompt(ImageRequest{Word: "apple"})
	if !strings.Contains(word, "Plain light pastel background") || strings.Contains(word, "lighting, on a plain") {
		t.Errorf("word prompt background:\n%s", word)
	}
	scene := ImagePrompt(ImageRequest{Scene: &SceneBrief{Text: "A story."}})
	if !strings.Contains(scene, "Background: the full setting of the scene") || strings.Contains(scene, "plain light pastel background") {
		t.Errorf("scene prompt background:\n%s", scene)
	}
	if strings.Contains(scene, "meant for the word cards") {
		t.Errorf("default style needs no background note:\n%s", scene)
	}
	custom := ImagePrompt(ImageRequest{Scene: &SceneBrief{Text: "A story."}, Style: "Flat, on a plain light background, no text. "})
	if !strings.Contains(custom, "Style: Flat, on a plain light background, no text.\n") {
		t.Errorf("style line:\n%s", custom)
	}
	if !strings.Contains(custom, "meant for the word cards") {
		t.Errorf("custom background not overridden:\n%s", custom)
	}
	if vi := ImagePrompt(ImageRequest{Scene: &SceneBrief{}, Style: "tranh màu nước, nền trắng"}); !strings.Contains(vi, "meant for the word cards") {
		t.Errorf("Vietnamese background not noticed:\n%s", vi)
	}
}
