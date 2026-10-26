package lesson

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/luongtran/luna/backend/internal/ai"
)

// Import (added 2026-10-26) saves a lesson written outside the app, by an admin or an agent, with
// everything the AI jobs would have written already in it, in the very shapes the AI returns: the
// annotations and extras of ProcessAnnotate, the practice and steps of ProcessPractice and the
// glosses of ProcessGlosses. They pass the same checks as the AI's, so an imported lesson is like a
// generated one, and no AI job is queued.

// Source and license of an imported lesson.
const (
	ImportSource  = "Nhập"
	ImportLicense = "Nội dung soạn sẵn"
)

// ImportContent is what the AI would have written for a lesson.
type ImportContent struct {
	Annotate ai.LessonExtras
	Practice ai.Practice
	// Glosses may be empty: the reading popup then falls back to the dictionary.
	Glosses []ai.Gloss
}

// ImportPart tells how much of one part of an import the checks kept and, when some was dropped,
// why: RuleVi is the rule of the part, Reasons the problem of each dropped item where known.
type ImportPart struct {
	// Key names the part as in the import JSON ("annotate.questions"), LabelVi for the admin.
	Key     string
	LabelVi string
	Given   int
	Kept    int
	RuleVi  string
	Reasons []string
}

// ImportReport is what the checks made of an import: the parts, with what each kept, and the
// sentences and words of the content.
type ImportReport struct {
	Sentences int
	Parts     []ImportPart
	// FromDictionary counts the target words the import left without an annotation, filled from
	// the dictionary as for an AI lesson.
	FromDictionary int
}

// Import checks a lesson and its content and, unless dryRun, stores it with its annotations,
// practice and glosses done. dryRun returns the lesson as it would be saved, without an ID.
func (s *Service) Import(ctx context.Context, in Input, c ImportContent, dryRun bool) (Lesson, ImportReport, error) {
	in, sentences, err := ValidateInput(in)
	if err != nil {
		return Lesson{}, ImportReport{}, err
	}
	t, err := s.topic(ctx, in.TopicID)
	if err != nil {
		return Lesson{}, ImportReport{}, err
	}
	if err := checkGrammarPoint(in.GrammarPointID, in.Level); err != nil {
		return Lesson{}, ImportReport{}, err
	}
	now := s.Now()
	l := Lesson{
		Title: in.Title, Content: in.Content, Level: in.Level, TopicID: t.ID,
		Source: in.Source, License: in.License, GrammarPointID: in.GrammarPointID, TargetWords: in.TargetWords,
		Revision: 1, Sentences: toSentences(sentences), Draft: in.Draft, CreatedAt: now, UpdatedAt: now,
	}
	rep := ImportReport{Sentences: len(sentences)}
	fields := map[string]string{}

	// The annotations, as ProcessAnnotate keeps them.
	focus, only := focusWords(l.Content, l.TargetWords), true
	if len(focus) == 0 {
		focus, only = focusWords(l.Content, t.WordTexts()), false
	}
	anns, _ := CleanAnnotations(c.Annotate.Annotations, sentences)
	if only {
		anns = onlyFocus(anns, focus)
	}
	kept := len(anns)
	if anns, err = addMissedFocus(ctx, anns, focus, sentences, s.Dict); err != nil {
		return Lesson{}, ImportReport{}, err
	}
	rep.FromDictionary = len(anns) - kept
	rep.Parts = append(rep.Parts, ImportPart{Key: "annotate.annotations", LabelVi: "Chú thích từ", Given: len(c.Annotate.Annotations), Kept: kept})
	if len(anns) == 0 {
		fields["annotate.annotations"] = "Không có chú thích nào hợp lệ: mỗi chú thích cần text, lemma, meaningVi và text phải có trong câu"
	}
	extras := CleanExtras(c.Annotate, l.Content)
	extras.SentenceTranslations = CleanTranslations(c.Annotate.SentenceTranslations, len(sentences))
	if title := GrammarTitle(l.GrammarPointID); title != "" && extras.GrammarNote != nil {
		extras.GrammarNote.Title = title
	}
	rep.Parts = append(rep.Parts,
		ImportPart{Key: "annotate.questions", LabelVi: "Câu hỏi đọc hiểu", Given: len(c.Annotate.Questions), Kept: len(extras.Questions)},
		ImportPart{Key: "annotate.grammarNote", LabelVi: "Ghi chú ngữ pháp", Given: one(c.Annotate.GrammarNote != nil), Kept: one(extras.GrammarNote != nil)},
		ImportPart{Key: "annotate.writingPrompt", LabelVi: "Đề viết", Given: one(c.Annotate.WritingPrompt != ""), Kept: one(extras.WritingPrompt != "")},
		ImportPart{Key: "annotate.sentenceTranslations", LabelVi: "Bản dịch từng câu", Given: len(c.Annotate.SentenceTranslations), Kept: len(extras.SentenceTranslations)},
	)
	l.Annotations, l.Extras = anns, extras

	// The practice and steps, as ProcessPractice keeps them.
	p, err := CleanPractice(c.Practice, practiceWords(l))
	if err != nil {
		fields["practice"] = "Phần luyện tập không có ví dụ, hội thoại hay câu dịch nào hợp lệ"
	}
	p.Steps = CleanSteps(c.Practice, l, p.Dialogue)
	rep.Parts = append(rep.Parts, practiceParts(c.Practice, p)...)
	l.Practice = &p

	glosses := CleanGlosses(c.Glosses, sentences)
	rep.Parts = append(rep.Parts, ImportPart{Key: "glosses", LabelVi: "Nghĩa trong bài (popup)", Given: len(c.Glosses), Kept: len(glosses)})

	explainImport(rep.Parts, c, sentences, l.Content, focus, only)
	if len(fields) > 0 {
		return Lesson{}, rep, &ValidationError{Fields: fields}
	}
	if dryRun {
		l.AnnotationStatus, l.PracticeStatus = StatusDone, StatusDone
		return l, rep, nil
	}
	saved, err := s.saveImport(ctx, l, glosses, in.AppendToRoadmap)
	return saved, rep, err
}

// saveImport stores a checked import. MongoDB runs without transactions, so the lesson is created,
// filled and only then put in its roadmap; on failure it is deleted again.
func (s *Service) saveImport(ctx context.Context, l Lesson, glosses []Gloss, appendToRoadmap bool) (Lesson, error) {
	anns, extras, p := l.Annotations, l.Extras, *l.Practice
	l.Annotations, l.Extras, l.Practice, l.AnnotationStatus = nil, Extras{}, nil, StatusRunning
	created, err := s.Lessons.Create(ctx, l)
	if err != nil {
		return Lesson{}, fmt.Errorf("lesson: create: %w", err)
	}
	if err := s.fillImport(ctx, created, anns, extras, p, glosses); err != nil {
		if derr := s.Lessons.Delete(ctx, created.ID); derr != nil {
			s.Log.ErrorContext(ctx, "lesson: remove lesson after failed import",
				slog.String("lesson", created.ID), slog.Any("error", derr))
		}
		return Lesson{}, err
	}
	if appendToRoadmap {
		if err := s.appendToRoadmap(ctx, created); err != nil {
			return Lesson{}, err
		}
	}
	return s.Lessons.Get(ctx, created.ID)
}

func (s *Service) fillImport(ctx context.Context, l Lesson, anns []Annotation, extras Extras, p Practice, glosses []Gloss) error {
	if _, err := s.Lessons.SaveAnnotations(ctx, l.ID, l.Revision, anns, extras); err != nil {
		return fmt.Errorf("lesson: save annotations: %w", err)
	}
	// SaveAnnotations dropped the practice; it is saved over the version that left.
	cur, err := s.Lessons.Get(ctx, l.ID)
	if err != nil {
		return fmt.Errorf("lesson: load: %w", err)
	}
	if _, err := s.Lessons.SavePractice(ctx, l.ID, l.Revision, cur.PracticeVersion, p); err != nil {
		return fmt.Errorf("lesson: save practice: %w", err)
	}
	// An empty list too, so the gloss job is never queued for the lesson.
	if s.Glosses != nil {
		if glosses == nil {
			glosses = []Gloss{}
		}
		if err := s.Glosses.Save(ctx, l.ID, l.Revision, glosses); err != nil {
			return fmt.Errorf("lesson: save glosses: %w", err)
		}
	}
	return nil
}

// practiceParts reports what CleanPractice and CleanSteps kept of in.
func practiceParts(in ai.Practice, p Practice) []ImportPart {
	turns := 0
	if p.Dialogue != nil {
		turns = len(p.Dialogue.Turns)
	}
	st := p.Steps
	if st == nil {
		st = &Steps{}
	}
	exercises := 0
	if st.Grammar != nil {
		exercises = len(st.Grammar.Exercises)
	}
	givenExercises := 0
	if in.GrammarStep != nil {
		givenExercises = len(in.GrammarStep.Exercises)
	}
	return []ImportPart{
		{Key: "practice.objectiveVi", LabelVi: "Mục tiêu bài", Given: one(in.ObjectiveVi != ""), Kept: one(p.ObjectiveVi != "")},
		{Key: "practice.examples", LabelVi: "Câu ví dụ của từ (Từ vựng)", Given: len(in.Examples), Kept: len(p.Examples)},
		{Key: "practice.dialogue.turns", LabelVi: "Lượt hội thoại (Nghe)", Given: len(in.Dialogue.Turns), Kept: turns},
		{Key: "practice.listeningQuestions", LabelVi: "Câu hỏi nghe", Given: len(in.ListeningQuestions), Kept: len(st.ListeningQuestions)},
		{Key: "practice.readingGuideVi", LabelVi: "Câu hỏi dẫn (Đọc)", Given: one(in.ReadingGuideVi != ""), Kept: one(st.ReadingGuideVi != "")},
		{Key: "practice.grammarStep", LabelVi: "Bước Ngữ pháp", Given: one(in.GrammarStep != nil), Kept: one(st.Grammar != nil)},
		{Key: "practice.grammarStep.exercises", LabelVi: "Bài tập ngữ pháp", Given: givenExercises, Kept: exercises},
		{Key: "practice.translations", LabelVi: "Câu dịch xếp chữ", Given: len(in.Translations), Kept: len(p.Translations)},
		{Key: "practice.readAloud", LabelVi: "Câu đọc to (Nói)", Given: len(in.ReadAloud), Kept: len(st.ReadAloud)},
		{Key: "practice.speakQuestions", LabelVi: "Câu hỏi nói", Given: len(in.SpeakQuestions), Kept: len(st.SpeakQuestions)},
		{Key: "practice.writingHelp", LabelVi: "Gợi ý bước Viết", Given: one(in.WritingHelp != nil), Kept: one(st.Writing != nil)},
	}
}

func one(b bool) int {
	if b {
		return 1
	}
	return 0
}
