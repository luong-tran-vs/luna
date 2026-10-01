package mongo

import (
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestReadingAnswerDocToAnswer(t *testing.T) {
	t.Parallel()
	uid, lid := bson.NewObjectID(), bson.NewObjectID()
	at := time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC)
	a := readingAnswerDoc{
		UserID: uid, LessonID: lid, QuizVersion: 3, QuestionIndex: 2, Choice: 1, Correct: true, AnsweredAt: at,
	}.toAnswer()
	if a.UserID != uid.Hex() || a.LessonID != lid.Hex() || a.QuizVersion != 3 || a.QuestionIndex != 2 ||
		a.Choice != 1 || !a.Correct || !a.AnsweredAt.Equal(at) {
		t.Fatalf("answer = %+v", a)
	}
}
