package lesson

import (
	"context"
	"fmt"
	"log/slog"
)

// LessonWord is a word a lesson teaches: an annotated lemma with its meaning in the lesson. Target
// says it is one of the lesson's target words, chosen for it, rather than a word the AI annotated.
type LessonWord struct {
	Lemma     string
	MeaningVi string
	Target    bool
}

// VocabSink keeps the words of the lessons in step with the topics and the word bank (added
// 2026-11-16): the words of a lesson join its topic's list, with the lesson's level, and the bank.
type VocabSink interface {
	// LessonWords shares the words of lessons of the topic at level and returns how many joined the
	// topic's list.
	LessonWords(ctx context.Context, topicID string, level Level, words []LessonWord) (int, error)
}

// lessonWords lists the annotated words, one per normalized lemma, in the order of the annotations,
// marking those among targets.
func lessonWords(anns []Annotation, targets []string) []LessonWord {
	isTarget := make(map[string]bool, len(targets))
	for _, t := range targets {
		isTarget[normalize(t)] = true
	}
	seen := map[string]bool{}
	out := []LessonWord{}
	for _, a := range anns {
		lemma := normalize(a.Lemma)
		if lemma == "" || seen[lemma] {
			continue
		}
		seen[lemma] = true
		out = append(out, LessonWord{Lemma: lemma, MeaningVi: a.MeaningVi, Target: isTarget[lemma]})
	}
	return out
}

// shareWords gives the words of a lesson just annotated to the Vocab sink. It is best effort: the
// annotations are saved either way, and "Đồng bộ từ vựng" catches up on a failure.
func (s *Service) shareWords(ctx context.Context, l Lesson, anns []Annotation) {
	if s.Vocab == nil || l.TopicID == "" {
		return
	}
	words := lessonWords(anns, l.TargetWords)
	if len(words) == 0 {
		return
	}
	if _, err := s.Vocab.LessonWords(ctx, l.TopicID, l.Level, words); err != nil {
		s.Log.WarnContext(ctx, "lesson: share words with topic and word bank",
			slog.String("lesson", l.ID), slog.Any("error", err))
	}
}

// shareBatch is how many lessons ShareAllWords reads at once.
const shareBatch = 50

// ShareAllWords gives the words of every annotated lesson to the Vocab sink, one call for each
// topic and level, and returns how many words joined the topics' lists ("Đồng bộ từ vựng").
func (s *Service) ShareAllWords(ctx context.Context) (int, error) {
	if s.Vocab == nil {
		return 0, nil
	}
	refs, err := s.Lessons.Annotated(ctx)
	if err != nil {
		return 0, fmt.Errorf("lesson: annotated lessons: %w", err)
	}
	type place struct {
		topic string
		level Level
	}
	type group struct {
		anns    []Annotation
		targets []string
	}
	var order []place
	groups := map[place]*group{}
	for start := 0; start < len(refs); start += shareBatch {
		ids := make([]string, 0, shareBatch)
		for _, r := range refs[start:min(start+shareBatch, len(refs))] {
			ids = append(ids, r.ID)
		}
		lessons, err := s.Lessons.GetMany(ctx, ids)
		if err != nil {
			return 0, fmt.Errorf("lesson: load lessons: %w", err)
		}
		for _, l := range lessons {
			if l.TopicID == "" || len(l.Annotations) == 0 {
				continue
			}
			p := place{l.TopicID, l.Level}
			g, ok := groups[p]
			if !ok {
				g = &group{}
				groups[p] = g
				order = append(order, p)
			}
			g.anns = append(g.anns, l.Annotations...)
			g.targets = append(g.targets, l.TargetWords...)
		}
	}
	added := 0
	for _, p := range order {
		g := groups[p]
		// One topic that fails (deleted meanwhile, list full) does not stop the others.
		n, err := s.Vocab.LessonWords(ctx, p.topic, p.level, lessonWords(g.anns, g.targets))
		if err != nil {
			s.Log.WarnContext(ctx, "lesson: share words of a topic",
				slog.String("topic", p.topic), slog.String("level", string(p.level)), slog.Any("error", err))
			continue
		}
		added += n
	}
	return added, nil
}
