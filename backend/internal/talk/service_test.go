package talk

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/luongtran/luna/backend/internal/ai"
)

type fakeRepo struct {
	sessions map[string]Session
	next     int
}

func (r *fakeRepo) Create(_ context.Context, s Session) (Session, error) {
	r.next++
	s.ID = fmt.Sprintf("s%d", r.next)
	r.sessions[s.ID] = s
	return s, nil
}

func (r *fakeRepo) Get(_ context.Context, id string) (Session, error) {
	s, ok := r.sessions[id]
	if !ok {
		return Session{}, ErrNotFound
	}
	return s, nil
}

func (r *fakeRepo) Save(_ context.Context, s Session) error {
	r.sessions[s.ID] = s
	return nil
}

func (r *fakeRepo) List(_ context.Context, userID string, limit int) ([]Session, error) {
	var out []Session
	for _, s := range r.sessions {
		if s.UserID == userID {
			out = append(out, s)
		}
	}
	slices.SortFunc(out, func(a, b Session) int { return b.CreatedAt.Compare(a.CreatedAt) })
	return out[:min(limit, len(out))], nil
}

type fakeStudies struct {
	lessons  []StudiedLesson
	material Material
	asked    []string
}

func (f *fakeStudies) Studied(context.Context, string) ([]StudiedLesson, error) { return f.lessons, nil }

func (f *fakeStudies) Material(_ context.Context, ids []string) (Material, error) {
	f.asked = ids
	return f.material, nil
}

type fakeAI struct {
	talks   []ai.TalkRequest
	reviews []ai.TalkReviewRequest
	reply   ai.TalkReply
	review  ai.TalkReview
	err     error
}

func (f *fakeAI) Talk(_ context.Context, req ai.TalkRequest) (ai.TalkReply, error) {
	f.talks = append(f.talks, req)
	return f.reply, f.err
}

func (f *fakeAI) ReviewTalk(_ context.Context, req ai.TalkReviewRequest) (ai.TalkReview, error) {
	f.reviews = append(f.reviews, req)
	return f.review, f.err
}

type env struct {
	svc     *Service
	repo    *fakeRepo
	studies *fakeStudies
	ai      *fakeAI
}

func newEnv(t *testing.T) *env {
	t.Helper()
	e := &env{
		repo: &fakeRepo{sessions: map[string]Session{}},
		studies: &fakeStudies{
			lessons: []StudiedLesson{
				{ID: "l3", Level: "A2", TopicID: "music", TopicName: "Âm nhạc"},
				{ID: "l9", Level: "A1", TopicID: "family", TopicName: "Gia đình"},
				{ID: "l1", Level: "A1", TopicID: "music", TopicName: "Âm nhạc"},
				{ID: "lx", Level: "A1"}, // in no topic
			},
			material: Material{Words: []string{"song", "Song", " guitar ", "listen to"}, Grammar: []string{"can (S + can + V)"}},
		},
		ai: &fakeAI{reply: ai.TalkReply{
			ScenarioVi: "Bạn gặp Sam ở buổi hoà nhạc.", Reply: "Hi! Do you  like\nthis band?", ReplyVi: "Chào!", Reaction: "happy",
			Hints: []ai.TalkHint{
				{Text: "Yes, I love them.", Vi: "Có."}, {Text: " yes, i love them "}, {Text: ""}, {Text: "No, not really."},
			},
		}},
	}
	e.svc = NewService(Deps{Repo: e.repo, Studies: e.studies, AI: e.ai, Now: func() time.Time {
		return time.Date(2026, 10, 20, 9, 0, 0, 0, time.UTC)
	}})
	return e
}

func TestTopicsGroupTheStudiedLessons(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	got, err := e.svc.Topics(t.Context(), "u1")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].ID != "music" || got[0].Lessons != 2 || got[0].Level != "A2" || got[1].ID != "family" {
		t.Fatalf("topics = %+v", got)
	}
}

func TestStartAsksTheAIForASituationAboutWhatWasStudied(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s, err := e.svc.Start(t.Context(), "u1", "music")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(e.studies.asked, []string{"l3", "l1"}) {
		t.Fatalf("material of %v", e.studies.asked)
	}
	req := e.ai.talks[0]
	if len(req.Turns) != 0 || req.Topic.Name != "Âm nhạc" || req.Topic.Level != "A2" ||
		!slices.Equal(req.Topic.Words, []string{"song", "guitar", "listen to"}) {
		t.Fatalf("request = %+v", req)
	}
	if s.ID == "" || s.Status != StatusOpen || s.ScenarioVi != "Bạn gặp Sam ở buổi hoà nhạc." ||
		len(s.Turns) != 1 || s.Turns[0].Text != "Hi! Do you like this band?" || s.Turns[0].Reaction != "happy" ||
		len(s.Hints) != 2 || s.Hints[0] != (Hint{Text: "Yes, I love them.", Vi: "Có."}) || s.Hints[1].Text != "No, not really." {
		t.Fatalf("session = %+v", s)
	}
	if _, err := e.svc.Start(t.Context(), "u1", "sport"); !errors.Is(err, ErrTopicNotStudied) {
		t.Fatalf("not studied: %v", err)
	}
}

func TestSaySendsTheWholeConversation(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s, _ := e.svc.Start(t.Context(), "u1", "music")
	e.ai.reply = ai.TalkReply{
		Reply: "Cool! Who is your favourite singer?", ReplyVi: "Hay!", Reaction: "angry",
		Hints: []ai.TalkHint{{Text: "I like Adele."}},
	}
	s, err := e.svc.Say(t.Context(), "u1", s.ID, " Yes I like  them ", true)
	if err != nil {
		t.Fatal(err)
	}
	req := e.ai.talks[1]
	if req.ScenarioVi == "" || len(req.Turns) != 2 || req.Turns[1] != (ai.TalkTurn{Role: "learner", Text: "Yes I like them"}) || req.Last {
		t.Fatalf("request = %+v", req)
	}
	if len(s.Turns) != 3 || !s.Turns[1].Spoken || s.Turns[2].Text != "Cool! Who is your favourite singer?" ||
		s.Turns[2].Reaction != "" || len(s.Hints) != 1 || s.Hints[0].Text != "I like Adele." {
		t.Fatalf("session = %+v", s)
	}
	if _, err := e.svc.Say(t.Context(), "u2", s.ID, "hi", false); !errors.Is(err, ErrNotFound) {
		t.Fatalf("another user: %v", err)
	}
	var ve *ValidationError
	if _, err := e.svc.Say(t.Context(), "u1", s.ID, "  ", false); !errors.As(err, &ve) {
		t.Fatalf("empty: %v", err)
	}
}

func TestAFailedReplyStoresNothing(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s, _ := e.svc.Start(t.Context(), "u1", "music")
	e.ai.err = ai.ErrQuota
	if _, err := e.svc.Say(t.Context(), "u1", s.ID, "Yes", false); !errors.Is(err, ai.ErrQuota) || !IsAIError(err) {
		t.Fatalf("err = %v", err)
	}
	if got := e.repo.sessions[s.ID]; len(got.Turns) != 1 {
		t.Fatalf("turns stored: %+v", got.Turns)
	}
	e.ai.err, e.ai.reply = nil, ai.TalkReply{Reply: "  "}
	if _, err := e.svc.Say(t.Context(), "u1", s.ID, "Yes", false); !errors.Is(err, ErrUnusableReply) {
		t.Fatalf("unusable: %v", err)
	}
}

func TestTheLastTurnSaysGoodbye(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s, _ := e.svc.Start(t.Context(), "u1", "music")
	var err error
	for i := range MaxTurns {
		if s, err = e.svc.Say(t.Context(), "u1", s.ID, fmt.Sprintf("line %d", i), false); err != nil {
			t.Fatal(err)
		}
	}
	if last := e.ai.talks[len(e.ai.talks)-1]; !last.Last || len(s.Hints) != 0 {
		t.Fatalf("last request = %+v, hints %v", last.Last, s.Hints)
	}
	if _, err := e.svc.Say(t.Context(), "u1", s.ID, "more", false); !errors.Is(err, ErrTurnLimit) {
		t.Fatalf("over the limit: %v", err)
	}
}

func TestEndReviewsTheLearnersLines(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s, _ := e.svc.Start(t.Context(), "u1", "music")
	s, _ = e.svc.Say(t.Context(), "u1", s.ID, "I very like song", true)
	e.ai.review = ai.TalkReview{
		OverallVi: "Tốt lắm.",
		Fixes: []ai.TalkFix{
			{Said: "I very like song", Better: "I really like this song.", NoteVi: "Không nói very like."},
			{Said: "Yes.", Better: "yes", NoteVi: "same"},
		},
		Good:      []ai.TalkGood{{Said: "I very like song.", NoteVi: "fixed, so not good"}, {Said: "Hi", NoteVi: "ok"}},
		WordsUsed: []string{"Song", "piano"},
	}
	s, err := e.svc.End(t.Context(), "u1", s.ID)
	if err != nil {
		t.Fatal(err)
	}
	r := s.Review
	if s.Status != StatusEnded || s.EndedAt.IsZero() || r == nil || len(r.Fixes) != 1 || len(r.Good) != 1 || r.Good[0].Said != "Hi" ||
		!slices.Equal(r.WordsUsed, []string{"song"}) || !slices.Equal(r.WordsUnused, []string{"guitar", "listen to"}) {
		t.Fatalf("session = %+v, review = %+v", s, r)
	}
	if _, err := e.svc.Say(t.Context(), "u1", s.ID, "hi", false); !errors.Is(err, ErrEnded) {
		t.Fatalf("after the end: %v", err)
	}
	if _, err := e.svc.End(t.Context(), "u1", s.ID); err != nil || len(e.ai.reviews) != 1 {
		t.Fatalf("ending again: %v, %d reviews", err, len(e.ai.reviews))
	}
}

func TestEndingWithoutALineNeedsNoReview(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s, _ := e.svc.Start(t.Context(), "u1", "music")
	s, err := e.svc.End(t.Context(), "u1", s.ID)
	if err != nil || s.Status != StatusEnded || s.Review != nil || len(e.ai.reviews) != 0 {
		t.Fatalf("session = %+v, err %v", s, err)
	}
}

func TestAFailedReviewKeepsTheSessionOpen(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s, _ := e.svc.Start(t.Context(), "u1", "music")
	s, _ = e.svc.Say(t.Context(), "u1", s.ID, "Yes", false)
	e.ai.err = errors.New("boom")
	if _, err := e.svc.End(t.Context(), "u1", s.ID); !IsAIError(err) {
		t.Fatalf("err = %v", err)
	}
	if e.repo.sessions[s.ID].Status != StatusOpen {
		t.Fatal("session ended without a review")
	}
}
