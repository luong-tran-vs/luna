package mongo

import (
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/luongtran/luna/backend/internal/lesson"
)

func TestAILookupDocToResult(t *testing.T) {
	t.Parallel()
	lid := bson.NewObjectID()
	at := time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC)
	got := aiLookupDoc{
		LessonID: lid, Revision: 2, SentenceIndex: 3, TextLower: "make up for",
		Lemma: "make up for", MeaningVi: "bù lại", NoteVi: "Ghi chú.", CreatedAt: at,
	}.toResult()
	want := lesson.AskResult{
		AskKey: lesson.AskKey{LessonID: lid.Hex(), Revision: 2, SentenceIndex: 3, Text: "make up for"},
		Lemma:  "make up for", MeaningVi: "bù lại", NoteVi: "Ghi chú.", CreatedAt: at,
	}
	if got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
	if f, err := askFilter(lesson.AskKey{LessonID: "bad"}); err == nil || f != nil {
		t.Fatalf("bad id filter = %v, %v", f, err)
	}
}
