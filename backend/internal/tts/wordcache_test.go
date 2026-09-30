package tts

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/luongtran/luna/backend/internal/platform/httpx"
)

type countingSynth struct {
	calls atomic.Int32
	err   error
	delay time.Duration
}

func (s *countingSynth) Synthesize(_ context.Context, text string) ([]byte, error) {
	s.calls.Add(1)
	time.Sleep(s.delay)
	if s.err != nil {
		return nil, s.err
	}
	return []byte("mp3:" + text), nil
}

func TestNormalizeWord(t *testing.T) {
	t.Parallel()

	ok := []struct{ in, want string }{
		{"  Give   Up ", "give up"}, {"don't", "don't"}, {"well-known", "well-known"}, {"Café", "café"},
	}
	for _, tt := range ok {
		if got, err := NormalizeWord(tt.in); err != nil || got != tt.want {
			t.Errorf("NormalizeWord(%q) = %q, %v; want %q", tt.in, got, err, tt.want)
		}
	}
	for _, bad := range []string{"", "   ", "abc1", "a/b", "hi!", string(make([]byte, 61))} {
		if _, err := NormalizeWord(bad); err == nil {
			t.Errorf("NormalizeWord(%q) accepted", bad)
		}
	}
}

func TestWordAudioCaches(t *testing.T) {
	t.Parallel()
	synth := &countingSynth{}
	wa := NewWordAudio(synth, t.TempDir())

	p1, err := wa.Path(t.Context(), "go")
	if err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(p1); string(data) != "mp3:go" {
		t.Fatalf("file content = %q", data)
	}
	if filepath.Base(filepath.Dir(p1)) != "words" {
		t.Fatalf("path = %s, want inside words/", p1)
	}
	p2, _ := wa.Path(t.Context(), "go")
	if p1 != p2 || synth.calls.Load() != 1 {
		t.Fatalf("second call synthesized again (%d calls)", synth.calls.Load())
	}
}

func TestWordAudioSingleflight(t *testing.T) {
	t.Parallel()
	synth := &countingSynth{delay: 50 * time.Millisecond}
	wa := NewWordAudio(synth, t.TempDir())

	var wg sync.WaitGroup
	for range 10 {
		wg.Go(func() {
			if _, err := wa.Path(context.Background(), "park"); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	if n := synth.calls.Load(); n != 1 {
		t.Fatalf("synthesized %d times, want 1", n)
	}
}

func TestWordAudioErrorLeavesNoFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	wa := NewWordAudio(&countingSynth{err: errors.New("kokoro down")}, dir)
	if _, err := wa.Path(t.Context(), "go"); err == nil {
		t.Fatal("error = nil")
	}
	files, _ := filepath.Glob(filepath.Join(dir, "words", "*"))
	if len(files) != 0 {
		t.Fatalf("files left: %v", files)
	}
}

func TestWordAudioHandler(t *testing.T) {
	t.Parallel()
	resolve := func(_ context.Context, token string) (httpx.Principal, error) {
		if token == "ok" {
			return httpx.Principal{UserID: "u1"}, nil
		}
		return httpx.Principal{}, errors.New("no session")
	}
	get := func(mux http.Handler, text, token string) *httptest.ResponseRecorder {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/tts/word?text="+url.QueryEscape(text), http.NoBody)
		if token != "" {
			req.AddCookie(&http.Cookie{Name: httpx.SessionCookieName, Value: token})
		}
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		return rec
	}

	mux := http.NewServeMux()
	NewWordAudio(&countingSynth{}, t.TempDir()).Register(mux, httpx.RequireAuth(resolve))
	rec := get(mux, "Give up", "ok")
	if rec.Code != http.StatusOK || rec.Body.String() != "mp3:give up" ||
		rec.Header().Get("Content-Type") != "audio/mpeg" ||
		rec.Header().Get("Cache-Control") != "private, max-age=31536000, immutable" {
		t.Fatalf("ok: %d %q %v", rec.Code, rec.Body, rec.Header())
	}
	if rec := get(mux, "hi!", "ok"); rec.Code != http.StatusBadRequest {
		t.Errorf("invalid text: %d", rec.Code)
	}
	if rec := get(mux, "go", ""); rec.Code != http.StatusUnauthorized {
		t.Errorf("anonymous: %d", rec.Code)
	}

	failing := http.NewServeMux()
	NewWordAudio(&countingSynth{err: errors.New("down")}, t.TempDir()).Register(failing, httpx.RequireAuth(resolve))
	if rec := get(failing, "go", "ok"); rec.Code != http.StatusServiceUnavailable || !strings.Contains(rec.Body.String(), "tts_unavailable") {
		t.Errorf("tts down: %d %s", rec.Code, rec.Body)
	}
}
