package curriculum

import (
	"strings"
	"testing"

	"github.com/luongtran/luna/backend/internal/lesson"
)

func TestGrammarUses(t *testing.T) {
	cases := []struct {
		point, text string
		want        int
	}{
		{"a1-to-be", "I am Minh. She is from Hue and they're friends. It isn't far.", 4},
		{"a1-have-has", "I have a cat. She has two dogs.", 2},
		{"a1-there-is-are", "There is a bed. There are two chairs. Is there a lamp?", 3},
		{"a1-prepositions-time", "I get up at 6 and go to school on Monday in the morning.", 3},
		{"a1-like-love-ing", "I like singing. She loves dancing but hates cooking.", 3},
		{"a1-imperatives", "Go straight. Then turn left! \"Don't stop,\" he says.", 3},
		{"a1-can", "Birds can fly. Fish can't walk.", 2},
	}
	for _, c := range cases {
		got, ok := GrammarUses(c.point, c.text)
		if !ok || got != c.want {
			t.Errorf("GrammarUses(%s) = %d, %v; want %d", c.point, got, ok, c.want)
		}
	}
	if _, ok := GrammarUses("a2-will", "I will go."); ok {
		t.Error("an A2 point has no pattern")
	}
}

func items(c Checks) map[string]Check {
	out := map[string]Check{}
	for _, it := range c.Items {
		out[it.Key] = it
	}
	return out
}

func TestInspect(t *testing.T) {
	content := "Hello! My name is Minh. I am twelve and I am from Da Nang. This is my friend Anna. " +
		"She is from London. She is not a student at my school, but we are good friends. Are you from Da Nang too? " +
		"Anna is anxious about the exam."
	l := lesson.Lesson{
		Level: "A1", Content: content, GrammarPointID: "a1-to-be", TargetWords: []string{"friend", "city"},
		Sentences: []lesson.Sentence{{Text: "This is my friend Anna."}},
		Practice: &lesson.Practice{Dialogue: &lesson.Dialogue{Turns: []lesson.Turn{
			{Text: "Where are you from?"}, {Text: "This is my friend Anna."}, {Text: "Is she nice?"},
		}}},
	}
	got := items(Inspect(l))
	if g := got["grammar"]; !g.OK || !strings.Contains(g.TextVi, "lần") {
		t.Errorf("grammar %+v", g)
	}
	for _, k := range []string{"otherPerson", "negative", "question"} {
		if !got[k].OK {
			t.Errorf("%s failed: %+v", k, got[k])
		}
	}
	if tw := got["targetWords"]; tw.OK || !tw.Warn || !strings.Contains(tw.TextVi, "city") {
		t.Errorf("target words %+v", tw)
	}
	if ln := got["length"]; ln.OK || !ln.Warn {
		t.Errorf("a 50-word text is short for A1: %+v", ln)
	}
	if d := got["dialogue"]; d.OK || !strings.Contains(d.TextVi, "2 lượt hỏi, 1 câu chép") {
		t.Errorf("dialogue %+v", d)
	}
	if lv := got["level"]; lv.OK || !lv.Warn || !strings.Contains(lv.TextVi, "anxious (A2)") {
		t.Errorf("level %+v", lv)
	}

	none := items(Inspect(lesson.Lesson{Level: "A1", Content: "I like it."}))
	if g := none["grammar"]; g.OK || g.Warn {
		t.Errorf("no grammar point is a failure: %+v", g)
	}
	if _, ok := none["dialogue"]; ok {
		t.Error("no practice, no dialogue check")
	}
}
