package grammar

import "testing"

func TestPublishedSources(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	choice := Exercise{ID: "p1", Kind: KindChoice, Text: "She ___ a teacher.", Options: []string{"am", "is", "are"}, AnswerIndex: 1}
	fix := Exercise{ID: "m1", Kind: KindFix, Text: "She are a teacher.", Sentence: "She is a teacher."}
	e.lessons.lessons["a1-to-be"] = Lesson{PointID: "a1-to-be", Status: StatusPublished, Content: Content{
		Practice: []Exercise{choice}, Mastery: []Exercise{fix}, Mistakes: []Mistake{{Wrong: "I is", Right: "I am", NoteVi: "I đi với am"}},
	}}
	e.lessons.lessons["a1-present-simple"] = Lesson{PointID: "a1-present-simple", Status: StatusDraft, Content: Content{Practice: []Exercise{choice}}}

	got, err := e.svc.PublishedSources(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Point.ID != "a1-to-be" || got[0].Point.TitleVi == "" {
		t.Fatalf("sources = %+v", got)
	}
	if len(got[0].Exercises) != 2 || got[0].Exercises[0].ID != "p1" || got[0].Exercises[1].ID != "m1" {
		t.Fatalf("exercises = %+v", got[0].Exercises)
	}
	// The lesson's mistake comes first, then the hand-written ones of the point.
	if len(got[0].Mistakes) < 2 || got[0].Mistakes[0].Wrong != "I is" || len(got[0].Mistakes) != 1+len(MistakesOf("a1-to-be")) {
		t.Fatalf("mistakes = %+v", got[0].Mistakes)
	}
}
