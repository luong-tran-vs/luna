package service

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/luongtran/luna/backend/internal/ai"
	"github.com/luongtran/luna/backend/internal/ai/llm"
	"github.com/luongtran/luna/backend/internal/aimodel"
	"github.com/luongtran/luna/backend/internal/aiusage"
)

// aiTimeout bounds one request to the AI provider: five minutes since 2026-10-23, as the slower
// models the admin can choose take minutes to write a lesson or a course.
const aiTimeout = 5 * time.Minute

// initAI picks the annotation provider from configuration (constitution III).
func (c *Container) initAI() {
	c.aiUsage = aiusage.NewService(c.store.AIUsage(), time.Now, c.log)
	if c.cfg.AIProvider != "gemini" || c.cfg.GeminiAPIKey == "" {
		c.log.Info("ai provider disabled: annotations will fail until configured", slog.String("provider", c.cfg.AIProvider))
		c.ai = ai.Disabled{}
		c.imageAI = ai.DisabledImages{}
		c.meaningAI = ai.DisabledMeanings{}
		c.glossAI = ai.DisabledGlosses{}
		c.talkAI = ai.DisabledTalk{}
		c.aiModel = aimodel.NewService(c.store.AIModel(), c.cfg.GeminiModel, false, false, nil, time.Now, c.log)
		return
	}
	text := llm.New(c.cfg.GeminiAPIKey, c.cfg.GeminiModel, &http.Client{Timeout: aiTimeout}, c.log)
	text.Usage = c.aiUsage
	for _, key := range c.cfg.GeminiBackupKeys {
		text.AddBackupKey(key)
	}
	// The admin picks the text model; GEMINI_MODEL is the default (added 2026-10-23).
	// Groq models are offered once GROQ_API_KEY is set (added 2026-10-23).
	text.SetGroq(c.cfg.GroqAPIKey)
	// ChatGPT models are offered once OPENAI_API_KEY is set (added 2026-10-26).
	text.SetOpenAI(c.cfg.OpenAIAPIKey)
	c.aiModel = aimodel.NewService(c.store.AIModel(), c.cfg.GeminiModel, c.cfg.GroqAPIKey != "", c.cfg.OpenAIAPIKey != "", text, time.Now, c.log)
	text.ModelOf = c.aiModel.ModelFor
	c.ai = text
	c.meaningAI = text
	c.glossAI = text
	c.talkAI = text
	images := llm.New(c.cfg.GeminiAPIKey, c.cfg.GeminiImageModel, &http.Client{Timeout: aiTimeout}, c.log)
	images.Usage = c.aiUsage
	for _, key := range c.cfg.GeminiBackupKeys {
		images.AddBackupKey(key)
	}
	c.imageAI = images
}

// queueMissingGlosses queues the glosses of lessons annotated before they existed or rewritten
// since. Without an AI provider it waits, like queueMissingPractice.
func (c *Container) queueMissingGlosses(ctx context.Context) {
	if _, off := c.ai.(ai.Disabled); off {
		return
	}
	n, err := c.lesson.QueueMissingGlosses(ctx)
	if err != nil {
		c.log.ErrorContext(ctx, "queue missing glosses failed", slog.Any("error", err))
	}
	if n > 0 {
		c.log.InfoContext(ctx, "queued glosses for older lessons", slog.Int("count", n))
	}
}

// queueMissingPractice queues the practice of lessons annotated before F17. Without an AI
// provider it waits: the jobs would only fail, and failed lessons are not queued again.
func (c *Container) queueMissingPractice(ctx context.Context) {
	if _, off := c.ai.(ai.Disabled); off {
		return
	}
	n, err := c.lesson.QueueMissingPractice(ctx)
	if err != nil {
		c.log.ErrorContext(ctx, "queue missing practice failed", slog.Any("error", err))
	}
	if n > 0 {
		c.log.InfoContext(ctx, "queued practice for older lessons", slog.Int("count", n))
	}
	// Added 2026-10-22: lessons whose practice was made before the six steps get a new one.
	n, err = c.lesson.QueueMissingSteps(ctx)
	if err != nil {
		c.log.ErrorContext(ctx, "queue missing steps failed", slog.Any("error", err))
	}
	if n > 0 {
		c.log.InfoContext(ctx, "queued six-step practice for older lessons", slog.Int("count", n))
	}
}
