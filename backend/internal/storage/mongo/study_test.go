package mongo

import (
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/luongtran/luna/backend/internal/progress"
)

func TestProgressDocStepTimes(t *testing.T) {
	t.Parallel()
	at := time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC)
	raw, err := bson.Marshal(progressDoc{
		UserID: bson.NewObjectID(), LessonID: bson.NewObjectID(),
		Steps:       map[string]bool{"read": true, "listen": true},
		StepsDoneAt: map[string]time.Time{"read": at},
	})
	if err != nil {
		t.Fatal(err)
	}
	var d progressDoc
	if err := bson.Unmarshal(raw, &d); err != nil {
		t.Fatal(err)
	}
	p := d.toProgress()
	if !p.Done[progress.StepRead] || !p.Done[progress.StepListen] || len(p.DoneAt) != 1 || !p.DoneAt[progress.StepRead].Equal(at) {
		t.Fatalf("progress = %+v", p)
	}

	// Progress saved before the step times has none.
	raw, _ = bson.Marshal(progressDoc{Steps: map[string]bool{"read": true}})
	if bson.Raw(raw).Lookup("stepsDoneAt").Type != 0 {
		t.Fatal("empty stepsDoneAt is stored")
	}
	d = progressDoc{}
	_ = bson.Unmarshal(raw, &d)
	if p := d.toProgress(); p.DoneAt != nil || !p.Done[progress.StepRead] {
		t.Fatalf("old progress = %+v", p)
	}
}

func TestWithSince(t *testing.T) {
	t.Parallel()
	base := bson.D{{Key: "userId", Value: "u"}}
	if got := withSince(base, "createdAt", nil); len(got) != 1 {
		t.Fatalf("nil since = %v", got)
	}
	since := time.Date(2026, 9, 28, 0, 0, 0, 0, time.FixedZone("ICT", 7*3600))
	got := withSince(base, "createdAt", &since)
	if len(got) != 2 || got[1].Key != "createdAt" {
		t.Fatalf("since = %v", got)
	}
	cond, _ := got[1].Value.(bson.D)
	at, _ := cond[0].Value.(time.Time)
	if cond[0].Key != "$gte" || !at.Equal(since) || at.Location() != time.UTC {
		t.Fatalf("condition = %v", cond)
	}
}
