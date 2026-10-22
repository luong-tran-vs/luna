package lesson

import (
	"errors"
	"slices"
	"testing"

	"github.com/luongtran/luna/backend/internal/ai"
)

func TestCleanQuestionKinds(t *testing.T) {
	t.Parallel()
	four := []string{"Hanoi", "Hue", "Da Nang", "Can Tho"}
	tests := []struct {
		name      string
		in        Question
		wantKind  string
		wantQuote string
		wantOpts  []string
	}{
		{name: "gist", in: Question{Kind: " gist ", Options: four}, wantKind: KindGist, wantOpts: four},
		{name: "detail", in: Question{Kind: "detail", Options: four}, wantKind: KindDetail, wantOpts: four},
		{name: "unknown kind", in: Question{Kind: "why", Options: four}, wantOpts: four},
		{name: "tfn gets its options", in: Question{Kind: "tfn", Options: four}, wantKind: KindTFN, wantOpts: TFNOptions},
		{name: "tfn without options", in: Question{Kind: "tfn"}, wantKind: KindTFN, wantOpts: TFNOptions},
		{name: "vocab quote in content", in: Question{Kind: "vocab", Quote: " GOES to school ", Options: four},
			wantKind: KindVocab, wantQuote: "GOES to school", wantOpts: four},
		{name: "vocab quote not in content", in: Question{Kind: "vocab", Quote: "flies away", Options: four},
			wantKind: KindVocab, wantOpts: four},
		{name: "quote of another kind", in: Question{Kind: "detail", Quote: "goes to school", Options: four},
			wantKind: KindDetail, wantOpts: four},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := cleanQuestion(tt.in, extrasContent)
			if got.Kind != tt.wantKind || got.Quote != tt.wantQuote || !slices.Equal(got.Options, tt.wantOpts) {
				t.Fatalf("question = %+v", got)
			}
		})
	}

	// The tfn options are a copy.
	q := cleanQuestion(Question{Kind: KindTFN}, extrasContent)
	q.Options[0] = "x"
	if TFNOptions[0] != "Đúng" {
		t.Fatal("TFNOptions changed")
	}
}

func TestCleanExtrasQuestionKinds(t *testing.T) {
	t.Parallel()
	three := []string{"Hanoi", "Hue", "Da Nang"}
	in := ai.LessonExtras{Questions: []ai.Question{
		{Kind: "tfn", Prompt: "Tom lives in Hanoi.", PromptVi: "Tom sống ở Hà Nội.", AnswerIndex: 0, ExplanationVi: "Câu 1."},
		{Kind: "tfn", Prompt: "Tom has a cat.", AnswerIndex: 3, ExplanationVi: "Không có."}, // only 0..2
		{Kind: "gist", Prompt: "Three options?", Options: three, ExplanationVi: "x"},
		{Kind: "vocab", Prompt: "What does 'visit' mean?", Quote: "they visit Grandma", Options: []string{"thăm", "ăn", "ngủ", "chơi"},
			AnswerIndex: 0, ExplanationVi: "visit = thăm."},
	}}
	got := CleanExtras(in, extrasContent).Questions
	if len(got) != 2 {
		t.Fatalf("questions = %+v", got)
	}
	if got[0].Kind != KindTFN || got[0].PromptVi != "Tom sống ở Hà Nội." || !slices.Equal(got[0].Options, TFNOptions) {
		t.Errorf("tfn = %+v", got[0])
	}
	if got[1].Kind != KindVocab || got[1].Quote != "they visit Grandma" {
		t.Errorf("vocab = %+v", got[1])
	}
}

func TestValidateExtrasTFN(t *testing.T) {
	t.Parallel()
	x, err := ValidateExtras(ExtrasInput{Questions: []Question{
		{Kind: KindTFN, Prompt: "Tom lives in Hanoi.", Options: []string{"a", "b", "c", "d"}, AnswerIndex: 2, ExplanationVi: "x"},
	}}, extrasContent)
	if err != nil || !slices.Equal(x.Questions[0].Options, TFNOptions) {
		t.Fatalf("tfn: %+v, %v", x.Questions, err)
	}
	_, err = ValidateExtras(ExtrasInput{Questions: []Question{
		{Kind: KindTFN, Prompt: "Tom lives in Hanoi.", AnswerIndex: 3, ExplanationVi: "x"},
		{Prompt: "Three options?", Options: []string{"a", "b", "c"}, ExplanationVi: "x"},
	}}, extrasContent)
	var verr *ValidationError
	if !errors.As(err, &verr) || verr.Fields["questions.0.answerIndex"] == "" || verr.Fields["questions.1.options"] != "Cần đúng 4 lựa chọn" {
		t.Fatalf("err = %v", err)
	}
}

func TestAnswerTFNChoice(t *testing.T) {
	t.Parallel()
	r, lessons, _, l := newQuizEnv(t)
	qs := append(slices.Clone(quizQuestions), Question{
		Kind: KindTFN, Prompt: "We went home.", Options: slices.Clone(TFNOptions), AnswerIndex: 0, ExplanationVi: "Câu 4.",
	})
	if _, err := lessons.update(l.ID, func(x *Lesson) bool { x.Extras.Questions = qs; return true }); err != nil {
		t.Fatal(err)
	}
	for choice := range 3 {
		if _, err := answer(t, r, "u1", l.ID, 2, 3, choice); err != nil {
			t.Errorf("choice %d: %v", choice, err)
		}
	}
	var verr *ValidationError
	if _, err := answer(t, r, "u1", l.ID, 2, 3, 3); !errors.As(err, &verr) || verr.Fields["choice"] == "" {
		t.Errorf("choice 3: %v", err)
	}
	// A four-option question still takes 3.
	if _, err := answer(t, r, "u1", l.ID, 2, 0, 3); err != nil {
		t.Errorf("four options, choice 3: %v", err)
	}
	if _, err := answer(t, r, "u1", l.ID, 2, 3, -1); !errors.As(err, &verr) || verr.Fields["choice"] == "" {
		t.Errorf("choice -1: %v", err)
	}
}
