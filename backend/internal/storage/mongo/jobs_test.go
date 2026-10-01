package mongo

import (
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/luongtran/luna/backend/internal/job"
)

func TestJobDocTargets(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC)
	wid := bson.NewObjectID().Hex()

	d, err := newJobDoc(job.Job{Type: job.TypeGrade, TargetID: wid}, now)
	if err != nil {
		t.Fatal(err)
	}
	if !d.LessonID.IsZero() || d.Status != "pending" {
		t.Fatalf("grade doc = %+v", d)
	}
	if j := d.toJob(); j.LessonID != "" || j.TargetID != wid || j.Type != job.TypeGrade {
		t.Fatalf("grade job = %+v", j)
	}

	lid := bson.NewObjectID().Hex()
	d, err = newJobDoc(job.Job{Type: job.TypeTTS, LessonID: lid, Revision: 2}, now)
	if err != nil {
		t.Fatal(err)
	}
	if j := d.toJob(); j.LessonID != lid || j.TargetID != "" || j.Revision != 2 {
		t.Fatalf("tts job = %+v", j)
	}

	if _, err := newJobDoc(job.Job{Type: job.TypeGrade, TargetID: "nope"}, now); err == nil {
		t.Fatal("bad target id accepted")
	}
}
