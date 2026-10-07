package mysql

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	drv "github.com/go-sql-driver/mysql"
)

// tableOptions ends every CREATE TABLE: InnoDB, utf8mb4, binary collation (exact comparisons).
const tableOptions = "ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin"

// errDuplicate is MySQL's "Duplicate entry for key" error number.
const errDuplicate = 1062

// errDuplicateColumn is MySQL's "Duplicate column name" error number.
const errDuplicateColumn = 1060

// errDuplicateKeyName is MySQL's "Duplicate key name": the key already exists.
const errDuplicateKeyName = 1061

// errCantDropKey is MySQL's "Can't DROP ...; check that column/key exists": the key is already gone.
const errCantDropKey = 1091

// newID returns a new 24-character lowercase hex id (12 random bytes, like a Mongo ObjectID).
func newID() string {
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic("mysql: no randomness: " + err.Error()) // crypto/rand does not fail on supported systems
	}
	return hex.EncodeToString(b[:])
}

// isDuplicate reports whether err is a unique-key violation.
func isDuplicate(err error) bool {
	var me *drv.MySQLError
	return errors.As(err, &me) && me.Number == errDuplicate
}

// placeholders returns "?,?,?" for n values, for IN (...) lists; n must be at least 1.
func placeholders(n int) string {
	return strings.TrimSuffix(strings.Repeat("?,", n), ",")
}

// args converts a slice to the []any a query takes.
func args[T any](items []T) []any {
	out := make([]any, len(items))
	for i, v := range items {
		out[i] = v
	}
	return out
}

// utc returns t in UTC, the zone every stored time uses.
func utc(t time.Time) time.Time { return t.UTC() }

// nullTime stores a zero time as NULL.
func nullTime(t time.Time) sql.NullTime {
	if t.IsZero() {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: t.UTC(), Valid: true}
}

// timeOf reads a nullable time column back; NULL is the zero time.
func timeOf(n sql.NullTime) time.Time {
	if !n.Valid {
		return time.Time{}
	}
	return n.Time.UTC()
}

// toJSON encodes v for a JSON column. A nil slice or map is stored as "null"; use jsonOr when the
// column must be an array or an object.
func toJSON(v any) ([]byte, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("mysql: encode json: %w", err)
	}
	return b, nil
}

// fromJSON decodes a JSON column into v. NULL and empty input leave v as it is.
func fromJSON(raw []byte, v any) error {
	if len(raw) == 0 {
		return nil
	}
	if err := json.Unmarshal(raw, v); err != nil {
		return fmt.Errorf("mysql: decode json: %w", err)
	}
	return nil
}

// execer is what a *sql.DB and a *sql.Tx share, so a helper works in or out of a transaction.
type execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// inTx runs fn in a transaction: committed when fn returns nil, rolled back otherwise.
func inTx(ctx context.Context, db *sql.DB, fn func(tx *sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("mysql: begin: %w", err)
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("mysql: commit: %w", err)
	}
	return nil
}

// changed reports whether an UPDATE touched a row. Note MySQL counts a row as affected only if a value
// actually changed, unless the DSN sets clientFoundRows; Open sets it, so a matched row counts.
func changed(res sql.Result) (bool, error) {
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("mysql: rows affected: %w", err)
	}
	return n > 0, nil
}

// sinceClause narrows a WHERE clause to rows whose column is at or after since; nil adds nothing.
func sinceClause(column string, since *time.Time) (string, []any) {
	if since == nil {
		return "", nil
	}
	return " AND " + column + " >= ?", []any{utc(*since)}
}
