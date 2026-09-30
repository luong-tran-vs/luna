package dictionary

import (
	"database/sql"
	"path/filepath"
	"testing"
)

// buildTestDB creates a small dictionary with the same schema as minhqnd/dictionary v2.0.0.
func buildTestDB(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "dictionary.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	stmts := []string{
		`CREATE TABLE words (id INTEGER PRIMARY KEY, word TEXT NOT NULL, source_id INTEGER DEFAULT 1, lang_code TEXT NOT NULL DEFAULT 'vi')`,
		`CREATE UNIQUE INDEX idx_words_unique ON words(word, lang_code)`,
		`CREATE TABLE definitions (id INTEGER PRIMARY KEY AUTOINCREMENT, definition TEXT NOT NULL, pos TEXT, sub_pos TEXT, definition_lang TEXT DEFAULT 'vi', links TEXT)`,
		`CREATE TABLE word_definitions (id INTEGER PRIMARY KEY AUTOINCREMENT, word_id INTEGER NOT NULL, definition_id INTEGER NOT NULL, example TEXT, source_id INTEGER DEFAULT 1)`,
		`CREATE TABLE pronunciations (id INTEGER PRIMARY KEY AUTOINCREMENT, word_id INTEGER NOT NULL, ipa TEXT NOT NULL, region TEXT)`,
		`INSERT INTO words (id, word, lang_code) VALUES (1, 'go', 'en'), (2, 'park', 'en'), (3, 'study', 'en'), (4, 'child', 'en'), (5, 'go', 'vi')`,
		`INSERT INTO definitions (id, definition, pos, definition_lang) VALUES
			(1, 'Đi, đi đến.', 'V', 'vi'), (2, 'Thành, hoá thành.', 'V', 'vi'), (3, 'Trôi qua.', 'V', 'vi'),
			(4, 'Chết, tiêu tan.', 'V', 'vi'), (5, 'to move', 'V', 'en'), (6, 'Công viên.', 'N', 'vi'),
			(7, 'Học, nghiên cứu.', 'V', 'vi'), (8, 'Đứa trẻ.', 'N', 'vi'), (9, 'tiếng Việt go', 'N', 'vi')`,
		`INSERT INTO word_definitions (word_id, definition_id) VALUES (1, 5), (1, 1), (1, 2), (1, 3), (1, 4), (2, 6), (3, 7), (4, 8), (5, 9)`,
		`INSERT INTO pronunciations (word_id, ipa, region) VALUES (1, 'ˈɡoʊ', 'US'), (1, '/ˈɡəʊ/', NULL), (2, '/ˈpɑːrk/', NULL)`,
	}
	for _, s := range stmts {
		if _, err := db.ExecContext(t.Context(), s); err != nil {
			t.Fatalf("%s: %v", s, err)
		}
	}
	return path
}

func TestSQLiteLookup(t *testing.T) {
	t.Parallel()

	d, err := Open(t.Context(), buildTestDB(t))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = d.Close() })

	e, ok, err := d.Lookup(t.Context(), "Go")
	if err != nil || !ok {
		t.Fatalf("Lookup(Go) = %v, %v", ok, err)
	}
	if e.Word != "go" || e.IPA != "/ˈɡəʊ/" {
		t.Errorf("entry = %+v, want word go and the /…/ IPA", e)
	}
	want := []Meaning{{POS: "V", Text: "Đi, đi đến."}, {POS: "V", Text: "Thành, hoá thành."}, {POS: "V", Text: "Trôi qua."}}
	if len(e.Meanings) != 3 {
		t.Fatalf("meanings = %+v, want 3 Vietnamese meanings", e.Meanings)
	}
	for i := range want {
		if e.Meanings[i] != want[i] {
			t.Errorf("meaning %d = %+v, want %+v", i, e.Meanings[i], want[i])
		}
	}

	if e, ok, _ := d.Lookup(t.Context(), "child"); !ok || e.IPA != "" || e.Meanings[0].Text != "Đứa trẻ." {
		t.Errorf("child = %+v, %v", e, ok)
	}
	if _, ok, err := d.Lookup(t.Context(), "banana"); ok || err != nil {
		t.Errorf("unknown word: ok=%v err=%v", ok, err)
	}
}

func TestOpenMissingFile(t *testing.T) {
	t.Parallel()
	if _, err := Open(t.Context(), filepath.Join(t.TempDir(), "missing.db")); err == nil {
		t.Fatal("Open(missing) error = nil")
	}
}

func TestNone(t *testing.T) {
	t.Parallel()
	if _, ok, err := (None{}).Lookup(t.Context(), "go"); ok || err != nil {
		t.Fatalf("None.Lookup = %v, %v", ok, err)
	}
}
