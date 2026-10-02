// Command api runs the Luna HTTP backend.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"syscall"
	"time"

	"github.com/luongtran/luna/backend/internal/ai"
	"github.com/luongtran/luna/backend/internal/ai/gemini"
	"github.com/luongtran/luna/backend/internal/auth"
	"github.com/luongtran/luna/backend/internal/dictionary"
	"github.com/luongtran/luna/backend/internal/export"
	"github.com/luongtran/luna/backend/internal/health"
	"github.com/luongtran/luna/backend/internal/job"
	"github.com/luongtran/luna/backend/internal/lesson"
	"github.com/luongtran/luna/backend/internal/platform/config"
	"github.com/luongtran/luna/backend/internal/platform/httpx"
	"github.com/luongtran/luna/backend/internal/platform/logger"
	"github.com/luongtran/luna/backend/internal/progress"
	"github.com/luongtran/luna/backend/internal/settings"
	"github.com/luongtran/luna/backend/internal/storage/mongo"
	"github.com/luongtran/luna/backend/internal/topic"
	"github.com/luongtran/luna/backend/internal/vocab"
	"github.com/luongtran/luna/backend/internal/writing"
)

const (
	shutdownTimeout    = 10 * time.Second
	disconnectTimeout  = 5 * time.Second
	healthPingTimeout  = 2 * time.Second
	indexRetryInterval = 10 * time.Second
	aiTimeout          = 130 * time.Second
)

func main() {
	// Local development: read backend/.env when run from the backend directory.
	// Real environment variables (Docker, shell) always take precedence.
	if _, err := config.LoadDotEnv(".env"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	cfg, err := config.Load(os.Getenv)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	log := logger.New(os.Stdout, cfg.LogLevel)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, cfg, log); err != nil {
		log.Error("server stopped with error", slog.Any("error", err))
		stop()
		os.Exit(1) //nolint:gocritic // stop() is called explicitly above
	}
}

// run wires dependencies, serves HTTP until ctx is cancelled, then shuts down gracefully.
func run(ctx context.Context, cfg config.Config, log *slog.Logger) error {
	db, err := mongo.Connect(cfg.MongoURI)
	if err != nil {
		return err
	}
	defer func() {
		dctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), disconnectTimeout)
		defer cancel()
		if err := db.Disconnect(dctx); err != nil {
			log.WarnContext(dctx, "mongo disconnect failed", slog.Any("error", err))
		}
	}()

	database := db.Database(cfg.MongoDatabase)
	mongo.PrepareInBackground(ctx, database, indexRetryInterval, log)

	authSvc, err := auth.NewService(
		mongo.NewUsers(database), mongo.NewSessions(database), auth.NewLockout(time.Now), time.Now,
	)
	if err != nil {
		return err
	}
	authHandler := auth.NewHandler(authSvc, cfg.CookieSecure, log)
	requireAuth := httpx.RequireAuth(authHandler.ResolvePrincipal)

	dict := openDictionary(ctx, cfg.DictionaryPath, log)
	defer func() { _ = dict.Close() }()

	lessons := mongo.NewLessons(database)
	aiProvider := newAIProvider(cfg, log)
	topicSvc := topic.NewService(mongo.NewTopics(database), topicLessons{lessons}, aiProvider, time.Now)
	lessonTopics := lessonTopicsPort{topicSvc}
	var worker *job.Worker
	lessonSvc := lesson.NewService(lesson.Deps{
		Lessons: lessons,
		Topics:  lessonTopics,
		Jobs:    mongo.NewJobs(database),
		AI:      aiProvider,
		Dict:    dict,
		Notify:  func() { worker.Notify() },
		Now:     time.Now,
		Log:     log,
	})
	// The writing service needs the study service (Write step) and the other way round; the
	// adapter gets the study service once it exists.
	writeSteps := &writingSteps{}
	writingSvc := writing.NewService(writing.Deps{
		Repo:    mongo.NewWritings(database),
		Lessons: writingLessons{lessons},
		Steps:   writeSteps,
		Jobs:    mongo.NewJobs(database),
		AI:      aiProvider,
		Notify:  func() { worker.Notify() },
		Now:     time.Now,
		Log:     log,
	})
	worker = job.NewWorker(mongo.NewJobs(database), map[job.Type]job.Handler{
		job.TypeAnnotate: lessonSvc.ProcessAnnotate,
		job.TypePractice: lessonSvc.ProcessPractice,
		job.TypeGrade:    writingSvc.ProcessGrade,
	}, func(ctx context.Context, j job.Job, err error) {
		// Grade jobs belong to a writing, the others to a lesson revision.
		if j.Type == job.TypeGrade {
			writingSvc.JobFailed(ctx, j, err)
			return
		}
		lessonSvc.JobFailed(ctx, j, err)
	}, time.Now, log)

	queueMissingPractice(ctx, lessonSvc, aiProvider, log)

	// The worker must stop before the database connection closes (deferred above).
	workerDone := make(chan struct{})
	go func() {
		defer close(workerDone)
		worker.Run(ctx)
	}()
	defer func() { <-workerDone }()

	mux := http.NewServeMux()
	mux.Handle("GET /api/health", health.NewHandler(db, healthPingTimeout, log))
	mux.HandleFunc("POST /api/auth/register", authHandler.Register)
	mux.HandleFunc("POST /api/auth/login", authHandler.Login)
	mux.HandleFunc("POST /api/auth/logout", authHandler.Logout)
	mux.Handle("GET /api/auth/me", requireAuth(http.HandlerFunc(authHandler.Me)))
	settingsSvc := settings.NewService(mongo.NewSettings(database))
	settings.NewHandler(settingsSvc, log).Register(mux, requireAuth)
	exportSvc := export.NewService(mongo.NewExport(database), exportSettings{settingsSvc}, time.Now)
	export.NewHandler(exportSvc, log).Register(mux, requireAuth)
	lesson.NewHandler(lessonSvc, log).Register(mux, requireAuth)
	reader := lesson.NewReader(lessons, dict, lessonTopics, mongo.NewReadingAnswers(database), mongo.NewAILookups(database), aiProvider)
	topic.NewHandler(topicSvc, log).Register(mux, requireAuth)
	vocabSvc := vocab.NewService(vocab.Deps{
		Repo: mongo.NewCards(database),
		Logs: mongo.NewReviewLogs(database),
		LessonExists: func(ctx context.Context, id string) (bool, error) {
			_, err := lessons.Get(ctx, id)
			if errors.Is(err, lesson.ErrNotFound) {
				return false, nil
			}
			return err == nil, err
		},
		Timezones:  settingsSvc,
		Vocabulary: lessonVocabulary{reader},
		Titles:     lessonTitles{lessons},
		Now:        time.Now,
	})
	vocab.NewHandler(vocabSvc, log).Register(mux, requireAuth)
	progressSvc := progress.NewService(mongo.NewDictationResults(database), progressLessons{lessons}, time.Now)
	studySvc := progress.NewStudyService(progress.StudyDeps{
		Goals:     mongo.NewGoals(database),
		Progress:  mongo.NewLessonProgress(database),
		Days:      mongo.NewStudyDays(database),
		Dictation: progressSvc,
		Lessons:   progressLessons{lessons},
		Roadmaps:  topicRoadmaps{topicSvc},
		Titles:    lessonTitles{lessons},
		Reviews:   dailyReviews{vocabSvc},
		Timezones: settingsSvc,
		Quiz:      readingQuiz{reader},
		Writings:  writingSvc,
		Now:       time.Now,
	})
	studyHandler := progress.NewStudyHandler(studySvc, log)
	studyHandler.Register(mux, requireAuth)
	// Lesson content is guarded: learners open only the lesson being studied and lessons already started (L).
	lesson.NewReadingHandler(reader, log).Register(mux, requireAuth, studyHandler.Guard)
	progress.NewHandler(progressSvc, log).Register(mux, requireAuth, studyHandler.Guard)
	writeSteps.svc = studySvc
	writing.NewHandler(writingSvc, log).Register(mux, requireAuth, studyHandler.Guard)

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           httpx.Chain(mux, httpx.RequestID, httpx.Logger(log), httpx.Recover(log)),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serveErr := make(chan error, 1)
	go func() {
		log.InfoContext(ctx, "server listening", slog.String("addr", cfg.HTTPAddr))
		serveErr <- srv.ListenAndServe()
	}()

	select {
	case err := <-serveErr:
		return fmt.Errorf("listen: %w", err)
	case <-ctx.Done():
	}

	log.InfoContext(ctx, "shutting down", slog.String("timeout", shutdownTimeout.String()))
	sctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(sctx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	if err := <-serveErr; err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("listen: %w", err)
	}
	log.InfoContext(ctx, "server stopped")
	return nil
}

// newAIProvider picks the annotation provider from configuration (constitution III).
func newAIProvider(cfg config.Config, log *slog.Logger) ai.Provider {
	if cfg.AIProvider != "gemini" || cfg.GeminiAPIKey == "" {
		log.Info("ai provider disabled: annotations will fail until configured", slog.String("provider", cfg.AIProvider))
		return ai.Disabled{}
	}
	return gemini.New(cfg.GeminiAPIKey, cfg.GeminiModel, &http.Client{Timeout: aiTimeout}, log)
}

// queueMissingPractice queues the practice of lessons annotated before F17. Without an AI
// provider it waits: the jobs would only fail, and failed lessons are not queued again.
func queueMissingPractice(ctx context.Context, svc *lesson.Service, p ai.Provider, log *slog.Logger) {
	if _, off := p.(ai.Disabled); off {
		return
	}
	n, err := svc.QueueMissingPractice(ctx)
	if err != nil {
		log.ErrorContext(ctx, "queue missing practice failed", slog.Any("error", err))
	}
	if n > 0 {
		log.InfoContext(ctx, "queued practice for older lessons", slog.Int("count", n))
	}
}

// closableDictionary is the dictionary used by the Reading step.
type closableDictionary interface {
	lesson.Dictionary
	Close() error
}

type noDictionary struct{ dictionary.None }

func (noDictionary) Close() error { return nil }

// openDictionary opens the offline dictionary; without the file, lookups use annotations only.
func openDictionary(ctx context.Context, path string, log *slog.Logger) closableDictionary {
	d, err := dictionary.Open(ctx, path)
	if err != nil {
		log.WarnContext(ctx, "dictionary unavailable: run deploy/fetch-dictionary.sh", slog.String("path", path), slog.Any("error", err))
		return noDictionary{}
	}
	log.InfoContext(ctx, "dictionary loaded", slog.String("path", path))
	return d
}

// progressLessons adapts the lesson repository to progress.Lessons.
type progressLessons struct {
	repo lesson.Repository
}

func (p progressLessons) Info(ctx context.Context, id string) (revision, sentenceCount int, err error) {
	l, err := p.repo.Get(ctx, id)
	if errors.Is(err, lesson.ErrNotFound) {
		return 0, 0, progress.ErrLessonNotFound
	}
	if err != nil {
		return 0, 0, fmt.Errorf("get lesson: %w", err)
	}
	return l.Revision, len(l.Sentences), nil
}

// lessonTitles adapts the lesson repository to vocab.LessonTitles.
type lessonTitles struct {
	repo lesson.Repository
}

func (l lessonTitles) Titles(ctx context.Context, ids []string) (map[string]string, error) {
	summaries, err := l.repo.Summaries(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("lesson summaries: %w", err)
	}
	out := make(map[string]string, len(summaries))
	for _, s := range summaries {
		out[s.ID] = s.Title
	}
	return out, nil
}

// lessonVocabulary adapts lesson.Reader to vocab.LessonVocabulary.
type lessonVocabulary struct {
	reader *lesson.Reader
}

func (l lessonVocabulary) Vocabulary(ctx context.Context, id string) ([]vocab.VocabItem, error) {
	items, _, err := l.reader.Vocabulary(ctx, id)
	if errors.Is(err, lesson.ErrNotFound) {
		return nil, vocab.ErrLessonNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("lesson vocabulary: %w", err)
	}
	out := make([]vocab.VocabItem, len(items))
	for i, it := range items {
		out[i] = vocab.VocabItem(it)
	}
	return out, nil
}

// topicLessons adapts the lesson repository to topic.Lessons.
type topicLessons struct {
	repo *mongo.Lessons
}

func (t topicLessons) CountByTopic(ctx context.Context) (map[string]int, error) {
	return t.repo.CountByTopic(ctx)
}

func (t topicLessons) TopicOf(ctx context.Context, ids []string) (map[string]string, error) {
	return t.repo.TopicOf(ctx, ids)
}

func (t topicLessons) Texts(ctx context.Context, topicIDs []string) (map[string][]topic.LessonText, error) {
	return t.repo.TopicTexts(ctx, topicIDs)
}

func (t topicLessons) SetLevelByTopic(ctx context.Context, topicID, level string) error {
	return t.repo.SetLevelByTopic(ctx, topicID, level)
}

func (t topicLessons) Refs(ctx context.Context, ids []string) ([]topic.LessonRef, error) {
	sums, err := t.repo.Summaries(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("lesson summaries: %w", err)
	}
	out := make([]topic.LessonRef, len(sums))
	for i, s := range sums {
		out[i] = topic.LessonRef{
			ID: s.ID, Title: s.Title, Level: string(s.Level), TopicID: s.TopicID,
			AnnotationStatus: string(s.AnnotationStatus), CreatedAt: s.CreatedAt,
		}
	}
	return out, nil
}

// lessonTopicsPort adapts topic.Service to lesson.Topics.
type lessonTopicsPort struct {
	svc *topic.Service
}

func toTopicRef(t topic.Topic) lesson.TopicRef {
	return lesson.TopicRef{ID: t.ID, Name: t.Name, Level: lesson.Level(t.Level), Words: t.Words}
}

func (p lessonTopicsPort) Get(ctx context.Context, id string) (lesson.TopicRef, error) {
	t, err := p.svc.Get(ctx, id)
	if errors.Is(err, topic.ErrNotFound) {
		return lesson.TopicRef{}, lesson.ErrTopicNotFound
	}
	if err != nil {
		return lesson.TopicRef{}, fmt.Errorf("get topic: %w", err)
	}
	return toTopicRef(t), nil
}

func (p lessonTopicsPort) Names(ctx context.Context) (map[string]lesson.TopicRef, error) {
	all, err := p.svc.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list topics: %w", err)
	}
	out := make(map[string]lesson.TopicRef, len(all))
	for _, t := range all {
		out[t.ID] = toTopicRef(t)
	}
	return out, nil
}

func (p lessonTopicsPort) RoadmapLessonIDs(ctx context.Context) (map[string]bool, error) {
	return p.svc.RoadmapLessonIDs(ctx)
}

func (p lessonTopicsPort) AppendLesson(ctx context.Context, topicID, lessonID string) error {
	err := p.svc.AppendLesson(ctx, topicID, lessonID)
	if errors.Is(err, topic.ErrNotFound) {
		return lesson.ErrTopicNotFound
	}
	return err
}

func (p lessonTopicsPort) MoveLesson(ctx context.Context, lessonID, from, to string) error {
	return p.svc.MoveLesson(ctx, lessonID, from, to)
}

func (p lessonTopicsPort) Position(ctx context.Context, topicID, lessonID string) (int, error) {
	t, err := p.svc.Get(ctx, topicID)
	if errors.Is(err, topic.ErrNotFound) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("get topic: %w", err)
	}
	return slices.Index(t.LessonIDs, lessonID) + 1, nil
}

// topicRoadmaps adapts topic.Service to progress.Roadmaps.
type topicRoadmaps struct {
	svc *topic.Service
}

func (t topicRoadmaps) Roadmap(ctx context.Context, topicID string) (progress.TopicInfo, error) {
	tp, err := t.svc.Get(ctx, topicID)
	if errors.Is(err, topic.ErrNotFound) {
		return progress.TopicInfo{}, progress.ErrTopicNotFound
	}
	if err != nil {
		return progress.TopicInfo{}, fmt.Errorf("get topic: %w", err)
	}
	return progress.TopicInfo{ID: tp.ID, Name: tp.Name, Level: tp.Level, LessonIDs: tp.LessonIDs}, nil
}

// dailyReviews adapts vocab.Service to progress.Reviews.
type dailyReviews struct {
	svc *vocab.Service
}

func (d dailyReviews) DueCount(ctx context.Context, userID string) (int, error) {
	list, err := d.svc.Due(ctx, userID, 1)
	if err != nil {
		return 0, fmt.Errorf("due cards: %w", err)
	}
	return list.Total, nil
}

func (d dailyReviews) DueBefore(ctx context.Context, userID string, before, createdBefore time.Time) (int, error) {
	return d.svc.DueBefore(ctx, userID, before, createdBefore)
}

func (d dailyReviews) CardCount(ctx context.Context, userID string) (int, error) {
	return d.svc.Count(ctx, userID)
}

// exportSettings adapts settings.Service to export.Settings.
type exportSettings struct {
	svc *settings.Service
}

func (e exportSettings) Values(ctx context.Context, userID string) (map[string]any, error) {
	s, err := e.svc.Get(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get settings: %w", err)
	}
	return map[string]any{"theme": string(s.Theme), "dailyReviewLimit": s.DailyReviewLimit, "timezone": s.Timezone}, nil
}

func (e exportSettings) Location(ctx context.Context, userID string) (*time.Location, error) {
	return e.svc.Location(ctx, userID)
}

// readingQuiz adapts lesson.Reader to progress.ReadingQuiz (F15).
type readingQuiz struct {
	reader *lesson.Reader
}

func (q readingQuiz) Status(ctx context.Context, userID, lessonID string) (questions, answered int, err error) {
	return q.reader.QuizStatus(ctx, userID, lessonID)
}

func (q readingQuiz) Totals(ctx context.Context, userID string) (answered, correct int, err error) {
	return q.reader.Totals(ctx, userID)
}

// writingLessons adapts the lesson repository to writing.Lessons (F8).
type writingLessons struct {
	repo lesson.Repository
}

func (w writingLessons) Info(ctx context.Context, id string) (writing.LessonInfo, error) {
	l, err := w.repo.Get(ctx, id)
	if errors.Is(err, lesson.ErrNotFound) {
		return writing.LessonInfo{}, writing.ErrNotFound
	}
	if err != nil {
		return writing.LessonInfo{}, fmt.Errorf("get lesson: %w", err)
	}
	return writing.LessonInfo{
		Title: l.Title, Level: string(l.Level), Content: l.Content, WritingPrompt: l.Extras.WritingPrompt, Revision: l.Revision,
	}, nil
}

// writingSteps adapts progress.StudyService to writing.Steps; svc is set once it exists.
type writingSteps struct {
	svc *progress.StudyService
}

func (w *writingSteps) CanWrite(ctx context.Context, userID, lessonID string) (bool, error) {
	return w.svc.CanWrite(ctx, userID, lessonID)
}
