package games

import (
	"slices"
	"testing"
)

func TestSpellableAndSameForm(t *testing.T) {
	t.Parallel()
	for w, want := range map[string]bool{"house": true, "cat": true, "computer": true, "go": false, "keyboards": false,
		"ice cream": false, "t-shirt": false, "café": false} {
		if got := spellable(w); got != want {
			t.Errorf("spellable(%q) = %v", w, got)
		}
	}
	for _, p := range [][2]string{{"apple", "apples"}, {"play", "played"}, {"box", "boxes"}, {"go", "going"}, {"Apple", "apple"}} {
		if !sameForm(p[0], p[1]) {
			t.Errorf("sameForm(%q, %q) = false", p[0], p[1])
		}
	}
	for _, p := range [][2]string{{"pear", "peach"}, {"car", "card"}, {"cat", "category"}} {
		if sameForm(p[0], p[1]) {
			t.Errorf("sameForm(%q, %q) = true", p[0], p[1])
		}
	}
}

func TestWordPoolOrderAndFallbacks(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	pool, err := e.svc.wordPool(t.Context(), "u1", 100, nil, e.svc.newRand())
	if err != nil || len(pool) != len(fruit) {
		t.Fatalf("pool = %+v, %v", pool, err)
	}
	// The two due cards come first.
	if !pool[0].Due || !pool[1].Due || pool[2].Due {
		t.Fatalf("due first: %+v", pool[:3])
	}
	if pool[0].ImageURL == "" || pool[0].MeaningVi == "" {
		t.Fatalf("decorated: %+v", pool[0])
	}

	// A learner without cards: the studied lessons, then the lesson to study now.
	e.lessons.words["L1"] = []LessonWord{{Lemma: "house"}, {Lemma: "apple", MeaningVi: "quả táo"}}
	e.lessons.words["L2"] = []LessonWord{{Lemma: "school", MeaningVi: "trường học"}, {Lemma: "apple"}}
	e.study.studied["u9"] = []string{"L1"}
	e.study.current["u9"] = "L2"
	pool, err = e.svc.wordPool(t.Context(), "u9", 100, nil, e.svc.newRand())
	if err != nil {
		t.Fatal(err)
	}
	var lemmas []string
	for _, w := range pool {
		lemmas = append(lemmas, w.Lemma)
	}
	if len(pool) != 3 || pool[2].Lemma != "school" || pool[2].Tier != 2 || !slices.Contains(lemmas, "house") {
		t.Fatalf("fallback pool = %+v", pool)
	}
	// The word bank fills a missing meaning.
	for _, w := range pool {
		if w.Lemma == "house" && w.MeaningVi != "ngôi nhà" {
			t.Fatalf("house = %+v", w)
		}
	}
	// Enough words already: the lessons are not read.
	e.cards.cards["u9"] = []Card{{Lemma: "dog", MeaningVi: "con chó"}}
	pool, _ = e.svc.wordPool(t.Context(), "u9", 1, nil, e.svc.newRand())
	if len(pool) != 1 || pool[0].Lemma != "dog" {
		t.Fatalf("enough = %+v", pool)
	}
}
