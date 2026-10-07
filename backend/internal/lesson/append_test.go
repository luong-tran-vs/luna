package lesson

import (
	"errors"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/luongtran/luna/backend/internal/job"
)

// --- F7: saving a draft appends it to the roadmap in the same request ---

func TestCreateAppendsToRoadmap(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.topics.setRoadmap("topic-a1", "old")
	l := e.create(t, func(in *Input) { in.TopicID, in.Level = "topic-a1", "A1"; in.AppendToRoadmap = true })

	if got := e.topics.roadmap("topic-a1"); !slices.Equal(got, []string{"old", l.ID}) {
		t.Fatalf("roadmap = %v, want lesson at the end", got)
	}
	if jobs := e.jobs.all(); len(jobs) != 1 || jobs[0].Type != job.TypeAnnotate {
		t.Fatalf("jobs = %+v", jobs)
	}
}

func TestCreateWithoutFlagDoesNotAppend(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.create(t, func(in *Input) { in.TopicID, in.Level = "topic-a1", "A1" })
	if got := e.topics.roadmap("topic-a1"); len(got) != 0 {
		t.Fatalf("roadmap = %v, want empty", got)
	}
}

func TestCreateAppendFailureRemovesLesson(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	boom := errors.New("boom")
	e.topics.failAppend = boom

	_, err := e.svc.Create(t.Context(), Input{
		Title: "Park", Content: sampleContent, TopicID: "topic-a1", Level: "A1", Source: "AI sinh", License: "Nội dung do AI tạo",
		AppendToRoadmap: true,
	})
	if !errors.Is(err, boom) {
		t.Fatalf("error = %v, want boom", err)
	}
	if items, _ := e.lessons.List(t.Context(), Filter{}); len(items) != 0 {
		t.Fatalf("lessons = %+v, want the created lesson removed", items)
	}
	if jobs := e.jobs.all(); len(jobs) != 0 {
		t.Fatalf("jobs = %+v, want none", jobs)
	}
}

func TestCreateAppendValidatesFirst(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	var verr *ValidationError
	if _, err := e.svc.Create(t.Context(), Input{TopicID: "topic-a1", Level: "A1", AppendToRoadmap: true}); !errors.As(err, &verr) {
		t.Fatalf("error = %v, want ValidationError", err)
	}
	if got := e.topics.roadmap("topic-a1"); len(got) != 0 {
		t.Fatalf("roadmap = %v", got)
	}
}

func TestCreateEndpointAppendToRoadmap(t *testing.T) {
	t.Parallel()
	a := newAPI(t)
	body := strings.Replace(validBody, `"license":"CC BY"`, `"license":"CC BY","appendToRoadmap":true`, 1)
	rec := a.do(t, http.MethodPost, "/api/admin/lessons", "admin", body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", rec.Code, rec.Body)
	}
	l := decodeBody(t, rec)["lesson"].(map[string]any)
	if l["inRoadmap"] != true || l["annotationStatus"] != "running" {
		t.Fatalf("lesson = %v", l)
	}

	// PUT ignores the flag: editing never touches the roadmap.
	id := l["id"].(string)
	a.env.topics.setRoadmap("topic-b1")
	if rec := a.do(t, http.MethodPut, "/api/admin/lessons/"+id, "admin", body); rec.Code != http.StatusOK {
		t.Fatalf("update: %d %s", rec.Code, rec.Body)
	}
	if got := a.env.topics.roadmap("topic-b1"); len(got) != 0 {
		t.Fatalf("roadmap after PUT = %v", got)
	}
}
