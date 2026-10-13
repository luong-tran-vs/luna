package dictionary

import (
	"database/sql"
	"path/filepath"
	"reflect"
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
	want := []Meaning{
		{POS: "V", Text: "Đi, đi đến."}, {POS: "V", Text: "Thành, hoá thành."},
		{POS: "V", Text: "Trôi qua."}, {POS: "V", Text: "Chết, tiêu tan."},
	}
	if len(e.Meanings) != len(want) {
		t.Fatalf("meanings = %+v, want %d Vietnamese meanings", e.Meanings, len(want))
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

func TestPickMeanings(t *testing.T) {
	t.Parallel()

	// "like": its adjective senses come first in the dictionary, its verb ones last.
	like := []Meaning{
		{POS: "A", Text: "Giống, giống nhau."}, {POS: "A", Text: "Thực đúng."}, {POS: "A", Text: "Thích, sẵn lòng."},
		{POS: "E", Text: "Như."}, {POS: "C", Text: "Thay cho as như."},
		{POS: "N", Text: "Cái thích."}, {POS: "V", Text: "Thích ưa, chuộng, yêu."}, {POS: "V", Text: "Muốn."},
	}
	got := pickMeanings(like, 4)
	want := []Meaning{like[0], like[5], like[6], like[3]}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("like = %+v, want first sense, then noun and verb, then preposition: %+v", got, want)
	}

	// One part of speech: the senses in order.
	run := []Meaning{{POS: "N", Text: "1"}, {POS: "N", Text: "2"}, {POS: "N", Text: "3"}, {POS: "N", Text: "4"}, {POS: "N", Text: "5"}}
	if got := pickMeanings(run, 4); !reflect.DeepEqual(got, run[:4]) {
		t.Errorf("one part of speech = %+v", got)
	}
	// Two parts of speech: one sense of each, then the next senses.
	light := []Meaning{{POS: "N", Text: "n1"}, {POS: "N", Text: "n2"}, {POS: "N", Text: "n3"}, {POS: "N", Text: "n4"}, {POS: "V", Text: "v1"}}
	if got := pickMeanings(light, 4); !reflect.DeepEqual(got, []Meaning{light[0], light[4], light[1], light[2]}) {
		t.Errorf("two parts of speech = %+v", got)
	}
	// Few senses: all of them.
	if got := pickMeanings(like[:2], 4); len(got) != 2 {
		t.Errorf("few senses = %+v", got)
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
