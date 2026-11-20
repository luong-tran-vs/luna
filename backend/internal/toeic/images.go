package toeic

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"time"

	"github.com/luongtran/luna/backend/internal/ai"
	"github.com/luongtran/luna/backend/internal/imageutil"
)

// Pictures of the groups (FR-036…FR-039): the Part 1 photo ("photo") and, optionally, a picture for
// each text of Part 6/7 ("doc-1", "doc-2"...). The admin draws them with an engine chosen every time
// (Gemini or Codex, no default), uploads their own, or draws the missing Part 1 photos in a batch.

// Engines draw the pictures.
const (
	EngineGemini = "gemini"
	EngineCodex  = "codex"
)

// Drawer draws pictures with the engine named (internal/service wraps the image providers).
type Drawer interface {
	// Draw returns ErrDrawUnavailable / ErrCodexUnavailable when the engine is not set up here.
	Draw(ctx context.Context, engine string, req ai.ImageRequest) (ai.Image, error)
	CodexAvailable() bool
}

var (
	// ErrDrawUnavailable means the image AI is not configured on this server.
	ErrDrawUnavailable = errors.New("toeic: image AI not configured")
	// ErrCodexUnavailable means the admin chose Codex but this server does not run it.
	ErrCodexUnavailable = errors.New("toeic: codex images not enabled on this server")
	// ErrCodexNotLoggedIn means codex runs but is not logged in.
	ErrCodexNotLoggedIn = errors.New("toeic: codex not logged in")
	// ErrDrawFailed means the AI answered without a usable picture.
	ErrDrawFailed = errors.New("toeic: drawing failed")
	// ErrBatchRunning means a photo batch is running already; one runs at a time.
	ErrBatchRunning = errors.New("toeic: a photo batch is running")
)

// MaxUploadBytes bounds an uploaded picture.
const MaxUploadBytes = 5 << 20

// imageSide is the longest side a picture is stored with.
const imageSide = 800

// drawTimeout bounds one picture.
const drawTimeout = 5 * time.Minute

// DocSlot is the picture slot of the text at index i of a group: the one the file names, else
// "doc-<i+1>".
func DocSlot(d Doc, i int) string {
	if d.ImageSlot != "" {
		return d.ImageSlot
	}
	return "doc-" + strconv.Itoa(i+1)
}

// ImageSlots are the slots a group can have a picture in.
func ImageSlots(g Group) []string {
	if g.Part == 1 {
		return []string{SlotPhoto}
	}
	out := []string{}
	for i, d := range g.Docs {
		out = append(out, DocSlot(d, i))
	}
	return out
}

// imageRequest is what a slot of a group is drawn from: for the photo, the scene and the right
// statement only (never the wrong ones, FR-037); for a text, the text itself.
func imageRequest(g Group, slot string) (ai.ImageRequest, error) {
	if slot == SlotPhoto && g.Part == 1 {
		brief := ai.PhotoBrief{Scene: g.Scene}
		if len(g.Questions) > 0 {
			q := g.Questions[0]
			if q.Answer >= 0 && q.Answer < len(q.Options) {
				brief.Statement = q.Options[q.Answer]
			}
		}
		return ai.ImageRequest{Photo: &brief}, nil
	}
	for i, d := range g.Docs {
		if DocSlot(d, i) == slot {
			return ai.ImageRequest{DocPicture: &ai.DocPictureBrief{Kind: d.Kind, Title: d.Title, Text: docText(d)}}, nil
		}
	}
	return ai.ImageRequest{}, invalid("slot", "Cụm này không có ô ảnh "+slot)
}

// checkEngine refuses a missing or unknown engine: the admin chooses every time.
func checkEngine(engine string) error {
	if engine != EngineGemini && engine != EngineCodex {
		return invalid("engine", "Chọn Gemini hoặc Codex để vẽ ảnh")
	}
	return nil
}

// DrawImage draws the picture of a slot of a group now (one AI request) and stores it, replacing the
// one there was.
func (s *Service) DrawImage(ctx context.Context, groupKey, slot, engine string) (Image, error) {
	if err := checkEngine(engine); err != nil {
		return Image{}, err
	}
	g, err := s.GetGroup(ctx, groupKey)
	if err != nil {
		return Image{}, err
	}
	req, err := imageRequest(g, slot)
	if err != nil {
		return Image{}, err
	}
	return s.draw(ctx, g.Key, slot, engine, req)
}

func (s *Service) draw(ctx context.Context, key, slot, engine string, req ai.ImageRequest) (Image, error) {
	if s.d.Drawer == nil {
		return Image{}, ErrDrawUnavailable
	}
	dctx, cancel := context.WithTimeout(ctx, drawTimeout)
	defer cancel()
	pic, err := s.d.Drawer.Draw(dctx, engine, req)
	if engine == EngineCodex && errors.Is(err, ai.ErrNotConfigured) {
		return Image{}, fmt.Errorf("%w: %w", ErrCodexNotLoggedIn, err)
	}
	switch {
	case errors.Is(err, ErrDrawUnavailable), errors.Is(err, ErrCodexUnavailable), errors.Is(err, ai.ErrNotConfigured),
		errors.Is(err, ai.ErrInvalidKey), errors.Is(err, ai.ErrQuota), errors.Is(err, ai.ErrNoImageTool),
		errors.Is(err, context.DeadlineExceeded):
		return Image{}, err
	case err != nil:
		return Image{}, fmt.Errorf("%w: %w", ErrDrawFailed, err)
	}
	data, err := imageutil.Shrink(pic.Data, imageSide)
	if err != nil {
		return Image{}, fmt.Errorf("%w: %w", ErrDrawFailed, err)
	}
	img := Image{GroupKey: key, Slot: slot, MIME: http.DetectContentType(data), Data: data, Origin: engine, CreatedAt: s.now()}
	if err := s.d.Images.Save(ctx, img); err != nil {
		return Image{}, err
	}
	return img, nil
}

// UploadImage stores the admin's own picture for a slot (JPEG, PNG, GIF or WebP, at most 5 MB),
// shrunk like a drawn one.
func (s *Service) UploadImage(ctx context.Context, groupKey, slot string, data []byte) (Image, error) {
	g, err := s.GetGroup(ctx, groupKey)
	if err != nil {
		return Image{}, err
	}
	if !slices.Contains(ImageSlots(g), slot) {
		return Image{}, invalid("slot", "Cụm này không có ô ảnh "+slot)
	}
	if len(data) == 0 || len(data) > MaxUploadBytes {
		return Image{}, invalid("image", "Ảnh tối đa 5 MB")
	}
	mime := http.DetectContentType(data)
	if !slices.Contains([]string{"image/jpeg", "image/png", "image/gif", "image/webp"}, mime) {
		return Image{}, invalid("image", "Chỉ nhận ảnh JPEG, PNG, GIF hoặc WebP")
	}
	small, err := imageutil.Shrink(data, imageSide)
	if err != nil {
		return Image{}, invalid("image", "Không đọc được ảnh này")
	}
	img := Image{GroupKey: g.Key, Slot: slot, MIME: http.DetectContentType(small), Data: small, Origin: "upload", CreatedAt: s.now()}
	if err := s.d.Images.Save(ctx, img); err != nil {
		return Image{}, err
	}
	return img, nil
}

// DeleteImage removes the picture of a slot.
func (s *Service) DeleteImage(ctx context.Context, groupKey, slot string) error {
	if _, err := s.GetGroup(ctx, groupKey); err != nil {
		return err
	}
	return s.d.Images.Delete(ctx, groupKey, slot)
}

// SlotState is a picture slot of a group in the admin's view.
type SlotState struct {
	Slot  string `json:"slot"`
	Label string `json:"label"`
	Has   bool   `json:"has"`
	URL   string `json:"url"`
}

// GroupImages returns the picture slots of a group and which ones have a picture.
func (s *Service) GroupImages(ctx context.Context, groupKey string) ([]SlotState, error) {
	g, err := s.GetGroup(ctx, groupKey)
	if err != nil {
		return nil, err
	}
	has, err := s.d.Images.Has(ctx, []string{g.Key})
	if err != nil {
		return nil, err
	}
	out := []SlotState{}
	for i, slot := range ImageSlots(g) {
		label := "Ảnh Part 1"
		if g.Part != 1 {
			label = fmt.Sprintf("Văn bản %d", i+1)
			if t := g.Docs[i].Title; t != "" {
				label += ": " + t
			}
		}
		out = append(out, SlotState{Slot: slot, Label: label, Has: slices.Contains(has[g.Key], slot), URL: *imageURL(g.Key, slot)})
	}
	return out, nil
}

// DrawErrorMessage is the Vietnamese message of a failed drawing.
func DrawErrorMessage(err error) string {
	_, _, msg := drawErrorOf(err)
	if msg == "" {
		return "Sinh ảnh thất bại, vui lòng thử lại."
	}
	return msg
}

// drawErrorOf maps a drawing error to an HTTP status, code and message; status 0 when it is not one.
func drawErrorOf(err error) (int, string, string) {
	switch {
	case errors.Is(err, ErrCodexUnavailable):
		return http.StatusServiceUnavailable, "codex_unavailable", "Máy chủ này không bật sinh ảnh bằng Codex CLI (CODEX_IMAGES)."
	case errors.Is(err, ErrCodexNotLoggedIn):
		return http.StatusServiceUnavailable, "codex_unavailable", "Codex CLI trên máy chủ chưa đăng nhập. Chạy codex login rồi thử lại."
	case errors.Is(err, ai.ErrNoImageTool):
		return http.StatusBadGateway, "codex_no_image_tool", "Codex CLI không có công cụ tạo ảnh (image_gen). Hãy dùng Gemini."
	case errors.Is(err, ErrDrawUnavailable), errors.Is(err, ai.ErrNotConfigured):
		return http.StatusServiceUnavailable, "ai_not_configured", "AI vẽ ảnh chưa được cấu hình."
	case errors.Is(err, ai.ErrInvalidKey):
		return http.StatusServiceUnavailable, "ai_not_configured", "Khoá AI không hợp lệ."
	case errors.Is(err, ai.ErrQuota):
		return http.StatusTooManyRequests, "ai_quota", "Đã hết lượt AI, vui lòng thử lại sau."
	case errors.Is(err, ErrDrawFailed), errors.Is(err, context.DeadlineExceeded):
		return http.StatusBadGateway, "ai_failed", "Sinh ảnh thất bại, vui lòng thử lại."
	case errors.Is(err, ErrBatchRunning):
		return http.StatusConflict, "batch_running", "Đang vẽ một loạt ảnh, đợi xong rồi thử lại."
	}
	return 0, "", ""
}
