package lesson

import (
	"slices"
	"testing"

	"github.com/luongtran/luna/backend/internal/ai"
	"github.com/luongtran/luna/backend/internal/job"
)

var focusDict = fakeDict{
	"family":        entry("family", "/ˈfæməli/", "Gia đình.", "Họ hàng."),
	"take a shower": entry("take a shower", "", "Tắm vòi sen."),
}

func TestFocusWords(t *testing.T) {
	t.Parallel()
	got := focusWords("My families took a shower. The season is cold.", []string{"Family", "take a shower", "son", "Uncle"})
	if want := []string{"Family", "take a shower"}; !slices.Equal(got, want) {
		t.Fatalf("focus = %q", got)
	}
	if got := focusWords("Hello.", nil); len(got) != 0 {
		t.Fatalf("no words = %q", got)
	}
}

func TestAddMissedFocus(t *testing.T) {
	t.Parallel()
	sentences := []string{"We went home.", "My families took a shower.", "Uncle Tom is here."}
	anns := []Annotation{
		{Text: "went", Lemma: "go", MeaningVi: "đã đi", SentenceIndex: 0},
		{Text: "Uncle", Lemma: "uncle", MeaningVi: "chú", SentenceIndex: 2},
	}
	got, err := addMissedFocus(t.Context(), anns, []string{"Family", "take a shower", "Uncle", "go", "cousin"}, sentences, focusDict)
	if err != nil {
		t.Fatal(err)
	}
	want := append(slices.Clone(anns),
		Annotation{Text: "families", Lemma: "family", MeaningVi: "Gia đình.", SentenceIndex: 1},
		Annotation{Text: "took a shower", Lemma: "take a shower", MeaningVi: "Tắm vòi sen.", SentenceIndex: 1},
	)
	if !slices.Equal(got, want) {
		t.Fatalf("annotations =\n%+v\nwant\n%+v", got, want)
	}
	if got, _ := addMissedFocus(t.Context(), anns, []string{"Family"}, sentences, nil); len(got) != len(anns) {
		t.Fatalf("no dictionary = %+v", got)
	}
}

func TestProcessAnnotateFocusWords(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.svc.Dict = focusDict
	e.topics.mu.Lock()
	b1 := e.topics.topics["topic-b1"]
	b1.Words = []string{"park", "Family", "give up"}
	e.topics.topics["topic-b1"] = b1
	e.topics.mu.Unlock()
	l := e.create(t, func(in *Input) { in.Content = "We went to the park with my family. He gave up smoking." })
	e.ai.result = []ai.Annotation{{Text: "gave up", Lemma: "give up", MeaningVi: "đã bỏ", SentenceIndex: 1}}

	if err := e.svc.ProcessAnnotate(t.Context(), jobFor(l, job.TypeAnnotate)); err != nil {
		t.Fatal(err)
	}
	if e.ai.calls != 1 || !slices.Equal(e.ai.annotateReq.FocusWords, []string{"park", "Family", "give up"}) {
		t.Fatalf("calls %d request %+v", e.ai.calls, e.ai.annotateReq)
	}
	got, _ := e.lessons.Get(t.Context(), l.ID)
	var texts []string
	for _, a := range got.Annotations {
		texts = append(texts, a.Text)
	}
	// "park" is not in the dictionary: skipped; "family" is added.
	if want := []string{"gave up", "family"}; !slices.Equal(texts, want) {
		t.Fatalf("annotations = %q", texts)
	}
}

func TestProcessAnnotateWithoutTopicWords(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.svc.Dict = focusDict
	l := e.create(t)
	e.ai.result = []ai.Annotation{{Text: "gave up", Lemma: "give up", MeaningVi: "đã bỏ", SentenceIndex: 1}}
	if err := e.svc.ProcessAnnotate(t.Context(), jobFor(l, job.TypeAnnotate)); err != nil {
		t.Fatal(err)
	}
	if len(e.ai.annotateReq.FocusWords) != 0 || e.ai.annotateReq.Level != "B1" || len(e.ai.annotateReq.Sentences) != 3 {
		t.Fatalf("request = %+v", e.ai.annotateReq)
	}
}
