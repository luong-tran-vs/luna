package dictionary

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"

	_ "modernc.org/sqlite" // pure-Go SQLite driver, registers "sqlite"
)

// maxMeanings is how many Vietnamese meanings a lookup returns.
const maxMeanings = 3

// Meaning is one Vietnamese definition with its part of speech code (N, V, A, …).
type Meaning struct {
	POS  string
	Text string
}

// Entry is an English word with its pronunciation and Vietnamese meanings.
type Entry struct {
	Word     string
	IPA      string
	Meanings []Meaning
}

// SQLite reads the minhqnd/dictionary database (read-only).
type SQLite struct {
	db *sql.DB
}

// Open opens the dictionary file read-only. A missing file is an error.
func Open(ctx context.Context, path string) (*SQLite, error) {
	if _, err := os.Stat(path); err != nil { //nolint:gosec // DICTIONARY_PATH is operator configuration
		return nil, fmt.Errorf("dictionary: %w", err)
	}
	db, err := sql.Open("sqlite", "file:"+path+"?mode=ro&immutable=1")
	if err != nil {
		return nil, fmt.Errorf("dictionary: open: %w", err)
	}
	db.SetMaxOpenConns(4)
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("dictionary: open: %w", err)
	}
	return &SQLite{db: db}, nil
}

// Close releases the database.
func (d *SQLite) Close() error { return d.db.Close() }

// Lookup finds an English word (case-insensitive). ok is false when it is not in the dictionary
// or has no Vietnamese meaning.
func (d *SQLite) Lookup(ctx context.Context, word string) (Entry, bool, error) {
	w := strings.ToLower(strings.TrimSpace(word))
	var id int64
	err := d.db.QueryRowContext(ctx, `SELECT id FROM words WHERE word = ? AND lang_code = 'en'`, w).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return Entry{}, false, nil
	}
	if err != nil {
		return Entry{}, false, fmt.Errorf("dictionary: find word: %w", err)
	}

	rows, err := d.db.QueryContext(ctx, `
		SELECT COALESCE(d.pos, ''), d.definition FROM word_definitions wd
		JOIN definitions d ON d.id = wd.definition_id
		WHERE wd.word_id = ? AND d.definition_lang = 'vi'
		ORDER BY wd.id LIMIT ?`, id, maxMeanings)
	if err != nil {
		return Entry{}, false, fmt.Errorf("dictionary: meanings: %w", err)
	}
	defer rows.Close() //nolint:errcheck // read-only query

	e := Entry{Word: w}
	for rows.Next() {
		var m Meaning
		if err := rows.Scan(&m.POS, &m.Text); err != nil {
			return Entry{}, false, fmt.Errorf("dictionary: scan meaning: %w", err)
		}
		e.Meanings = append(e.Meanings, m)
	}
	if err := rows.Err(); err != nil {
		return Entry{}, false, fmt.Errorf("dictionary: meanings: %w", err)
	}
	if len(e.Meanings) == 0 {
		return Entry{}, false, nil
	}

	// Prefer the slash-delimited transcription ("/ˈɡəʊ/"); none is fine.
	err = d.db.QueryRowContext(ctx, `
		SELECT ipa FROM pronunciations WHERE word_id = ?
		ORDER BY (ipa LIKE '/%') DESC, id LIMIT 1`, id).Scan(&e.IPA)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return Entry{}, false, fmt.Errorf("dictionary: ipa: %w", err)
	}
	return e, true, nil
}

// None is used when no dictionary file is installed: every lookup misses.
type None struct{}

// Lookup always reports not found.
func (None) Lookup(context.Context, string) (Entry, bool, error) { return Entry{}, false, nil }
