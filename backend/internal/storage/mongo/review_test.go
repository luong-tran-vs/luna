package mongo

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/luongtran/luna/backend/internal/lesson"
)

func sampleReview() *lesson.Review {
	t := time.Date(2026, 10, 5, 8, 0, 0, 0, time.UTC)
	return &lesson.Review{
		CheckedAt:  t,
		VerifiedAt: t.Add(time.Hour),
		Flags: []lesson.Flag{
			{Area: lesson.AreaSentence, Index: 2, Kind: lesson.FlagWrong, NoteVi: "Câu sai."},
			{Area: lesson.AreaQuestion, Index: 0, Kind: lesson.FlagMismatch, NoteVi: "AI chọn khác.", Confirmed: true},
		},
	}
}

func TestLessonDocReview(t *testing.T) {
	t.Parallel()
	in := sampleReview()
	raw, err := bson.Marshal(fromLesson(lesson.Lesson{Title: "t", Review: in}))
	if err != nil {
		t.Fatal(err)
	}
	var back lessonDoc
	if err := bson.Unmarshal(raw, &back); err != nil {
		t.Fatal(err)
	}
	if got := back.toLesson().Review; !reflect.DeepEqual(got, in) {
		t.Fatalf("Review = %+v, want %+v", got, in)
	}

	// Not verified: the zero time comes back as the zero time.
	open := sampleReview()
	open.VerifiedAt = time.Time{}
	raw, _ = bson.Marshal(fromLesson(lesson.Lesson{Review: open}))
	var openBack lessonDoc
	if err := bson.Unmarshal(raw, &openBack); err != nil {
		t.Fatal(err)
	}
	if got := openBack.toLesson().Review; got == nil || !got.VerifiedAt.IsZero() {
		t.Fatalf("unverified Review = %+v", got)
	}

	// Never checked: the field is left out of the document and reads back as nil.
	raw, err = bson.Marshal(fromLesson(lesson.Lesson{Title: "t"}))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := bson.Raw(raw).LookupErr("review"); err == nil {
		t.Fatal("nil review should be omitted")
	}
	var plain lessonDoc
	if err := bson.Unmarshal(raw, &plain); err != nil || plain.toLesson().Review != nil {
		t.Fatalf("plain = %+v, %v", plain.toLesson().Review, err)
	}
}

func TestReviewDocFlagsAreAlwaysAnArray(t *testing.T) {
	t.Parallel()
	raw, err := bson.Marshal(fromReview(&lesson.Review{CheckedAt: time.Now()}))
	if err != nil {
		t.Fatal(err)
	}
	if v, err := bson.Raw(raw).LookupErr("flags"); err != nil || v.Type != bson.TypeArray {
		t.Fatalf("flags = %v, %v", v, err)
	}
}

func TestSummaryDocReview(t *testing.T) {
	t.Parallel()
	d := summaryDoc{Title: "t", Review: fromReview(sampleReview())}
	s := d.toSummary()
	if s.Flags != 1 || !s.Checked || !s.Verified {
		t.Fatalf("summary = %+v", s)
	}
	s = summaryDoc{Title: "t"}.toSummary()
	if s.Flags != 0 || s.Checked || s.Verified {
		t.Fatalf("unchecked summary = %+v", s)
	}
	found := false
	for _, e := range summaryProjection {
		found = found || e.Key == "review"
	}
	if !found {
		t.Error("summaryProjection does not read review")
	}
}

func TestReviewUnsetAndMalformedID(t *testing.T) {
	t.Parallel()
	if len(reviewUnset) != 1 || reviewUnset[0].Key != "review" {
		t.Fatalf("reviewUnset = %v", reviewUnset)
	}
	if _, err := (&Lessons{}).SaveReview(t.Context(), "bad", 1, sampleReview()); !errors.Is(err, lesson.ErrNotFound) {
		t.Fatalf("malformed id: %v", err)
	}
}
