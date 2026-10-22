package lesson

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/luongtran/luna/backend/internal/ai"
	"github.com/luongtran/luna/backend/internal/platform/httpx"
)

// fakeSpeakAI is a fakeAI that also checks spoken answers.
type fakeSpeakAI struct {
	fakeAI
	smu      sync.Mutex
	feedback []ai.SpeakFeedback
	speakErr error
	calls    int
	req      ai.SpeakCheckRequest
}

func (f *fakeSpeakAI) CheckSpeaking(_ context.Context, req ai.SpeakCheckRequest) ([]ai.SpeakFeedback, error) {
	f.smu.Lock()
	defer f.smu.Unlock()
	f.calls++
	f.req = req
	return f.feedback, f.speakErr
}

func (f *fakeSpeakAI) speakCalls() (int, ai.SpeakCheckRequest) {
	f.smu.Lock()
	defer f.smu.Unlock()
	return f.calls, f.req
}

func newSpeakEnv(t *testing.T, provider ai.Provider) (*Reader, Lesson) {
	t.Helper()
	lessons := newFakeLessons()
	l, _ := lessons.Create(context.Background(), Lesson{
		Title: "Park", Level: "A1", TopicID: "topic-a1", Content: readingContent, Revision: 1,
		Sentences: toSentences(SplitSentences(readingContent)),
	})
	return NewReader(lessons, readingDict, newFakeTopics(), newFakeAnswers(), newFakeAsks(), provider), l
}

func TestCheckSpeaking(t *testing.T) {
	t.Parallel()
	f := &fakeSpeakAI{feedback: []ai.SpeakFeedback{
		{Index: 0, Corrected: "I like music.", NoteVi: "Thiếu chủ ngữ."},
		{Index: 1, Corrected: "she goes to school", NoteVi: "Không cần sửa."}, // same as said
		{Index: 5, Corrected: "Out of range."},
	}}
	r, l := newSpeakEnv(t, f)
	answers := []ai.SpeakAnswer{
		{Question: "What do you like?", Said: " like  music "},
		{Question: "Do you write?", Said: "  "},
		{Question: "Where does she go?", Said: "She goes to school."},
	}
	got, err := r.CheckSpeaking(t.Context(), "u1", l.ID, answers, false)
	if err != nil {
		t.Fatal(err)
	}
	want := []SpeakResult{
		{Said: "like music", Corrected: "I like music.", NoteVi: "Thiếu chủ ngữ."},
		{Said: "", Corrected: ""},
		{Said: "She goes to school.", Corrected: "She goes to school."},
	}
	if len(got) != len(want) {
		t.Fatalf("results = %+v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("result %d = %+v, want %+v", i, got[i], want[i])
		}
	}
	calls, req := f.speakCalls()
	if calls != 1 || req.Level != "A1" || len(req.Answers) != 2 || req.Answers[1].Said != "She goes to school." {
		t.Fatalf("calls %d request %+v", calls, req)
	}
}

func TestCheckSpeakingEmptyAnswers(t *testing.T) {
	t.Parallel()
	f := &fakeSpeakAI{}
	r, l := newSpeakEnv(t, f)
	got, err := r.CheckSpeaking(t.Context(), "u1", l.ID, []ai.SpeakAnswer{{Question: "Q?", Said: " "}, {Question: "Q2?"}}, false)
	if err != nil || len(got) != 2 || got[0] != (SpeakResult{}) {
		t.Fatalf("results %+v, %v", got, err)
	}
	if calls, _ := f.speakCalls(); calls != 0 {
		t.Fatalf("AI calls = %d", calls)
	}
	// Without a speaking checker, empty answers still work.
	r2, l2 := newSpeakEnv(t, &fakeAI{})
	if _, err := r2.CheckSpeaking(t.Context(), "u1", l2.ID, []ai.SpeakAnswer{{Question: "Q?"}}, false); err != nil {
		t.Fatalf("no checker, empty: %v", err)
	}
}

func TestCheckSpeakingErrors(t *testing.T) {
	t.Parallel()
	said := func(n int) []ai.SpeakAnswer {
		out := make([]ai.SpeakAnswer, n)
		for i := range out {
			out[i] = ai.SpeakAnswer{Question: "Q?", Said: "I like it."}
		}
		return out
	}
	tests := []struct {
		name     string
		provider ai.Provider
		lessonID string
		answers  []ai.SpeakAnswer
		field    string
		err      error
	}{
		{name: "no answers", answers: nil, field: "answers"},
		{name: "four answers", answers: said(4), field: "answers"},
		{name: "long answer", answers: []ai.SpeakAnswer{{Question: "Q?", Said: strings.Repeat("a", maxSpeakText+1)}}, field: "answers.0"},
		{name: "long question", answers: []ai.SpeakAnswer{{Question: strings.Repeat("q", maxSpeakText+1), Said: "Hi."}}, field: "answers.0"},
		{name: "missing lesson", lessonID: "missing", answers: said(1), err: ErrNotFound},
		{name: "no checker", provider: &fakeAI{}, answers: said(1), err: ai.ErrNotConfigured},
		{name: "AI fails", provider: &fakeSpeakAI{speakErr: ai.ErrQuota}, answers: said(1), err: ai.ErrQuota},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p := tt.provider
			if p == nil {
				p = &fakeSpeakAI{}
			}
			r, l := newSpeakEnv(t, p)
			id := l.ID
			if tt.lessonID != "" {
				id = tt.lessonID
			}
			_, err := r.CheckSpeaking(t.Context(), "u1", id, tt.answers, false)
			if tt.field != "" {
				var verr *ValidationError
				if !errors.As(err, &verr) || verr.Fields[tt.field] == "" {
					t.Fatalf("err = %v, want field %s", err, tt.field)
				}
				return
			}
			if !errors.Is(err, tt.err) {
				t.Fatalf("err = %v, want %v", err, tt.err)
			}
		})
	}
}

func TestCheckSpeakingLimit(t *testing.T) {
	t.Parallel()
	f := &fakeSpeakAI{}
	r, l := newSpeakEnv(t, f)
	other, _ := r.lessons.Create(t.Context(), Lesson{Title: "x", Level: "A1", Content: "Hi there."})
	answers := func() []ai.SpeakAnswer { return []ai.SpeakAnswer{{Question: "Q?", Said: "I like it."}} }

	for i := range SpeakChecksPerDay {
		if _, err := r.CheckSpeaking(t.Context(), "u1", l.ID, answers(), false); err != nil {
			t.Fatalf("check %d: %v", i+1, err)
		}
	}
	if _, err := r.CheckSpeaking(t.Context(), "u1", l.ID, answers(), false); !errors.Is(err, ErrSpeakLimit) {
		t.Fatalf("over limit: %v", err)
	}
	// Other learners, other lessons and admins are not limited.
	if _, err := r.CheckSpeaking(t.Context(), "u2", l.ID, answers(), false); err != nil {
		t.Fatalf("other user: %v", err)
	}
	if _, err := r.CheckSpeaking(t.Context(), "u1", other.ID, answers(), false); err != nil {
		t.Fatalf("other lesson: %v", err)
	}
	if _, err := r.CheckSpeaking(t.Context(), "u1", l.ID, answers(), true); err != nil {
		t.Fatalf("unlimited: %v", err)
	}
	// Empty answers cost nothing.
	if _, err := r.CheckSpeaking(t.Context(), "u1", l.ID, []ai.SpeakAnswer{{Question: "Q?"}}, false); err != nil {
		t.Fatalf("empty over limit: %v", err)
	}
	if calls, _ := f.speakCalls(); calls != SpeakChecksPerDay+3 {
		t.Fatalf("AI calls = %d", calls)
	}
}

func postSpeak(t *testing.T, mux http.Handler, lessonID, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/lessons/"+lessonID+"/speak/check", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.AddCookie(&http.Cookie{Name: httpx.SessionCookieName, Value: token})
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestCheckSpeakingEndpoint(t *testing.T) {
	t.Parallel()
	f := &fakeSpeakAI{feedback: []ai.SpeakFeedback{{Index: 0, Corrected: "I like music.", NoteVi: "Thiếu chủ ngữ."}}}
	r, l := newSpeakEnv(t, f)
	mux := http.NewServeMux()
	NewReadingHandler(r, slog.New(slog.DiscardHandler)).Register(mux, httpx.RequireAuth(resolver), allowAll)
	body := `{"answers":[{"question":"What do you like?","said":"like music"},{"question":"Q?","said":""}]}`

	rec := postSpeak(t, mux, l.ID, "learner", body)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d %s", rec.Code, rec.Body)
	}
	var out struct {
		Results []map[string]string `json:"results"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Results) != 2 || out.Results[0]["said"] != "like music" || out.Results[0]["corrected"] != "I like music." ||
		out.Results[0]["noteVi"] != "Thiếu chủ ngữ." || out.Results[1]["corrected"] != "" {
		t.Fatalf("results = %+v", out.Results)
	}

	if rec := postSpeak(t, mux, l.ID, "learner", `{"answers":[]}`); rec.Code != http.StatusBadRequest {
		t.Errorf("no answers: %d", rec.Code)
	}
	if rec := postSpeak(t, mux, "missing", "learner", body); rec.Code != http.StatusNotFound {
		t.Errorf("missing lesson: %d", rec.Code)
	}
	if rec := postSpeak(t, mux, l.ID, "", body); rec.Code != http.StatusUnauthorized {
		t.Errorf("anonymous: %d", rec.Code)
	}
	// The learner reaches the limit; the admin does not.
	for range SpeakChecksPerDay - 1 {
		postSpeak(t, mux, l.ID, "learner", body)
	}
	if rec := postSpeak(t, mux, l.ID, "learner", body); rec.Code != http.StatusTooManyRequests || !strings.Contains(rec.Body.String(), "speak_limit") {
		t.Errorf("limit: %d %s", rec.Code, rec.Body)
	}
	for range SpeakChecksPerDay + 1 {
		if rec := postSpeak(t, mux, l.ID, "admin", body); rec.Code != http.StatusOK {
			t.Fatalf("admin: %d", rec.Code)
		}
	}
}

func TestCheckSpeakingEndpointErrors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		provider ai.Provider
		code     int
		errCode  string
	}{
		{name: "no checker", provider: &fakeAI{}, code: http.StatusServiceUnavailable, errCode: "ai_not_configured"},
		{name: "invalid key", provider: &fakeSpeakAI{speakErr: ai.ErrInvalidKey}, code: http.StatusServiceUnavailable, errCode: "ai_not_configured"},
		{name: "quota", provider: &fakeSpeakAI{speakErr: ai.ErrQuota}, code: http.StatusTooManyRequests, errCode: "ai_quota"},
		{name: "other", provider: &fakeSpeakAI{speakErr: errors.New("boom")}, code: http.StatusBadGateway, errCode: "ai_failed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r, l := newSpeakEnv(t, tt.provider)
			mux := http.NewServeMux()
			NewReadingHandler(r, slog.New(slog.DiscardHandler)).Register(mux, httpx.RequireAuth(resolver), allowAll)
			rec := postSpeak(t, mux, l.ID, "learner", `{"answers":[{"question":"Q?","said":"I like it."}]}`)
			if rec.Code != tt.code || !strings.Contains(rec.Body.String(), tt.errCode) {
				t.Fatalf("status %d %s", rec.Code, rec.Body)
			}
		})
	}
}
