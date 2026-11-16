package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/luongtran/luna/backend/internal/games"
	"github.com/luongtran/luna/backend/internal/grammar"
	"github.com/luongtran/luna/backend/internal/progress"
	"github.com/luongtran/luna/backend/internal/verbs"
	"github.com/luongtran/luna/backend/internal/vocab"
	"github.com/luongtran/luna/backend/internal/wordbank"
)

// initGames builds the learner's games (F26). It reads the vocabulary, the word bank, the verbs, the
// grammar lessons and the study flow, so it runs after initProgress.
func (c *Container) initGames() {
	c.games = games.NewService(games.Deps{
		Bests:     c.store.GameBests(),
		Daily:     c.store.DailyChallenges(),
		Spots:     c.store.GrammarSpotItems(),
		Cards:     gameCards{c.vocab},
		Pictures:  c.wordBank,
		Meanings:  gameMeanings{c.wordBank},
		Lessons:   gameLessonWords{lessonVocabulary{c.reader}},
		Study:     gameStudy{c.study},
		Words:     c.vocab,
		Verbs:     gameVerbs{c.verbs},
		Grammar:   gameGrammar{svc: c.grammar, progress: c.store.GrammarProgress()},
		AI:        c.ai,
		StudyDays: c.studyDays,
		Timezones: c.settings,
		Now:       time.Now,
	})
}

// gameCards adapts vocab.Service to games.Cards.
type gameCards struct{ svc *vocab.Service }

func (g gameCards) Cards(ctx context.Context, userID string) ([]games.Card, error) {
	cards, err := g.svc.GameCards(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]games.Card, len(cards))
	for i, c := range cards {
		out[i] = games.Card{Lemma: c.Lemma, Text: c.Text, MeaningVi: c.MeaningVi, LessonID: c.LessonID, Due: c.Schedule.Due}
	}
	return out, nil
}

// gameMeanings adapts the word bank to games.Meanings.
type gameMeanings struct{ bank *wordbank.Service }

func (g gameMeanings) MeaningsOf(ctx context.Context, lemmas []string) (map[string]string, error) {
	words, err := g.bank.Words(ctx, lemmas)
	if err != nil {
		return nil, fmt.Errorf("word bank words: %w", err)
	}
	out := make(map[string]string, len(lemmas))
	for _, l := range lemmas {
		if w, ok := words[wordbank.Normalize(l)]; ok && w.MeaningVi != "" {
			out[l] = w.MeaningVi
		}
	}
	return out, nil
}

// gameLessonWords adapts the lesson vocabulary to games.LessonWords: a lesson gone or not annotated
// gives no words.
type gameLessonWords struct{ lessons lessonVocabulary }

func (g gameLessonWords) LessonWords(ctx context.Context, lessonID string) ([]games.LessonWord, error) {
	items, err := g.lessons.Vocabulary(ctx, lessonID)
	if errors.Is(err, vocab.ErrLessonNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	out := make([]games.LessonWord, len(items))
	for i, it := range items {
		out[i] = games.LessonWord{Lemma: it.Lemma, Text: it.Text, MeaningVi: it.MeaningVi}
	}
	return out, nil
}

// gameStudy adapts progress.StudyService to games.Study.
type gameStudy struct{ svc *progress.StudyService }

func (g gameStudy) StudiedLessons(ctx context.Context, userID string) ([]string, error) {
	return g.svc.StudiedLessons(ctx, userID)
}

func (g gameStudy) CurrentLesson(ctx context.Context, userID string) (string, error) {
	return g.svc.CurrentLesson(ctx, userID)
}

func (g gameStudy) CanOpen(ctx context.Context, userID string, isAdmin bool, lessonID string) (bool, error) {
	return g.svc.CanOpen(ctx, userID, isAdmin, lessonID)
}

func (g gameStudy) Level(ctx context.Context, userID string) (string, error) {
	goals, err := g.svc.Goals(ctx, userID)
	if err != nil {
		return "", err
	}
	if goals.Active == nil {
		return "", nil
	}
	return goals.Active.Level, nil
}

// gameVerbs adapts verbs.Service to games.Verbs.
type gameVerbs struct{ svc *verbs.Service }

func (g gameVerbs) GameMisses(ctx context.Context, userID string, misses []games.VerbMiss) error {
	results := make([]verbs.Result, 0, len(misses))
	for _, m := range misses {
		results = append(results, verbs.Result{Base: m.Base, Wrong: []verbs.Mistake{{Form: m.Form, Typed: m.Typed}}})
	}
	return g.svc.RecordGame(ctx, userID, results)
}

func (g gameVerbs) LearnedBases(ctx context.Context, userID string) (due, other []string, err error) {
	list, err := g.svc.List(ctx, userID)
	if err != nil {
		return nil, nil, err
	}
	now := time.Now()
	for _, p := range list {
		switch {
		case !p.Learned():
		case !p.DueAt.After(now):
			due = append(due, p.Base)
		default:
			other = append(other, p.Base)
		}
	}
	return due, other, nil
}

// gameGrammar adapts the grammar service and progress to games.Grammar.
type gameGrammar struct {
	svc      *grammar.Service
	progress grammar.ProgressRepository
}

func (g gameGrammar) PublishedSources(ctx context.Context) ([]games.PointSource, error) {
	list, err := g.svc.PublishedSources(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]games.PointSource, len(list))
	for i, s := range list {
		p := games.PointSource{
			PointID: s.Point.ID, Level: s.Point.Level, TitleVi: s.Point.TitleVi, TitleEn: s.Point.TitleEn,
			Pattern: s.Point.Pattern, HintVi: s.Point.HintVi, Examples: s.Point.Examples,
		}
		for _, e := range s.Exercises {
			p.Exercises = append(p.Exercises, games.SourceExercise{
				ID: e.ID, Kind: string(e.Kind), Text: e.Text, Options: e.Options, AnswerIndex: e.AnswerIndex,
				Sentence: e.Sentence, ExplanationVi: e.ExplanationVi,
			})
		}
		for _, m := range s.Mistakes {
			p.Mistakes = append(p.Mistakes, games.PointMistake{Wrong: m.Wrong, Right: m.Right, NoteVi: m.NoteVi})
		}
		out[i] = p
	}
	return out, nil
}

func (g gameGrammar) StartedPoints(ctx context.Context, userID string) ([]string, error) {
	list, err := g.progress.List(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("grammar progress: %w", err)
	}
	out := make([]string, len(list))
	for i, p := range list {
		out[i] = p.PointID
	}
	return out, nil
}
