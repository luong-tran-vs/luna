package gemini_test

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/luongtran/luna/backend/internal/ai"
)

func TestGenerateImage(t *testing.T) {
	t.Parallel()
	var body struct {
		Contents []struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"contents"`
		GenerationConfig map[string]any `json:"generationConfig"`
	}
	png := []byte("\x89PNG fake")
	c, _ := newClient(t, "secret-key", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1beta/models/gemini-test:generateContent" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode: %v", err)
		}
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"Here it is"},` +
			`{"inlineData":{"mimeType":"image/png","data":"` + base64.StdEncoding.EncodeToString(png) + `"}}]}}]}`))
	})

	img, err := c.GenerateImage(t.Context(), ai.ImageRequest{
		Word: "coffee", MeaningVi: "cà phê", Sentence: "I drink coffee.", Style: "watercolor",
	})
	if err != nil {
		t.Fatal(err)
	}
	if img.MIME != "image/png" || string(img.Data) != string(png) {
		t.Fatalf("image = %s %q", img.MIME, img.Data)
	}
	prompt := body.Contents[0].Parts[0].Text
	for _, want := range []string{`"coffee"`, `"cà phê"`, `"I drink coffee."`, "Style: watercolor.", "Do not draw any letters"} {
		if !strings.Contains(prompt, want) {
			t.Errorf("prompt misses %q:\n%s", want, prompt)
		}
	}
	if mods, _ := body.GenerationConfig["responseModalities"].([]any); len(mods) != 1 || mods[0] != "IMAGE" {
		t.Errorf("generationConfig = %v", body.GenerationConfig)
	}
}

func TestGenerateImageDefaultStyleAndErrors(t *testing.T) {
	t.Parallel()
	var prompt string
	c, _ := newClient(t, "k", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Contents []struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"contents"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		prompt = body.Contents[0].Parts[0].Text
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"I cannot draw that."}]}}]}`))
	})
	if _, err := c.GenerateImage(t.Context(), ai.ImageRequest{Word: "run"}); err == nil {
		t.Fatal("a response without a picture should fail")
	}
	if !strings.Contains(prompt, "flat illustration") {
		t.Errorf("default style missing:\n%s", prompt)
	}

	quota, _ := newClient(t, "k", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusTooManyRequests) })
	if _, err := quota.GenerateImage(t.Context(), ai.ImageRequest{Word: "run"}); !errors.Is(err, ai.ErrQuota) {
		t.Fatalf("quota err = %v", err)
	}
	noKey, calls := newClient(t, "", func(http.ResponseWriter, *http.Request) {})
	if _, err := noKey.GenerateImage(t.Context(), ai.ImageRequest{Word: "run"}); !errors.Is(err, ai.ErrNotConfigured) || calls.Load() != 0 {
		t.Fatalf("no key: %v, %d calls", err, calls.Load())
	}
}
