package lesson

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/luongtran/luna/backend/internal/ai"
	"github.com/luongtran/luna/backend/internal/job"
)

const maxAnnotations = 100

// Deps are the collaborators of Service.
type Deps struct {
	Lessons Repository
	Topics  Topics
	Jobs    job.Repository
	AI      ai.Provider
	// Dict gives the meaning of topic words the AI did not annotate (F18); may be nil.
	Dict Dictionary
	// Notify wakes the job worker after jobs are enqueued; may be nil.
	Notify func()
	Now    func() time.Time
	Log    *slog.Logger
	// GenerateTimeout bounds the AI call of a generation batch; 0 means 60 seconds.
	GenerateTimeout time.Duration
	// ImageStore keeps the word pictures (F23); nil turns them off.
	ImageStore ImageRepository
	// ImageAI draws the word pictures; nil fails every drawing.
	ImageAI ai.ImageProvider
	// FetchImage downloads a picture from a link an admin pasted (NewImageFetcher); nil turns links off.
	FetchImage ImageFetcher
}

// Service implements lesson management and the background work on lessons.
type Service struct {
	Deps
}

// NewService returns a Service. Notify defaults to a no-op.
func NewService(d Deps) *Service {
	if d.Notify == nil {
		d.Notify = func() {}
	}
	return &Service{Deps: d}
}

// Create validates and stores a lesson, then queues its annotation.
func (s *Service) Create(ctx context.Context, in Input) (Lesson, error) {
	in, sentences, err := ValidateInput(in)
	if err != nil {
		return Lesson{}, err
	}
	topic, err := s.topic(ctx, in.TopicID)
	if err != nil {
		return Lesson{}, err
	}
	if err := checkGrammarPoint(in.GrammarPointID, in.Level); err != nil {
		return Lesson{}, err
	}
	var images ImageInput
	if in.Images != nil && in.Images.Enabled {
		if s.ImageStore == nil {
			return Lesson{}, ErrImagesUnavailable
		}
		if images, err = validateImageInput(*in.Images); err != nil {
			return Lesson{}, err
		}
	}
	now := s.Now()
	l, err := s.Lessons.Create(ctx, Lesson{
		Title: in.Title, Content: in.Content, Level: in.Level, TopicID: topic.ID,
		Source: in.Source, License: in.License, GrammarPointID: in.GrammarPointID, TargetWords: in.TargetWords,
		Revision:         1,
		Sentences:        toSentences(sentences),
		AnnotationStatus: StatusRunning,
		Draft:            in.Draft,
		CreatedAt:        now,
		UpdatedAt:        now,
	})
	if err != nil {
		return Lesson{}, fmt.Errorf("lesson: create: %w", err)
	}
	if in.AppendToRoadmap {
		if err := s.appendToRoadmap(ctx, l); err != nil {
			return Lesson{}, err
		}
	}
	// Saved before the annotation is queued: the drawing starts once the words are known (F23).
	if images.Enabled {
		if err := s.applyImages(ctx, l, images); err != nil {
			return Lesson{}, err
		}
	}
	if err := s.enqueue(ctx, l.ID, l.Revision, job.TypeAnnotate); err != nil {
		return Lesson{}, err
	}
	return l, nil
}

// appendToRoadmap adds a just-created lesson at the end of its topic roadmap. MongoDB runs
// without transactions, so on failure the lesson is deleted again: a saved draft is either in
// the roadmap or not saved at all.
func (s *Service) appendToRoadmap(ctx context.Context, l Lesson) error {
	err := s.Topics.AppendLesson(ctx, Place{TopicID: l.TopicID, Level: l.Level}, l.ID)
	if err == nil {
		return nil
	}
	if derr := s.Lessons.Delete(ctx, l.ID); derr != nil {
		s.Log.ErrorContext(ctx, "lesson: remove lesson after failed roadmap append",
			slog.String("lesson", l.ID), slog.Any("error", derr))
	}
	return fmt.Errorf("lesson: append to roadmap: %w", err)
}

// Get returns a lesson or ErrNotFound.
func (s *Service) Get(ctx context.Context, id string) (Lesson, error) {
	return s.Lessons.Get(ctx, id)
}

// topic returns the topic of a lesson being saved; an unknown topic is a field error.
func (s *Service) topic(ctx context.Context, id string) (TopicRef, error) {
	t, err := s.Topics.Get(ctx, id)
	if errors.Is(err, ErrTopicNotFound) {
		return TopicRef{}, &ValidationError{Fields: map[string]string{"topicId": "Chủ đề không tồn tại"}}
	}
	if err != nil {
		return TopicRef{}, fmt.Errorf("lesson: topic: %w", err)
	}
	return t, nil
}

// List returns lesson summaries, newest first, with their topic name and whether they are in
// their topic's roadmap.
func (s *Service) List(ctx context.Context, f Filter) ([]Summary, error) {
	if f.Level != "" && !ValidLevel(string(f.Level)) {
		return nil, &ValidationError{Fields: map[string]string{"level": "Trình độ không hợp lệ"}}
	}
	items, err := s.Lessons.List(ctx, f)
	if err != nil {
		return nil, fmt.Errorf("lesson: list: %w", err)
	}
	names, err := s.Topics.Names(ctx)
	if err != nil {
		return nil, fmt.Errorf("lesson: topic names: %w", err)
	}
	inRoadmap, err := s.Topics.RoadmapLessonIDs(ctx)
	if err != nil {
		return nil, fmt.Errorf("lesson: roadmap: %w", err)
	}
	for i := range items {
		items[i].TopicName = names[items[i].TopicID].Name
		items[i].InRoadmap = inRoadmap[items[i].ID]
	}
	return items, nil
}

// InRoadmap reports whether a lesson is in its topic's roadmap.
func (s *Service) InRoadmap(ctx context.Context, id string) (bool, error) {
	ids, err := s.Topics.RoadmapLessonIDs(ctx)
	if err != nil {
		return false, fmt.Errorf("lesson: roadmap: %w", err)
	}
	return ids[id], nil
}

// TopicName is the name of a topic, "" when it does not exist.
func (s *Service) TopicName(ctx context.Context, id string) (string, error) {
	t, err := s.Topics.Get(ctx, id)
	if errors.Is(err, ErrTopicNotFound) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("lesson: topic: %w", err)
	}
	return t.Name, nil
}

// Update edits a lesson. Changing the content re-splits it and redoes the annotations (then the
// practice); other fields keep the existing sentences and annotations. Changing the topic moves
// the lesson to the end of the new topic's roadmap if it was in the old one.
func (s *Service) Update(ctx context.Context, id string, in Input) (Lesson, error) {
	cur, err := s.Lessons.Get(ctx, id)
	if err != nil {
		return Lesson{}, err
	}
	in, sentences, err := ValidateInput(in)
	if err != nil {
		return Lesson{}, err
	}
	topic, err := s.topic(ctx, in.TopicID)
	if err != nil {
		return Lesson{}, err
	}
	if in.KeepGrammarPoint {
		in.GrammarPointID = cur.GrammarPointID
	}
	if err := checkGrammarPoint(in.GrammarPointID, in.Level); err != nil {
		return Lesson{}, err
	}
	info := Info{
		Title: in.Title, Level: in.Level, TopicID: topic.ID, Source: in.Source, License: in.License,
		GrammarPointID: in.GrammarPointID,
	}
	from, to := Place{TopicID: cur.TopicID, Level: cur.Level}, Place{TopicID: topic.ID, Level: in.Level}
	if from != to {
		if err := s.Topics.MoveLesson(ctx, id, from, to); err != nil {
			return Lesson{}, fmt.Errorf("lesson: move to topic: %w", err)
		}
	}

	if in.Content == cur.Content {
		if err := s.Lessons.UpdateInfo(ctx, id, info); err != nil {
			return Lesson{}, fmt.Errorf("lesson: update info: %w", err)
		}
		return s.Lessons.Get(ctx, id)
	}

	next := cur
	next.Title, next.Level, next.TopicID, next.Source, next.License = info.Title, info.Level, info.TopicID, info.Source, info.License
	next.GrammarPointID = info.GrammarPointID
	next.Content = in.Content
	next.Revision = cur.Revision + 1
	next.Sentences = toSentences(sentences)
	next.Annotations = nil
	next.Extras, next.ExtrasEditedByAdmin = Extras{}, false
	next.QuizVersion = cur.QuizVersion + 1 // old answers belong to the old text
	next.AnnotationStatus, next.AnnotationError = StatusRunning, ""
	next.Practice, next.PracticeStatus, next.PracticeError = nil, StatusNone, "" // redone after the annotations
	next.Review = nil                                                            // its flags point into the old text
	next.UpdatedAt = s.Now()
	if err := s.Lessons.ReplaceContent(ctx, next); err != nil {
		return Lesson{}, fmt.Errorf("lesson: replace content: %w", err)
	}
	if err := s.Jobs.DeletePending(ctx, id); err != nil {
		return Lesson{}, fmt.Errorf("lesson: drop old jobs: %w", err)
	}
	if err := s.enqueue(ctx, id, next.Revision, job.TypeAnnotate); err != nil {
		return Lesson{}, err
	}
	return next, nil
}

// Delete removes a lesson that is not in its topic's roadmap, with its jobs.
func (s *Service) Delete(ctx context.Context, id string) error {
	if _, err := s.Lessons.Get(ctx, id); err != nil {
		return err
	}
	if in, err := s.InRoadmap(ctx, id); err != nil {
		return err
	} else if in {
		return ErrInRoadmap
	}
	if err := s.Lessons.Delete(ctx, id); err != nil {
		return fmt.Errorf("lesson: delete: %w", err)
	}
	if err := s.Jobs.DeleteForLesson(ctx, id); err != nil {
		return fmt.Errorf("lesson: delete jobs: %w", err)
	}
	if s.ImageStore != nil {
		if err := s.ImageStore.DeleteForLesson(ctx, id); err != nil {
			return fmt.Errorf("lesson: delete images: %w", err)
		}
	}
	return nil
}

// SetPublished shows the lesson to learners (published) or hides it again as a draft.
func (s *Service) SetPublished(ctx context.Context, id string, published bool) (Lesson, error) {
	if err := s.Lessons.SetDraft(ctx, id, !published); err != nil {
		if errors.Is(err, ErrNotFound) {
			return Lesson{}, err
		}
		return Lesson{}, fmt.Errorf("lesson: set draft: %w", err)
	}
	return s.Lessons.Get(ctx, id)
}

// Retry queues the annotation again, also when done, so lessons from before F15 get their
// questions (it replaces admin edits, which the admin page warns about).
func (s *Service) Retry(ctx context.Context, id string, t job.Type) (Lesson, error) {
	l, err := s.Lessons.Get(ctx, id)
	if err != nil {
		return Lesson{}, err
	}
	if l.StatusOf(t) == StatusRunning {
		return Lesson{}, ErrAnnotationRunning
	}
	if _, err := s.Lessons.SetStatus(ctx, id, l.Revision, t, StatusRunning, ""); err != nil {
		return Lesson{}, fmt.Errorf("lesson: set status: %w", err)
	}
	if err := s.enqueue(ctx, id, l.Revision, t); err != nil {
		return Lesson{}, err
	}
	return s.Lessons.Get(ctx, id)
}

// AnnotationInput is one annotation as edited by an admin.
type AnnotationInput struct {
	Text      string
	Lemma     string
	MeaningVi string
}

// UpdateAnnotations replaces the annotations with an admin-edited list. New items and items
// whose lemma or meaning changed are marked as edited by the admin.
func (s *Service) UpdateAnnotations(ctx context.Context, id string, items []AnnotationInput) (Lesson, error) {
	l, err := s.Lessons.Get(ctx, id)
	if err != nil {
		return Lesson{}, err
	}
	if l.AnnotationStatus == StatusRunning {
		return Lesson{}, ErrAnnotationRunning
	}
	if len(items) > maxAnnotations {
		return Lesson{}, &ValidationError{Fields: map[string]string{
			"annotations": fmt.Sprintf("Tối đa %d chú thích", maxAnnotations),
		}}
	}

	sentences := sentenceTexts(l.Sentences)
	existing := map[string]Annotation{}
	for _, a := range l.Annotations {
		existing[strings.ToLower(a.Text)] = a
	}

	fields := map[string]string{}
	out := make([]Annotation, 0, len(items))
	for i, it := range items {
		a := Annotation{
			Text:      strings.TrimSpace(it.Text),
			Lemma:     strings.TrimSpace(it.Lemma),
			MeaningVi: strings.TrimSpace(it.MeaningVi),
		}
		key := fmt.Sprintf("annotations.%d.", i)
		switch {
		case a.Text == "":
			fields[key+"text"] = "Vui lòng nhập từ hoặc cụm từ"
		case utf8.RuneCountInString(a.Text) > 100:
			fields[key+"text"] = "Tối đa 100 ký tự"
		}
		if a.Lemma == "" {
			fields[key+"lemma"] = "Vui lòng nhập dạng gốc"
		}
		if a.MeaningVi == "" {
			fields[key+"meaningVi"] = "Vui lòng nhập nghĩa"
		}
		if a.Text == "" {
			continue
		}
		prev, had := existing[strings.ToLower(a.Text)]
		idx, ok := findSentence(a.Text, prev.SentenceIndex, sentences)
		if !ok {
			fields[key+"text"] = "Cụm từ không có trong bài"
			continue
		}
		a.SentenceIndex = idx
		a.EditedByAdmin = !had || prev.EditedByAdmin || prev.Lemma != a.Lemma || prev.MeaningVi != a.MeaningVi
		out = append(out, a)
	}
	if len(fields) > 0 {
		return Lesson{}, &ValidationError{Fields: fields}
	}
	if err := s.Lessons.ReplaceAnnotations(ctx, id, out); err != nil {
		return Lesson{}, fmt.Errorf("lesson: save annotations: %w", err)
	}
	s.keepReview(ctx, l, AreaAnnotation, annotationKeys(l.Annotations), annotationKeys(out))
	// New words get their picture too (F23).
	if err := s.queueImages(ctx, id, l.Revision); err != nil {
		return Lesson{}, err
	}
	return s.Lessons.Get(ctx, id)
}

// ProcessAnnotate asks the AI provider once for the whole lesson and stores the cleaned
// annotations with the questions, grammar note and writing prompt (F15). Topic words found in
// the lesson are sent along, and the ones the AI skipped are added from the dictionary (F18).
func (s *Service) ProcessAnnotate(ctx context.Context, j job.Job) error {
	l, ok, err := s.current(ctx, j)
	if !ok {
		return err
	}
	// A lesson generated around target words teaches exactly those found in it; when none is
	// (the content was rewritten), it is annotated like any other lesson of its topic.
	focus, only := focusWords(l.Content, l.TargetWords), true
	if len(focus) == 0 {
		if focus, err = s.topicFocus(ctx, l); err != nil {
			return err
		}
		only = false
	}
	res, err := s.AI.Annotate(ctx, ai.AnnotateRequest{Sentences: sentenceTexts(l.Sentences), Level: string(l.Level), FocusWords: focus,
		OnlyFocus: only, GrammarFocus: grammarFocus(l.GrammarPointID)})
	if err != nil {
		if errors.Is(err, ai.ErrNotConfigured) || errors.Is(err, ai.ErrInvalidKey) {
			return job.Permanent(err)
		}
		return err
	}
	anns, err := CleanAnnotations(res.Annotations, sentenceTexts(l.Sentences))
	if only {
		// The AI may still add words of its own, or give nothing usable: keep only the targets
		// and fill the gaps from the dictionary.
		anns, err = onlyFocus(anns, focus), nil
	}
	if err != nil {
		return err
	}
	if anns, err = addMissedFocus(ctx, anns, focus, sentenceTexts(l.Sentences), s.Dict); err != nil {
		return err
	}
	extras := CleanExtras(res, l.Content)
	// The note is about the point the lesson was given; its title is the syllabus's, whatever the AI wrote.
	if title := GrammarTitle(l.GrammarPointID); title != "" && extras.GrammarNote != nil {
		extras.GrammarNote.Title = title
	}
	saved, err := s.Lessons.SaveAnnotations(ctx, l.ID, l.Revision, anns, extras)
	if err != nil {
		return fmt.Errorf("lesson: save annotations: %w", err)
	}
	if !saved {
		return nil
	}
	// SaveAnnotations dropped the practice of the old annotations; write a new one (F17).
	if err := s.enqueue(ctx, l.ID, l.Revision, job.TypePractice); err != nil {
		return err
	}
	return s.queueImages(ctx, l.ID, l.Revision)
}

// JobFailed records a job that gave up on the lesson, with a short Vietnamese reason.
func (s *Service) JobFailed(ctx context.Context, j job.Job, err error) {
	if j.Type == job.TypeImages {
		s.imagesFailed(ctx, j, err)
		return
	}
	if j.Type != job.TypeAnnotate && j.Type != job.TypePractice {
		// Jobs of removed kinds (audio before Kokoro was dropped) have nothing to record.
		s.Log.WarnContext(ctx, "job of an unknown kind failed", slog.String("type", string(j.Type)), slog.Any("error", err))
		return
	}
	if _, serr := s.Lessons.SetStatus(ctx, j.LessonID, j.Revision, j.Type, StatusFailed, failureMessage(j.Type, err)); serr != nil &&
		!errors.Is(serr, ErrNotFound) {
		s.Log.ErrorContext(ctx, "record job failure", slog.String("lesson_id", j.LessonID), slog.Any("error", serr))
	}
}

func failureMessage(t job.Type, err error) string {
	switch {
	case errors.Is(err, ai.ErrNotConfigured):
		return "AI chưa được cấu hình"
	case errors.Is(err, ai.ErrInvalidKey):
		return "Khoá API của AI không hợp lệ"
	case errors.Is(err, ai.ErrQuota):
		return "AI hết hạn mức, thử lại sau"
	case errors.Is(err, ErrNoValidAnnotations):
		return "AI không trả về chú thích hợp lệ"
	case errors.Is(err, ErrNoValidPractice):
		return "AI không trả về phần luyện tập hợp lệ"
	case errors.Is(err, ErrAnnotationNotDone):
		return "Bài chưa có chú thích"
	case t == job.TypePractice:
		return "Không tạo được phần luyện tập"
	case t == job.TypeImages:
		return "Không sinh được ảnh cho một số từ"
	default:
		return "Không chú thích được bài"
	}
}

// current loads the job's lesson. ok is false (with a nil error) when the job is stale,
// and false with a permanent error when the lesson no longer exists.
func (s *Service) current(ctx context.Context, j job.Job) (Lesson, bool, error) {
	l, err := s.Lessons.Get(ctx, j.LessonID)
	if errors.Is(err, ErrNotFound) {
		return Lesson{}, false, job.Permanent(err)
	}
	if err != nil {
		return Lesson{}, false, fmt.Errorf("lesson: load: %w", err)
	}
	if l.Revision != j.Revision {
		return Lesson{}, false, nil
	}
	return l, true, nil
}

func (s *Service) enqueue(ctx context.Context, id string, revision int, types ...job.Type) error {
	now := s.Now()
	for _, t := range types {
		err := s.Jobs.Enqueue(ctx, job.Job{
			Type: t, LessonID: id, Revision: revision, Status: job.StatusPending,
			RunAt: now, CreatedAt: now, UpdatedAt: now,
		})
		if err != nil {
			return fmt.Errorf("lesson: enqueue %s: %w", t, err)
		}
	}
	s.Notify()
	return nil
}

func toSentences(texts []string) []Sentence {
	out := make([]Sentence, len(texts))
	for i, t := range texts {
		out[i] = Sentence{Index: i, Text: t}
	}
	return out
}

func sentenceTexts(ss []Sentence) []string {
	out := make([]string, len(ss))
	for i, s := range ss {
		out[i] = s.Text
	}
	return out
}

// topicFocus returns the words of the lesson's topic found in its content (F18); none when the
// topic no longer exists.
func (s *Service) topicFocus(ctx context.Context, l Lesson) ([]string, error) {
	t, err := s.Topics.Get(ctx, l.TopicID)
	if errors.Is(err, ErrTopicNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("lesson: topic: %w", err)
	}
	return focusWords(l.Content, t.WordTexts()), nil
}
