package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/luongtran/luna/backend/internal/wordbank"
)

// WordBank implements wordbank.Repository (F24): the words in "word_bank" and their pictures, as
// binary data, in "word_bank_images".
type WordBank struct {
	db *sql.DB
}

// NewWordBank returns the WordBank repository.
func NewWordBank(db *sql.DB) *WordBank { return &WordBank{db: db} }

var _ wordbank.Repository = (*WordBank)(nil)

// wordBankSchema is the DDL of this domain (see migrate.go). image_at is NULL while the word has no
// picture.
func wordBankSchema() []string {
	return []string{
		`CREATE TABLE IF NOT EXISTS word_bank (
	lemma      VARCHAR(191) NOT NULL PRIMARY KEY,
	meaning_vi TEXT         NOT NULL,
	ipa        VARCHAR(191) NOT NULL,
	image_at   DATETIME(6)  NULL,
	created_at DATETIME(6)  NOT NULL,
	updated_at DATETIME(6)  NOT NULL
) ` + tableOptions,
		`CREATE TABLE IF NOT EXISTS word_bank_images (
	lemma      VARCHAR(191) NOT NULL PRIMARY KEY,
	mime       VARCHAR(64)  NOT NULL,
	data       MEDIUMBLOB   NOT NULL,
	created_at DATETIME(6)  NOT NULL
) ` + tableOptions,
	}
}

// insertMissingBatch is how many words one INSERT of InsertMissing carries.
const insertMissingBatch = 500

const wordBankColumns = "lemma, meaning_vi, ipa, image_at, created_at, updated_at"

func scanBankWord(row interface{ Scan(...any) error }) (wordbank.Word, error) {
	var (
		w       wordbank.Word
		imageAt sql.NullTime
	)
	if err := row.Scan(&w.Lemma, &w.MeaningVi, &w.IPA, &imageAt, &w.CreatedAt, &w.UpdatedAt); err != nil {
		return wordbank.Word{}, err
	}
	w.ImageAt = timeOf(imageAt)
	return w, nil
}

func (r *WordBank) query(ctx context.Context, q string, a ...any) ([]wordbank.Word, error) {
	rows, err := r.db.QueryContext(ctx, q, a...)
	if err != nil {
		return nil, fmt.Errorf("query word bank: %w", err)
	}
	defer rows.Close()
	var out []wordbank.Word
	for rows.Next() {
		w, err := scanBankWord(rows)
		if err != nil {
			return nil, fmt.Errorf("scan word bank: %w", err)
		}
		out = append(out, w)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read word bank: %w", err)
	}
	return out, nil
}

// List returns a page of the matching words by lemma, and how many match.
func (r *WordBank) List(ctx context.Context, q wordbank.ListQuery) ([]wordbank.Word, int, error) {
	where, a := "1 = 1", []any{}
	if q.Search != "" {
		pat := "%" + cardsLikeEscaper.Replace(strings.ToLower(q.Search)) + "%"
		where += " AND (LOWER(lemma) LIKE ? ESCAPE '!' OR LOWER(meaning_vi) LIKE ? ESCAPE '!')"
		a = append(a, pat, pat)
	}
	switch q.Missing {
	case wordbank.MissingImage:
		where += " AND image_at IS NULL"
	case wordbank.MissingIPA:
		where += " AND ipa = ''"
	case wordbank.MissingMeaning:
		where += " AND meaning_vi = ''"
	}
	var total int
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM word_bank WHERE "+where, a...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count word bank: %w", err)
	}
	words, err := r.query(ctx, "SELECT "+wordBankColumns+" FROM word_bank WHERE "+where+" ORDER BY lemma LIMIT ? OFFSET ?",
		append(a, max(q.Limit, 0), max(q.Skip, 0))...)
	if err != nil {
		return nil, 0, err
	}
	return words, total, nil
}

// Get returns wordbank.ErrNotFound when the bank has no such word.
func (r *WordBank) Get(ctx context.Context, lemma string) (wordbank.Word, error) {
	w, err := scanBankWord(r.db.QueryRowContext(ctx, "SELECT "+wordBankColumns+" FROM word_bank WHERE lemma = ?", lemma))
	if errors.Is(err, sql.ErrNoRows) {
		return wordbank.Word{}, wordbank.ErrNotFound
	}
	if err != nil {
		return wordbank.Word{}, fmt.Errorf("get bank word: %w", err)
	}
	return w, nil
}

// Find returns the words of the bank among lemmas.
func (r *WordBank) Find(ctx context.Context, lemmas []string) ([]wordbank.Word, error) {
	if len(lemmas) == 0 {
		return nil, nil
	}
	return r.query(ctx, "SELECT "+wordBankColumns+" FROM word_bank WHERE lemma IN ("+placeholders(len(lemmas))+")", args(lemmas)...)
}

// Create returns wordbank.ErrExists when the lemma is taken.
func (r *WordBank) Create(ctx context.Context, w wordbank.Word) error {
	_, err := r.db.ExecContext(ctx, "INSERT INTO word_bank ("+wordBankColumns+") VALUES (?, ?, ?, NULL, ?, ?)",
		w.Lemma, w.MeaningVi, w.IPA, utc(w.CreatedAt), utc(w.UpdatedAt))
	if isDuplicate(err) {
		return wordbank.ErrExists
	}
	if err != nil {
		return fmt.Errorf("insert bank word: %w", err)
	}
	return nil
}

// InsertMissing adds the words not in the bank yet; existing lemmas are skipped.
func (r *WordBank) InsertMissing(ctx context.Context, ws []wordbank.Word) (int, error) {
	added := 0
	for start := 0; start < len(ws); start += insertMissingBatch {
		batch := ws[start:min(start+insertMissingBatch, len(ws))]
		values := make([]string, len(batch))
		a := make([]any, 0, len(batch)*5)
		for i, w := range batch {
			values[i] = "(?, ?, ?, NULL, ?, ?)"
			a = append(a, w.Lemma, w.MeaningVi, w.IPA, utc(w.CreatedAt), utc(w.UpdatedAt))
		}
		res, err := r.db.ExecContext(ctx,
			"INSERT IGNORE INTO word_bank ("+wordBankColumns+") VALUES "+strings.Join(values, ", "), a...)
		if err != nil {
			return added, fmt.Errorf("insert bank words: %w", err)
		}
		n, err := res.RowsAffected()
		if err != nil {
			return added, fmt.Errorf("insert bank words: %w", err)
		}
		added += int(n)
	}
	return added, nil
}

// Update sets the meaning, IPA and update time of a word.
func (r *WordBank) Update(ctx context.Context, w wordbank.Word) error {
	res, err := r.db.ExecContext(ctx, "UPDATE word_bank SET meaning_vi = ?, ipa = ?, updated_at = ? WHERE lemma = ?",
		w.MeaningVi, w.IPA, utc(w.UpdatedAt), w.Lemma)
	if err != nil {
		return fmt.Errorf("update bank word: %w", err)
	}
	if ok, err := changed(res); err != nil {
		return err
	} else if !ok {
		return wordbank.ErrNotFound
	}
	return nil
}

// Delete removes a word and its picture.
func (r *WordBank) Delete(ctx context.Context, lemma string) error {
	return inTx(ctx, r.db, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, "DELETE FROM word_bank WHERE lemma = ?", lemma)
		if err != nil {
			return fmt.Errorf("delete bank word: %w", err)
		}
		if n, err := res.RowsAffected(); err != nil {
			return fmt.Errorf("delete bank word: %w", err)
		} else if n == 0 {
			return wordbank.ErrNotFound
		}
		if _, err := tx.ExecContext(ctx, "DELETE FROM word_bank_images WHERE lemma = ?", lemma); err != nil {
			return fmt.Errorf("delete bank image: %w", err)
		}
		return nil
	})
}

// Image returns wordbank.ErrImageNotFound when the word has no picture.
func (r *WordBank) Image(ctx context.Context, lemma string) (wordbank.Image, error) {
	img := wordbank.Image{Lemma: lemma}
	err := r.db.QueryRowContext(ctx, "SELECT mime, data, created_at FROM word_bank_images WHERE lemma = ?", lemma).
		Scan(&img.MIME, &img.Data, &img.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return wordbank.Image{}, wordbank.ErrImageNotFound
	}
	if err != nil {
		return wordbank.Image{}, fmt.Errorf("get bank image: %w", err)
	}
	return img, nil
}

// SaveImage replaces the picture of a word and marks the word as having one.
func (r *WordBank) SaveImage(ctx context.Context, img wordbank.Image) error {
	return inTx(ctx, r.db, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, "UPDATE word_bank SET image_at = ? WHERE lemma = ?", utc(img.CreatedAt), img.Lemma)
		if err != nil {
			return fmt.Errorf("mark bank image: %w", err)
		}
		if ok, err := changed(res); err != nil {
			return err
		} else if !ok {
			return wordbank.ErrNotFound
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO word_bank_images (lemma, mime, data, created_at) VALUES (?, ?, ?, ?)
ON DUPLICATE KEY UPDATE mime = VALUES(mime), data = VALUES(data), created_at = VALUES(created_at)`,
			img.Lemma, img.MIME, img.Data, utc(img.CreatedAt))
		if err != nil {
			return fmt.Errorf("save bank image: %w", err)
		}
		return nil
	})
}

// DeleteImage removes the picture of a word.
func (r *WordBank) DeleteImage(ctx context.Context, lemma string) error {
	return inTx(ctx, r.db, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, "UPDATE word_bank SET image_at = NULL WHERE lemma = ?", lemma); err != nil {
			return fmt.Errorf("unmark bank image: %w", err)
		}
		if _, err := tx.ExecContext(ctx, "DELETE FROM word_bank_images WHERE lemma = ?", lemma); err != nil {
			return fmt.Errorf("delete bank image: %w", err)
		}
		return nil
	})
}
