package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/luongtran/luna/backend/internal/ai"
	"github.com/luongtran/luna/backend/internal/aiusage"
)

// AIUsage implements aiusage.Repository on "ai_usage".
type AIUsage struct {
	db *sql.DB
}

// NewAIUsage returns the AIUsage repository.
func NewAIUsage(db *sql.DB) *AIUsage { return &AIUsage{db: db} }

var _ aiusage.Repository = (*AIUsage)(nil)

// aiUsageSchema is the DDL of this domain (see migrate.go).
func aiUsageSchema() []string {
	return []string{
		`CREATE TABLE IF NOT EXISTS ai_usage (
	id             BIGINT       NOT NULL AUTO_INCREMENT PRIMARY KEY,
	at             DATETIME(6)  NOT NULL,
	model          VARCHAR(100) NOT NULL,
	op             VARCHAR(50)  NOT NULL,
	status         INT          NOT NULL,
	prompt_tokens  INT          NOT NULL,
	output_tokens  INT          NOT NULL,
	thought_tokens INT          NOT NULL,
	total_tokens   INT          NOT NULL,
	duration_ms    BIGINT       NOT NULL,
	KEY ai_usage_at (at)
) ` + tableOptions,
	}
}

// Add stores one request.
func (r *AIUsage) Add(ctx context.Context, u ai.Usage) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO ai_usage
	(at, model, op, status, prompt_tokens, output_tokens, thought_tokens, total_tokens, duration_ms)
	VALUES (?,?,?,?,?,?,?,?,?)`,
		utc(u.At), u.Model, u.Op, u.Status, u.PromptTokens, u.OutputTokens, u.ThoughtTokens, u.TotalTokens,
		u.Duration.Milliseconds())
	if err != nil {
		return fmt.Errorf("insert ai usage: %w", err)
	}
	return nil
}

// Since returns the requests made at or after from, oldest first.
func (r *AIUsage) Since(ctx context.Context, from time.Time) ([]ai.Usage, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT at, model, op, status, prompt_tokens, output_tokens, thought_tokens,
	total_tokens, duration_ms FROM ai_usage WHERE at >= ? ORDER BY at, id`, utc(from))
	if err != nil {
		return nil, fmt.Errorf("query ai usage: %w", err)
	}
	defer rows.Close() //nolint:errcheck // read-only query
	var out []ai.Usage
	for rows.Next() {
		var u ai.Usage
		var ms int64
		if err := rows.Scan(&u.At, &u.Model, &u.Op, &u.Status, &u.PromptTokens, &u.OutputTokens, &u.ThoughtTokens,
			&u.TotalTokens, &ms); err != nil {
			return nil, fmt.Errorf("scan ai usage: %w", err)
		}
		u.Duration = time.Duration(ms) * time.Millisecond
		out = append(out, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("query ai usage: %w", err)
	}
	return out, nil
}

// DeleteBefore removes the requests made before t.
func (r *AIUsage) DeleteBefore(ctx context.Context, t time.Time) error {
	if _, err := r.db.ExecContext(ctx, "DELETE FROM ai_usage WHERE at < ?", utc(t)); err != nil {
		return fmt.Errorf("delete ai usage: %w", err)
	}
	return nil
}
