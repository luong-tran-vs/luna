package lesson

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"

	"github.com/luongtran/luna/backend/internal/ai"
	"github.com/luongtran/luna/backend/internal/job"
)

// Practice audio kinds, as used in file names and URLs.
const (
	audioExample = "example"
	audioTurn    = "turn"
	audioAnswer  = "answer"
)

// ProcessPractice asks the AI provider once for the vocabulary practice of a lesson (F17),
// keeps the parts that pass the checks and queues their audio. Nothing usable is a permanent
// failure, so a generation never costs more than one successful AI request.
func (s *Service) ProcessPractice(ctx context.Context, j job.Job) error {
	l, ok, err := s.current(ctx, j)
	if !ok {
		return err
	}
	if l.AnnotationStatus != StatusDone || len(l.Annotations) == 0 {
		return job.Permanent(ErrAnnotationNotDone)
	}
	words := practiceWords(l)
	res, err := s.AI.Practice(ctx, ai.PracticeRequest{
		Level: string(l.Level), Title: l.Title, Sentences: sentenceTexts(l.Sentences), Words: words,
	})
	if err != nil {
		if errors.Is(err, ai.ErrNotConfigured) || errors.Is(err, ai.ErrInvalidKey) {
			return job.Permanent(err)
		}
		return err
	}
	p, err := CleanPractice(res, words)
	if err != nil {
		return job.Permanent(err)
	}
	saved, err := s.Lessons.SavePractice(ctx, l.ID, l.Revision, l.PracticeVersion, p)
	if err != nil {
		return fmt.Errorf("lesson: save practice: %w", err)
	}
	if !saved { // content or practice changed meanwhile
		return nil
	}
	// The practice is saved: a failure from here on must not retry this job, which would call the
	// AI again. Without audio the practice still works (the page hides the play buttons).
	if err := s.enqueue(ctx, l.ID, l.Revision, job.TypePracticeAudio); err != nil {
		s.Log.ErrorContext(ctx, "queue practice audio failed", slog.String("lesson_id", l.ID), slog.Any("error", err))
	}
	return nil
}

// ProcessPracticeAudio makes the mp3 of every example, dialogue turn and translation answer of
// the current practice version. Existing files are kept, so a failed job resumes without
// calling the AI again; files of other versions are removed once all are made.
func (s *Service) ProcessPracticeAudio(ctx context.Context, j job.Job) error {
	l, ok, err := s.current(ctx, j)
	if !ok {
		return err
	}
	if l.Practice == nil {
		return nil
	}
	dir := s.practiceVersionDir(l.ID, l.Revision, l.PracticeVersion)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("lesson: practice audio dir: %w", err)
	}
	for _, it := range practiceAudioItems(*l.Practice) {
		file := filepath.Join(dir, it.name+".mp3")
		if info, err := os.Stat(file); err == nil && info.Size() > 0 {
			continue
		}
		audio, err := s.TTS.Synthesize(ctx, it.text)
		if err != nil {
			return fmt.Errorf("lesson: synthesize practice %s: %w", it.name, err)
		}
		if err := writeFileAtomic(dir, file, audio); err != nil {
			return err
		}
	}
	s.removeOtherPracticeVersions(ctx, l.ID, l.Revision, l.PracticeVersion)
	return nil
}

// RegeneratePractice queues a new practice for a lesson whose annotations are done (F17). The
// current practice stays visible until the new one is saved.
func (s *Service) RegeneratePractice(ctx context.Context, id string) (Lesson, error) {
	l, err := s.Lessons.Get(ctx, id)
	if err != nil {
		return Lesson{}, err
	}
	if l.AnnotationStatus != StatusDone || len(l.Annotations) == 0 {
		return Lesson{}, ErrAnnotationNotDone
	}
	if l.PracticeStatus == StatusRunning {
		return Lesson{}, ErrPracticeRunning
	}
	if _, err := s.Lessons.SetStatus(ctx, id, l.Revision, job.TypePractice, StatusRunning, ""); err != nil {
		return Lesson{}, fmt.Errorf("lesson: set practice status: %w", err)
	}
	if err := s.enqueue(ctx, id, l.Revision, job.TypePractice); err != nil {
		return Lesson{}, err
	}
	return s.Lessons.Get(ctx, id)
}

// QueueMissingPractice queues a practice for every lesson annotated before F17, which never got
// one, and returns how many were queued. Each lesson is queued once: its practice status leaves
// StatusNone, so a failed generation is not queued again here (the admin can regenerate it).
func (s *Service) QueueMissingPractice(ctx context.Context) (int, error) {
	refs, err := s.Lessons.WithoutPractice(ctx)
	if err != nil {
		return 0, fmt.Errorf("lesson: lessons without practice: %w", err)
	}
	queued := 0
	for _, r := range refs {
		ok, err := s.Lessons.SetStatus(ctx, r.ID, r.Revision, job.TypePractice, StatusRunning, "")
		if err != nil {
			return queued, fmt.Errorf("lesson: set practice status: %w", err)
		}
		if !ok { // edited meanwhile: the new revision gets its own practice
			continue
		}
		if err := s.enqueue(ctx, r.ID, r.Revision, job.TypePractice); err != nil {
			return queued, err
		}
		queued++
	}
	return queued, nil
}

type practiceAudioItem struct {
	name string
	text string
}

// practiceAudioItems lists the practice texts read aloud, by file name ("turn-2").
func practiceAudioItems(p Practice) []practiceAudioItem {
	var out []practiceAudioItem
	for i, e := range p.Examples {
		out = append(out, practiceAudioItem{audioName(audioExample, i), e.Sentence})
	}
	if p.Dialogue != nil {
		for i, t := range p.Dialogue.Turns {
			out = append(out, practiceAudioItem{audioName(audioTurn, i), t.Text})
		}
	}
	for i, t := range p.Translations {
		out = append(out, practiceAudioItem{audioName(audioAnswer, i), t.En})
	}
	return out
}

func audioName(kind string, i int) string { return kind + "-" + strconv.Itoa(i) }

func (s *Service) practiceDir(id string, revision int) string {
	return filepath.Join(s.revisionDir(id, revision), "practice")
}

func (s *Service) practiceVersionDir(id string, revision, version int) string {
	return filepath.Join(s.practiceDir(id, revision), strconv.Itoa(version))
}

func (s *Service) removeOtherPracticeVersions(ctx context.Context, id string, revision, keep int) {
	dir := s.practiceDir(id, revision)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() && e.Name() != strconv.Itoa(keep) {
			if err := os.RemoveAll(filepath.Join(dir, e.Name())); err != nil {
				s.Log.WarnContext(ctx, "remove old practice audio failed", slog.String("lesson_id", id), slog.Any("error", err))
			}
		}
	}
}

// PracticeAudioURL is the URL the backend serves a practice mp3 at.
func PracticeAudioURL(id string, revision, version int, kind string, index int) string {
	return fmt.Sprintf("/api/audio/%s/%d/practice/%d/%s/%d", id, revision, version, kind, index)
}
