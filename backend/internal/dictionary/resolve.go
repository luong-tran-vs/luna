package dictionary

import (
	"context"
	"regexp"
	"strings"
)

// Lookuper finds one exact (lowercase) word.
type Lookuper interface {
	Lookup(ctx context.Context, word string) (Entry, bool, error)
}

// inflectionNote matches dictionary definitions that only say which word an inflected form
// belongs to, e.g. "động từ quá khứ của go.", "Số nhiều của child",
// "Động từ chia ở ngôi thứ ba số ít của study", "cấp so sánh của good".
var inflectionNote = regexp.MustCompile(
	`(?i)^\s*(?:dạng\s+|động từ\s+)?(?:quá khứ|phân từ|số nhiều|chia ở ngôi|ngôi thứ|cấp so sánh|so sánh|hiện tại)[^.]*?\bcủa\s+([a-z][a-z' -]*?)\s*\.?\s*$`)

// inflectionBase returns the base word named by an inflection note, or "".
func inflectionBase(definition string) string {
	m := inflectionNote.FindStringSubmatch(definition)
	if m == nil {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(m[1]))
}

// Resolve looks word up and returns the entry of its base form: Entry.Word is the lemma.
// Order: the word itself (following its first meaning when that meaning is only an inflection
// note, "went" → "go"), then irregular and rule-based candidates ("stopped" → "stop").
func Resolve(ctx context.Context, d Lookuper, word string) (Entry, bool, error) {
	for _, c := range Candidates(word) {
		e, ok, err := d.Lookup(ctx, c)
		if err != nil {
			return Entry{}, false, err
		}
		if !ok {
			continue
		}
		if base := inflectionBase(e.Meanings[0].Text); base != "" && base != c {
			b, ok, err := d.Lookup(ctx, base)
			if err != nil {
				return Entry{}, false, err
			}
			if ok {
				return b, true, nil
			}
		}
		return e, true, nil
	}
	return Entry{}, false, nil
}

// Resolve is Resolve on this dictionary.
func (d *SQLite) Resolve(ctx context.Context, word string) (Entry, bool, error) {
	return Resolve(ctx, d, word)
}

// Resolve always reports not found.
func (None) Resolve(context.Context, string) (Entry, bool, error) { return Entry{}, false, nil }
