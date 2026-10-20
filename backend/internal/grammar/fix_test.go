package grammar

import (
	"testing"

	"github.com/luongtran/luna/backend/internal/ai"
)

// Added 2026-10-20: the "fix" exercises, the hand-written mistakes in the request and the check of
// the Vietnamese against the answer.

func goodFix(n string) ai.GrammarExercise {
	return ai.GrammarExercise{
		Kind: "fix", PromptVi: "Tìm và sửa lỗi", Text: "She are a teacher " + n + ".", Sentence: "She is a teacher " + n + ".",
		Answers: []string{"She's a teacher " + n + "."}, Options: []string{"x"}, Words: []string{"y"},
		ExplanationVi: "She đi với is.",
	}
}

func TestFixExercisesAreValidatedAndCleaned(t *testing.T) {
	t.Parallel()
	c := Normalize(Content{Practice: []Exercise{fromAIExercise(goodFix("a"))}})
	e := c.Practice[0]
	if e.Options != nil || e.Words != nil || e.Sentence != "She is a teacher a." || len(e.Answers) != 1 {
		t.Fatalf("exercise = %+v", e)
	}
	if p := exerciseProblems(e); p != nil {
		t.Fatalf("valid fix rejected: %v", p)
	}
	for name, f := range map[string]func(*Exercise){
		"no correction":       func(e *Exercise) { e.Sentence = "" },
		"same as the mistake": func(e *Exercise) { e.Sentence = "she are a teacher a" },
		"too short":           func(e *Exercise) { e.Sentence = "She is" },
		"answer is the text":  func(e *Exercise) { e.Answers = []string{"She are a teacher a."} },
	} {
		bad := e
		bad.Answers = append([]string(nil), e.Answers...)
		f(&bad)
		if exerciseProblems(bad) == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

// fromAIExercise converts one AI exercise the way Generate does.
func fromAIExercise(x ai.GrammarExercise) Exercise {
	return exercisesFromAI([]ai.GrammarExercise{x})[0]
}

func TestGenerateSendsTheHandWrittenMistakes(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	if _, _, err := e.svc.Generate(t.Context(), point, false); err != nil {
		t.Fatal(err)
	}
	got := e.ai.req.CommonMistakes
	if len(got) == 0 || got[0] != (ai.GrammarMistake{Wrong: "I happy.", Right: "I am happy.", NoteVi: MistakesOf(point)[0].NoteVi}) {
		t.Fatalf("mistakes = %+v", got)
	}
}

func TestCheckComparesFixesAndFlagsVietnameseThatDiffers(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	c := aiContent()
	c.Practice[0], c.Practice[3] = goodFix("a"), goodFix("b")
	e.ai.content = c
	e.ai.solve = func(req ai.SolveRequest) ([]ai.Solution, error) {
		out := solveLikeFixtures(req)
		for i := range out {
			switch out[i].ExerciseID {
			case "p1": // another accepted correction: fine
				out[i].Sentence = "she's a teacher a"
			case "p4": // a correction not stored
				out[i].Sentence = "She was a teacher b."
			case "p3": // a reorder whose Vietnamese says something else
				out[i].MeaningOff, out[i].NoteVi = true, "Tiếng Việt là câu hỏi."
			case "m3": // no note: a default one
				out[i].MeaningOff = true
			}
		}
		return out, nil
	}
	l, _, err := e.svc.Generate(t.Context(), point, false)
	if err != nil {
		t.Fatal(err)
	}
	got := flagByID(l)
	if len(got) != 3 || got["p4"].Kind != CheckMismatch || got["p3"].Kind != CheckMeaning ||
		got["p3"].NoteVi != "Tiếng Việt là câu hỏi." || got["m3"].Kind != CheckMeaning || got["m3"].NoteVi == "" {
		t.Fatalf("checks = %+v", l.Checks)
	}
}

// Fixed 2026-10-20: the AI often writes the tiles without the full stop of the sentence.
func TestReorderTilesIgnorePunctuationAndCase(t *testing.T) {
	t.Parallel()
	e := fromAIExercise(ai.GrammarExercise{
		Kind: "reorder", PromptVi: "Sắp xếp", Text: "Tôi có hai anh trai.", Sentence: "I have two brothers.",
		Words: []string{"i", "have", "two", "brothers", "dog"}, ExplanationVi: "Số nhiều thêm -s.",
	})
	e = Normalize(Content{Practice: []Exercise{e}}).Practice[0]
	if p := exerciseProblems(e); p != nil {
		t.Fatalf("rejected: %v", p)
	}
	e.Words = []string{"I", "have", "two", "dog"}
	if exerciseProblems(e) == nil {
		t.Fatal("a missing word was accepted")
	}
}
