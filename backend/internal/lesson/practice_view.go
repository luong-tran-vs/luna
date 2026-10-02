package lesson

import (
	"context"
	"fmt"
)

// PracticeView is the practice of a lesson as the learner page shows it (F17). Answers are
// included: the page checks them and nothing is sent back. The page reads every text aloud with
// the browser's voice.
type PracticeView struct {
	Status Status
	// LessonNumber is the lesson's place in its topic roadmap, 0 when it is not there.
	LessonNumber int
	ObjectiveVi  string
	Examples     []Example
	Dialogue     *Dialogue
	Fill         *Fill
	GrammarTipVi string
	Translations []TranslationView
}

// TranslationView is one sentence to translate: the answer tiles in order, and every tile
// (answer then distractors) for the word bank.
type TranslationView struct {
	Vi     string
	Answer []string
	Tiles  []string
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
	v := PracticeView{Status: l.PracticeStatus, LessonNumber: n, Examples: []Example{}, Translations: []TranslationView{}}
	p := l.Practice
	if p == nil {
		return v, nil
	}

	v.ObjectiveVi, v.GrammarTipVi = p.ObjectiveVi, p.GrammarTipVi
	v.Examples = append(v.Examples, p.Examples...)
	if p.Dialogue != nil {
		v.Dialogue = p.Dialogue
		v.Fill = BuildFill(p.Dialogue, practiceWords(l))
	}
	for _, t := range p.Translations {
		answer := answerTiles(t.En)
		v.Translations = append(v.Translations, TranslationView{
			Vi: t.Vi, Answer: answer, Tiles: append(append([]string{}, answer...), t.Distractors...),
		})
	}
	return v, nil
}
