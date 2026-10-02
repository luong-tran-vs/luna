package lesson

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/luongtran/luna/backend/internal/ai"
)

// withFamilyWords gives topic-a1 a word list.
func withFamilyWords(e *env) {
	e.topics.mu.Lock()
	defer e.topics.mu.Unlock()
	t := e.topics.topics["topic-a1"]
	t.Words = []string{"Family", "Brother", "take a shower", "Uncle"}
	e.topics.topics["topic-a1"] = t
}

func TestGenerateTargetWordsValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		targets [][]string
		field   string
	}{
		{"wrong group count", [][]string{{"Family"}}, "targetWords"},
		{"unknown word", [][]string{{"Family"}, {"banana"}, {}}, "targetWords.1.0"},
		{"duplicate in group", [][]string{{"Family", "family"}, {}, {}}, "targetWords.0.1"},
		{"too many", [][]string{slices.Repeat([]string{"x"}, 16), {}, {}}, "targetWords.0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			e := newEnv(t)
			withFamilyWords(e)
			in := validGenerate()
			in.TargetWords = tt.targets
			_, err := e.svc.Generate(t.Context(), "topic-a1", in)
			var verr *ValidationError
			if !errors.As(err, &verr) || verr.Fields[tt.field] == "" {
				t.Fatalf("err = %v, want field %s", err, tt.field)
			}
			if _, calls := e.ai.lastRequest(); calls != 0 {
				t.Fatalf("AI called %d times", calls)
			}
		})
	}
}

func TestGenerateTargetWords(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	withFamilyWords(e)
	long := text(100)
	e.ai.drafts = []ai.LessonDraft{
		{Title: "", Content: text(10)}, // dropped (no title): its group must not move to the next draft
		{Title: "Brothers", Content: "My brothers took a shower. " + text(95)},
		{Title: "Plain", Content: long},
	}
	in := validGenerate()
	in.TargetWords = [][]string{{"uncle"}, {"brother", " TAKE A SHOWER ", "family"}, {}}

	res, err := e.svc.Generate(t.Context(), "topic-a1", in)
	if err != nil {
		t.Fatal(err)
	}
	req, _ := e.ai.lastRequest()
	if want := [][]string{{"Uncle"}, {"Brother", "take a shower", "Family"}, {}}; len(req.TargetWords) != 3 ||
		!slices.Equal(req.TargetWords[0], want[0]) || !slices.Equal(req.TargetWords[1], want[1]) || len(req.TargetWords[2]) != 0 {
		t.Fatalf("AI target words = %q", req.TargetWords)
	}
	if len(res.Drafts) != 2 {
		t.Fatalf("drafts = %+v", res.Drafts)
	}
	d := res.Drafts[0]
	if !slices.Equal(d.TargetWords, []string{"Brother", "take a shower", "Family"}) || !slices.Equal(d.MissingWords, []string{"Family"}) {
		t.Fatalf("draft 0 = %q / %q", d.TargetWords, d.MissingWords)
	}
	if d := res.Drafts[1]; len(d.TargetWords) != 0 || d.TargetWords == nil || len(d.MissingWords) != 0 || d.MissingWords == nil {
		t.Fatalf("draft 1 = %#v / %#v", d.TargetWords, d.MissingWords)
	}
}

func TestGenerateWithoutTargetWords(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	withFamilyWords(e)
	e.ai.drafts = []ai.LessonDraft{{Title: "Plain", Content: text(100)}}
	res, err := e.svc.Generate(t.Context(), "topic-a1", validGenerate())
	if err != nil {
		t.Fatal(err)
	}
	if req, _ := e.ai.lastRequest(); req.TargetWords != nil {
		t.Fatalf("AI target words = %q", req.TargetWords)
	}
	if d := res.Drafts[0]; d.TargetWords == nil || len(d.TargetWords) != 0 || d.MissingWords == nil {
		t.Fatalf("draft = %#v", d)
	}
}

func TestGenerateEndpointTargetWords(t *testing.T) {
	t.Parallel()
	a := newAPI(t)
	withFamilyWords(a.env)
	a.env.ai.drafts = []ai.LessonDraft{{Title: "Brothers", Content: "My brother is here. " + text(96)}}

	rec := a.do(t, "POST", "/api/admin/topics/topic-a1/generate", "admin",
		`{"count":1,"words":100,"kind":"reading","idea":"","targetWords":[["Brother","Uncle"]]}`)
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"targetWords":["Brother","Uncle"],"missingWords":["Uncle"]`) {
		t.Fatalf("generate: %d %s", rec.Code, rec.Body)
	}
	rec = a.do(t, "POST", "/api/admin/topics/topic-a1/generate", "admin",
		`{"count":1,"words":100,"kind":"reading","idea":"","targetWords":[["banana"]]}`)
	if rec.Code != 400 || !strings.Contains(rec.Body.String(), `"targetWords.0.0"`) {
		t.Fatalf("invalid: %d %s", rec.Code, rec.Body)
	}
}
