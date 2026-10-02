package mongo

import (
	"slices"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestTopicDocWords(t *testing.T) {
	t.Parallel()

	raw, err := bson.Marshal(topicDoc{Name: "Gia đình", Words: []string{"Family", "take a shower"}, WordsSeeded: true})
	if err != nil {
		t.Fatal(err)
	}
	var d topicDoc
	if err := bson.Unmarshal(raw, &d); err != nil {
		t.Fatal(err)
	}
	if got := d.toTopic(); !slices.Equal(got.Words, []string{"Family", "take a shower"}) || !got.WordsSeeded {
		t.Fatalf("topic = %+v", got)
	}

	// A topic from before F18 has neither field.
	raw, _ = bson.Marshal(bson.D{{Key: "name", Value: "Chung"}})
	d = topicDoc{}
	if err := bson.Unmarshal(raw, &d); err != nil {
		t.Fatal(err)
	}
	if got := d.toTopic(); len(got.Words) != 0 || got.WordsSeeded {
		t.Fatalf("old topic = %+v", got)
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
