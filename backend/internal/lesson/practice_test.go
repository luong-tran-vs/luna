package lesson

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/luongtran/luna/backend/internal/ai"
)

var practiceTestWords = []ai.PracticeWord{
	{Lemma: "meet", Text: "met", MeaningVi: "gặp"},
	{Lemma: "introduce", Text: "introduce", MeaningVi: "giới thiệu"},
	{Lemma: "give up", Text: "gave up", MeaningVi: "từ bỏ"},
	{Lemma: "friend", Text: "friends", MeaningVi: "bạn"},
}

func TestContainsWord(t *testing.T) {
	t.Parallel()

	meet := practiceTestWords[0]
	giveUp := practiceTestWords[2]
	tests := []struct {
		sentence string
		word     ai.PracticeWord
		want     bool
	}{
		{"Nice to MEET you.", meet, true},
		{"We met yesterday.", meet, true},
		{"The meeting starts now.", meet, false},
		{"He never gave up.", giveUp, true},
		{"Don't give  up!", giveUp, true},
		{"He gave it up.", giveUp, false},
		{"", meet, false},
	}
	for _, tt := range tests {
		if got := containsWord(tt.sentence, tt.word); got != tt.want {
			t.Errorf("containsWord(%q, %q) = %v, want %v", tt.sentence, tt.word.Lemma, got, tt.want)
		}
	}
}

func goodAIPractice() ai.Practice {
	return ai.Practice{
		ObjectiveVi: "  Bạn có thể   chào hỏi. ",
		Examples: []ai.Example{
			{Lemma: "meet", Sentence: "Nice to meet you."},
			{Lemma: "introduce", Sentence: "Let me introduce Anna."},
			{Lemma: "friend", Sentence: "He is my best buddy."},    // no word: dropped
			{Lemma: "meet", Sentence: "We meet every day."},        // duplicate lemma: dropped
			{Lemma: "unknown", Sentence: "This is unknown to me."}, // not a lesson word: dropped
		},
		Dialogue: ai.Dialogue{Speakers: []string{"Minh", " Anna "}, Turns: []ai.Turn{
			{Speaker: 0, Text: "Hi, I'm Minh.", MeaningVi: "Chào, mình là Minh."},
			{Speaker: 1, Text: "Nice to meet you.", MeaningVi: "Rất vui được gặp bạn."},
			{Speaker: 2, Text: "Bad speaker.", MeaningVi: "Sai."},
			{Speaker: 0, Text: "Let me introduce my friends.", MeaningVi: "Để mình giới thiệu bạn mình."},
			{Speaker: 1, Text: "", MeaningVi: "Rỗng."},
			{Speaker: 1, Text: "Hello, friends!", MeaningVi: ""},
			{Speaker: 1, Text: "Hello, friends!", MeaningVi: "Chào các bạn!"},
		}},
		GrammarTipVi: "Dùng \"Nice to meet you\" khi gặp lần đầu.",
		Translations: []ai.Translation{
			{Vi: "Rất vui được gặp bạn.", En: "Nice to meet you.", Distractors: []string{" see ", "glad", "Glad", "meet", "you", "us", "they"}},
			{Vi: "Chào.", En: "Hello.", Distractors: []string{"hi"}},                           // one tile: dropped
			{Vi: "Tôi đi học.", En: "I go to school.", Distractors: []string{"run"}},           // no word: dropped
			{Vi: "", En: "Meet my friends.", Distractors: nil},                                 // empty vi: dropped
			{Vi: "Anh ấy đã bỏ cuộc.", En: "He gave up.", Distractors: []string{"give", "up"}}, // "up" is an answer tile
		},
	}
}

func TestCleanPractice(t *testing.T) {
	t.Parallel()

	p, err := CleanPractice(goodAIPractice(), practiceTestWords)
	if err != nil {
		t.Fatalf("CleanPractice: %v", err)
	}
	if p.ObjectiveVi != "Bạn có thể chào hỏi." {
		t.Errorf("objective = %q", p.ObjectiveVi)
	}
	wantEx := []Example{{Lemma: "meet", Sentence: "Nice to meet you."}, {Lemma: "introduce", Sentence: "Let me introduce Anna."}}
	if !slices.Equal(p.Examples, wantEx) {
		t.Errorf("examples = %+v", p.Examples)
	}
	if p.Dialogue == nil {
		t.Fatal("dialogue dropped")
	}
	if !slices.Equal(p.Dialogue.Speakers, []string{"Minh", "Anna"}) || len(p.Dialogue.Turns) != 4 {
		t.Errorf("dialogue = %+v", p.Dialogue)
	}
	if len(p.Translations) != 2 {
		t.Fatalf("translations = %+v", p.Translations)
	}
	if got := p.Translations[0].Distractors; !slices.Equal(got, []string{"see", "glad", "us", "they"}) {
		t.Errorf("distractors = %q", got)
	}
	if got := p.Translations[1].Distractors; !slices.Equal(got, []string{"give"}) {
		t.Errorf("distractors = %q", got)
	}
	if p.GrammarTipVi == "" {
		t.Error("grammar tip dropped")
	}
}

func TestCleanPracticeDialogueRules(t *testing.T) {
	t.Parallel()

	turn := ai.Turn{Speaker: 0, Text: "Nice to meet you.", MeaningVi: "Rất vui."}
	withTurns := func(n int, speakers ...string) ai.Practice {
		p := goodAIPractice()
		p.Dialogue = ai.Dialogue{Speakers: speakers, Turns: slices.Repeat([]ai.Turn{turn}, n)}
		return p
	}

	if p, _ := CleanPractice(withTurns(3, "A", "B"), practiceTestWords); p.Dialogue != nil {
		t.Error("3 turns must drop the dialogue")
	}
	if p, _ := CleanPractice(withTurns(12, "A", "B"), practiceTestWords); p.Dialogue == nil || len(p.Dialogue.Turns) != 10 {
		t.Errorf("12 turns must keep 10: %+v", p.Dialogue)
	}
	if p, _ := CleanPractice(withTurns(6, "A"), practiceTestWords); p.Dialogue != nil {
		t.Error("one speaker must drop the dialogue")
	}
	if p, _ := CleanPractice(withTurns(6, "A", " "), practiceTestWords); p.Dialogue != nil {
		t.Error("an empty speaker name must drop the dialogue")
	}
	long := withTurns(5, "A", "B")
	long.Dialogue.Turns[0].Text = strings.Repeat("meet ", 50)
	if p, _ := CleanPractice(long, practiceTestWords); p.Dialogue == nil || len(p.Dialogue.Turns) != 4 {
		t.Error("a turn over 200 characters must be dropped")
	}
}

func TestCleanPracticeLimits(t *testing.T) {
	t.Parallel()

	p := goodAIPractice()
	tr := ai.Translation{Vi: "Gặp bạn.", En: "Nice to meet you.", Distractors: []string{"a"}}
	p.Translations = slices.Repeat([]ai.Translation{tr}, 7)
	p.Translations = append(p.Translations, ai.Translation{Vi: "Dài.", En: strings.Repeat("meet ", 16)})
	p.ObjectiveVi = strings.Repeat("a", 201)
	p.GrammarTipVi = strings.Repeat("b", 401)

	got, err := CleanPractice(p, practiceTestWords)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Translations) != 5 {
		t.Errorf("translations = %d, want 5", len(got.Translations))
	}
	if got.ObjectiveVi != "" || got.GrammarTipVi != "" {
		t.Error("too long objective and tip must be dropped")
	}
}

func TestCleanPracticeNothingUsable(t *testing.T) {
	t.Parallel()

	p := ai.Practice{
		ObjectiveVi: "Bạn có thể chào hỏi.",
		Examples:    []ai.Example{{Lemma: "meet", Sentence: "Hello."}},
		Dialogue:    ai.Dialogue{Speakers: []string{"A", "B"}},
	}
	if _, err := CleanPractice(p, practiceTestWords); !errors.Is(err, ErrNoValidPractice) {
		t.Fatalf("err = %v, want ErrNoValidPractice", err)
	}
}

func TestAnswerTiles(t *testing.T) {
	t.Parallel()
	got := answerTiles("  Nice to meet  you, Anna. ")
	if want := []string{"Nice", "to", "meet", "you,", "Anna."}; !slices.Equal(got, want) {
		t.Fatalf("answerTiles = %q", got)
	}
}

func dialogueOf(lines ...string) *Dialogue {
	d := &Dialogue{Speakers: []string{"Minh", "Anna"}}
	for i, l := range lines {
		d.Turns = append(d.Turns, Turn{Speaker: i % 2, Text: l, MeaningVi: "nghĩa " + l})
	}
	return d
}

// render rebuilds the turns with "[answer]" in place of each blank.
func render(f *Fill) []string {
	out := make([]string, len(f.Turns))
	for i, t := range f.Turns {
		var b strings.Builder
		for _, p := range t.Parts {
			if p.Blank != nil {
				b.WriteString("[" + f.Blanks[*p.Blank].Answer + "]")
			} else {
				b.WriteString(p.Text)
			}
		}
		out[i] = b.String()
	}
	return out
}

func TestBuildFill(t *testing.T) {
	t.Parallel()

	d := dialogueOf(
		"We met at school, my friends.",
		"Let me introduce Lan. She never gave up.",
		"No words here.",
		"Nice to MEET you!",
	)
	f := BuildFill(d, practiceTestWords)
	if f == nil {
		t.Fatal("nil fill")
	}
	want := []string{
		"We [met] at school, my [friends].",
		"Let me [introduce] Lan. She never [gave up].",
		"No words here.",
		"Nice to [MEET] you!",
	}
	if got := render(f); !slices.Equal(got, want) {
		t.Errorf("turns =\n%q\nwant\n%q", got, want)
	}
	if len(f.Blanks) != 5 {
		t.Errorf("blanks = %d", len(f.Blanks))
	}
	for i, tr := range f.Turns {
		if tr.TurnIndex != i || tr.Speaker != i%2 || tr.MeaningVi == "" {
			t.Errorf("turn %d = %+v", i, tr)
		}
	}
	if !slices.Equal(f.WordBank, []string{"met", "friends", "introduce", "gave up", "MEET"}) {
		t.Errorf("word bank = %q", f.WordBank)
	}
}

func TestBuildFillSpreadsAndLimits(t *testing.T) {
	t.Parallel()

	d := dialogueOf(
		"meet meet meet meet meet",
		"introduce",
		"friends",
		"meet",
	)
	f := BuildFill(d, practiceTestWords)
	got := render(f)
	want := []string{
		"[meet] [meet] meet meet meet",
		"[introduce]",
		"[friends]",
		"[meet]",
	}
	if !slices.Equal(got, want) {
		t.Errorf("turns = %q, want %q", got, want)
	}
	// Two answers are unused words, so the bank adds "gave up" only.
	if !slices.Equal(f.WordBank, []string{"meet", "meet", "introduce", "friends", "meet", "gave up"}) {
		t.Errorf("word bank = %q", f.WordBank)
	}
}

func TestBuildFillAdjacentBlanks(t *testing.T) {
	t.Parallel()

	f := BuildFill(dialogueOf("met met friends"), practiceTestWords)
	if got := render(f); !slices.Equal(got, []string{"[met] [met] [friends]"}) {
		t.Errorf("turns = %q", got)
	}
	if f.Turns[0].Parts[1].Text != " " {
		t.Errorf("parts = %+v", f.Turns[0].Parts)
	}
	// Answers use meet and friend; two other words fill the bank.
	if !slices.Equal(f.WordBank, []string{"met", "met", "friends", "introduce", "gave up"}) {
		t.Errorf("word bank = %q", f.WordBank)
	}
}

func TestBuildFillWithoutCandidates(t *testing.T) {
	t.Parallel()
	if f := BuildFill(dialogueOf("Hello.", "Hi."), practiceTestWords); f != nil {
		t.Errorf("fill = %+v, want nil", f)
	}
	if f := BuildFill(nil, practiceTestWords); f != nil {
		t.Error("nil dialogue must give nil fill")
	}
}

func TestBuildFillPrefersUnusedWords(t *testing.T) {
	t.Parallel()

	// One turn, seven candidates: the five blanks take each word once before repeating one.
	f := BuildFill(dialogueOf("met met met met friends introduce gave up"), practiceTestWords)
	if got := render(f); !slices.Equal(got, []string{"[met] [met] met met [friends] [introduce] [gave up]"}) {
		t.Errorf("turns = %q", got)
	}
}

func TestPracticeWords(t *testing.T) {
	t.Parallel()

	l := Lesson{Annotations: []Annotation{
		{Text: "gave up", Lemma: "Give up", MeaningVi: "từ bỏ", SentenceIndex: 2},
		{Text: "went", Lemma: "go", MeaningVi: "đã đi", SentenceIndex: 0},
		{Text: "goes", Lemma: "GO", MeaningVi: "đi", SentenceIndex: 1},
		{Text: "x", Lemma: " ", MeaningVi: "rỗng", SentenceIndex: 1},
	}}
	want := []ai.PracticeWord{
		{Lemma: "go", Text: "went", MeaningVi: "đã đi"},
		{Lemma: "give up", Text: "gave up", MeaningVi: "từ bỏ"},
	}
	if got := practiceWords(l); !slices.Equal(got, want) {
		t.Fatalf("practiceWords = %+v", got)
	}

	many := Lesson{}
	for i := range 40 {
		many.Annotations = append(many.Annotations, Annotation{Text: "w", Lemma: "w" + string(rune('a'+i%26)) + string(rune('a'+i/26))})
	}
	if got := practiceWords(many); len(got) != maxPracticeWords {
		t.Fatalf("words = %d, want %d", len(got), maxPracticeWords)
	}
}
