package games

import (
	"errors"
	"testing"
)

func TestPictureRound(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	// "apples" is another form of "apple" and "táo" the same meaning: never with apple.
	e.cards.cards["u1"] = append(e.cards.cards["u1"],
		Card{Lemma: "apples", MeaningVi: "những quả táo", LessonID: "L1"}, Card{Lemma: "malus", MeaningVi: "quả táo", LessonID: "L1"})
	qs, err := e.svc.PictureRound(t.Context(), "u1")
	if err != nil {
		t.Fatal(err)
	}
	if len(qs) != pictureQuestions {
		t.Fatalf("%d questions", len(qs))
	}
	seen := map[string]int{}
	for i, q := range qs {
		if len(q.Options) != 4 || q.Options[q.Answer] != q.Lemma || q.ImageURL == "" {
			t.Fatalf("question %d = %+v", i, q)
		}
		distinct := map[string]bool{}
		for _, o := range q.Options {
			distinct[o] = true
			if o != q.Lemma && sameForm(o, q.Lemma) {
				t.Fatalf("question %d: %q is a form of %q", i, o, q.Lemma)
			}
			if q.Lemma == "apple" && o == "malus" {
				t.Fatalf("question %d: same meaning", i)
			}
		}
		if len(distinct) != 4 {
			t.Fatalf("question %d repeats an option: %v", i, q.Options)
		}
		seen[q.Lemma]++
	}
	// 8 words with a picture for 40 questions: each asked 5 times, none skipped.
	for _, f := range fruit {
		if seen[f.lemma] != 5 {
			t.Fatalf("asked %v", seen)
		}
	}
	// The first 8 questions are 8 different words.
	first := map[string]bool{}
	for _, q := range qs[:8] {
		first[q.Lemma] = true
	}
	if len(first) != 8 {
		t.Fatalf("a word repeats while others wait: %v", first)
	}
}

func TestPictureRoundNotEnough(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.cards.cards["u9"] = []Card{{Lemma: "apple", MeaningVi: "táo"}, {Lemma: "pear", MeaningVi: "lê"}, {Lemma: "plane", MeaningVi: "máy bay"}}
	if _, err := e.svc.PictureRound(t.Context(), "u9"); !errors.Is(err, ErrNotEnough) {
		t.Fatalf("3 pictures: %v", err)
	}
}

func TestLettersRound(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.cards.cards["u1"] = append(e.cards.cards["u1"], Card{Lemma: "ice cream", MeaningVi: "kem"}, Card{Lemma: "strawberry", MeaningVi: "dâu tây"})
	words, err := e.svc.LettersRound(t.Context(), "u1")
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, w := range words {
		if !spellable(w.Lemma) || w.MeaningVi == "" || seen[w.Lemma] {
			t.Fatalf("word = %+v", w)
		}
		seen[w.Lemma] = true
	}
	if len(words) != len(fruit) {
		t.Fatalf("%d words", len(words))
	}
	e.cards.cards["u9"] = []Card{{Lemma: "cat", MeaningVi: "mèo"}, {Lemma: "dog", MeaningVi: "chó"}}
	if _, err := e.svc.LettersRound(t.Context(), "u9"); !errors.Is(err, ErrNotEnough) {
		t.Fatalf("2 words: %v", err)
	}
}

func TestMemoryRound(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	pairs, err := e.svc.MemoryRound(t.Context(), "u1", false, "")
	if err != nil || len(pairs) != memoryPairs {
		t.Fatalf("pairs = %+v, %v", pairs, err)
	}
	// Due words first.
	due := 0
	for _, p := range pairs {
		if p.Lemma == "apple" || p.Lemma == "banana" {
			due++
		}
	}
	if due != 2 {
		t.Fatalf("due words missing: %+v", pairs)
	}

	// A lesson's words; a word without a picture shows its meaning.
	e.lessons.words["L5"] = []LessonWord{
		{Lemma: "house", MeaningVi: "ngôi nhà"}, {Lemma: "apple", MeaningVi: "quả táo"}, {Lemma: "pear", MeaningVi: "quả lê"},
		{Lemma: "lemon", MeaningVi: "quả chanh"}, {Lemma: "mango", MeaningVi: "quả xoài"}, {Lemma: "grape", MeaningVi: "quả nho"},
	}
	if _, err := e.svc.MemoryRound(t.Context(), "u1", false, "L5"); !errors.Is(err, ErrLessonLocked) {
		t.Fatalf("locked lesson: %v", err)
	}
	e.study.open["L5"] = true
	pairs, err = e.svc.MemoryRound(t.Context(), "u1", false, "L5")
	if err != nil || len(pairs) != 6 {
		t.Fatalf("lesson pairs = %+v, %v", pairs, err)
	}
	for _, p := range pairs {
		if p.Lemma == "house" && (p.ImageURL != "" || p.MeaningVi != "ngôi nhà") {
			t.Fatalf("house = %+v", p)
		}
	}
	// The last word has no picture: it comes after the five with one.
	if pairs[5].Lemma != "house" {
		t.Fatalf("pictures first: %+v", pairs)
	}
	e.lessons.words["L6"] = e.lessons.words["L5"][:4]
	e.study.open["L6"] = true
	if _, err := e.svc.MemoryRound(t.Context(), "u1", false, "L6"); !errors.Is(err, ErrNotEnough) {
		t.Fatalf("4 words: %v", err)
	}
}
