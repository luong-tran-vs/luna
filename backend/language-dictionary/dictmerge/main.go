// Command dictmerge builds english-dictionary.db, the offline English–Vietnamese dictionary of
// the Reading step, from two sources:
//
//   - minhqnd/dictionary v2.0.0 (MIT, built from Wiktionary): its English words with Vietnamese
//     meanings, kept as they are and first;
//   - Hồ Ngọc Đức's Free Vietnamese Dictionary Project, anhviet109K.txt (GPL): its meanings are
//     added after them, except a meaning whose every idea is already given; its pronunciation is
//     used when a word has none.
//
// The result has the tables dictionary.SQLite reads, so the backend uses it as is. Run it through
// language-dictionary/build.sh, which downloads and checks both sources:
//
//	go run ./language-dictionary/dictmerge -minhqnd source-wiktionary.db \
//		-hnd source-ho-ngoc-duc.txt -out english-dictionary.db
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
)

func main() {
	minhqnd := flag.String("minhqnd", "", "minhqnd/dictionary SQLite file (dictionary.db)")
	hnd := flag.String("hnd", "", "Hồ Ngọc Đức English–Vietnamese text file (anhviet109K.txt)")
	out := flag.String("out", "english-dictionary.db", "merged SQLite file to write")
	flag.Parse()
	if *minhqnd == "" || *hnd == "" {
		flag.Usage()
		os.Exit(2)
	}
	if err := run(context.Background(), *minhqnd, *hnd, *out); err != nil {
		fmt.Fprintln(os.Stderr, "dictmerge:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, minhqndPath, hndPath, outPath string) error {
	d := newDictionary()
	if err := readMinhqnd(ctx, minhqndPath, d); err != nil {
		return fmt.Errorf("minhqnd: %w", err)
	}

	f, err := os.Open(hndPath) //nolint:gosec // operator-given path
	if err != nil {
		return fmt.Errorf("hnd: %w", err)
	}
	entries, err := parseHND(f)
	_ = f.Close()
	if err != nil {
		return fmt.Errorf("hnd: %w", err)
	}
	for _, e := range entries {
		d.addHND(e)
	}

	if err := write(ctx, outPath, d); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	s := d.stats
	fmt.Printf("minhqnd: %d words, %d meanings\n", s.MinhqndWords, s.MinhqndMeanings)
	fmt.Printf("Hồ Ngọc Đức: %d entries read, %d meanings added, %d skipped as already given\n",
		len(entries), s.HNDAdded, s.HNDSkipped)
	phrases := 0
	for _, e := range entries {
		if strings.Contains(e.Word, " ") {
			phrases++
		}
	}
	fmt.Printf("  %d new words or phrases (%d phrases read), %d words that had no meaning, %d pronunciations added\n",
		s.HNDOnlyWords, phrases, s.HNDFilledWords, s.HNDIPAs)
	fmt.Printf("wrote %s\n", outPath)
	return nil
}
