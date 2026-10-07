package mongo

import (
	"reflect"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/luongtran/luna/backend/internal/lesson"
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

func TestAnswerUpsertReplacesByQuestion(t *testing.T) {
	t.Parallel()
	uid, lid := bson.NewObjectID(), bson.NewObjectID()
	at := time.Date(2026, 10, 1, 8, 0, 0, 0, time.FixedZone("ICT", 7*3600))
	filter, update := answerUpsert(uid, lid, lesson.Answer{QuizVersion: 3, QuestionIndex: 2, Choice: 1, Correct: true, AnsweredAt: at})
	want := bson.D{
		{Key: "userId", Value: uid}, {Key: "lessonId", Value: lid}, {Key: "quizVersion", Value: 3}, {Key: "questionIndex", Value: 2},
	}
	if !reflect.DeepEqual(filter, want) {
		t.Fatalf("filter = %v", filter)
	}
	set := update[0].Value.(bson.D)
	if update[0].Key != "$set" || len(set) != 3 || set[0].Value != 1 || set[1].Value != true ||
		set[2].Value.(time.Time).Location() != time.UTC || !set[2].Value.(time.Time).Equal(at) {
		t.Fatalf("update = %v", update)
	}
}
