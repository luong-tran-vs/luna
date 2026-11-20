package toeic

import "testing"

func TestRepairEvidence(t *testing.T) {
	t.Parallel()
	g := Group{Part: 7, Docs: []Doc{
		{Kind: "notice", Body: []Block{{Kind: "p", Text: "Welcome."}, {Kind: "p", Text: "The garden opens at 9 a.m. on Saturdays."}}},
		{Kind: "email", Body: []Block{{Kind: "p", Text: "I’d like to book the “Blue Room” for Friday."}}},
	}, Questions: []Question{{No: 1, Evidence: []Ref{
		{Doc: 0, Block: 1, Text: "opens at 9 a.m."},                    // right: kept as it is
		{Doc: 0, Block: 0, Text: "OPENS AT 9  a.m. on"},                // wrong block, other case and spacing
		{Doc: 0, Block: 0, Text: "I'd like to book the \"Blue Room\""}, // wrong text, plain quotes
		{Doc: 1, Block: 0, Text: "closes at noon"},                     // nowhere: dropped
	}}}}
	repairEvidence(&g)
	got := g.Questions[0].Evidence
	want := []Ref{
		{Doc: 0, Block: 1, Text: "opens at 9 a.m."},
		{Doc: 0, Block: 1, Text: "opens at 9 a.m. on"},
		{Doc: 1, Block: 0, Text: "I’d like to book the “Blue Room”"},
	}
	if len(got) != len(want) {
		t.Fatalf("evidence = %+v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("evidence[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
	if errs := validateGroup(Group{Key: "p7-x", Part: 7, Level: 600, Docs: g.Docs, Questions: []Question{{
		No: 1, Prompt: "Why?", Options: []string{"a", "b", "c", "d"}, ExplainVi: "Vì.", Tags: []string{"detail"}, Evidence: got,
	}, {No: 2, Prompt: "What?", Options: []string{"a", "b", "c", "d"}, ExplainVi: "Vì.", Tags: []string{"detail"}}}}); len(errs) > 0 {
		t.Fatalf("repaired group still invalid: %v", errs)
	}
}

func TestRepairEvidenceInScript(t *testing.T) {
	t.Parallel()
	g := Group{Part: 3, Script: []Turn{{Speaker: "m", Text: "Hello there."}, {Speaker: "w", Text: "Our printer broke down this morning."}},
		Questions: []Question{{No: 1, Evidence: []Ref{{Doc: -1, Turn: 0, Text: "printer broke down"}}}}}
	repairEvidence(&g)
	if e := g.Questions[0].Evidence; len(e) != 1 || e[0].Turn != 1 || e[0].Doc != -1 {
		t.Fatalf("evidence = %+v", e)
	}
}

func TestRepairKinds(t *testing.T) {
	t.Parallel()
	g := Group{Docs: []Doc{{Kind: "Advertisement", Body: []Block{{Kind: "paragraph"}, {Kind: "Heading"}, {Kind: "p"}}}},
		Graphic: &Graphic{Kind: "Floor plan"}}
	repairKinds(&g)
	if g.Docs[0].Kind != "ad" || g.Docs[0].Body[0].Kind != "p" || g.Docs[0].Body[1].Kind != "h" || g.Docs[0].Body[2].Kind != "p" ||
		g.Graphic.Kind != "floor-plan" {
		t.Fatalf("kinds = %+v %+v", g.Docs[0], g.Graphic)
	}
}

func TestNotEnglish(t *testing.T) {
	t.Parallel()
	en := p7("p7-en", 2)
	en.Docs[0].Meta = []KV{{K: "From", V: "Café Lumière"}}
	if got := notEnglish(en); len(got) != 0 {
		t.Fatalf("English group flagged: %v", got)
	}
	vi := p7("p7-vi", 2)
	vi.Docs[0].Meta = []KV{{K: "Gửi", V: "Tất cả nhân viên"}}
	vi.Questions[1].Options = []string{"Một", "Hai", "Ba", "Bốn"}
	got := notEnglish(vi)
	if got["docs[0]"] == "" || got["questions[1]"] == "" || got["questions[0]"] != "" {
		t.Fatalf("Vietnamese group = %v", got)
	}
	// The Vietnamese fields themselves are fine.
	ok := p5("p5-ok", 1)
	ok.Questions[0].TranslateVi = "Bản dịch tiếng Việt."
	if got := notEnglish(ok); len(got) != 0 {
		t.Fatalf("translation flagged: %v", got)
	}
}

func TestDraftDropsGroupsWrittenInVietnamese(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	vi := p5("", 0)
	vi.Questions[0].Prompt = "Báo cáo đã được viết ___ bởi nhóm."
	e.svc.d.Writer = &fakeWriter{text: draftJSON(t, p5("", 1), vi)}
	rep, err := e.svc.DraftGroups(t.Context(), DraftInput{Part: 5, Level: 600})
	if err != nil || len(rep.Created) != 1 || len(rep.Dropped) != 1 || rep.Dropped[0].Fields["questions[0]"] == "" {
		t.Fatalf("report = %+v, %v", rep, err)
	}
}

func TestSplitSingles(t *testing.T) {
	t.Parallel()
	g := p5("", 0)
	q2 := g.Questions[0]
	q2.No = 2
	g.Questions = append(g.Questions, q2)
	out := splitSingles([]Group{g, p7("p7", 3)})
	if len(out) != 3 || len(out[0].Questions) != 1 || out[1].Questions[0].No != 1 || len(out[2].Questions) != 3 {
		t.Fatalf("split = %+v", out)
	}
}

func TestDropHeaderBlocks(t *testing.T) {
	t.Parallel()
	g := Group{Docs: []Doc{
		{Kind: "memo", Body: []Block{{Kind: "p", Text: "To: All staff"}, {Kind: "p", Text: "From: HR"}, {Kind: "p", Text: "We will [[1]] soon."}}},
		{Kind: "email", Body: []Block{{Kind: "p", Text: "To: Ann\nSubject: Hello\nDate: May 2\nThanks for the [[2]]."}}},
		{Kind: "memo", Body: []Block{{Kind: "p", Text: "To: everyone, the party is on Friday."}}},
	}}
	dropHeaderBlocks(&g)
	if len(g.Docs[0].Body) != 1 || g.Docs[0].Body[0].Text != "We will [[1]] soon." {
		t.Fatalf("doc 0 = %+v", g.Docs[0].Body)
	}
	if g.Docs[1].Body[0].Text != "Thanks for the [[2]]." {
		t.Fatalf("doc 1 = %q", g.Docs[1].Body[0].Text)
	}
	if len(g.Docs[2].Body) != 1 {
		t.Fatalf("a text of one block is kept: %+v", g.Docs[2].Body)
	}
}
