package tts_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/luongtran/luna/backend/internal/tts"
)

func TestKokoroSynthesize(t *testing.T) {
	t.Parallel()

	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/audio/speech" {
			t.Errorf("request %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("decode body: %v", err)
		}
		w.Header().Set("Content-Type", "audio/mpeg")
		_, _ = w.Write([]byte("ID3fake-mp3"))
	}))
	defer srv.Close()

	k := tts.NewKokoro(srv.URL, "af_heart", srv.Client())
	audio, err := k.Synthesize(t.Context(), "He was late!")
	if err != nil {
		t.Fatalf("Synthesize: %v", err)
	}
	if string(audio) != "ID3fake-mp3" {
		t.Fatalf("audio = %q", audio)
	}
	want := map[string]any{"model": "kokoro", "input": "He was late!", "voice": "af_heart", "response_format": "mp3", "speed": 1.0}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("body[%s] = %v, want %v", k, got[k], v)
		}
	}
}

func TestKokoroErrors(t *testing.T) {
	t.Parallel()

	tests := map[string]http.HandlerFunc{
		"server error": func(w http.ResponseWriter, _ *http.Request) { http.Error(w, "boom", http.StatusInternalServerError) },
		"empty body":   func(http.ResponseWriter, *http.Request) {},
		"timeout":      func(http.ResponseWriter, *http.Request) { time.Sleep(300 * time.Millisecond) },
	}
	for name, h := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(h)
			defer srv.Close()

			client := srv.Client()
			client.Timeout = 100 * time.Millisecond
			if _, err := tts.NewKokoro(srv.URL, "af_heart", client).Synthesize(t.Context(), "Hi."); err == nil {
				t.Fatal("Synthesize error = nil")
			}
		})
	}
}
