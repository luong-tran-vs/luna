package dictionary

import (
	"os"
	"testing"
	"time"
)

// TestRealDictionary runs against the downloaded dictionary when LUNA_DICTIONARY points to it:
//
//	LUNA_DICTIONARY=<absolute path to dictionary.db> go test ./internal/dictionary -run Real -v
func TestRealDictionary(t *testing.T) {
	path := os.Getenv("LUNA_DICTIONARY")
	if path == "" {
		t.Skip("LUNA_DICTIONARY not set")
	}
	d, err := Open(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })

	want := map[string]string{
		"went": "go", "park": "park", "studies": "study", "stopped": "stop",
		"children": "child", "taken": "take", "goes": "go", "running": "", "laughing": "",
	}
	for word, lemma := range want {
		start := time.Now()
		e, ok, err := d.Resolve(t.Context(), word)
		elapsed := time.Since(start)
		if err != nil || !ok {
			t.Errorf("%s: not found (%v)", word, err)
			continue
		}
		t.Logf("%-9s → %-8s %-14s %-10v %s", word, e.Word, e.IPA, elapsed, e.Meanings[0].Text)
		if lemma != "" && e.Word != lemma {
			t.Errorf("%s → %s, want %s", word, e.Word, lemma)
		}
		if elapsed > 100*time.Millisecond {
			t.Errorf("%s took %v", word, elapsed)
		}
	}
}
