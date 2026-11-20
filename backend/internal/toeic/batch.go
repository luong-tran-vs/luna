package toeic

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"sync"
	"time"

	"github.com/luongtran/luna/backend/internal/ai"
)

// Picture batches: the server draws, one after another in the background, the pictures still
// missing in a test (its Part 1 photos) or in a part of the bank (Part 1 photos; Part 6/7 pictures
// of the texts that carry one on the real test). One batch runs at a time, inside this process, not
// through the job queue, which the local server shares with production (only the local one runs
// Codex). A batch cut short is simply started again: what got a picture is no longer missing.

// PictureParts are the parts whose groups can have pictures.
var PictureParts = []Part{1, 6, 7}

// BatchInput starts a batch: the pictures missing in some groups (Keys), the Part 1 photos missing in
// a test (TestKey), or the pictures missing in a part of the bank (Part 1, 6 or 7).
type BatchInput struct {
	TestKey string `json:"testKey"`
	Part    Part   `json:"part"`
	// Keys, when set, are the groups to draw (the ones "Nhờ AI soạn" has just written).
	Keys   []string `json:"keys"`
	Engine string   `json:"engine"`
}

// PhotoBatchStatus is how the batch is going; the zero value means none ran since the server started.
type PhotoBatchStatus struct {
	Running  bool   `json:"running"`
	Stopping bool   `json:"stopping"`
	Engine   string `json:"engine"`
	TestKey  string `json:"testKey"`
	Part     Part   `json:"part"`
	// Keys are the groups of a batch started for some groups ("Nhờ AI soạn"), empty otherwise.
	Keys  []string `json:"keys"`
	Total int      `json:"total"`
	// Drawing, Saved and the keys of Failed are pictures as "groupKey/slot".
	Drawing string            `json:"drawing"`
	Saved   []string          `json:"saved"`
	Failed  map[string]string `json:"failed"`
	// StopReason is why the batch ended early by itself (no quota, AI not set up...), "" otherwise.
	StopReason string    `json:"stopReason"`
	StartedAt  time.Time `json:"startedAt"`
}

// photoBatch is the one batch of the service.
type photoBatch struct {
	mu     sync.Mutex
	status PhotoBatchStatus
	stop   bool
}

// pictureJob is one picture to draw.
type pictureJob struct {
	group Group
	slot  string
}

func (j pictureJob) id() string { return j.group.Key + "/" + j.slot }

// StartPhotoBatch starts drawing the missing pictures of a test or of a part in the background and
// returns at once with the batch's status.
func (s *Service) StartPhotoBatch(ctx context.Context, in BatchInput, log *slog.Logger) (PhotoBatchStatus, error) {
	if err := checkEngine(in.Engine); err != nil {
		return PhotoBatchStatus{}, err
	}
	if len(in.Keys) > maxBatchKeys {
		return PhotoBatchStatus{}, invalid("keys", fmt.Sprintf("Mỗi lần vẽ tối đa %d cụm", maxBatchKeys))
	}
	if len(in.Keys) == 0 && in.TestKey == "" && !slices.Contains(PictureParts, in.Part) {
		return PhotoBatchStatus{}, invalid("part", "Chỉ Part 1, 6 và 7 có ảnh")
	}
	if s.d.Drawer == nil {
		return PhotoBatchStatus{}, ErrDrawUnavailable
	}
	if in.Engine == EngineCodex && !s.d.Drawer.CodexAvailable() {
		return PhotoBatchStatus{}, ErrCodexUnavailable
	}
	var jobs []pictureJob
	var err error
	if len(in.Keys) > 0 {
		jobs, err = s.missingInGroups(ctx, in.Keys)
	} else {
		jobs, err = s.missingPictures(ctx, in.TestKey, in.Part)
	}
	if err != nil {
		return PhotoBatchStatus{}, err
	}
	b := &s.batch
	b.mu.Lock()
	if b.status.Running {
		b.mu.Unlock()
		return PhotoBatchStatus{}, ErrBatchRunning
	}
	part, keys := in.Part, []string{}
	switch {
	case len(in.Keys) > 0:
		part, keys = 0, slices.Clone(in.Keys)
	case in.TestKey != "":
		part = 0
	}
	b.status = PhotoBatchStatus{Running: len(jobs) > 0, Engine: in.Engine, TestKey: in.TestKey, Part: part, Keys: keys, Total: len(jobs),
		Saved: []string{}, Failed: map[string]string{}, StartedAt: s.now()}
	b.stop = false
	st := b.status
	b.mu.Unlock()
	if len(jobs) > 0 {
		go s.runPhotoBatch(context.WithoutCancel(ctx), jobs, in.Engine, log)
	}
	return st, nil
}

func (s *Service) runPhotoBatch(ctx context.Context, jobs []pictureJob, engine string, log *slog.Logger) {
	b := &s.batch
	defer func() {
		b.mu.Lock()
		b.status.Running, b.status.Stopping, b.status.Drawing = false, false, ""
		b.mu.Unlock()
	}()
	for _, j := range jobs {
		b.mu.Lock()
		if b.stop {
			b.mu.Unlock()
			return
		}
		b.status.Drawing = j.id()
		b.mu.Unlock()

		req, err := imageRequest(j.group, j.slot)
		if err == nil {
			_, err = s.draw(ctx, j.group.Key, j.slot, engine, req)
		}

		b.mu.Lock()
		if err == nil {
			b.status.Saved = append(b.status.Saved, j.id())
		} else {
			b.status.Failed[j.id()] = err.Error()
			if log != nil {
				log.WarnContext(ctx, "toeic picture batch: draw failed", slog.String("picture", j.id()), slog.Any("error", err))
			}
			if batchFatal(err) {
				b.status.StopReason = DrawErrorMessage(err)
				b.mu.Unlock()
				return
			}
		}
		b.mu.Unlock()
	}
}

// PhotoBatch returns how the batch is going.
func (s *Service) PhotoBatch() PhotoBatchStatus {
	b := &s.batch
	b.mu.Lock()
	defer b.mu.Unlock()
	st := b.status
	st.Saved = slices.Clone(st.Saved)
	if st.Saved == nil {
		st.Saved = []string{}
	}
	st.Keys = slices.Clone(st.Keys)
	if st.Keys == nil {
		st.Keys = []string{}
	}
	failed := make(map[string]string, len(st.Failed))
	for k, v := range st.Failed {
		failed[k] = v
	}
	st.Failed = failed
	return st
}

// StopPhotoBatch asks the batch to stop after the picture being drawn.
func (s *Service) StopPhotoBatch() PhotoBatchStatus {
	b := &s.batch
	b.mu.Lock()
	if b.status.Running {
		b.stop, b.status.Stopping = true, true
	}
	b.mu.Unlock()
	return s.PhotoBatch()
}

// MissingPictures counts the pictures a batch of each picture part would draw.
func (s *Service) MissingPictures(ctx context.Context) (map[Part]int, error) {
	out := map[Part]int{}
	for _, p := range PictureParts {
		jobs, err := s.missingPictures(ctx, "", p)
		if err != nil {
			return nil, err
		}
		out[p] = len(jobs)
	}
	return out, nil
}

// missingPictures are the pictures still to draw: the Part 1 photos of a test (testKey), or, in a
// part of the bank, the Part 1 photos and the pictures of the Part 6/7 texts of a picture kind. The
// groups keep their order (a test's, or by key); hidden groups are left out.
func (s *Service) missingPictures(ctx context.Context, testKey string, part Part) ([]pictureJob, error) {
	var groups []Group
	if testKey != "" {
		t, err := s.d.Tests.Get(ctx, testKey)
		if err != nil {
			return nil, err
		}
		byKey, err := s.groupsOf(ctx, t.Groups)
		if err != nil {
			return nil, err
		}
		for _, k := range t.Groups {
			if g, ok := byKey[k]; ok && g.Part == 1 {
				groups = append(groups, g)
			}
		}
	} else {
		all, err := s.d.Groups.List(ctx, GroupFilter{Part: part})
		if err != nil {
			return nil, err
		}
		for _, g := range all {
			if g.Status != StatusHidden {
				groups = append(groups, g)
			}
		}
	}
	keys := make([]string, len(groups))
	for i, g := range groups {
		keys[i] = g.Key
	}
	has := map[string][]string{}
	if len(keys) > 0 {
		var err error
		if has, err = s.d.Images.Has(ctx, keys); err != nil {
			return nil, err
		}
	}
	var out []pictureJob
	for _, g := range groups {
		for _, slot := range batchSlots(g) {
			if !slices.Contains(has[g.Key], slot) {
				out = append(out, pictureJob{group: g, slot: slot})
			}
		}
	}
	return out, nil
}

// batchSlots are the slots of a group a batch draws: the Part 1 photo, or every text of a Part 6/7
// group (emails, memos and chats too: each gets a picture of its situation).
func batchSlots(g Group) []string {
	return ImageSlots(g)
}

// batchFatal says the next pictures would fail the same way, so the batch stops.
func batchFatal(err error) bool {
	return errors.Is(err, ErrDrawUnavailable) || errors.Is(err, ErrCodexUnavailable) || errors.Is(err, ErrCodexNotLoggedIn) ||
		errors.Is(err, ai.ErrNotConfigured) || errors.Is(err, ai.ErrInvalidKey) || errors.Is(err, ai.ErrQuota) ||
		errors.Is(err, ai.ErrNoImageTool)
}

// maxBatchKeys bounds the groups of a batch started for some groups.
const maxBatchKeys = 50

// missingInGroups are the pictures still to draw in the given groups, in their order: the Part 1
// photo, or the pictures of the Part 6/7 texts of a picture kind. Unknown keys are left out.
func (s *Service) missingInGroups(ctx context.Context, keys []string) ([]pictureJob, error) {
	byKey, err := s.groupsOf(ctx, keys)
	if err != nil {
		return nil, err
	}
	has, err := s.d.Images.Has(ctx, keys)
	if err != nil {
		return nil, err
	}
	var out []pictureJob
	seen := map[string]bool{}
	for _, k := range keys {
		g, ok := byKey[k]
		if !ok || seen[k] {
			continue
		}
		seen[k] = true
		for _, slot := range batchSlots(g) {
			if !slices.Contains(has[g.Key], slot) {
				out = append(out, pictureJob{group: g, slot: slot})
			}
		}
	}
	return out, nil
}
