package topic

import (
	"slices"
	"testing"
)

func TestLoadSeedIsValid(t *testing.T) {
	t.Parallel()

	seed, err := LoadSeed()
	if err != nil {
		t.Fatalf("LoadSeed: %v", err)
	}
	if len(seed.Topics) != 42 {
		t.Fatalf("topics = %d, want 42", len(seed.Topics))
	}
	total := 0
	for _, st := range seed.Topics {
		words, err := CleanWords(words(st.Words...))
		if err != nil {
			t.Errorf("%s: %v", st.Name, err)
		}
		total += len(words)
	}
	if total != 1257 {
		t.Fatalf("words = %d, want 1257", total)
	}
}

func TestSeedKey(t *testing.T) {
	t.Parallel()
	pairs := [][2]string{
		{"Sức khoẻ", "sức  khỏe"},
		{"Thuý", "thúy"},
		{"HOÀ BÌNH", "hòa bình"},
		{"  Gia   đình ", "gia đình"},
	}
	for _, p := range pairs {
		if SeedKey(p[0]) != SeedKey(p[1]) {
			t.Errorf("SeedKey(%q) = %q, SeedKey(%q) = %q", p[0], SeedKey(p[0]), p[1], SeedKey(p[1]))
		}
	}
	if SeedKey("Gia đình") == SeedKey("Gia dinh") {
		t.Error("diacritics must still matter")
	}
}

func TestPlanSeed(t *testing.T) {
	t.Parallel()

	seed := Seed{Topics: []SeedTopic{
		{Name: "Gia đình", Words: []string{"Family", " Parents "}},
		{Name: "Sức khoẻ", Words: []string{"Health"}},
	}}
	targets := []SeedTarget{
		{ID: "a1", Name: "gia đình"},
		{ID: "a2", Name: "Gia Đình"},
		{ID: "b1", Name: "Sức khỏe"},
		{ID: "c1", Name: "Chung"},
	}
	got := PlanSeed(targets, seed)
	want := []SeedWrite{
		{ID: "a1", Words: []string{"Family", "Parents"}, Matched: true},
		{ID: "a2", Words: []string{"Family", "Parents"}, Matched: true},
		{ID: "b1", Words: []string{"Health"}, Matched: true},
		{ID: "c1", Words: []string{}, Matched: false},
	}
	if len(got) != len(want) {
		t.Fatalf("writes = %+v", got)
	}
	for i := range want {
		if got[i].ID != want[i].ID || got[i].Matched != want[i].Matched || !slices.Equal(got[i].Words, want[i].Words) ||
			got[i].Words == nil {
			t.Errorf("write %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}
