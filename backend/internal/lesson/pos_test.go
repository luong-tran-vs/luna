package lesson

import (
	"testing"

	"github.com/luongtran/luna/backend/internal/dictionary"
)

// posDict has "watch" with its noun senses first and the verb "xem" only after the first three
// meanings, like the real dictionary, plus a word with one part of speech only.
var posDict = fakeDict{
	"watch": {
		Word: "watch",
		Meanings: []dictionary.Meaning{
			{POS: "N", Text: "Đồng hồ đeo tay."}, {POS: "N", Text: "Sự canh gác."}, {POS: "N", Text: "Người canh gác."},
		},
		Senses: []dictionary.Meaning{
			{POS: "N", Text: "Đồng hồ đeo tay."}, {POS: "N", Text: "Sự canh gác."}, {POS: "N", Text: "Người canh gác."},
			{POS: "N", Text: "Phiên gác."}, {POS: "V", Text: "Xem, nhìn, theo dõi."}, {POS: "V", Text: "Canh gác, trông nom."},
		},
	},
	"park":  {Word: "park", Meanings: []dictionary.Meaning{{POS: "N", Text: "Công viên."}}},
	"light": {Word: "light", Meanings: []dictionary.Meaning{{POS: "N", Text: "Ánh sáng."}, {POS: "A", Text: "Nhẹ."}}},
}

func TestDictionaryPOS(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name, lemma, text, meaning, want string
	}{
		{"meaning matches a later verb sense", "watch", "watch", "xem", POSVerb},
		{"meaning matches the noun sense", "watch", "watch", "đồng hồ", POSNoun},
		{"meaning with filler words", "watch", "watching", "đang theo dõi", POSVerb},
		{"two parts of speech match equally", "watch", "watch", "canh gác", ""},
		{"no sense matches: do not guess", "watch", "watch", "quan sát", ""},
		{"-ed form is a verb", "watch", "watched", "", POSVerb},
		{"one part of speech only", "park", "park", "chỗ đỗ xe", POSNoun},
		{"unknown word", "zorb", "zorb", "bóng", ""},
		{"several words", "give up", "gave up", "từ bỏ", POSPhrase},
		{"adjective sense by meaning", "light", "light", "nhẹ", POSAdjective},
	} {
		got, err := dictionaryPOS(t.Context(), posDict, c.lemma, c.text, c.meaning)
		if err != nil || got != c.want {
			t.Errorf("%s: %q, %v; want %q", c.name, got, err, c.want)
		}
	}
	if got, _ := dictionaryPOS(t.Context(), nil, "watch", "watch", "xem"); got != "" {
		t.Errorf("no dictionary: %q", got)
	}
}
