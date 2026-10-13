package main

import (
	"reflect"
	"strings"
	"testing"
)

const sample = "\ufeff@music /'mju:zik/\n" +
	"*  danh từ\n" +
	"- nhạc, âm nhạc\n" +
	"=to have an ear for music+ có năng khiếu về âm nhạc\n" +
	"\n" +
	"@acid-proof /'æsid'pru:f/ (acid-resisting) /'æsidri'zistiɳ/\n" +
	"-resisting) \n" +
	"*  tính từ\n" +
	"- chịu axit\n" +
	"\n" +
	"@went /gou/\n" +
	"* xem go\n" +
	"\n" +
	"@party /'pɑ:ti/\n" +
	"*  danh từ\n" +
	"- đảng\n" +
	"!the Communist Party\n" +
	"- đảng Cộng sản\n" +
	"*  ngoại động từ+ (party) /'pɑ:ti/\n" +
	"- mở tiệc\n" +
	"\n" +
	"@problem /problem/\n" +
	"*  danh từ\n" +
	"- vấn đề\n"

func TestParseHND(t *testing.T) {
	got, err := parseHND(strings.NewReader(sample))
	if err != nil {
		t.Fatal(err)
	}
	want := []hndEntry{
		{Word: "music", IPA: "/ˈmjuːzik/", Meanings: []meaning{{POS: "N", Text: "nhạc, âm nhạc", Source: sourceHND}}},
		// Only the first pronunciation; "-resisting)" is not a meaning (no space after "-").
		{Word: "acid-proof", IPA: "/ˈæsidˈpruːf/", Meanings: []meaning{{POS: "A", Text: "chịu axit", Source: sourceHND}}},
		// "went" has no meaning ("* xem go") and is left out; idiom meanings are skipped.
		{Word: "party", IPA: "/ˈpɑːti/", Meanings: []meaning{
			{POS: "N", Text: "đảng", Source: sourceHND},
			{POS: "V", Text: "mở tiệc", Source: sourceHND},
		}},
		// A "pronunciation" that is the word itself is dropped.
		{Word: "problem", Meanings: []meaning{{POS: "N", Text: "vấn đề", Source: sourceHND}}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseHND =\n%+v\nwant\n%+v", got, want)
	}
}

func TestHNDIPA(t *testing.T) {
	for _, c := range []struct{ word, in, want string }{
		{"understand", ",ʌndə'stænd", "/ˌʌndəˈstænd/"},
		{"a", "ei, ə", "/ei, ə/"},
		{"seeing", "'si:iɳ", "/ˈsiːiŋ/"},
		{"usually", "'ju: u li", ""},
		{"problem", "problem", ""},
		{"x", "", ""},
	} {
		if got := hndIPA(c.word, c.in); got != c.want {
			t.Errorf("hndIPA(%q, %q) = %q, want %q", c.word, c.in, got, c.want)
		}
	}
}

func TestGlosses(t *testing.T) {
	for _, c := range []struct {
		in   string
		want []string
	}{
		{"Bài hát, điệu hát.", []string{"bài hát", "điệu hát"}},
		{"(âm nhạc) la; nốt La", []string{"la", "nốt la"}},
		{"(viết tắt)", []string{"(viết tắt)"}},
	} {
		if got := glosses(c.in); !reflect.DeepEqual(got, c.want) {
			t.Errorf("glosses(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestMerge(t *testing.T) {
	d := newDictionary()
	d.addMinhqnd("song", []meaning{
		{POS: "N", Text: "Tiếng hát; tiếng hót.", Source: sourceMinhqnd},
		{POS: "N", Text: "Bài hát, điệu hát.", Source: sourceMinhqnd},
	}, []pronunciation{{IPA: "/ˈsɔŋ/"}})
	d.addMinhqnd("music", nil, nil) // in the first source without a Vietnamese meaning

	d.addHND(hndEntry{Word: "song", IPA: "/sɔɳ/", Meanings: []meaning{
		{POS: "N", Text: "bài hát", Source: sourceHND},           // already given: skipped
		{POS: "N", Text: "bài hát, khúc hát", Source: sourceHND}, // "khúc hát" is new: added
	}})
	d.addHND(hndEntry{Word: "music", IPA: "/ˈmjuːzik/", Meanings: []meaning{
		{POS: "N", Text: "nhạc, âm nhạc", Source: sourceHND},
	}})
	d.addHND(hndEntry{Word: "guitarist", Meanings: []meaning{{POS: "N", Text: "người chơi ghita", Source: sourceHND}}})

	song := d.entries["song"]
	if texts := meaningTexts(song); !reflect.DeepEqual(texts, []string{"Tiếng hát; tiếng hót.", "Bài hát, điệu hát.", "bài hát, khúc hát"}) {
		t.Errorf("song meanings = %q", texts)
	}
	if !reflect.DeepEqual(song.IPAs, []pronunciation{{IPA: "/ˈsɔŋ/"}}) {
		t.Errorf("song keeps its own pronunciation, got %v", song.IPAs)
	}
	music := d.entries["music"]
	if texts := meaningTexts(music); !reflect.DeepEqual(texts, []string{"nhạc, âm nhạc"}) || music.IPAs[0].IPA != "/ˈmjuːzik/" {
		t.Errorf("music = %+v", music)
	}
	want := stats{
		MinhqndWords: 1, MinhqndMeanings: 2,
		HNDAdded: 3, HNDSkipped: 1, HNDOnlyWords: 1, HNDFilledWords: 1, HNDIPAs: 1,
	}
	if d.stats != want {
		t.Errorf("stats = %+v, want %+v", d.stats, want)
	}
	if !reflect.DeepEqual(d.order, []string{"song", "music", "guitarist"}) {
		t.Errorf("order = %v", d.order)
	}
}

func meaningTexts(e *entry) []string {
	var out []string
	for _, m := range e.Meanings {
		out = append(out, m.Text)
	}
	return out
}
