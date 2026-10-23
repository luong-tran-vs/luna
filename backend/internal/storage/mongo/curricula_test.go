package mongo

import (
	"reflect"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/luongtran/luna/backend/internal/curriculum"
)

func TestCurriculumPlanDocRoundTrip(t *testing.T) {
	p := &curriculum.Plan{Level: "A2", Topics: []curriculum.Topic{{
		TopicID: "t1", Name: "Du lịch", CanDoVi: "kể về chuyến đi.", Grammar: []string{"a2-past-simple-regular"},
		Slots: []curriculum.Slot{{Key: "a2-01-1", Kind: curriculum.KindMain, Title: "My Trip", Grammar: []string{"a2-past-simple-regular"}, Idea: "Lan visits Hue."}},
	}}}
	data, err := bson.Marshal(curriculumDoc{Level: "A2", Draft: toCurriculumPlanDoc(p)})
	if err != nil {
		t.Fatal(err)
	}
	var d curriculumDoc
	if err := bson.Unmarshal(data, &d); err != nil {
		t.Fatal(err)
	}
	if got := d.Draft.plan("A2"); !reflect.DeepEqual(got, p) {
		t.Errorf("round trip\n got %+v\nwant %+v", got, p)
	}
	if d.Approved.plan("A2") != nil || toCurriculumPlanDoc(nil) != nil {
		t.Error("no plan stays nil")
	}
}
