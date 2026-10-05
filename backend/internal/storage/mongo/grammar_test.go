package mongo

import (
	"reflect"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/luongtran/luna/backend/internal/grammar"
)

func sampleGrammarLesson() grammar.Lesson {
	t := time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC)
	return grammar.Lesson{
		PointID: "a1-to-be", Status: grammar.StatusPublished, Edited: true,
		CreatedAt: t, UpdatedAt: t.Add(time.Hour), PublishedAt: t.Add(2 * time.Hour),
		CheckedAt: t.Add(90 * time.Minute), VerifiedAt: t.Add(100 * time.Minute),
		Checks: []grammar.Check{
			{ExerciseID: "p1", Kind: grammar.CheckMismatch, NoteVi: "AI chọn đáp án khác"},
			{ExerciseID: "m1", Kind: grammar.CheckAmbiguous, NoteVi: "mơ hồ", Confirmed: true},
		},
		Content: grammar.Content{
			Objective:   "Bạn có thể dùng to be.",
			Explanation: []string{"một", "hai"},
			Usage:       []string{"dùng khi …"},
			Structures:  []grammar.Structure{{Label: "Khẳng định", Pattern: "S + am/is/are", Example: "I am here."}},
			Examples:    []grammar.Example{{En: "I am a student.", Vi: "Tôi là học sinh."}},
			Mistakes:    []grammar.Mistake{{Wrong: "She are", Right: "She is", NoteVi: "chia động từ"}},
			Practice: []grammar.Exercise{
				{ID: "p1", Kind: grammar.KindChoice, PromptVi: "Chọn", Text: "She ___ a teacher.",
					Options: []string{"am", "is", "are", "be"}, AnswerIndex: 1, ExplanationVi: "vì she"},
				{ID: "p2", Kind: grammar.KindFill, Text: "I ___ ok.", Answers: []string{"am"}, ExplanationVi: "vì I"},
			},
			Mastery: []grammar.Exercise{
				{ID: "m1", Kind: grammar.KindReorder, Text: "Tôi là học sinh.", Sentence: "I am a student",
					Words: []string{"I", "am", "a", "student", "is"}, ExplanationVi: "trật tự"},
			},
		},
	}
}

func TestGrammarLessonDocRoundTrip(t *testing.T) {
	t.Parallel()
	in := sampleGrammarLesson()
	raw, err := bson.Marshal(fromGrammarLesson(in))
	if err != nil {
		t.Fatal(err)
	}
	var back grammarLessonDoc
	if err := bson.Unmarshal(raw, &back); err != nil {
		t.Fatal(err)
	}
	if got := back.toLesson(); !reflect.DeepEqual(got, in) {
		t.Fatalf("round trip:\n got %+v\nwant %+v", got, in)
	}
}

func TestGrammarLessonDocEmptyAndUnpublished(t *testing.T) {
	t.Parallel()
	in := grammar.Lesson{PointID: "a1-x", Status: grammar.StatusDraft}
	raw, err := bson.Marshal(fromGrammarLesson(in))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := bson.Raw(raw).LookupErr("publishedAt"); err == nil {
		t.Fatal("a zero publishedAt should be omitted")
	}
	if _, err := bson.Raw(raw).LookupErr("checkedAt"); err == nil {
		t.Fatal("a zero checkedAt should be omitted")
	}
	if _, err := bson.Raw(raw).LookupErr("verifiedAt"); err == nil {
		t.Fatal("a zero verifiedAt should be omitted")
	}
	if v, err := bson.Raw(raw).LookupErr("checks"); err != nil || v.Type != bson.TypeArray {
		t.Fatalf("checks must be stored as an array: %v %v", v, err)
	}
	var back grammarLessonDoc
	if err := bson.Unmarshal(raw, &back); err != nil {
		t.Fatal(err)
	}
	got := back.toLesson()
	if !got.PublishedAt.IsZero() || got.Content.Explanation == nil || got.Content.Practice == nil ||
		got.Checks == nil || len(got.Checks) != 0 || !got.CheckedAt.IsZero() || !got.VerifiedAt.IsZero() {
		t.Fatalf("lists must read back non-nil: %+v", got)
	}
}

func TestGrammarReportDoc(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 10, 2, 9, 0, 0, 0, time.UTC)
	d := grammarReportDoc{
		ID: bson.NewObjectID(), UserID: bson.NewObjectID(), PointID: "a1-to-be", ExerciseID: "p3",
		Reason: "typo", Note: "lỗi", Status: "open", CreatedAt: now, UpdatedAt: now.Add(time.Hour),
	}
	raw, err := bson.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := bson.Raw(raw).LookupErr("resolvedAt"); err == nil {
		t.Fatal("a zero resolvedAt should be omitted")
	}
	var back grammarReportDoc
	if err := bson.Unmarshal(raw, &back); err != nil {
		t.Fatal(err)
	}
	want := grammar.Report{
		ID: d.ID.Hex(), UserID: d.UserID.Hex(), PointID: "a1-to-be", ExerciseID: "p3", Reason: grammar.ReasonTypo,
		Note: "lỗi", Status: grammar.ReportOpen, CreatedAt: now, UpdatedAt: now.Add(time.Hour),
	}
	if got := back.toReport(); !reflect.DeepEqual(got, want) {
		t.Fatalf("report = %+v, want %+v", got, want)
	}
}

func TestGrammarProgressDoc(t *testing.T) {
	t.Parallel()
	uid := bson.NewObjectID()
	d := grammarProgressDoc{
		UserID: uid, PointID: "a1-to-be", Status: "learning", PracticeAttempts: 2, LastPractice: 50,
		BestMastery: 70, UpdatedAt: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
	}
	raw, err := bson.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := bson.Raw(raw).LookupErr("masteredAt"); err == nil {
		t.Fatal("a zero masteredAt should be omitted")
	}
	var back grammarProgressDoc
	if err := bson.Unmarshal(raw, &back); err != nil {
		t.Fatal(err)
	}
	p := back.toProgress()
	if p.UserID != uid.Hex() || p.PointID != "a1-to-be" || p.PracticeAttempts != 2 || p.BestMastery != 70 ||
		!p.MasteredAt.IsZero() || p.Weak == nil || len(p.Weak) != 0 {
		t.Fatalf("progress = %+v", p)
	}
}

func TestGrammarSummaryDocCountsUnconfirmed(t *testing.T) {
	t.Parallel()
	raw, err := bson.Marshal(fromGrammarLesson(sampleGrammarLesson()))
	if err != nil {
		t.Fatal(err)
	}
	var sum grammarSummaryDoc
	if err := bson.Unmarshal(raw, &sum); err != nil {
		t.Fatal(err)
	}
	if n := unconfirmed(sum.Checks); n != 1 || sum.VerifiedAt.IsZero() || sum.CheckedAt.IsZero() {
		t.Fatalf("summary = %+v, unconfirmed = %d", sum, n)
	}
}
