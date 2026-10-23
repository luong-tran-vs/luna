package progress

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"
)

type fakeRecords struct {
	rec     Records
	lessons []string
}

func (f *fakeRecords) Records(_ context.Context, _ string, lessonIDs []string) (Records, error) {
	f.lessons = lessonIDs
	return f.rec, nil
}

func TestTopicSummary(t *testing.T) {
	t.Parallel()
	e := newStudyEnv()
	rec := &fakeRecords{rec: Records{
		Words: 12, Remembered: 5,
		Quiz:     Score{Correct: 7, Total: 8},
		Review:   Score{Correct: 9, Total: 10},
		Writings: 2, WritingAverage: 4,
		Mistakes: []MistakeCount{{Type: "tense", Count: 3}},
	}}
	e.svc.d.Records = rec
	e.svc.d.Course = func(_ context.Context, level, topicID, _ string) (CourseTopic, bool) {
		return CourseTopic{CanDoVi: "giới thiệu gia đình", GrammarVi: []string{"have got"}}, level == "A1" && topicID == "family"
	}
	ctx := t.Context()

	// Nothing studied yet: the lessons are listed, nothing is rated.
	s, err := e.svc.TopicSummary(ctx, "u1", "family", "A1")
	if err != nil {
		t.Fatal(err)
	}
	if s.TopicName != "Gia đình" || len(s.Lessons) != 3 || s.Completed != 0 || s.Rating != RatingNone ||
		len(s.Skills) != 0 || rec.lessons != nil || s.CanDoVi != "giới thiệu gia đình" {
		t.Fatalf("empty summary = %+v", s)
	}

	day1 := time.Date(2026, 9, 28, 8, 0, 0, 0, hcm)
	day2 := day1.AddDate(0, 0, 2)
	for _, p := range []LessonProgress{
		{UserID: "u1", LessonID: "f1", DayKey: "2026-09-28", StartedAt: day1, CompletedAt: day1.Add(time.Hour)},
		{UserID: "u1", LessonID: "f2", DayKey: "2026-09-30", StartedAt: day2, CompletedAt: day2.Add(time.Hour)},
		// Another topic's lesson and a lesson not finished do not count.
		{UserID: "u1", LessonID: "s1", DayKey: "2026-09-30", StartedAt: day2, CompletedAt: day2},
		{UserID: "u1", LessonID: "f3", DayKey: "2026-09-30", StartedAt: day2},
	} {
		if err := e.progress.Upsert(ctx, p); err != nil {
			t.Fatal(err)
		}
	}
	// Listening: 8 of 10 words written down right in f1.
	if _, err := e.dictation.Record(ctx, "u1", "f1", Input{SentenceIndex: 0, Typed: "a b c", CorrectWords: 8, TotalWords: 10}); err != nil {
		t.Fatal(err)
	}

	s, err = e.svc.TopicSummary(ctx, "u1", "family", "A1")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(rec.lessons, []string{"f1", "f2"}) {
		t.Errorf("records asked for %v, want the lessons done", rec.lessons)
	}
	if s.Completed != 2 || s.Days != 2 || !s.StartedAt.Equal(day1) || !s.FinishedAt.Equal(day2.Add(time.Hour)) ||
		!s.Lessons[0].Done || s.Lessons[2].Done || s.Lessons[1].Title != "Family 2" {
		t.Errorf("lessons = %+v", s)
	}
	want := []SkillScore{
		{Skill: SkillListening, Percent: 80, Correct: 8, Total: 10},
		{Skill: SkillReading, Percent: 88, Correct: 7, Total: 8},
		{Skill: SkillWriting, Percent: 80, Correct: 2},
		{Skill: SkillReview, Percent: 90, Correct: 9, Total: 10},
	}
	if !slices.Equal(s.Skills, want) {
		t.Errorf("skills = %+v", s.Skills)
	}
	// (80 + 88 + 80 + 90) / 4 = 84.5, rounded.
	if s.Overall != 85 || s.Rating != RatingGood || s.Words != 12 || s.Remembered != 5 || s.Mistakes[0].Count != 3 {
		t.Errorf("overall %d, rating %s, %+v", s.Overall, s.Rating, s)
	}

	var verr *ValidationError
	if _, err := e.svc.TopicSummary(ctx, "u1", "family", "Z9"); !errors.As(err, &verr) {
		t.Errorf("bad level: %v", err)
	}
	if _, err := e.svc.TopicSummary(ctx, "u1", "nope", "A1"); !errors.Is(err, ErrTopicNotFound) {
		t.Errorf("unknown topic: %v", err)
	}
}

func TestRate(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		overall int
		want    Rating
	}{{95, RatingExcellent}, {90, RatingExcellent}, {89, RatingGood}, {75, RatingGood}, {74, RatingFair}, {60, RatingFair}, {59, RatingPractice}} {
		if got := rate(c.overall); got != c.want {
			t.Errorf("rate(%d) = %s, want %s", c.overall, got, c.want)
		}
	}
}
