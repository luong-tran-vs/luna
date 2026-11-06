package grammar

import (
	"errors"
	"strings"
	"testing"

	"github.com/luongtran/luna/backend/internal/ai"
)

func pictureExercise(n string) ai.GrammarExercise {
	return ai.GrammarExercise{
		Kind: "picture", PromptVi: "Nhìn ảnh, chọn từ đúng.", Text: "The cat is ___ the sofa " + n + ".",
		Options: []string{"in", "on", "under"}, AnswerIndex: 2, ExplanationVi: "Con mèo ở dưới ghế.",
		Picture: "a small orange cat sleeping under a blue sofa " + n,
	}
}

func TestPictureExerciseValidation(t *testing.T) {
	t.Parallel()
	good := normalizeExercises(exercisesFromAI([]ai.GrammarExercise{pictureExercise("1")}), "p")[0]
	if p := exerciseProblems(good); p != nil {
		t.Fatalf("good picture: %v", p)
	}
	if good.Answers != nil || good.Words != nil || good.Picture == "" {
		t.Fatalf("normalized = %+v", good)
	}
	for name, edit := range map[string]func(*ai.GrammarExercise){
		"no blank":       func(e *ai.GrammarExercise) { e.Text = "The cat is under the sofa." },
		"one option":     func(e *ai.GrammarExercise) { e.Options = []string{"under"} },
		"answer outside": func(e *ai.GrammarExercise) { e.AnswerIndex = 3 },
		"no picture":     func(e *ai.GrammarExercise) { e.Picture = " " },
	} {
		e := pictureExercise("1")
		edit(&e)
		if p := exerciseProblems(normalizeExercises(exercisesFromAI([]ai.GrammarExercise{e}), "p")[0]); p == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	// Another kind never keeps a picture description.
	c := goodChoice("1")
	c.Picture = "stale"
	if got := normalizeExercises(exercisesFromAI([]ai.GrammarExercise{c}), "p")[0]; got.Picture != "" {
		t.Fatalf("choice kept its picture: %q", got.Picture)
	}
}

func TestDrawablePointsAskForPictureExercises(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	if _, _, err := e.svc.Generate(t.Context(), "a1-prepositions-place", false); err != nil {
		t.Fatal(err)
	}
	if !e.ai.req.Pictures {
		t.Fatal("a drawable point did not ask for picture exercises")
	}
	if _, _, err := e.svc.Generate(t.Context(), "a1-articles", false); err != nil {
		t.Fatal(err)
	}
	if e.ai.req.Pictures {
		t.Fatal("a point that cannot be drawn asked for picture exercises")
	}
}

func TestPictureExercisesReachLearnersOnceDrawn(t *testing.T) {
	t.Parallel()
	e, store, draw := newImageEnv(t)
	e.ai.content.Practice = append(exercises("p", 7), pictureExercise("1"))
	if _, _, err := e.svc.Generate(t.Context(), "a1-to-be", true); err != nil {
		t.Fatal(err)
	}
	if _, err := e.svc.Publish(t.Context(), "a1-to-be", true); err != nil {
		t.Fatal(err)
	}
	d, err := e.svc.Get(t.Context(), "u1", "a1-to-be")
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Content.Practice) != 7 || len(d.Pictures) != 0 {
		t.Fatalf("before its picture: %d practice, pictures %v", len(d.Content.Practice), d.Pictures)
	}
	// A round of the 7 shown exercises is accepted.
	if _, _, err := e.svc.RecordAttempt(t.Context(), "u1", "a1-to-be", Attempt{Kind: AttemptPractice, Correct: 7, Total: 7}); err != nil {
		t.Fatalf("attempt: %v", err)
	}

	// The batch draws the examples and the picture of the exercise, told what it shows.
	if _, err := e.svc.StartImageBatch(t.Context(), ImageBatchRequest{PointIDs: []string{"a1-to-be"}}); err != nil {
		t.Fatal(err)
	}
	st := waitImageBatch(t, e.svc)
	if st.Total != 4 || len(st.Saved) != 4 {
		t.Fatalf("batch = %+v", st)
	}
	last := draw.requests[len(draw.requests)-1].Example
	if !last.Exercise || !strings.Contains(last.En, "orange cat") {
		t.Fatalf("picture request = %+v", last)
	}
	if _, ok := store.images["a1-to-be/"+PictureKey(Exercise{Picture: "a small orange cat sleeping under a blue sofa 1"})]; !ok {
		t.Fatal("picture not stored under its key")
	}

	d, _ = e.svc.Get(t.Context(), "u1", "a1-to-be")
	pic := d.Content.Practice[len(d.Content.Practice)-1]
	if len(d.Content.Practice) != 8 || pic.Kind != KindPicture || pic.Picture != "" || d.Pictures[pic.ID] == "" {
		t.Fatalf("after: %d practice, last %+v, pictures %v", len(d.Content.Practice), pic, d.Pictures)
	}
}

func TestPointsThatCannotBeDrawnGetNoPictures(t *testing.T) {
	t.Parallel()
	e, _, _ := newImageEnv(t)
	if _, _, err := e.svc.Generate(t.Context(), "a1-articles", false); err != nil {
		t.Fatal(err)
	}
	var verr *ValidationError
	if _, err := e.svc.StartImageBatch(t.Context(), ImageBatchRequest{PointIDs: []string{"a1-articles"}}); !errors.As(err, &verr) {
		t.Fatalf("not drawable: %v", err)
	}
	if p, _ := e.svc.point("a1-articles"); p.Drawable {
		t.Fatal("a1-articles is drawable")
	}
	if p, _ := e.svc.point("a1-prepositions-place"); !p.Drawable {
		t.Fatal("a1-prepositions-place is not drawable")
	}
}
