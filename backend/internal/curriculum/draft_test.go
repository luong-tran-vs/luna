package curriculum

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/luongtran/luna/backend/internal/ai"
	"github.com/luongtran/luna/backend/internal/grammar"
	"github.com/luongtran/luna/backend/internal/topic"
)

// fakeWriter answers the outlines in turn, and the lessons of every topic.
type fakeWriter struct {
	outlines [][]ai.OutlineTopic
	outReqs  []ai.OutlineRequest
	slots    []ai.SlotDraft
	slotReqs []ai.SlotsRequest
	err      error
}

func (f *fakeWriter) OutlineCurriculum(_ context.Context, req ai.OutlineRequest) ([]ai.OutlineTopic, error) {
	f.outReqs = append(f.outReqs, req)
	if f.err != nil {
		return nil, f.err
	}
	out := f.outlines[0]
	if len(f.outlines) > 1 {
		f.outlines = f.outlines[1:]
	}
	return out, nil
}

func (f *fakeWriter) WriteTopicSlots(_ context.Context, req ai.SlotsRequest) ([]ai.SlotDraft, error) {
	f.slotReqs = append(f.slotReqs, req)
	if f.err != nil {
		return nil, f.err
	}
	out := slices.Clone(f.slots)
	for i := range out {
		out[i].Title = fmt.Sprintf("%s %s", req.Topic.Name, out[i].Title)
	}
	return out, nil
}

// a2Outline is n A2 topics, the first named as a catalogue topic.
func a2Outline(n int) []ai.OutlineTopic {
	pts := grammar.Default().ByLevel("A2")
	out := make([]ai.OutlineTopic, n)
	for i := range out {
		out[i] = ai.OutlineTopic{Name: fmt.Sprintf("Chủ đề %d", i+1), CanDoVi: "kể chuyện.", Grammar: []string{pts[i%len(pts)].ID}}
	}
	out[0].Name = " âm nhạc "
	return out
}

func eightSlots() []ai.SlotDraft {
	out := make([]ai.SlotDraft, 8)
	for i := range out {
		out[i] = ai.SlotDraft{Title: fmt.Sprintf("Lesson %d", i+1), Idea: "Two friends talk.", Grammar: []string{"a2-past-simple-regular"}}
	}
	return out
}

func TestSeedIsSavedOnce(t *testing.T) {
	s, _, _ := newService()
	repo := s.Repo.(*fakeRepo)
	for range 2 {
		if _, err := s.Board(context.Background(), "A1"); err != nil {
			t.Fatal(err)
		}
	}
	if repo.saves != 1 || repo.courses["A1"].Approved == nil || repo.courses["A1"].ApprovedAt.IsZero() {
		t.Fatalf("saves %d, course %+v", repo.saves, repo.courses["A1"])
	}
	b, _ := s.Board(context.Background(), "A1")
	if !slices.ContainsFunc(b.Warnings, func(w string) bool { return strings.Contains(w, "bài thêm, nên 4–5") }) {
		t.Errorf("A1 has 2–3 extra lessons, warnings %v", b.Warnings)
	}
}

func TestDraftWithAI(t *testing.T) {
	s, _, _ := newService()
	ctx := context.Background()
	if _, err := s.DraftWithAI(ctx, "A2"); !errors.Is(err, ErrNoAI) {
		t.Fatalf("no writer: %v", err)
	}
	// The first outline has too few topics and a point of another level: it is sent back once.
	bad := a2Outline(5)
	bad[1].Grammar = []string{"a1-to-be"}
	w := &fakeWriter{outlines: [][]ai.OutlineTopic{bad, a2Outline(7)}}
	s.Writer = w
	b, err := s.DraftWithAI(ctx, "A2")
	if err != nil {
		t.Fatal(err)
	}
	if len(w.outReqs) != 2 || len(w.outReqs[1].PreviousErrors) != 2 {
		t.Fatalf("requests %d, errors of the 2nd %v", len(w.outReqs), w.outReqs[len(w.outReqs)-1].PreviousErrors)
	}
	req := w.outReqs[0]
	if len(req.Points) != len(grammar.Default().ByLevel("A2")) || len(req.Suggestions) == 0 || !slices.Contains(req.Topics, "Âm nhạc") {
		t.Errorf("request %+v", req)
	}
	if !slices.Contains(req.Earlier, "A1: Âm nhạc") {
		t.Errorf("earlier topics %v", req.Earlier)
	}
	if b.Status != StatusDraft || b.HasApproved || len(b.Topics) != 7 || b.Topics[0].Name != "Âm nhạc" || b.Topics[0].TopicID != "music" {
		t.Fatalf("board %s %v, topic 1 %+v", b.Status, b.HasApproved, b.Topics[0].Topic)
	}
	// Learners never see a draft.
	if _, _, ok := s.PointTopics(ctx, "A2"); ok {
		t.Error("a draft has no point topics")
	}
	// An outline still wrong is not saved.
	w.outlines = [][]ai.OutlineTopic{a2Outline(3)}
	var perr *PlanError
	if _, err := s.DraftWithAI(ctx, "B1"); !errors.As(err, &perr) {
		t.Errorf("wrong twice: %v", err)
	}
	w.err = ai.ErrQuota
	if _, err := s.DraftWithAI(ctx, "B1"); !errors.Is(err, ErrAIFailed) || !errors.Is(err, ai.ErrQuota) {
		t.Errorf("AI error: %v", err)
	}
}

func TestWriteTopicAndApprove(t *testing.T) {
	s, ft, _ := newService()
	ctx := context.Background()
	w := &fakeWriter{outlines: [][]ai.OutlineTopic{a2Outline(7)}, slots: eightSlots()}
	s.Writer = w
	if _, err := s.WriteTopic(ctx, "A2", 1, false); !errors.Is(err, ErrNoDraftPlan) {
		t.Fatalf("no draft: %v", err)
	}
	if _, err := s.DraftWithAI(ctx, "A2"); err != nil {
		t.Fatal(err)
	}
	var perr *PlanError
	if _, err := s.Approve(ctx, "A2"); !errors.As(err, &perr) {
		t.Fatalf("topics without lessons cannot be approved: %v", err)
	}
	// The AI repeats a title, uses a B1 point and forgets an idea: those are dropped or fixed.
	w.slots[1].Title = "lesson 1"
	w.slots[2].Grammar = []string{"b1-present-perfect-continuous"}
	w.slots[3].Idea = ""
	b, err := s.WriteTopic(ctx, "A2", 2, false)
	if err != nil {
		t.Fatal(err)
	}
	got := b.Topics[1].Slots
	if len(got) != 6 || got[0].Key != "a2-02-1" || got[2].Kind != KindMain || got[3].Kind != KindExtra {
		t.Fatalf("slots %+v", got)
	}
	if !slices.Equal(got[1].Slot.Grammar, []string{b.Topics[1].Topic.Grammar[0]}) {
		t.Errorf("a lesson left without an A2 point takes the topic's: %v", got[1].Slot.Grammar)
	}
	req := w.slotReqs[0]
	if req.Main != MainPerTopic || req.Extra != MaxExtra || !slices.Contains(req.Avoid, "A Concert in the Park") {
		t.Errorf("request main %d extra %d, avoid %d titles", req.Main, req.Extra, len(req.Avoid))
	}
	w.slots = eightSlots()
	for n := 1; n <= 7; n++ {
		if _, err := s.WriteTopic(ctx, "A2", n, false); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.WriteTopic(ctx, "A2", 8, false); !errors.Is(err, ErrSlotNotFound) {
		t.Errorf("topic 8: %v", err)
	}
	before := len(ft.topics)
	b, err = s.Approve(ctx, "A2")
	if err != nil {
		t.Fatal(err)
	}
	if b.Status != StatusApproved || !b.HasApproved || len(ft.topics) != before+6 {
		t.Fatalf("approved %s, %d topics (was %d)", b.Status, len(ft.topics), before)
	}
	if b.Topics[0].TopicID != "music" || b.Topics[1].TopicID == "" {
		t.Errorf("topic ids %q %q", b.Topics[0].TopicID, b.Topics[1].TopicID)
	}
	if info, ok := s.LearnerTopic(ctx, "A2", b.Topics[1].TopicID, "renamed"); !ok || info.Number != 2 || len(info.Extra) != 5 {
		t.Errorf("learner topic by id: %+v %v", info, ok)
	}
	if _, err := s.GenerateSlot(ctx, "a2-02-1"); !errors.Is(err, ErrNoDraft) {
		t.Errorf("an approved slot can be generated (the fake AI gives no draft): %v", err)
	}
}

func TestSaveEditDiscard(t *testing.T) {
	s, _, _ := newService()
	ctx := context.Background()
	if _, err := s.SaveDraft(ctx, "A1", Plan{}); !errors.Is(err, ErrNoDraftPlan) {
		t.Fatalf("no draft yet: %v", err)
	}
	b, err := s.EditApproved(ctx, "A1")
	if err != nil || b.Status != StatusDraft || !b.HasApproved || len(b.Topics) != 12 {
		t.Fatalf("edit: %s %v %v", b.Status, b.HasApproved, err)
	}
	// The admin moves topic 2 first and renames a lesson.
	p := Plan{}
	for _, tv := range b.Topics {
		p.Topics = append(p.Topics, tv.Topic)
	}
	p.Topics[0], p.Topics[1] = p.Topics[1], p.Topics[0]
	p.Topics[0].Slots[0].Title = "Meet My Family"
	b, err = s.SaveDraft(ctx, "A1", p)
	if err != nil {
		t.Fatal(err)
	}
	if b.Topics[0].Name != "Gia đình" || b.Topics[0].Slots[0].Key != "a1-01-1" || b.Topics[0].Slots[0].Title != "Meet My Family" {
		t.Errorf("saved %+v", b.Topics[0].Slots[0].Slot)
	}
	// Learners still see the approved course.
	if info, ok := s.LearnerTopic(ctx, "A1", "", "Gia đình"); !ok || info.Number != 2 {
		t.Errorf("approved number %+v", info)
	}
	p.Topics[3].Slots[1].Title = ""
	var perr *PlanError
	if _, err := s.SaveDraft(ctx, "A1", p); !errors.As(err, &perr) || !strings.Contains(perr.Message, "chưa có tên") {
		t.Errorf("empty title: %v", err)
	}
	b, err = s.DiscardDraft(ctx, "A1")
	if err != nil || b.Status != StatusApproved || b.Topics[0].Name != "Chào hỏi và giới thiệu bản thân" {
		t.Errorf("discard: %s %v", b.Status, err)
	}
	if _, err := s.DiscardDraft(ctx, "A1"); !errors.Is(err, ErrNoDraftPlan) {
		t.Errorf("discard twice: %v", err)
	}
	if _, err := s.EditApproved(ctx, "B2"); !errors.Is(err, ErrNoPlan) {
		t.Errorf("B2 has nothing to edit: %v", err)
	}
}

func TestWriteTopicExtraOnly(t *testing.T) {
	s, _, _ := newService()
	ctx := context.Background()
	w := &fakeWriter{slots: eightSlots()[:3]}
	s.Writer = w
	if _, err := s.EditApproved(ctx, "A1"); err != nil {
		t.Fatal(err)
	}
	// Topic 9 (Âm nhạc) has 3 main and 2 extra lessons; its lessons stay, 3 extra ones are added.
	b, err := s.WriteTopic(ctx, "A1", 9, true)
	if err != nil {
		t.Fatal(err)
	}
	got := b.Topics[8].Slots
	if len(got) != 8 || got[0].Title != "My Favorite Song" || got[5].Title != "Âm nhạc Lesson 1" || got[7].Kind != KindExtra {
		t.Fatalf("slots %d: %+v", len(got), got)
	}
	req := w.slotReqs[0]
	if req.Main != 0 || req.Extra != 3 || !slices.Contains(req.Avoid, "My Favorite Song") {
		t.Errorf("request main %d extra %d avoid %v", req.Main, req.Extra, req.Avoid)
	}
	// The slot with a lesson keeps it.
	if got[1].Lesson == nil {
		t.Error("the draft lesson still fills its slot")
	}
	var perr *PlanError
	if _, err := s.WriteTopic(ctx, "A1", 9, true); !errors.As(err, &perr) {
		t.Errorf("a full topic: %v", err)
	}
}

func TestSummaryAndWords(t *testing.T) {
	s, ft, _ := newService()
	ctx := context.Background()
	ft.find("music").Words = []topic.Word{{Text: "song"}, {Text: "piano", Level: "A1"}, {Text: "orchestra", Level: "B1"}}
	b, err := s.Board(ctx, "A1")
	if err != nil {
		t.Fatal(err)
	}
	if b.Topics[8].Words != 2 {
		t.Errorf("music has %d A1 words, want 2", b.Topics[8].Words)
	}
	if !slices.ContainsFunc(b.Warnings, func(w string) bool { return strings.Contains(w, "“Âm nhạc” mới có 2 từ vựng") }) {
		t.Errorf("warnings %v", b.Warnings)
	}
	if _, err := s.EditApproved(ctx, "A1"); err != nil {
		t.Fatal(err)
	}
	s.Writer = &fakeWriter{outlines: [][]ai.OutlineTopic{a2Outline(7)}}
	if _, err := s.DraftWithAI(ctx, "A2"); err != nil {
		t.Fatal(err)
	}
	sum, err := s.Summary(ctx)
	if err != nil {
		t.Fatal(err)
	}
	want := []LevelSummary{
		{Level: "A1", Status: StatusApproved, HasDraft: true, Applied: false, Empty: 62, Drafts: 1},
		{Level: "A2", Status: StatusDraft, HasDraft: true},
	}
	if !slices.Equal(sum, want) {
		t.Errorf("summary\n got %+v\nwant %+v", sum, want)
	}
}
