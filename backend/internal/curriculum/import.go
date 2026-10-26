package curriculum

import (
	"context"
	"fmt"

	"github.com/luongtran/luna/backend/internal/grammar"
	"github.com/luongtran/luna/backend/internal/lesson"
)

// Lessons written outside the app for a slot (added 2026-10-26): Brief gives the writer, an admin
// or an agent, what the AI gets when it writes the slot's lesson, and ImportSlot saves the lesson
// with all its parts written (lesson.Import), so no AI is called.

// BriefGrammar is a grammar point of a slot as the writer needs it.
type BriefGrammar struct {
	ID       string
	TitleVi  string
	TitleEn  string
	Pattern  string
	HintVi   string
	Examples []string
}

// Brief is everything the lesson of a slot is written from.
type Brief struct {
	Key         string
	Level       string
	TopicID     string
	TopicName   string
	TopicNumber int
	CanDoVi     string
	Kind        Kind
	// Title is the lesson's title: the slot is matched with its lesson by it.
	Title string
	Idea  string
	// Grammar are the slot's points, the first being the one the lesson is built on.
	Grammar []BriefGrammar
	// TargetWords are the topic words planned for the lesson, TopicWords every topic word of the
	// level.
	TargetWords []string
	TopicWords  []string
	// Words is about how many words the lesson should have; MinGrammarUses how many times it uses
	// its point.
	Words          int
	MinGrammarUses int
	Filled         bool
}

// Brief returns what the lesson of the slot key is written from.
func (s *Service) Brief(ctx context.Context, key string) (Brief, error) {
	o, err := s.openSlot(ctx, key)
	if err != nil {
		return Brief{}, err
	}
	b := Brief{
		Key: o.slot.Key, Level: o.level, TopicID: o.found.ID, TopicName: o.found.Name, TopicNumber: o.topic.Number,
		CanDoVi: o.topic.CanDoVi, Kind: o.slot.Kind, Title: o.slot.Title, Idea: o.slot.Idea,
		TargetWords: s.plannedWords(ctx, o), Words: lessonWords[o.level], MinGrammarUses: MinGrammarUses, Filled: o.filled,
		Grammar: []BriefGrammar{}, TopicWords: []string{},
	}
	for _, id := range o.slot.Grammar {
		if p, ok := grammar.Default().Get(id); ok {
			b.Grammar = append(b.Grammar, BriefGrammar{
				ID: id, TitleVi: p.TitleVi, TitleEn: p.TitleEn, Pattern: p.Pattern, HintVi: p.HintVi, Examples: p.Examples,
			})
		}
	}
	for _, w := range o.found.Words {
		if w.FitsLevel(o.level) {
			b.TopicWords = append(b.TopicWords, w.Text)
		}
	}
	return b, nil
}

// plannedWords are the topic words a new lesson of the slot gets, as GenerateSlot plans them;
// none when the topic has too few.
func (s *Service) plannedWords(ctx context.Context, o openedSlot) []string {
	per := targetWords[o.level]
	if per == 0 {
		return []string{}
	}
	plan, err := s.Topics.WordPlan(ctx, o.found.ID, o.level, 1, per)
	if err != nil || len(plan.Groups) != 1 {
		return []string{}
	}
	return plan.Groups[0]
}

// SlotImport is a lesson written for a slot: its content and what the AI would have written for
// it. TargetWords, if empty, are those of the brief.
type SlotImport struct {
	Content     string
	TargetWords []string
	lesson.ImportContent
}

// ImportResult is what an import of a slot did, or would do with dryRun: the slot with its lesson
// (none on a dry run), the checks of the lesson and what was kept of each part.
type ImportResult struct {
	Slot   SlotView
	Checks Checks
	Report lesson.ImportReport
}

// ImportSlot saves the lesson of an empty slot from in, as a draft at its place in the roadmap,
// titled as the slot and built on its first grammar point. dryRun only checks it.
func (s *Service) ImportSlot(ctx context.Context, key string, in SlotImport, dryRun bool) (ImportResult, error) {
	o, err := s.openSlot(ctx, key)
	if err != nil {
		return ImportResult{}, err
	}
	if o.filled {
		return ImportResult{}, ErrSlotFilled
	}
	words := in.TargetWords
	if len(words) == 0 {
		words = s.plannedWords(ctx, o)
	}
	l, rep, err := s.Lessons.Import(ctx, lesson.Input{
		Title: o.slot.Title, Content: in.Content, TopicID: o.found.ID, Level: lesson.Level(o.level),
		Source: lesson.ImportSource, License: lesson.ImportLicense, GrammarPointID: o.slot.Grammar[0], TargetWords: words,
		AppendToRoadmap: !dryRun, Draft: true,
	}, in.ImportContent, dryRun)
	if err != nil {
		return ImportResult{Report: rep}, err
	}
	if dryRun {
		published, err := s.published(ctx)
		if err != nil {
			return ImportResult{}, err
		}
		return ImportResult{
			Slot:   SlotView{Slot: o.slot, Grammar: grammarViews(o.slot.Grammar, published)},
			Checks: Inspect(l), Report: rep,
		}, nil
	}
	v, err := s.placed(ctx, o, l)
	if err != nil {
		return ImportResult{}, fmt.Errorf("curriculum: place imported lesson: %w", err)
	}
	return ImportResult{Slot: v, Checks: *v.Checks, Report: rep}, nil
}
