package lesson

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

// PracticeView is the practice of a lesson as the learner page shows it (F17). Answers are
// included: the page checks them and nothing is sent back.
type PracticeView struct {
	Status Status
	// LessonNumber is the lesson's place in its topic roadmap, 0 when it is not there.
	LessonNumber int
	ObjectiveVi  string
	Examples     []ExampleView
	Dialogue     *DialogueView
	Fill         *Fill
	GrammarTipVi string
	Translations []TranslationView
}

// ExampleView is an example sentence; AudioURL is empty until its audio exists.
type ExampleView struct {
	Lemma    string
	Sentence string
	AudioURL string
}

// DialogueView is the sample dialogue with the audio of each turn.
type DialogueView struct {
	Speakers []string
	Turns    []TurnView
}

// TurnView is one dialogue turn; AudioURL is empty until its audio exists.
type TurnView struct {
	Speaker   int
	Text      string
	MeaningVi string
	AudioURL  string
}

// TranslationView is one sentence to translate: the answer tiles in order, and every tile
// (answer then distractors) for the word bank.
type TranslationView struct {
	Vi       string
	Answer   []string
	Tiles    []string
	AudioURL string
}

// Practice returns the practice view of a lesson; parts are empty while it has no practice.
func (r *Reader) Practice(ctx context.Context, id string) (PracticeView, error) {
	l, err := r.lessons.Get(ctx, id)
	if err != nil {
		return PracticeView{}, err
	}
	n, err := r.topics.Position(ctx, l.TopicID, l.ID)
	if err != nil {
		return PracticeView{}, fmt.Errorf("lesson: roadmap position: %w", err)
	}
	v := PracticeView{Status: l.PracticeStatus, LessonNumber: n, Examples: []ExampleView{}, Translations: []TranslationView{}}
	p := l.Practice
	if p == nil {
		return v, nil
	}
	audio := func(kind string, i int) string {
		file := filepath.Join(r.audioDir, l.ID, strconv.Itoa(l.Revision), "practice", strconv.Itoa(l.PracticeVersion),
			audioName(kind, i)+".mp3")
		if info, err := os.Stat(file); err != nil || info.Size() == 0 { //nolint:gosec // path built from a stored lesson id and integers
			return ""
		}
		return PracticeAudioURL(l.ID, l.Revision, l.PracticeVersion, kind, i)
	}

	v.ObjectiveVi, v.GrammarTipVi = p.ObjectiveVi, p.GrammarTipVi
	for i, e := range p.Examples {
		v.Examples = append(v.Examples, ExampleView{Lemma: e.Lemma, Sentence: e.Sentence, AudioURL: audio(audioExample, i)})
	}
	if p.Dialogue != nil {
		v.Dialogue = &DialogueView{Speakers: p.Dialogue.Speakers}
		for i, t := range p.Dialogue.Turns {
			v.Dialogue.Turns = append(v.Dialogue.Turns, TurnView{
				Speaker: t.Speaker, Text: t.Text, MeaningVi: t.MeaningVi, AudioURL: audio(audioTurn, i),
			})
		}
		v.Fill = BuildFill(p.Dialogue, practiceWords(l))
	}
	for i, t := range p.Translations {
		answer := answerTiles(t.En)
		v.Translations = append(v.Translations, TranslationView{
			Vi: t.Vi, Answer: answer, Tiles: append(append([]string{}, answer...), t.Distractors...),
			AudioURL: audio(audioAnswer, i),
		})
	}
	return v, nil
}
