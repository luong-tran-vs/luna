package mongo

import (
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/luongtran/luna/backend/internal/writing"
)

func TestWritingDocConversion(t *testing.T) {
	t.Parallel()
	at := time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC)
	g := writing.Grade{
		Status: writing.GradeDone, Criteria: []writing.Criterion{{Name: "task", Score: 4, CommentVi: "Tốt."}},
		OverallVi: "Khá.", CorrectedText: "Fixed.", GradedAt: at, Seen: false,
	}
	d := writingDoc{
		ID: bson.NewObjectID(), UserID: bson.NewObjectID(), LessonID: bson.NewObjectID(), LessonRevision: 2,
		LessonTitle: "Family", Prompt: "P", Text: "T", Status: "submitted", SubmittedAt: at,
	}
	gd := fromGrade(g)
	d.Grade = &gd
	w := d.toWriting()
	if w.ID != d.ID.Hex() || w.Status != writing.StatusSubmitted || w.LessonRevision != 2 || !w.SubmittedAt.Equal(at) {
		t.Fatalf("writing = %+v", w)
	}
	if w.Grade == nil || w.Grade.Status != writing.GradeDone || w.Grade.Criteria[0] != g.Criteria[0] || w.Grade.Seen || !w.Grade.GradedAt.Equal(at) {
		t.Fatalf("grade = %+v", w.Grade)
	}
	d.Grade = nil
	if d.toWriting().Grade != nil {
		t.Fatal("draft has a grade")
	}
}
