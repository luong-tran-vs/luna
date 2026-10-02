package mongo

import (
	"reflect"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/luongtran/luna/backend/internal/lesson"
)

func TestPracticeDocRoundTrip(t *testing.T) {
	t.Parallel()

	p := &lesson.Practice{
		ObjectiveVi:  "Bạn có thể chào hỏi.",
		Examples:     []lesson.Example{{Lemma: "meet", Sentence: "Nice to meet you."}},
		Dialogue:     &lesson.Dialogue{Speakers: []string{"Minh", "Anna"}, Turns: []lesson.Turn{{Speaker: 1, Text: "Hi.", MeaningVi: "Chào."}}},
		GrammarTipVi: "Mẹo.",
		Translations: []lesson.Translation{{Vi: "Chào bạn.", En: "Hello, friend.", Distractors: []string{"bye"}}},
	}
	raw, err := bson.Marshal(lessonDoc{Practice: fromPractice(p), PracticeStatus: "done", PracticeVersion: 3})
	if err != nil {
		t.Fatal(err)
	}
	var d lessonDoc
	if err := bson.Unmarshal(raw, &d); err != nil {
		t.Fatal(err)
	}
	got := d.toLesson()
	if !reflect.DeepEqual(got.Practice, p) || got.PracticeStatus != lesson.StatusDone || got.PracticeVersion != 3 {
		t.Fatalf("got %+v (status %q, version %d)", got.Practice, got.PracticeStatus, got.PracticeVersion)
	}

	noDialogue := *p
	noDialogue.Dialogue = nil
	if back := fromPractice(&noDialogue).toPractice(); back.Dialogue != nil || back.ObjectiveVi != p.ObjectiveVi {
		t.Fatalf("nil dialogue = %+v", back)
	}
	if fromPractice(nil) != nil || (*practiceDoc)(nil).toPractice() != nil {
		t.Fatal("nil practice must stay nil")
	}
}
