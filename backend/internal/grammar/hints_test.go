package grammar

import (
	"errors"
	"testing"
)

func TestHintSentence(t *testing.T) {
	t.Parallel()
	e := Exercise{Kind: KindChoice, Text: "She ___  a teacher.", Options: []string{"am", "is"}, AnswerIndex: 1}
	if got := HintSentence(e); got != "She is a teacher." {
		t.Fatalf("sentence = %q", got)
	}
	for name, bad := range map[string]Exercise{
		"fill":      {Kind: KindFill, Text: "She ___ a teacher.", Options: []string{"is"}},
		"no blank":  {Kind: KindChoice, Text: "Pick one.", Options: []string{"am", "is"}, AnswerIndex: 1},
		"two":       {Kind: KindChoice, Text: "___ ___", Options: []string{"am", "is"}, AnswerIndex: 1},
		"bad index": {Kind: KindChoice, Text: "She ___.", Options: []string{"am"}, AnswerIndex: 3},
	} {
		if HintKey(bad) != "" {
			t.Errorf("%s: has a key", name)
		}
	}
	other := e
	other.AnswerIndex = 0
	if HintKey(other) == HintKey(e) {
		t.Fatal("the key does not follow the answer")
	}
}

func TestHintPicturesBatchAndLearnerURL(t *testing.T) {
	t.Parallel()
	e, _, draw := newImageEnv(t)
	list, err := e.svc.HintPictures(t.Context(), "a1-to-be")
	if err != nil || len(list) == 0 || list[0].Drawn {
		t.Fatalf("hints = %+v, %v", list, err)
	}
	n := len(list)
	if c, _ := e.svc.CountHintPictures(t.Context(), []string{"a1-to-be", "a1-to-be", "nope"}); c.Total != n || c.Drawn != 0 {
		t.Fatalf("count = %+v", c)
	}

	st, err := e.svc.StartImageBatch(t.Context(), ImageBatchRequest{PointIDs: []string{"a1-to-be"}, Hints: true})
	if err != nil {
		t.Fatal(err)
	}
	if st.Total != n {
		t.Fatalf("total = %d, want %d (only the hint pictures)", st.Total, n)
	}
	st = waitImageBatch(t, e.svc)
	if len(st.Saved) != n || draw.requests[0].Example == nil || draw.requests[0].Example.En != list[0].Sentence {
		t.Fatalf("batch = %+v, first request = %+v", st, draw.requests[0].Example)
	}
	if c, _ := e.svc.CountHintPictures(t.Context(), []string{"a1-to-be"}); c.Drawn != n {
		t.Fatalf("drawn = %+v", c)
	}
	var verr *ValidationError
	if _, err := e.svc.StartImageBatch(t.Context(), ImageBatchRequest{PointIDs: []string{"a1-to-be"}, Hints: true}); !errors.As(err, &verr) {
		t.Fatalf("nothing left: %v", err)
	}

	// The learner gets the key of each drawn hint picture under "hint:<id>".
	l, _ := e.svc.AdminGet(t.Context(), "a1-to-be")
	_, pictures, err := e.svc.learnerContent(t.Context(), "a1-to-be", l.Content)
	if err != nil || pictures[hintPrefix+list[0].ExerciseID] != list[0].Key {
		t.Fatalf("pictures = %v, %v", pictures, err)
	}
}
