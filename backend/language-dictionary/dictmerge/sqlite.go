package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	_ "modernc.org/sqlite" // pure-Go SQLite driver, registers "sqlite"
)

// readMinhqnd adds the English words of the minhqnd/dictionary database that have a Vietnamese
// meaning, with those meanings (in the database's order) and their pronunciations. Other
// languages and English-language definitions are left out.
func readMinhqnd(ctx context.Context, path string, d *dictionary) error {
	if _, err := os.Stat(path); err != nil {
		return err
	}
	db, err := sql.Open("sqlite", "file:"+path+"?mode=ro&immutable=1")
	if err != nil {
		return err
	}
	defer db.Close() //nolint:errcheck // read-only

	words := map[int64]string{}
	var ids []int64
	rows, err := db.QueryContext(ctx, `SELECT id, word FROM words WHERE lang_code = 'en' ORDER BY id`)
	if err != nil {
		return fmt.Errorf("words: %w", err)
	}
	for rows.Next() {
		var id int64
		var w string
		if err := rows.Scan(&id, &w); err != nil {
			return err
		}
		if w = normalizeWord(w); w != "" {
			words[id] = w
			ids = append(ids, id)
		}
	}
	if err := rows.Close(); err != nil {
		return err
	}

	meanings := map[int64][]meaning{}
	rows, err = db.QueryContext(ctx, `
		SELECT wd.word_id, COALESCE(d.pos, ''), d.definition
		FROM word_definitions wd
		JOIN definitions d ON d.id = wd.definition_id
		JOIN words w ON w.id = wd.word_id
		WHERE w.lang_code = 'en' AND d.definition_lang = 'vi'
		ORDER BY wd.word_id, wd.id`)
	if err != nil {
		return fmt.Errorf("definitions: %w", err)
	}
	for rows.Next() {
		var id int64
		var m meaning
		if err := rows.Scan(&id, &m.POS, &m.Text); err != nil {
			return err
		}
		m.Source = sourceMinhqnd
		meanings[id] = append(meanings[id], m)
	}
	if err := rows.Close(); err != nil {
		return err
	}

	ipas := map[int64][]pronunciation{}
	rows, err = db.QueryContext(ctx, `
		SELECT p.word_id, p.ipa, COALESCE(p.region, '')
		FROM pronunciations p JOIN words w ON w.id = p.word_id
		WHERE w.lang_code = 'en'
		ORDER BY p.word_id, p.id`)
	if err != nil {
		return fmt.Errorf("pronunciations: %w", err)
	}
	for rows.Next() {
		var id int64
		var p pronunciation
		if err := rows.Scan(&id, &p.IPA, &p.Region); err != nil {
			return err
		}
		ipas[id] = append(ipas[id], p)
	}
	if err := rows.Close(); err != nil {
		return err
	}

	for _, id := range ids {
		d.addMinhqnd(words[id], meanings[id], ipas[id])
	}
	return nil
}

// schema is the part of the minhqnd/dictionary schema that dictionary.SQLite reads, so the
// backend opens the merged file unchanged.
const schema = `
CREATE TABLE sources (
	id INTEGER PRIMARY KEY,
	name TEXT UNIQUE NOT NULL
);
CREATE TABLE words (
	id INTEGER PRIMARY KEY,
	word TEXT NOT NULL,
	lang_code TEXT NOT NULL DEFAULT 'en'
);
CREATE TABLE definitions (
	id INTEGER PRIMARY KEY,
	definition TEXT NOT NULL,
	pos TEXT,
	definition_lang TEXT NOT NULL DEFAULT 'vi'
);
CREATE TABLE word_definitions (
	id INTEGER PRIMARY KEY,
	word_id INTEGER NOT NULL REFERENCES words(id),
	definition_id INTEGER NOT NULL REFERENCES definitions(id),
	source_id INTEGER NOT NULL REFERENCES sources(id)
);
CREATE TABLE pronunciations (
	id INTEGER PRIMARY KEY,
	word_id INTEGER NOT NULL REFERENCES words(id),
	ipa TEXT NOT NULL,
	region TEXT
);`

const indexes = `
CREATE UNIQUE INDEX idx_words_unique ON words(word, lang_code);
CREATE UNIQUE INDEX idx_definitions_unique ON definitions(definition, pos);
CREATE INDEX idx_word_definitions_word_id ON word_definitions(word_id);
CREATE INDEX idx_pronunciations_word_id ON pronunciations(word_id);
ANALYZE;`

// write stores d in a new SQLite file at path, replacing it only once complete.
func write(ctx context.Context, path string, d *dictionary) error {
	tmp := path + ".tmp"
	_ = os.Remove(tmp)
	db, err := sql.Open("sqlite", "file:"+tmp)
	if err != nil {
		return err
	}
	db.SetMaxOpenConns(1)
	if err := fill(ctx, db, d); err != nil {
		_ = db.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := db.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func fill(ctx context.Context, db *sql.DB, d *dictionary) error {
	for _, pragma := range []string{"PRAGMA journal_mode = OFF", "PRAGMA synchronous = OFF"} {
		if _, err := db.ExecContext(ctx, pragma); err != nil {
			return err
		}
	}
	if _, err := db.ExecContext(ctx, schema); err != nil {
		return fmt.Errorf("schema: %w", err)
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck // no-op after Commit

	for id := sourceMinhqnd; id <= sourceHND; id++ {
		if _, err := tx.ExecContext(ctx, `INSERT INTO sources (id, name) VALUES (?, ?)`, id, sourceNames[id]); err != nil {
			return err
		}
	}
	insWord, err := tx.PrepareContext(ctx, `INSERT INTO words (id, word, lang_code) VALUES (?, ?, 'en')`)
	if err != nil {
		return err
	}
	insDef, err := tx.PrepareContext(ctx, `INSERT INTO definitions (id, definition, pos, definition_lang) VALUES (?, ?, ?, 'vi')`)
	if err != nil {
		return err
	}
	insLink, err := tx.PrepareContext(ctx, `INSERT INTO word_definitions (word_id, definition_id, source_id) VALUES (?, ?, ?)`)
	if err != nil {
		return err
	}
	insIPA, err := tx.PrepareContext(ctx, `INSERT INTO pronunciations (word_id, ipa, region) VALUES (?, ?, NULLIF(?, ''))`)
	if err != nil {
		return err
	}

	// The same definition text and part of speech is stored once and shared.
	defIDs := map[[2]string]int64{}
	var wordID int64
	for _, w := range d.order {
		e := d.entries[w]
		if len(e.Meanings) == 0 {
			continue
		}
		wordID++
		if _, err := insWord.ExecContext(ctx, wordID, e.Word); err != nil {
			return fmt.Errorf("word %q: %w", e.Word, err)
		}
		for _, m := range e.Meanings {
			key := [2]string{m.Text, m.POS}
			defID, ok := defIDs[key]
			if !ok {
				defID = int64(len(defIDs) + 1)
				defIDs[key] = defID
				if _, err := insDef.ExecContext(ctx, defID, m.Text, m.POS); err != nil {
					return fmt.Errorf("definition of %q: %w", e.Word, err)
				}
			}
			if _, err := insLink.ExecContext(ctx, wordID, defID, m.Source); err != nil {
				return err
			}
		}
		for _, p := range e.IPAs {
			if _, err := insIPA.ExecContext(ctx, wordID, p.IPA, p.Region); err != nil {
				return err
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, indexes); err != nil {
		return fmt.Errorf("indexes: %w", err)
	}
	_, err = db.ExecContext(ctx, "VACUUM")
	return err
}
