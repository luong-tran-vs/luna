package mongo

import (
	"slices"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/luongtran/luna/backend/internal/topic"
)

func TestTopicDocWordsAndRoadmaps(t *testing.T) {
	t.Parallel()

	lesson := bson.NewObjectID()
	raw, err := bson.Marshal(topicDoc{
		Name: "Gia đình", Words: toWordDocs([]topic.Word{{Text: "Family"}, {Text: "take a shower", Level: "A2"}}), WordsSeeded: true,
		Roadmaps: map[string][]bson.ObjectID{"A1": {lesson}},
	})
	if err != nil {
		t.Fatal(err)
	}
	var d topicDoc
	if err := bson.Unmarshal(raw, &d); err != nil {
		t.Fatal(err)
	}
	got := d.toTopic()
	if !slices.Equal(got.Words, []topic.Word{{Text: "Family"}, {Text: "take a shower", Level: "A2"}}) || !got.WordsSeeded {
		t.Fatalf("topic = %+v", got)
	}
	if !slices.Equal(got.Roadmap("A1"), []string{lesson.Hex()}) || len(got.Roadmap("B1")) != 0 {
		t.Fatalf("roadmaps = %v", got.Roadmaps)
	}

	// A topic from before F18 has neither field.
	raw, _ = bson.Marshal(bson.D{{Key: "name", Value: "Chung"}})
	d = topicDoc{}
	if err := bson.Unmarshal(raw, &d); err != nil {
		t.Fatal(err)
	}
	if got := d.toTopic(); len(got.Words) != 0 || got.WordsSeeded || got.Roadmaps == nil {
		t.Fatalf("old topic = %+v", got)
	}
}

func TestWordDocReadsPlainStrings(t *testing.T) {
	t.Parallel()
	// Words stored before they had a level are strings; the seed still writes them so.
	raw, _ := bson.Marshal(bson.D{{Key: "words", Value: bson.A{"Family", bson.D{{Key: "text", Value: "aunt"}, {Key: "level", Value: "B1"}}}}})
	var d topicDoc
	if err := bson.Unmarshal(raw, &d); err != nil {
		t.Fatal(err)
	}
	if got := fromWordDocs(d.Words); !slices.Equal(got, []topic.Word{{Text: "Family"}, {Text: "aunt", Level: "B1"}}) {
		t.Fatalf("words = %+v", got)
	}
}

func TestMergeTopicDocReadsBothShapes(t *testing.T) {
	t.Parallel()
	lesson := bson.NewObjectID()
	raw, _ := bson.Marshal(bson.D{
		{Key: "name", Value: "Gia đình"}, {Key: "level", Value: "A2"}, {Key: "lessonIds", Value: bson.A{lesson}},
		{Key: "words", Value: bson.A{"Family"}},
	})
	var d mergeTopicDoc
	if err := bson.Unmarshal(raw, &d); err != nil {
		t.Fatal(err)
	}
	s := d.toSource()
	if s.Level != "A2" || !slices.Equal(s.LessonIDs, []string{lesson.Hex()}) || len(s.Words) != 1 || s.Words[0].Text != "Family" {
		t.Fatalf("legacy source = %+v", s)
	}
}

func TestRoadmapField(t *testing.T) {
	t.Parallel()
	if f, err := roadmapField("B2"); err != nil || f != "roadmaps.B2" {
		t.Fatalf("field = %q, %v", f, err)
	}
	if _, err := roadmapField("$where"); err == nil {
		t.Fatal("bad level accepted")
	}
}

func TestTopicTextDocLemmas(t *testing.T) {
	t.Parallel()
	d := topicTextDoc{Content: "He went up.", Annotations: []annotationTextDoc{
		{Text: "Went", Lemma: "Go"}, {Text: "gave up", Lemma: "give up"},
	}}
	got := d.toLessonText()
	if got.Content != "He went up." || len(got.Lemmas) != 1 || got.Lemmas["went"] != "go" {
		t.Fatalf("lesson text = %+v", got)
	}
}
