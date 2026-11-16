package games

import (
	"errors"
	"testing"

	"github.com/luongtran/luna/backend/internal/ai"
)

func goodSpot() SpotItem {
	return SpotItem{Sentence: "She go to school every day.", Wrong: "go", Right: "goes", Distractors: []string{"going", "went"}, RuleVi: "She + V-s."}
}

func TestValidateSpot(t *testing.T) {
	t.Parallel()
	if f := ValidateSpot(goodSpot()); len(f) != 0 {
		t.Fatalf("good = %v", f)
	}
	cases := map[string]func(*SpotItem){
		"sentence":    func(s *SpotItem) { s.Sentence = "She go." },
		"wrong":       func(s *SpotItem) { s.Wrong = "went" },
		"right":       func(s *SpotItem) { s.Right = "Go" },
		"distractors": func(s *SpotItem) { s.Distractors = []string{"goes", "went"} },
		"ruleVi":      func(s *SpotItem) { s.RuleVi = " " },
	}
	for field, change := range cases {
		it := goodSpot()
		change(&it)
		if f := ValidateSpot(it); f[field] == "" {
			t.Errorf("%s: %v", field, f)
		}
	}
	twice := goodSpot()
	twice.Sentence = "They go and go every day."
	if f := ValidateSpot(twice); f["wrong"] == "" {
		t.Errorf("wrong twice: %v", f)
	}
	same := goodSpot()
	same.Distractors = []string{"went", "went"}
	if f := ValidateSpot(same); f["distractors"] == "" {
		t.Errorf("same distractors: %v", f)
	}
}

func TestGenerateSpots(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.grammar.sources = []PointSource{presentSimple()}
	e.ai.items = []ai.GrammarSpotItem{
		{Sentence: "My dog eat bones.", Wrong: "eat", Right: "eats", Distractors: []string{"eating", "ate"}, RuleVi: "Chủ ngữ số ít + V-s."},
		{Sentence: "Tom like milk.", Wrong: "loves", Right: "likes", Distractors: []string{"liking", "liked"}, RuleVi: "r"},     // mistake not in the sentence
		{Sentence: "He play football.", Wrong: "play", Right: "plays", Distractors: []string{"playing", "played"}, RuleVi: "r"}, // already a fix exercise
		{Sentence: "She watch TV.", Wrong: "watch", Right: "watches", Distractors: []string{"watches", "watched"}, RuleVi: "r"}, // distractor = right
	}
	res, err := e.svc.GenerateSpots(t.Context(), "a1-present-simple")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Created) != 1 || res.Dropped != 3 || res.Created[0].Status != SpotDraft || res.Created[0].Level != "A1" || res.Created[0].ID == "" {
		t.Fatalf("result = %+v", res)
	}
	if e.ai.req.TitleEn != "Present simple" || len(e.ai.req.Existing) == 0 || e.ai.req.Count != 10 {
		t.Fatalf("request = %+v", e.ai.req)
	}
	// A draft is not played until approved.
	if items, _ := e.spots.Approved(t.Context(), nil); len(items) != 0 {
		t.Fatalf("approved = %+v", items)
	}
	counts, _ := e.svc.SpotCounts(t.Context())
	if len(counts) != 1 || counts[0].Draft != 1 || counts[0].Spot != 0 || counts[0].Choice != 2 || counts[0].Fix != 2 || counts[0].Total != 4 {
		t.Fatalf("counts = %+v", counts)
	}
	if _, err := e.svc.ApproveSpot(t.Context(), res.Created[0].ID); err != nil {
		t.Fatal(err)
	}
	counts, _ = e.svc.SpotCounts(t.Context())
	if counts[0].Spot != 1 || counts[0].Draft != 0 || counts[0].Total != 5 {
		t.Fatalf("counts after approval = %+v", counts)
	}

	// Edits are checked again; the status stays.
	var verr *ValidationError
	if _, err := e.svc.UpdateSpot(t.Context(), res.Created[0].ID, SpotEdit{Sentence: "x"}); !errors.As(err, &verr) {
		t.Fatalf("bad edit: %v", err)
	}
	it, err := e.svc.UpdateSpot(t.Context(), res.Created[0].ID, SpotEdit(func() SpotEdit {
		g := goodSpot()
		return SpotEdit{Sentence: g.Sentence, Wrong: g.Wrong, Right: g.Right, Distractors: g.Distractors, RuleVi: g.RuleVi}
	}()))
	if err != nil || it.Sentence != goodSpot().Sentence || it.Status != SpotApproved {
		t.Fatalf("edit = %+v, %v", it, err)
	}
	if err := e.svc.DeleteSpot(t.Context(), it.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := e.svc.ApproveSpot(t.Context(), it.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("approve deleted: %v", err)
	}
}

func TestGenerateSpotsErrors(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.grammar.sources = []PointSource{presentSimple()}
	if _, err := e.svc.GenerateSpots(t.Context(), "b2-unknown"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown point: %v", err)
	}
	e.ai.err = ai.ErrNotConfigured
	_, err := e.svc.GenerateSpots(t.Context(), "a1-present-simple")
	if !errors.Is(err, ai.ErrNotConfigured) || !IsAIError(err) {
		t.Fatalf("AI off: %v", err)
	}
	if len(e.spots.items) != 0 {
		t.Fatalf("saved = %+v", e.spots.items)
	}
}

func TestImportSpots(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.grammar.sources = []PointSource{presentSimple()}
	items := []ai.GrammarSpotItem{
		{Sentence: " My dog eat bones. ", Wrong: "eat", Right: "eats", Distractors: []string{"eating", "ate"}, RuleVi: "Chủ ngữ số ít + V-s."},
		{Sentence: "Tom like milk.", Wrong: "loves", Right: "likes", Distractors: []string{"liking", "liked"}, RuleVi: "r"},
		{Sentence: "He play football.", Wrong: "play", Right: "plays", Distractors: []string{"playing", "played"}, RuleVi: "r"},
		{Sentence: "My dog eat bones.", Wrong: "eat", Right: "eats", Distractors: []string{"eaten", "ate"}, RuleVi: "r"},
	}
	dry, err := e.svc.ImportSpots(t.Context(), "a1-present-simple", items, true)
	if err != nil {
		t.Fatal(err)
	}
	if dry.Kept != 1 || dry.Dropped != 3 || len(dry.Created) != 0 || len(e.spots.items) != 0 {
		t.Fatalf("dry run = %+v", dry)
	}
	if r := dry.Reports; !r[0].Kept || r[1].Fields["wrong"] == "" || !r[2].Duplicate || !r[3].Duplicate {
		t.Fatalf("reports = %+v", r)
	}
	res, err := e.svc.ImportSpots(t.Context(), "a1-present-simple", items, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Created) != 1 || res.Created[0].Status != SpotDraft || res.Created[0].Sentence != "My dog eat bones." || res.Created[0].Level != "A1" {
		t.Fatalf("result = %+v", res)
	}
	// Importing again finds the saved sentence.
	again, _ := e.svc.ImportSpots(t.Context(), "a1-present-simple", items[:1], false)
	if again.Kept != 0 || !again.Reports[0].Duplicate {
		t.Fatalf("again = %+v", again)
	}
	if _, err := e.svc.ImportSpots(t.Context(), "zz", items, false); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown point: %v", err)
	}
	var verr *ValidationError
	if _, err := e.svc.ImportSpots(t.Context(), "a1-present-simple", nil, false); !errors.As(err, &verr) {
		t.Fatalf("no items: %v", err)
	}
}

func TestApproveDrafts(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.spots.items = []SpotItem{
		{ID: "a", PointID: "p1", Status: SpotDraft}, {ID: "b", PointID: "p1", Status: SpotApproved},
		{ID: "c", PointID: "p2", Status: SpotDraft}, {ID: "d", PointID: "p3", Status: SpotDraft},
	}
	if n, err := e.svc.ApproveDrafts(t.Context(), []string{"p1", "p2"}); err != nil || n != 2 {
		t.Fatalf("approve p1,p2 = %d, %v", n, err)
	}
	if e.spots.items[3].Status != SpotDraft {
		t.Fatalf("p3 approved: %+v", e.spots.items[3])
	}
	if n, _ := e.svc.ApproveDrafts(t.Context(), nil); n != 1 {
		t.Fatalf("approve all = %d", n)
	}
}
