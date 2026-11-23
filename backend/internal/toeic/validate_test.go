package toeic

import (
	"errors"
	"strings"
	"testing"
)

func fieldsOf(t *testing.T, err error) map[string]string {
	t.Helper()
	var verr *ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("want ValidationError, got %v", err)
	}
	return verr.Fields
}

func TestValidateGoodGroups(t *testing.T) {
	t.Parallel()
	gs := []Group{p1("p1-a"), p2("p2-a"), p3("p3-a"), p4("p4-a"), p5("p5-a", 1), p6("p6-a"), p7("p7-a", 3)}
	for i := range gs {
		Normalize(&gs[i])
	}
	if err := Validate(gs); err != nil {
		t.Fatalf("valid groups: %v", err)
	}
}

func TestValidateErrors(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		group func() Group
		field string
	}{
		{"answer out of range", func() Group { g := p5("p5-x", 0); g.Questions[0].Answer = 5; return g }, "questions[0].answer"},
		{"too few options", func() Group { g := p5("p5-x", 0); g.Questions[0].Options = g.Questions[0].Options[:3]; return g }, "questions[0].options"},
		{"duplicate options", func() Group { g := p5("p5-x", 0); g.Questions[0].Options[1] = "Alpha"; return g }, "questions[0].options"},
		{"part 2 needs three options", func() Group { g := p2("p2-x"); g.Questions[0].Options = append(g.Questions[0].Options, "x"); return g }, "questions[0].options"},
		{"unknown tag", func() Group { g := p5("p5-x", 0); g.Questions[0].Tags = []string{"nope"}; return g }, "questions[0].tags"},
		{"tag of another part", func() Group { g := p5("p5-x", 0); g.Questions[0].Tags = []string{"detail"}; return g }, "questions[0].tags"},
		{"no tag", func() Group { g := p5("p5-x", 0); g.Questions[0].Tags = nil; return g }, "questions[0].tags"},
		{"no explanation", func() Group { g := p5("p5-x", 0); g.Questions[0].ExplainVi = " "; return g }, "questions[0].explainVi"},
		{"part 5 blank", func() Group { g := p5("p5-x", 0); g.Questions[0].Prompt = "No blank here."; return g }, "questions[0].prompt"},
		{"part 3 one speaker", func() Group { g := p3("p3-x"); g.Script[1].Speaker = "m"; return g }, "script"},
		{"speaker given as a name", func() Group {
			g := p4("p4-x")
			for i := range g.Script {
				g.Script[i].Speaker = "Sarah Jenkins"
			}
			return g
		}, "script[0].speaker"},
		{"part 3 bad accent", func() Group { g := p3("p3-x"); g.Script[0].Accent = "fr"; return g }, "script[0].accent"},
		{"part 3 two questions", func() Group { g := p3("p3-x"); g.Questions = g.Questions[:2]; return g }, "questions"},
		{"part 6 missing blank", func() Group {
			g := p6("p6-x")
			g.Docs[0].Body[0].Text = strings.Replace(g.Docs[0].Body[0].Text, "[[3]]", "___", 1)
			return g
		}, "docs[0].body"},
		{"part 7 too many questions", func() Group { return p7("p7-x", 6) }, "questions"},
		{"evidence not in block", func() Group { g := p7("p7-x", 2); g.Questions[1].Evidence[0].Text = "closes at noon"; return g }, "questions[1].evidence[0].text"},
		{"evidence turn", func() Group {
			g := p3("p3-x")
			g.Questions[0].Evidence = []Ref{{Doc: -1, Turn: 1, Text: "brochures"}}
			return g
		}, "questions[0].evidence[0].text"},
		{"graphic tag without graphic", func() Group { g := p3("p3-x"); g.Questions[2].Tags = []string{"graphic"}; return g }, "graphic"},
		{"graphic title too long", func() Group {
			g := p3("p3-x")
			g.Graphic = testTable()
			g.Graphic.Title = strings.Repeat("format presentation ", 10)
			return g
		}, "graphic.title"},
		{"graphic table without rows", func() Group { g := p3("p3-x"); g.Graphic = testTable(); g.Graphic.Rows = nil; return g }, "graphic.rows"},
		{"graphic row missing a column", func() Group {
			g := p3("p3-x")
			g.Graphic = testTable()
			g.Graphic.Rows[1] = []string{"Room B"}
			return g
		}, "graphic.rows"},
		{"graphic floor plan without rooms", func() Group { g := p3("p3-x"); g.Graphic = &Graphic{Kind: "floor-plan"}; return g }, "graphic.cells"},
		{"graphic bar chart with one bar", func() Group {
			g := p3("p3-x")
			g.Graphic = &Graphic{Kind: "bar-chart", Bars: []Bar{{Label: "Sales", Value: 3}}}
			return g
		}, "graphic.bars"},
		{"bad key", func() Group { g := p5("P5 bad key!", 0); return g }, "key"},
		{"part 1 without scene", func() Group { g := p1("p1-x"); g.Scene = ""; return g }, "scene"},
		{"level", func() Group { g := p5("p5-x", 0); g.Level = 500; return g }, "level"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			g := c.group()
			Normalize(&g)
			f := fieldsOf(t, Validate([]Group{g}))
			if _, ok := f["groups[0]."+c.field]; !ok {
				t.Fatalf("fields = %v, want groups[0].%s", f, c.field)
			}
		})
	}
}

// testTable is a valid two-row schedule for graphic cases.
func testTable() *Graphic {
	return &Graphic{Kind: "schedule", Title: "Room Schedule", Columns: []string{"Room", "Time"},
		Rows: [][]string{{"Room A", "9:00"}, {"Room B", "10:00"}}}
}

func TestValidateGoodGraphics(t *testing.T) {
	t.Parallel()
	for _, gr := range []*Graphic{
		testTable(),
		{Kind: "floor-plan", Cells: []GraphicCell{{Label: "Lobby"}, {Label: "Room 1", Col: 1}}},
		{Kind: "bar-chart", Bars: []Bar{{Label: "A", Value: 1}, {Label: "B", Value: 2}}},
	} {
		g := p3("p3-g")
		g.Graphic = gr
		Normalize(&g)
		if err := Validate([]Group{g}); err != nil {
			t.Fatalf("graphic %s: %v", gr.Kind, err)
		}
	}
}

func TestValidateDuplicateKeyInFile(t *testing.T) {
	t.Parallel()
	f := fieldsOf(t, Validate([]Group{p5("p5-a", 0), p5("p5-a", 1)}))
	if _, ok := f["groups[1].key"]; !ok {
		t.Fatalf("fields = %v", f)
	}
}

func TestEstimateListening(t *testing.T) {
	t.Parallel()
	if m := estimateListening([]Group{p5("p5-a", 0)}); m != 0 {
		t.Fatalf("reading only = %v", m)
	}
	one := estimateListening([]Group{p2("p2-a")})
	two := estimateListening([]Group{p2("p2-a"), p2("p2-b")})
	if one <= 0 || two <= one {
		t.Fatalf("one = %v, two = %v", one, two)
	}
}
