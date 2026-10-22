package lesson

import (
	"slices"
	"strings"
	"testing"

	"github.com/luongtran/luna/backend/internal/ai"
)

// stepsContent has a sentence of 18 words, too long to read aloud.
const stepsContent = "We went to the park. He gave up smoking. " +
	"It was a sunny day and we walked along the river with our friends until the evening came."

func stepsLesson() Lesson {
	return Lesson{Level: "A1", Content: stepsContent, Sentences: toSentences(SplitSentences(stepsContent))}
}

func stepsDialogue() *Dialogue {
	return &Dialogue{Speakers: []string{"Minh", "Anna"}, Turns: []Turn{
		{Speaker: 0, Text: "Do you like the park?"},
		{Speaker: 1, Text: "Yes, I go there every Sunday."},
		{Speaker: 0, Text: "I play football there."},
		{Speaker: 1, Text: "My brother plays tennis."},
	}}
}

func listenQ(kind, prompt string, options []string, answer, turn int) ai.ListenQuestion {
	return ai.ListenQuestion{Kind: kind, Prompt: prompt, Options: options, AnswerIndex: answer, ExplanationVi: "Giải thích.", TurnIndex: turn}
}

func TestCleanStepsListeningQuestions(t *testing.T) {
	t.Parallel()
	three := []string{"A park", "A school", "A shop"}
	in := []ai.ListenQuestion{
		listenQ(" gist ", "What is it about?", three, 1, -1),
		listenQ("tfn", "Anna goes every Sunday.", nil, 2, 9), // turn out of range
		listenQ("tfn", "Bad tfn answer.", nil, 3, 0),
		listenQ("detail", "Two options?", []string{"a", "b"}, 0, 0),
		listenQ("detail", "Five options?", []string{"a", "b", "c", "d", "e"}, 0, 0),
		listenQ("detail", "Repeated option?", []string{"a", " A ", "c"}, 0, 0),
		listenQ("detail", "Empty option?", []string{"a", " ", "c"}, 0, 0),
		listenQ("why", "Unknown kind?", three, 0, 0),
		listenQ("detail", "Answer out of range?", three, 3, 0),
		{Kind: "detail", Prompt: "No explanation?", Options: three, TurnIndex: 0},
		listenQ("gist", "what is IT about?", three, 0, -1), // duplicate prompt
		listenQ("detail", "What does Minh play?", []string{"Football", "Tennis", "Chess", "Golf"}, 0, 2),
		listenQ("detail", "Turn below -1?", three, 0, -2),
		listenQ("detail", "One too many?", three, 0, 1),
	}

	got := CleanSteps(ai.Practice{ListeningQuestions: in}, stepsLesson(), stepsDialogue()).ListeningQuestions
	var prompts []string
	for _, q := range got {
		prompts = append(prompts, q.Prompt)
	}
	want := []string{"What is it about?", "Anna goes every Sunday.", "What does Minh play?", "Turn below -1?"}
	if !slices.Equal(prompts, want) {
		t.Fatalf("prompts = %q, want %q", prompts, want)
	}
	if got[0].Kind != KindGist || got[0].AnswerIndex != 1 || got[0].TurnIndex != -1 {
		t.Errorf("gist = %+v", got[0])
	}
	if !slices.Equal(got[1].Options, ListenTFNOptions) || got[1].TurnIndex != -1 || got[1].AnswerIndex != 2 {
		t.Errorf("tfn = %+v", got[1])
	}
	if got[2].TurnIndex != 2 || len(got[2].Options) != 4 {
		t.Errorf("detail = %+v", got[2])
	}
	if got[3].TurnIndex != -1 {
		t.Errorf("turn -2 = %d, want -1", got[3].TurnIndex)
	}
	// The tfn options are a copy.
	got[1].Options[0] = "x"
	if ListenTFNOptions[0] != "Đúng" {
		t.Fatal("ListenTFNOptions changed")
	}

	if qs := CleanSteps(ai.Practice{ListeningQuestions: in}, stepsLesson(), nil).ListeningQuestions; qs != nil {
		t.Fatalf("no dialogue: %+v", qs)
	}
}

func fillExercise(text string) ai.GrammarExercise {
	return ai.GrammarExercise{Kind: "fill", Text: text, Answers: []string{"goes"}, ExplanationVi: "Ngôi thứ ba thêm -es."}
}

func choiceExercise() ai.GrammarExercise {
	return ai.GrammarExercise{
		Kind: "choice", Text: "He ___ tea every day.", Options: []string{"drink", "drinks", "drinking", "drank"},
		AnswerIndex: 1, ExplanationVi: "He + động từ thêm -s.",
	}
}

func okGrammarStep() *ai.GrammarStep {
	return &ai.GrammarStep{
		TitleVi: " Hiện tại đơn ",
		Noticing: []ai.Noticing{
			{Sentence: "we went to the park", Highlights: []string{"WENT", "went", "flew"}}, // in the content
			{Sentence: "My brother plays tennis.", Highlights: []string{"plays"}},             // in the dialogue
			{Sentence: "She sings songs.", Highlights: []string{"sings"}},                      // not in the texts
			{Sentence: "He gave up smoking.", Highlights: []string{"stopped"}},                 // no highlight in it
		},
		QuestionVi: "Vì sao động từ có -s?", Options: []string{"Ngôi thứ ba", "Số nhiều", "Quá khứ"}, AnswerIndex: 0,
		ExplanationVi: "Vì chủ ngữ là ngôi thứ ba số ít.",
		Table: []ai.FormRow{
			{Subject: "I / you", Form: "play", Negative: "don't play"},
			{Subject: "He", Form: ""}, // no form
		},
		Mistakes: []ai.MistakePair{
			{Wrong: "He play.", Right: "He plays."},
			{Wrong: "He plays.", Right: "he  plays."}, // the same
		},
		Exercises: []ai.GrammarExercise{fillExercise("She ___ to school."), choiceExercise(), fillExercise("No blank here.")},
	}
}

func TestCleanStepsGrammar(t *testing.T) {
	t.Parallel()
	g := CleanSteps(ai.Practice{GrammarStep: okGrammarStep()}, stepsLesson(), stepsDialogue()).Grammar
	if g == nil {
		t.Fatal("grammar step dropped")
	}
	if g.TitleVi != "Hiện tại đơn" || g.QuestionVi == "" || len(g.Options) != 3 || g.ExplanationVi == "" {
		t.Errorf("grammar = %+v", g)
	}
	want := []Noticing{
		{Sentence: "we went to the park", Highlights: []string{"WENT"}},
		{Sentence: "My brother plays tennis.", Highlights: []string{"plays"}},
	}
	if len(g.Noticing) != len(want) {
		t.Fatalf("noticing = %+v", g.Noticing)
	}
	for i, n := range want {
		if g.Noticing[i].Sentence != n.Sentence || !slices.Equal(g.Noticing[i].Highlights, n.Highlights) {
			t.Errorf("noticing[%d] = %+v, want %+v", i, g.Noticing[i], n)
		}
	}
	if len(g.Table) != 1 || len(g.Mistakes) != 1 {
		t.Errorf("table %+v mistakes %+v", g.Table, g.Mistakes)
	}
	if len(g.Exercises) != 2 || g.Exercises[0].ID != "g1" || g.Exercises[1].ID != "g2" {
		t.Errorf("exercises = %+v", g.Exercises)
	}
}

func TestCleanStepsGrammarDropped(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		mutate func(*ai.GrammarStep)
		// nil step, else whether the noticing and the question are kept
		dropped, noticing, question bool
	}{
		{name: "no title", mutate: func(g *ai.GrammarStep) { g.TitleVi = " " }, dropped: true},
		{name: "one valid exercise", mutate: func(g *ai.GrammarStep) { g.Exercises = g.Exercises[1:] }, dropped: true},
		{name: "one noticing", mutate: func(g *ai.GrammarStep) { g.Noticing = g.Noticing[1:] }},
		{name: "bad answer", mutate: func(g *ai.GrammarStep) { g.AnswerIndex = 3 }, noticing: true},
		{name: "one option", mutate: func(g *ai.GrammarStep) { g.Options = g.Options[:1] }, noticing: true},
		{name: "five options", mutate: func(g *ai.GrammarStep) { g.Options = []string{"a", "b", "c", "d", "e"} }, noticing: true},
		{name: "no explanation", mutate: func(g *ai.GrammarStep) { g.ExplanationVi = "" }, noticing: true},
		{name: "valid", mutate: func(*ai.GrammarStep) {}, noticing: true, question: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			in := okGrammarStep()
			tt.mutate(in)
			g := CleanSteps(ai.Practice{GrammarStep: in}, stepsLesson(), stepsDialogue()).Grammar
			if tt.dropped {
				if g != nil {
					t.Fatalf("grammar = %+v, want nil", g)
				}
				return
			}
			if g == nil {
				t.Fatal("grammar dropped")
			}
			if (g.Noticing != nil) != tt.noticing {
				t.Errorf("noticing = %+v", g.Noticing)
			}
			if (g.QuestionVi != "") != tt.question || (g.Options != nil) != tt.question {
				t.Errorf("question %q options %q", g.QuestionVi, g.Options)
			}
			if !tt.question && (g.AnswerIndex != 0 || g.ExplanationVi != "") {
				t.Errorf("question not cleared: %+v", g)
			}
		})
	}
	if g := CleanSteps(ai.Practice{}, stepsLesson(), stepsDialogue()).Grammar; g != nil {
		t.Fatalf("no step: %+v", g)
	}
}

func TestCleanStepsReadAloudAndSpeak(t *testing.T) {
	t.Parallel()
	long := "It was a sunny day and we walked along the river with our friends until the evening came."
	p := ai.Practice{
		ReadAloud: []ai.ReadAloud{
			{Text: long, TipVi: "Quá dài."},
			{Text: "Hello there.", TipVi: "Không có trong bài."},
			{Text: " he GAVE up smoking ", TipVi: "Nối âm gave up."},
			{Text: "I play football there.", TipVi: "Trong hội thoại."},
			{Text: "We went to the park.", TipVi: "Thừa."},
		},
		SpeakQuestions: []ai.SpeakQuestion{
			{Question: "Do you like parks?", QuestionVi: "Bạn có thích công viên không?", Frame: "I ___ parks."},
			{Question: " "},
			{Question: "What do you do on Sunday?"},
			{Question: "Who do you go with?"},
			{Question: "One too many?"},
		},
	}
	s := CleanSteps(p, stepsLesson(), stepsDialogue())
	if len(s.ReadAloud) != 2 || s.ReadAloud[0].Text != "he GAVE up smoking" || s.ReadAloud[1].Text != "I play football there." ||
		s.ReadAloud[0].TipVi != "Nối âm gave up." {
		t.Errorf("read aloud = %+v", s.ReadAloud)
	}
	var qs []string
	for _, q := range s.SpeakQuestions {
		qs = append(qs, q.Question)
	}
	if want := []string{"Do you like parks?", "What do you do on Sunday?", "Who do you go with?"}; !slices.Equal(qs, want) {
		t.Errorf("speak questions = %q", qs)
	}
	if s.SpeakQuestions[0].Frame != "I ___ parks." {
		t.Errorf("frame = %q", s.SpeakQuestions[0].Frame)
	}
}

func TestCleanStepsWritingHelp(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		in        *ai.WritingHelp
		wantNil   bool
		wantLabel string
		wantAnyOf []string
		outline   int
	}{
		{name: "none", in: nil, wantNil: true},
		{
			name: "full",
			in: &ai.WritingHelp{
				GrammarLabelVi: "Dùng thì hiện tại đơn",
				AnyOf:          []string{" Goes ", "goes", "PLAYS  every", "", strings.Repeat("x", 31)},
				OutlineVi:      []string{"Mở bài", " ", "Thân bài", "Kết bài", "Thừa"},
				ModelAnswer:    "I go to the park.",
			},
			wantLabel: "Dùng thì hiện tại đơn", wantAnyOf: []string{"goes", "plays every"}, outline: 3,
		},
		{
			name:      "label without anyOf",
			in:        &ai.WritingHelp{GrammarLabelVi: "Dùng thì hiện tại đơn", AnyOf: []string{" "}, OutlineVi: []string{"Mở bài"}},
			wantLabel: "", outline: 1,
		},
		{
			name:      "anyOf without label",
			in:        &ai.WritingHelp{AnyOf: []string{"goes"}, ModelAnswer: "I go."},
			wantLabel: "",
		},
		{name: "only a label", in: &ai.WritingHelp{GrammarLabelVi: "Nhãn"}, wantNil: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := CleanSteps(ai.Practice{WritingHelp: tt.in}, stepsLesson(), nil).Writing
			if tt.wantNil {
				if w != nil {
					t.Fatalf("writing = %+v, want nil", w)
				}
				return
			}
			if w == nil {
				t.Fatal("writing dropped")
			}
			if w.GrammarLabelVi != tt.wantLabel || !slices.Equal(w.AnyOf, tt.wantAnyOf) || len(w.OutlineVi) != tt.outline {
				t.Fatalf("writing = %+v", w)
			}
		})
	}
}

func TestCleanStepsGrammarPointID(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name, lessonPoint, picked, want string
	}{
		{name: "picked", picked: " a1-present-simple ", want: "a1-present-simple"},
		{name: "lesson has one", lessonPoint: "a1-to-be", picked: "a1-present-simple"},
		{name: "other level", picked: "b1-present-perfect"},
		{name: "unknown", picked: "a1-nope"},
		{name: "none"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			l := stepsLesson()
			l.GrammarPointID = tt.lessonPoint
			if got := CleanSteps(ai.Practice{GrammarPointID: tt.picked}, l, nil).GrammarPointID; got != tt.want {
				t.Fatalf("GrammarPointID = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCleanStepsReadingGuide(t *testing.T) {
	t.Parallel()
	s := CleanSteps(ai.Practice{ReadingGuideVi: "  Tìm xem   họ đi đâu. "}, stepsLesson(), nil)
	if s.ReadingGuideVi != "Tìm xem họ đi đâu." {
		t.Errorf("guide = %q", s.ReadingGuideVi)
	}
	if s := CleanSteps(ai.Practice{ReadingGuideVi: strings.Repeat("a", maxReadingGuide+1)}, stepsLesson(), nil); s.ReadingGuideVi != "" {
		t.Errorf("long guide kept")
	}
}

func dictationTexts(items []DictationItem) []string {
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = it.Text
	}
	return out
}

func TestDictation(t *testing.T) {
	t.Parallel()
	all := []string{
		"We went to the park.", "He gave up smoking.",
		"It was a sunny day and we walked along the river with our friends until the evening came.",
	}

	// Without a practice, or with one made before the steps, every sentence.
	l := stepsLesson()
	if got := dictationTexts(Dictation(l)); !slices.Equal(got, all) {
		t.Fatalf("no practice = %q", got)
	}
	l.Practice = &Practice{Examples: []Example{{Lemma: "go", Sentence: "Go!"}}}
	if got := dictationTexts(Dictation(l)); !slices.Equal(got, all) {
		t.Fatalf("no steps = %q", got)
	}

	// With steps: the short examples first, then lesson sentences of at most 10 words.
	l.Practice = &Practice{Steps: &Steps{}, Examples: []Example{
		{Lemma: "go", Sentence: "Go!"}, // too short
		{Lemma: "sunny", Sentence: "We like sunny days a lot."},
		{Lemma: "friend", Sentence: "I go to school every single day with my friends."}, // 10 words
		{Lemma: "give up", Sentence: "Never give up."},
		{Lemma: "give up", Sentence: "NEVER  give up."}, // the same
	}}
	got := Dictation(l)
	want := []DictationItem{
		{Text: "Never give up.", Lemma: "give up"},
		{Text: "We like sunny days a lot.", Lemma: "sunny"},
		{Text: "He gave up smoking."},
	}
	if !slices.Equal(got, want) {
		t.Fatalf("dictation = %+v, want %+v", got, want)
	}

	// Enough examples: no lesson sentence.
	l.Practice.Examples = []Example{
		{Lemma: "a", Sentence: "One two three four five six."},
		{Lemma: "b", Sentence: "One two three four."},
		{Lemma: "c", Sentence: "One two three four five."},
		{Lemma: "d", Sentence: "One two three."},
	}
	if got := dictationTexts(Dictation(l)); !slices.Equal(got, []string{"One two three.", "One two three four.", "One two three four five."}) {
		t.Fatalf("examples only = %q", got)
	}

	// No example fits: the lesson sentences of at most 10 words.
	l.Practice.Examples = nil
	if got := dictationTexts(Dictation(l)); !slices.Equal(got, []string{"He gave up smoking.", "We went to the park."}) {
		t.Fatalf("sentences only = %q", got)
	}
}
