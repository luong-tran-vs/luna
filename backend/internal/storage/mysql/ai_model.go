package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/luongtran/luna/backend/internal/aimodel"
)

// AIModel implements aimodel.Repository on "ai_settings" (added 2026-10-23): one row, its id
// "text", with the model the admin chose and the ops given their own model (a JSON object).
type AIModel struct {
	db *sql.DB
}

var _ aimodel.Repository = (*AIModel)(nil)

// NewAIModel returns the AIModel repository.
func NewAIModel(db *sql.DB) *AIModel { return &AIModel{db: db} }

const aiModelID = "text"

// aiSettingsSchema is the DDL of this domain (see migrate.go).
func aiSettingsSchema() []string {
	return []string{`CREATE TABLE IF NOT EXISTS ai_settings (
  id         VARCHAR(16)  NOT NULL PRIMARY KEY,
  model      VARCHAR(64)  NOT NULL,
  updated_at DATETIME(6)  NOT NULL
) ` + tableOptions}
}

// aiSettingsByOpSchema adds the ops' own models (added 2026-10-23).
func aiSettingsByOpSchema() []string {
	return []string{`ALTER TABLE ai_settings ADD COLUMN by_op JSON NULL`}
}

// Get returns the saved choice, or a zero Choice.
func (r *AIModel) Get(ctx context.Context) (aimodel.Choice, error) {
	var c aimodel.Choice
	var at time.Time
	var byOp sql.NullString
	err := r.db.QueryRowContext(ctx, `SELECT model, by_op, updated_at FROM ai_settings WHERE id = ?`, aiModelID).
		Scan(&c.Model, &byOp, &at)
	if errors.Is(err, sql.ErrNoRows) {
		return aimodel.Choice{}, nil
	}
	if err != nil {
		return aimodel.Choice{}, fmt.Errorf("mysql: get ai model: %w", err)
	}
	if byOp.Valid {
		if err := json.Unmarshal([]byte(byOp.String), &c.ByOp); err != nil {
			return aimodel.Choice{}, fmt.Errorf("mysql: decode ai model by op: %w", err)
		}
	}
	c.UpdatedAt = at.UTC()
	return c, nil
}

// Save replaces the choice.
func (r *AIModel) Save(ctx context.Context, c aimodel.Choice) error {
	var byOp any
	if len(c.ByOp) > 0 {
		data, err := json.Marshal(c.ByOp)
		if err != nil {
			return fmt.Errorf("mysql: encode ai model by op: %w", err)
		}
		byOp = string(data)
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO ai_settings (id, model, by_op, updated_at) VALUES (?, ?, ?, ?)
ON DUPLICATE KEY UPDATE model = VALUES(model), by_op = VALUES(by_op), updated_at = VALUES(updated_at)`,
		aiModelID, c.Model, byOp, c.UpdatedAt.UTC())
	if err != nil {
		return fmt.Errorf("mysql: save ai model: %w", err)
	}
	return nil
}
