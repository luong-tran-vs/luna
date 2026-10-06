package service

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/luongtran/luna/backend/internal/ai"
	"github.com/luongtran/luna/backend/internal/ai/gemini"
)

// aiTimeout bounds one request to the AI provider (a generation batch can take two minutes).
const aiTimeout = 130 * time.Second

// initAI picks the annotation provider from configuration (constitution III).
func (c *Container) initAI() {
	if c.cfg.AIProvider != "gemini" || c.cfg.GeminiAPIKey == "" {
		c.log.Info("ai provider disabled: annotations will fail until configured", slog.String("provider", c.cfg.AIProvider))
		c.ai = ai.Disabled{}
		c.imageAI = ai.DisabledImages{}
		return
	}
	c.ai = gemini.New(c.cfg.GeminiAPIKey, c.cfg.GeminiModel, &http.Client{Timeout: aiTimeout}, c.log)
	c.imageAI = gemini.New(c.cfg.GeminiAPIKey, c.cfg.GeminiImageModel, &http.Client{Timeout: aiTimeout}, c.log)
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
}
