package tts

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	"golang.org/x/sync/singleflight"

	"github.com/luongtran/luna/backend/internal/platform/httpx"
)

const maxWordChars = 60

var wordText = regexp.MustCompile(`^\p{L}+(?:[ '’-]\p{L}+)*$`)

// ErrInvalidWord means the text is not a word or short phrase.
var ErrInvalidWord = errors.New("tts: invalid word")

// NormalizeWord trims, collapses spaces and lowercases text; only letters, spaces,
// apostrophes and hyphens (1–60 characters) are accepted.
func NormalizeWord(text string) (string, error) {
	w := strings.ToLower(strings.Join(strings.Fields(text), " "))
	if w == "" || utf8.RuneCountInString(w) > maxWordChars || !wordText.MatchString(w) {
		return "", ErrInvalidWord
	}
	return w, nil
}

// WordAudio caches pronunciation audio of words and phrases in dir/words, shared by all
// learners: each text is synthesized once.
type WordAudio struct {
	synth Synthesizer
	dir   string
	group singleflight.Group
}

// NewWordAudio returns a cache in dir/words.
func NewWordAudio(synth Synthesizer, dir string) *WordAudio {
	return &WordAudio{synth: synth, dir: filepath.Join(dir, "words")}
}

// Path returns the mp3 file for text, synthesizing it on first use. Concurrent calls for
// the same text share one synthesis.
func (a *WordAudio) Path(ctx context.Context, text string) (string, error) {
	w, err := NormalizeWord(text)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256([]byte(w))
	// The file name is a hex digest, so user text never reaches the path.
	file := filepath.Join(a.dir, hex.EncodeToString(sum[:])+".mp3")
	if info, err := os.Stat(file); err == nil && info.Size() > 0 { //nolint:gosec // hex digest path
		return file, nil
	}

	_, err, _ = a.group.Do(file, func() (any, error) {
		if info, err := os.Stat(file); err == nil && info.Size() > 0 { //nolint:gosec // hex digest path
			return nil, nil // written by a previous caller
		}
		audio, err := a.synth.Synthesize(ctx, w)
		if err != nil {
			return nil, err
		}
		return nil, writeAtomic(a.dir, file, audio)
	})
	if err != nil {
		return "", err
	}
	return file, nil
}

// Register adds GET /api/tts/word?text=… behind requireAuth.
func (a *WordAudio) Register(mux *http.ServeMux, requireAuth httpx.Middleware) {
	mux.Handle("GET /api/tts/word", requireAuth(http.HandlerFunc(a.serve)))
}

func (a *WordAudio) serve(w http.ResponseWriter, r *http.Request) {
	file, err := a.Path(r.Context(), r.URL.Query().Get("text"))
	switch {
	case errors.Is(err, ErrInvalidWord):
		httpx.WriteFieldErrors(w, map[string]string{"text": "Chỉ phát âm được từ hoặc cụm từ (tối đa 60 ký tự)"})
		return
	case err != nil:
		httpx.WriteError(w, http.StatusServiceUnavailable, "tts_unavailable", "Chưa phát được âm thanh")
		return
	}
	w.Header().Set("Content-Type", "audio/mpeg")
	w.Header().Set("Cache-Control", "private, max-age=31536000, immutable")
	http.ServeFile(w, r, file) //nolint:gosec // file name is a SHA-256 hex digest inside the cache dir
}

func writeAtomic(dir, file string, data []byte) error {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("tts: cache dir: %w", err)
	}
	tmp, err := os.CreateTemp(dir, "*.tmp")
	if err != nil {
		return fmt.Errorf("tts: temp file: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		return fmt.Errorf("tts: write: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmp.Name())
		return fmt.Errorf("tts: close: %w", err)
	}
	if err := os.Rename(tmp.Name(), file); err != nil { //nolint:gosec // hex digest path
		_ = os.Remove(tmp.Name())
		return fmt.Errorf("tts: rename: %w", err)
	}
	return nil
}
