package mongo

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/luongtran/luna/backend/internal/lesson"
)

func TestLessonDocGrammarPoint(t *testing.T) {
	t.Parallel()
	raw, err := bson.Marshal(fromLesson(lesson.Lesson{Title: "t", GrammarPointID: "a1-to-be"}))
	if err != nil {
		t.Fatal(err)
	}
	var back lessonDoc
	if err := bson.Unmarshal(raw, &back); err != nil {
		t.Fatal(err)
	}
	if got := back.toLesson().GrammarPointID; got != "a1-to-be" {
		t.Fatalf("GrammarPointID = %q", got)
	}

	// No point: the field is left out of the document.
	raw, err = bson.Marshal(fromLesson(lesson.Lesson{Title: "t"}))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := bson.Raw(raw).LookupErr("grammarPointId"); err == nil {
		t.Fatal("empty grammarPointId should be omitted")
	}
}
