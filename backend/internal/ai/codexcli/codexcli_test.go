package codexcli

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/luongtran/luna/backend/internal/ai"
)

func pngBytes(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 8, 8))); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

const doneLog = `{"type":"thread.started","thread_id":"t"}
{"type":"item.completed","item":{"id":"i0","type":"agent_message","text":"Saved the picture"}}
{"type":"turn.completed","usage":{"input_tokens":100,"output_tokens":20,"reasoning_output_tokens":5}}
`

var savedAs = regexp.MustCompile(`directory as (\S+\.png)\. `)

// save writes data where the prompt asks Codex to save the picture, as Codex would.
func save(t *testing.T, dir, stdin string, data []byte) {
	t.Helper()
	m := savedAs.FindStringSubmatch(stdin)
	if m == nil {
		t.Fatalf("prompt names no file:\n%s", stdin)
	}
	if err := os.WriteFile(filepath.Join(dir, m[1]), data, 0o600); err != nil {
		t.Fatal(err)
	}
}

type recorder struct{ got []ai.Usage }

func (r *recorder) RecordUsage(_ context.Context, u ai.Usage) { r.got = append(r.got, u) }

func TestGenerateImageReadsTheSavedPictureAndKeepsIt(t *testing.T) {
	pic := pngBytes(t)
	folder := filepath.Join(t.TempDir(), "image")
	var gotArgs []string
	var gotStdin, gotDir string
	usage := &recorder{}
	c := New(Options{Bin: "codex", Dir: folder, Usage: usage, Run: func(_ context.Context, dir, bin string, args []string, stdin string) ([]byte, []byte, error) {
		gotDir, gotArgs, gotStdin = dir, args, stdin
		if bin != "codex" {
			t.Errorf("bin = %q", bin)
		}
		save(t, dir, stdin, pic)
		return []byte(doneLog), nil, nil
	}})
	img, err := c.GenerateImage(context.Background(), ai.ImageRequest{Word: "Good morning", MeaningVi: "chào buổi sáng", Style: "watercolor"})
	if err != nil {
		t.Fatalf("GenerateImage: %v", err)
	}
	if img.MIME != "image/png" || !bytes.Equal(img.Data, pic) {
		t.Errorf("image = %s, %d bytes", img.MIME, len(img.Data))
	}
	if gotDir != folder {
		t.Errorf("work folder = %q, want %q", gotDir, folder)
	}
	for _, want := range []string{"exec", "--json", "--ephemeral", "image_generation", "-C", folder, "-"} {
		if !slices.Contains(gotArgs, want) {
			t.Errorf("args %v miss %q", gotArgs, want)
		}
	}
	for _, want := range []string{`"Good morning"`, "Style: watercolor.", "image_gen", "good-morning-", noImageTool} {
		if !strings.Contains(gotStdin, want) {
			t.Errorf("prompt misses %q:\n%s", want, gotStdin)
		}
	}
	kept, err := filepath.Glob(filepath.Join(folder, "good-morning-*.png"))
	if err != nil || len(kept) != 1 {
		t.Errorf("pictures kept in the image folder = %v, %v", kept, err)
	}
	if len(usage.got) != 1 || usage.got[0].Op != "image" || usage.got[0].Model != ModelName || usage.got[0].PromptTokens != 100 {
		t.Errorf("usage = %+v", usage.got)
	}
}

func TestGenerateImageErrors(t *testing.T) {
	tests := []struct {
		name   string
		stdout string
		stderr string
		runErr error
		want   error
	}{
		{"no tool", `{"type":"item.completed","item":{"type":"agent_message","text":"NO_IMAGE_TOOL"}}`, "", nil, ai.ErrNoImageTool},
		{"no file", doneLog, "", nil, ErrNoImage},
		{"usage limit", `{"type":"error","message":"You've hit your usage limit. Try again later."}`, "", errors.New("exit status 1"), ai.ErrQuota},
		{"not logged in", "", "Not logged in. Run codex login", errors.New("exit status 1"), ai.ErrNotConfigured},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New(Options{Dir: t.TempDir(), Run: func(context.Context, string, string, []string, string) ([]byte, []byte, error) {
				return []byte(tt.stdout), []byte(tt.stderr), tt.runErr
			}})
			_, err := c.GenerateImage(context.Background(), ai.ImageRequest{Word: "apple"})
			if !errors.Is(err, tt.want) {
				t.Errorf("err = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestGenerateImageRejectsNonPicture(t *testing.T) {
	c := New(Options{Dir: t.TempDir(), Run: func(_ context.Context, dir, _ string, _ []string, stdin string) ([]byte, []byte, error) {
		save(t, dir, stdin, []byte("<svg></svg>"))
		return []byte(doneLog), nil, nil
	}})
	if _, err := c.GenerateImage(context.Background(), ai.ImageRequest{Word: "apple"}); err == nil {
		t.Fatal("want an error for a file that is not a picture")
	}
}

func TestGenerateImageTimesOut(t *testing.T) {
	c := New(Options{Dir: t.TempDir(), Timeout: 20 * time.Millisecond, Run: func(ctx context.Context, _, _ string, _ []string, _ string) ([]byte, []byte, error) {
		<-ctx.Done()
		return nil, nil, ctx.Err()
	}})
	_, err := c.GenerateImage(context.Background(), ai.ImageRequest{Word: "apple"})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("err = %v, want deadline exceeded", err)
	}
}

func TestGenerateImageLimitsConcurrencyWithOwnFileNames(t *testing.T) {
	pic := pngBytes(t)
	folder := t.TempDir()
	var running, peak atomic.Int32
	c := New(Options{Dir: folder, Concurrency: 2, Run: func(_ context.Context, dir, _ string, _ []string, stdin string) ([]byte, []byte, error) {
		n := running.Add(1)
		for {
			p := peak.Load()
			if n <= p || peak.CompareAndSwap(p, n) {
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
		running.Add(-1)
		save(t, dir, stdin, pic)
		return []byte(doneLog), nil, nil
	}})
	done := make(chan error, 5)
	for range 5 {
		go func() {
			_, err := c.GenerateImage(context.Background(), ai.ImageRequest{Word: "apple"})
			done <- err
		}()
	}
	for range 5 {
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	}
	if peak.Load() != 2 {
		t.Errorf("peak concurrency = %d, want 2", peak.Load())
	}
	if kept, _ := filepath.Glob(filepath.Join(folder, "apple-*.png")); len(kept) != 5 {
		t.Errorf("pictures kept = %d, want 5 with their own names", len(kept))
	}
}

func TestFileName(t *testing.T) {
	at := time.Date(2026, 11, 4, 10, 42, 33, 0, time.UTC)
	tests := map[string]string{
		"apple":            "apple-20261104-104233-7.png",
		"  Good   morning": "good-morning-20261104-104233-7.png",
		"mother-in-law's":  "mother-in-law-s-20261104-104233-7.png",
		"cà phê":           "c-ph-20261104-104233-7.png",
		"...":              "word-20261104-104233-7.png",
	}
	for word, want := range tests {
		if got := fileName(word, at, 7); got != want {
			t.Errorf("fileName(%q) = %q, want %q", word, got, want)
		}
	}
}

func TestWithBinDirPutsTheCodexFolderFirstInPath(t *testing.T) {
	sep := string(os.PathListSeparator)
	env := []string{"HOME=/root", "PATH=/usr/bin" + sep + "/bin"}
	if got := withBinDir(env, "codex"); !slices.Equal(got, env) {
		t.Fatalf("bare name changed env: %v", got)
	}
	bin := filepath.Join("opt", "node", "bin", "codex")
	want := []string{"HOME=/root", "PATH=" + filepath.Dir(bin) + sep + "/usr/bin" + sep + "/bin"}
	if got := withBinDir(env, bin); !slices.Equal(got, want) {
		t.Fatalf("withBinDir = %v, want %v", got, want)
	}
	if got := withBinDir([]string{"HOME=/root"}, bin); !slices.Equal(got, []string{"HOME=/root", "PATH=" + filepath.Dir(bin)}) {
		t.Fatalf("no PATH: %v", got)
	}
}
