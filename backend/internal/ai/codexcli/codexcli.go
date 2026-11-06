// Package codexcli draws the vocabulary pictures with the Codex CLI (added 2026-11-04): the server
// runs `codex exec` in the image folder (CODEX_IMAGE_DIR), Codex's image_gen tool draws the
// picture with the ChatGPT account Codex is logged in with and saves it there under a name the
// server gives, and the server reads the file back (the file is kept). It is meant for the
// admin's own machine, where Codex is installed and logged in; the other server never runs it.
package codexcli

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/luongtran/luna/backend/internal/ai"
)

// ModelName is what the AI usage page shows for a picture drawn through Codex when no model is set.
const ModelName = "codex-cli"

// noImageTool is what Codex answers when it has no image_gen tool, so it does not draw with code.
const noImageTool = "NO_IMAGE_TOOL"

// maxImageBytes bounds the file read back: a 1536×1024 PNG is a few MB.
const maxImageBytes = 25 << 20

var (
	// ErrNoImage means Codex finished without saving a picture.
	ErrNoImage = errors.New("codex: no picture saved")
)

// Runner starts the command and waits for it; dir is its working folder and stdin its input.
// It is a field so the tests do not need Codex.
type Runner func(ctx context.Context, dir, bin string, args []string, stdin string) (stdout, stderr []byte, err error)

// Options configure a Client.
type Options struct {
	// Bin is the codex program: "codex" from PATH, or a full path to codex.exe.
	Bin string
	// Dir is the folder Codex works in and saves the pictures to (default "image"); the pictures
	// are kept there after being stored in the database.
	Dir string
	// Model, if set, is passed as `-m`; empty uses Codex's default.
	Model string
	// Concurrency is how many pictures may be drawn at once (at least 1).
	Concurrency int
	// Timeout bounds one picture.
	Timeout time.Duration
	Usage   ai.UsageRecorder
	Log     *slog.Logger
	// Run replaces the real command (tests).
	Run Runner
}

// Client is an ai.ImageProvider backed by the Codex CLI.
type Client struct {
	bin     string
	dir     string
	seq     atomic.Int64
	model   string
	timeout time.Duration
	sem     chan struct{}
	usage   ai.UsageRecorder
	log     *slog.Logger
	run     Runner
	now     func() time.Time
}

var _ ai.ImageProvider = (*Client)(nil)

// New returns a Client.
func New(o Options) *Client {
	if o.Concurrency < 1 {
		o.Concurrency = 1
	}
	if o.Timeout <= 0 {
		o.Timeout = 4 * time.Minute
	}
	if o.Bin == "" {
		o.Bin = "codex"
	}
	if o.Dir == "" {
		o.Dir = "image"
	}
	if o.Run == nil {
		o.Run = runCommand
	}
	if o.Log == nil {
		o.Log = slog.Default()
	}
	return &Client{
		bin: o.Bin, dir: o.Dir, model: o.Model, timeout: o.Timeout, sem: make(chan struct{}, o.Concurrency),
		usage: o.Usage, log: o.Log, run: o.Run, now: time.Now,
	}
}

// Available tells whether the codex program can be found, so the server can turn the mode off
// at startup instead of failing at the first picture.
func Available(bin string) error {
	if bin == "" {
		bin = "codex"
	}
	_, err := exec.LookPath(bin)
	return err
}

// GenerateImage runs Codex once for one picture. At most Concurrency pictures are drawn at once;
// the others wait for their turn (or for ctx).
func (c *Client) GenerateImage(ctx context.Context, req ai.ImageRequest) (ai.Image, error) {
	select {
	case c.sem <- struct{}{}:
	case <-ctx.Done():
		return ai.Image{}, ctx.Err()
	}
	defer func() { <-c.sem }()

	dir, err := filepath.Abs(c.dir)
	if err != nil {
		return ai.Image{}, fmt.Errorf("codex: image folder: %w", err)
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return ai.Image{}, fmt.Errorf("codex: make image folder: %w", err)
	}
	// Pictures drawn at once share the folder, so each one has its own name.
	name := fileName(req.Word, c.now(), c.seq.Add(1))

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	start := c.now()
	stdout, stderr, runErr := c.run(ctx, dir, c.bin, c.args(dir), prompt(req, name))
	ev := parseEvents(stdout)
	c.record(ctx, start, ev, runErr)

	if err := classify(ctx, runErr, ev, stderr); err != nil {
		c.log.WarnContext(ctx, "codex: draw picture failed", slog.Int("word_chars", len(req.Word)),
			slog.String("last_message", clip(ev.lastMessage, 300)), slog.String("stderr", clip(string(stderr), 500)),
			slog.Any("error", err))
		return ai.Image{}, err
	}
	img, err := readPicture(filepath.Join(dir, name))
	if err != nil {
		c.log.WarnContext(ctx, "codex: read picture failed", slog.String("last_message", clip(ev.lastMessage, 300)),
			slog.Any("error", err))
		return ai.Image{}, err
	}
	return img, nil
}

func (c *Client) args(dir string) []string {
	args := []string{
		"exec", "--skip-git-repo-check", "--ephemeral", "--json",
		"--sandbox", "workspace-write", "--enable", "image_generation", "-C", dir,
	}
	if c.model != "" {
		args = append(args, "-m", c.model)
	}
	// The prompt is read from stdin: no quoting through cmd.exe on Windows.
	return append(args, "-")
}

// prompt is the picture's description plus how Codex must hand it back: as name in the working
// folder (the image folder).
func prompt(req ai.ImageRequest, name string) string {
	return ai.ImagePrompt(req) + "\n\n" +
		"Use your built-in image_gen tool exactly once to create this picture. " +
		"Then copy the generated image file into the current working directory as " + name + ". " +
		"Never create or edit the picture with code (no Python, PIL, SVG, HTML, canvas or scripts) and do not touch any other file. " +
		"If the image_gen tool is not available to you, reply with only " + noImageTool + " and stop."
}

func (c *Client) record(ctx context.Context, start time.Time, ev events, runErr error) {
	if c.usage == nil {
		return
	}
	model := c.model
	if model == "" {
		model = ModelName
	}
	status := http.StatusOK
	if runErr != nil {
		status = 0
	}
	c.usage.RecordUsage(ctx, ai.Usage{
		At: start, Model: model, Op: "image", Status: status,
		PromptTokens: ev.inputTokens, OutputTokens: ev.outputTokens, ThoughtTokens: ev.reasoningTokens,
		TotalTokens: ev.inputTokens + ev.outputTokens, Duration: c.now().Sub(start),
	})
}

// classify turns a finished run into an error, nil when Codex says it is done.
func classify(ctx context.Context, runErr error, ev events, stderr []byte) error {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return fmt.Errorf("codex: timed out: %w", ctx.Err())
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	all := strings.ToLower(ev.errors + "\n" + string(stderr) + "\n" + ev.lastMessage)
	switch {
	case strings.Contains(all, "usage limit"), strings.Contains(all, "rate limit"):
		return fmt.Errorf("codex: %w", ai.ErrQuota)
	case strings.Contains(all, "not logged in"), strings.Contains(all, "codex login"):
		return fmt.Errorf("codex: not logged in: %w", ai.ErrNotConfigured)
	case strings.Contains(ev.lastMessage, noImageTool):
		return fmt.Errorf("codex: %w", ai.ErrNoImageTool)
	case runErr != nil:
		return fmt.Errorf("codex: run: %w (%s)", runErr, clip(strings.TrimSpace(string(stderr)+" "+ev.errors), 300))
	}
	return nil
}

// readPicture reads the picture Codex saved at path.
func readPicture(path string) (ai.Image, error) {
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return ai.Image{}, ErrNoImage
	}
	if err != nil {
		return ai.Image{}, fmt.Errorf("codex: stat picture: %w", err)
	}
	if info.Size() > maxImageBytes {
		return ai.Image{}, fmt.Errorf("codex: picture too large (%d bytes)", info.Size())
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ai.Image{}, fmt.Errorf("codex: read picture: %w", err)
	}
	mime := http.DetectContentType(data)
	switch mime {
	case "image/png", "image/jpeg", "image/gif":
		return ai.Image{MIME: mime, Data: data}, nil
	default:
		return ai.Image{}, fmt.Errorf("codex: picture is %s, want PNG or JPEG", mime)
	}
}

// events is what the JSONL of `codex exec --json` tells.
type events struct {
	lastMessage     string
	errors          string
	inputTokens     int
	outputTokens    int
	reasoningTokens int
}

func parseEvents(stdout []byte) events {
	var ev events
	sc := bufio.NewScanner(bytes.NewReader(stdout))
	sc.Buffer(make([]byte, 0, 64<<10), 8<<20)
	for sc.Scan() {
		line := bytes.TrimSpace(bytes.TrimPrefix(sc.Bytes(), []byte("\xef\xbb\xbf")))
		if len(line) == 0 || line[0] != '{' {
			continue
		}
		var e struct {
			Type    string `json:"type"`
			Message string `json:"message"`
			Item    *struct {
				Type    string `json:"type"`
				Text    string `json:"text"`
				Message string `json:"message"`
			} `json:"item"`
			Error *struct {
				Message string `json:"message"`
			} `json:"error"`
			Usage *struct {
				InputTokens     int `json:"input_tokens"`
				OutputTokens    int `json:"output_tokens"`
				ReasoningTokens int `json:"reasoning_output_tokens"`
			} `json:"usage"`
		}
		if json.Unmarshal(line, &e) != nil {
			continue
		}
		switch {
		case e.Item != nil && e.Item.Type == "agent_message":
			ev.lastMessage = e.Item.Text
		case e.Item != nil && e.Item.Type == "error":
			ev.errors += e.Item.Message + "\n"
		case e.Type == "error":
			ev.errors += e.Message + "\n"
		case e.Type == "turn.failed" && e.Error != nil:
			ev.errors += e.Error.Message + "\n"
		}
		if e.Usage != nil {
			ev.inputTokens += e.Usage.InputTokens
			ev.outputTokens += e.Usage.OutputTokens
			ev.reasoningTokens += e.Usage.ReasoningTokens
		}
	}
	return ev
}

func runCommand(ctx context.Context, dir, bin string, args []string, stdin string) ([]byte, []byte, error) {
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = dir
	cmd.Env = withBinDir(os.Environ(), bin)
	cmd.Stdin = strings.NewReader(stdin)
	var out, errOut bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errOut
	// codex.cmd starts node, which may outlive the killed shell: stop waiting for its pipes.
	cmd.WaitDelay = 10 * time.Second
	err := cmd.Run()
	return out.Bytes(), errOut.Bytes(), err
}

// withBinDir puts the folder of bin first in PATH when bin is a path (CODEX_BIN=/root/.nvm/.../bin/codex):
// the npm codex is a script run by `node`, which sits in the same folder, and a systemd service
// does not have that folder in its PATH (added 2026-11-04). A bare "codex" leaves env as it is.
func withBinDir(env []string, bin string) []string {
	if !strings.ContainsAny(bin, `/\`) {
		return env
	}
	binDir := filepath.Dir(bin)
	out := make([]string, 0, len(env)+1)
	path := ""
	for _, kv := range env {
		k, v, _ := strings.Cut(kv, "=")
		if strings.EqualFold(k, "PATH") {
			path = v
			continue
		}
		out = append(out, kv)
	}
	if path == "" {
		return append(out, "PATH="+binDir)
	}
	return append(out, "PATH="+binDir+string(os.PathListSeparator)+path)
}

func clip(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// fileName names a picture after its word and when it was drawn, e.g.
// "good-morning-20261104-104233-7.png"; seq tells apart pictures of the same second.
func fileName(word string, at time.Time, seq int64) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(word) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			dash = false
		case !dash && b.Len() > 0:
			b.WriteByte('-')
			dash = true
		}
		if b.Len() >= 40 {
			break
		}
	}
	slug := strings.Trim(b.String(), "-")
	if slug == "" {
		slug = "word"
	}
	return fmt.Sprintf("%s-%s-%d.png", slug, at.Format("20060102-150405"), seq)
}
