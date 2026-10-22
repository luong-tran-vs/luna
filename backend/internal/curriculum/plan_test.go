package curriculum

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/luongtran/luna/backend/internal/grammar"
)

func TestA1Plan(t *testing.T) {
	p, ok := ByLevel("A1")
	if !ok {
		t.Fatal("no A1 course")
	}
	if len(p.Topics) != 12 {
		t.Fatalf("A1 has %d topics, want 12", len(p.Topics))
	}
	main, extra := 0, 0
	taught := map[string]bool{}
	for i, top := range p.Topics {
		if top.Number != i+1 {
			t.Errorf("topic %q has number %d, want %d", top.Name, top.Number, i+1)
		}
		n := 0
		for _, s := range top.Slots {
			for _, id := range s.Grammar {
				taught[id] = true
			}
			if s.Kind == KindMain {
				main++
			} else {
				extra++
				n++
			}
		}
		if n < 2 || n > 3 {
			t.Errorf("topic %q has %d extra lessons, want 2 or 3", top.Name, n)
		}
	}
	if main != 36 || extra != 29 {
		t.Errorf("A1 has %d main and %d extra lessons, want 36 and 29", main, extra)
	}
	for _, pt := range grammar.Default().ByLevel("A1") {
		if !taught[pt.ID] {
			t.Errorf("no A1 lesson uses %s", pt.ID)
		}
	}
	if _, ok := ByLevel("B2"); ok {
		t.Error("B2 has no course yet")
	}
}

// The course names topics as the catalogue's seeds do, so that applying it finds them.
func TestA1TopicsAreInTheSeeds(t *testing.T) {
	names := map[string]bool{}
	for _, f := range []string{"../topic/seed/extra-topics.json", "../topic/seed/topic-words.json"} {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		var seed struct {
			Topics []struct {
				Name string `json:"name"`
			} `json:"topics"`
		}
		if err := json.Unmarshal(data, &seed); err != nil {
			t.Fatal(err)
		}
		for _, s := range seed.Topics {
			names[NameKey(s.Name)] = true
		}
	}
	p, _ := ByLevel("A1")
	for _, top := range p.Topics {
		if !names[NameKey(top.Name)] {
			t.Errorf("topic %q is not in the seeds", top.Name)
		}
	}
}

func TestValidateRejects(t *testing.T) {
	good := func() Plan {
		return Plan{Level: "A1", Topics: []Topic{{Name: "T", Grammar: []string{"a1-to-be"}, Slots: []Slot{
			{Key: "k1", Kind: KindMain, Title: "One", Grammar: []string{"a1-to-be"}, Idea: "i"},
			{Key: "k2", Kind: KindMain, Title: "Two", Grammar: []string{"a1-to-be"}, Idea: "i"},
			{Key: "k3", Kind: KindMain, Title: "Three", Grammar: []string{"a1-to-be"}, Idea: "i"},
			{Key: "k4", Kind: KindExtra, Title: "Four", Grammar: []string{"a1-to-be"}, Idea: "i"},
		}}}}
	}
	if err := good().Validate(grammar.Default()); err != nil {
		t.Fatalf("valid plan: %v", err)
	}
	for name, spoil := range map[string]func(*Plan){
		"A2 point":           func(p *Plan) { p.Topics[0].Slots[0].Grammar = []string{"a2-will"} },
		"unknown point":      func(p *Plan) { p.Topics[0].Grammar = []string{"nope"} },
		"repeated key":       func(p *Plan) { p.Topics[0].Slots[1].Key = "k1" },
		"repeated title":     func(p *Plan) { p.Topics[0].Slots[1].Title = "one" },
		"extra before main":  func(p *Plan) { p.Topics[0].Slots[2].Kind = KindExtra },
		"main after the 3rd": func(p *Plan) { p.Topics[0].Slots[3].Kind = KindMain },
		"no idea":            func(p *Plan) { p.Topics[0].Slots[0].Idea = " " },
	} {
		p := good()
		spoil(&p)
		if err := p.Validate(grammar.Default()); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
}

func TestPointTopicsAndLearnerTopic(t *testing.T) {
	topics, order, ok := PointTopics("A1")
	if !ok || len(order) != 18 {
		t.Fatalf("PointTopics = %d points, %v", len(order), ok)
	}
	if order[0] != "a1-to-be" || topics["a1-have-has"].Number != 2 || topics["a1-can"].Name != "Con vật" {
		t.Errorf("order %v, have-has %v, can %v", order[:3], topics["a1-have-has"], topics["a1-can"])
	}
	info, ok := LearnerTopic("A1", "  âm NHẠC ")
	if !ok || info.Number != 9 || len(info.GrammarVi) != 2 || !strings.HasPrefix(info.CanDoVi, "nói") {
		t.Fatalf("LearnerTopic = %+v, %v", info, ok)
	}
	if !info.Extra[TitleKey("A Concert in the Park")] || info.Extra[TitleKey("My Favorite Song")] {
		t.Errorf("extra lessons %v", info.Extra)
	}
	if _, ok := LearnerTopic("A1", "Công việc"); ok {
		t.Error("Công việc is not in the A1 course")
	}
}
