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

// maxMeanings is how many Vietnamese meanings a lookup shows; maxSenses how many it reads to tell
// the parts of speech of a word (the senses of one part of speech come first: "run" has 22 noun
// senses before its verb ones).
const (
	maxMeanings = 4
	maxSenses   = 60
)

// corePOS are the parts of speech a lookup shows first, one sense each, so "like" shows the verb
// "thích" next to its adjective senses.
var corePOS = map[string]bool{"N": true, "V": true, "A": true}

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
	// Senses are all the meanings read (up to maxSenses), Meanings the ones shown (pickMeanings).
	Senses []Meaning
}

// SQLite reads english-dictionary.db (read-only), built by language-dictionary/build.sh from
// minhqnd/dictionary and Hồ Ngọc Đức's dictionary with the minhqnd/dictionary tables.
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
		ORDER BY wd.id LIMIT ?`, id, maxSenses)
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
		e.Senses = append(e.Senses, m)
	}
	if err := rows.Err(); err != nil {
		return Entry{}, false, fmt.Errorf("dictionary: meanings: %w", err)
	}
	e.Meanings = pickMeanings(e.Senses, maxMeanings)
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

// pickMeanings chooses up to n senses to show, spread over the word's parts of speech: the first
// sense (Resolve reads it for inflection notes, "went" → "go"), then the first sense of each other
// core part of speech, then of each other part of speech, then the next senses, all in dictionary
// order within each step.
func pickMeanings(senses []Meaning, n int) []Meaning {
	if len(senses) <= n {
		return senses
	}
	picked := make([]bool, len(senses))
	shown := map[string]bool{}
	out := make([]Meaning, 0, n)
	take := func(i int) {
		picked[i] = true
		shown[senses[i].POS] = true
		out = append(out, senses[i])
	}
	take(0)
	for _, core := range []bool{true, false} {
		for i, s := range senses {
			if len(out) == n {
				return out
			}
			if !picked[i] && s.POS != "" && !shown[s.POS] && corePOS[s.POS] == core {
				take(i)
			}
		}
	}
	for i := range senses {
		if len(out) == n {
			break
		}
		if !picked[i] {
			take(i)
		}
	}
	return out
}

// None is used when no dictionary file is installed: every lookup misses.
type None struct{}

// Lookup always reports not found.
func (None) Lookup(context.Context, string) (Entry, bool, error) { return Entry{}, false, nil }
