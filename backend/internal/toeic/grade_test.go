package toeic

import "testing"

func TestScaled(t *testing.T) {
	t.Parallel()
	cases := []struct {
		sec            Section
		correct, total int
		want           int
	}{
		{Listening, 0, 100, 5}, {Listening, 50, 100, 245}, {Reading, 50, 100, 215}, {Listening, 100, 100, 495},
		{Reading, 100, 100, 495}, {Reading, 98, 100, 495}, {Listening, 25, 50, 245}, {Reading, 38, 50, 365},
	}
	for _, c := range cases {
		if got := scaled(c.sec, c.correct, c.total); got != c.want {
			t.Errorf("scaled(%s, %d, %d) = %d, want %d", c.sec, c.correct, c.total, got, c.want)
		}
	}
	// Between marks: 30 right → 105 + (5/25)·140 = 133 → 135.
	if got := scaled(Listening, 30, 100); got != 135 {
		t.Errorf("interpolated = %d", got)
	}
	for c := 0; c <= 100; c++ {
		if s := scaled(Reading, c, 100); s%5 != 0 || s < 5 || s > 495 {
			t.Fatalf("scaled(%d) = %d", c, s)
		}
	}
}

func TestGrade(t *testing.T) {
	t.Parallel()
	groups := map[string]Group{"p5-a": p5("p5-a", 1), "p5-b": p5("p5-b", 2), "p7-a": p7("p7-a", 2)}
	g := groups["p7-a"]
	g.Version = 3
	groups["p7-a"] = g
	a := Attempt{Mode: ModePractice, Source: Source{Kind: SourceTest},
		QuestionIDs: []string{"p5-a#1", "p5-b#1", "p7-a#1", "p7-a#2"},
		Answers:     map[string]int{"p5-a#1": 1, "p5-b#1": 0, "p7-a#1": 1}, ElapsedMs: 9000}
	r := grade(groups, a)
	if r.Correct != 2 || r.Wrong != 1 || r.Skipped != 1 || r.ByQuestion["p5-b#1"] != OutcomeWrong || r.ByQuestion["p7-a#2"] != OutcomeSkipped {
		t.Fatalf("result = %+v", r)
	}
	if r.Total != 0 || r.Listening != nil || r.Reading != nil || r.Version["p7-a"] != 3 || r.DurationMs != 9000 {
		t.Fatalf("practice scores = %+v", r)
	}
	// A whole reading section of a test gets an estimated reading score.
	a.Mode, a.QuestionIDs = ModeTest, []string{"p5-a#1", "p6-x#1", "p7-a#1"}
	groups["p6-x"] = p6("p6-x")
	r = grade(groups, a)
	if r.Reading == nil || r.Reading.Total != 3 || r.Listening != nil || r.Total != 0 {
		t.Fatalf("reading = %+v", r)
	}
}
