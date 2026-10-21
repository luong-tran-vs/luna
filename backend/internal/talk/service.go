package talk

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/luongtran/luna/backend/internal/ai"
)

// maxWords bounds the topic's words sent to the AI.
const maxWords = 60

// historyLimit is how many of the user's sessions are read for the topic list and the history.
const historyLimit = 200

// errAI wraps a failed AI request; the AI's own error (ai.ErrQuota…) stays visible to errors.Is.
var errAI = errors.New("talk: AI request failed")

// Deps are the collaborators of Service.
type Deps struct {
	Repo    Repository
	Studies Studies
	AI      ai.TalkProvider
	Now     func() time.Time
	Log     *slog.Logger
}

// Service runs the conversations.
type Service struct {
	d Deps
}

// NewService returns the talk service.
func NewService(d Deps) *Service {
	if d.Now == nil {
		d.Now = time.Now
	}
	return &Service{d: d}
}

// TopicChoice is a topic the learner can talk about: one where they studied a lesson.
type TopicChoice struct {
	ID    string
	Name  string
	Level string
	// Lessons is how many lessons of the topic the learner studied.
	Lessons int
	// Sessions is how many conversations they had about it, LastAt when the last one started.
	Sessions int
	LastAt   time.Time
}

// Topics lists the topics the learner studied, the most recently studied first.
func (s *Service) Topics(ctx context.Context, userID string) ([]TopicChoice, error) {
	studied, err := s.d.Studies.Studied(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("talk: studied lessons: %w", err)
	}
	out := []TopicChoice{}
	index := map[string]int{}
	for _, l := range studied {
		if l.TopicID == "" {
			continue
		}
		i, ok := index[l.TopicID]
		if !ok {
			i = len(out)
			index[l.TopicID] = i
			out = append(out, TopicChoice{ID: l.TopicID, Name: l.TopicName})
		}
		out[i].Lessons++
		out[i].Level = max(out[i].Level, l.Level)
	}
	sessions, err := s.d.Repo.List(ctx, userID, historyLimit)
	if err != nil {
		return nil, fmt.Errorf("talk: list sessions: %w", err)
	}
	for _, ss := range sessions {
		if i, ok := index[ss.TopicID]; ok {
			out[i].Sessions++
			if ss.CreatedAt.After(out[i].LastAt) {
				out[i].LastAt = ss.CreatedAt
			}
		}
	}
	return out, nil
}

// List returns the learner's sessions, newest first; with topicID only those about that topic.
func (s *Service) List(ctx context.Context, userID, topicID string) ([]Session, error) {
	sessions, err := s.d.Repo.List(ctx, userID, historyLimit)
	if err != nil {
		return nil, fmt.Errorf("talk: list sessions: %w", err)
	}
	if topicID == "" {
		return sessions, nil
	}
	return slices.DeleteFunc(sessions, func(ss Session) bool { return ss.TopicID != topicID }), nil
}

// Get returns one of the learner's sessions.
func (s *Service) Get(ctx context.Context, userID, id string) (Session, error) {
	ss, err := s.d.Repo.Get(ctx, id)
	if err != nil {
		return Session{}, err
	}
	if ss.UserID != userID {
		return Session{}, ErrNotFound
	}
	return ss, nil
}

// Start begins a conversation about a topic the learner studied: the AI picks a situation and says
// the first line. Nothing is stored when the AI fails.
func (s *Service) Start(ctx context.Context, userID, topicID string) (Session, error) {
	studied, err := s.d.Studies.Studied(ctx, userID)
	if err != nil {
		return Session{}, fmt.Errorf("talk: studied lessons: %w", err)
	}
	var ids []string
	ss := Session{UserID: userID, TopicID: topicID, Status: StatusOpen}
	for _, l := range studied {
		if l.TopicID == topicID {
			ids = append(ids, l.ID)
			ss.TopicName = cmp.Or(ss.TopicName, l.TopicName)
			ss.Level = max(ss.Level, l.Level)
		}
	}
	if topicID == "" || len(ids) == 0 {
		return Session{}, ErrTopicNotStudied
	}
	m, err := s.d.Studies.Material(ctx, ids)
	if err != nil {
		return Session{}, fmt.Errorf("talk: material: %w", err)
	}
	ss.Words = firstUnique(m.Words, maxWords)
	ss.Grammar = firstUnique(m.Grammar, 0)

	reply, err := s.d.AI.Talk(ctx, ai.TalkRequest{Topic: topicOf(ss)})
	if err != nil {
		return Session{}, fmt.Errorf("%w: %w", errAI, err)
	}
	if !usable(reply) {
		return Session{}, ErrUnusableReply
	}
	now := s.d.Now().UTC()
	ss.ScenarioVi = cmp.Or(clean(reply.ScenarioVi), "Trò chuyện về chủ đề "+ss.TopicName)
	ss.Turns = []Turn{aiTurn(reply, now)}
	ss.Hints = hintsOf(reply)
	ss.CreatedAt, ss.UpdatedAt = now, now
	return s.d.Repo.Create(ctx, ss)
}

// Say adds the learner's line and the AI's answer. After MaxTurns lines the AI says goodbye and
// nothing more can be said. When the AI fails nothing is stored, so the line can be sent again.
func (s *Service) Say(ctx context.Context, userID, id, text string, spoken bool) (Session, error) {
	text = clean(text)
	switch {
	case text == "":
		return Session{}, &ValidationError{Fields: map[string]string{"text": "Hãy nói hoặc gõ câu của bạn"}}
	case utf8.RuneCountInString(text) > MaxTextLength:
		return Session{}, &ValidationError{Fields: map[string]string{
			"text": fmt.Sprintf("Câu dài quá, tối đa %d ký tự", MaxTextLength),
		}}
	}
	ss, err := s.Get(ctx, userID, id)
	if err != nil {
		return Session{}, err
	}
	if ss.Status != StatusOpen {
		return Session{}, ErrEnded
	}
	said := ss.LearnerTurns()
	if said >= MaxTurns {
		return Session{}, ErrTurnLimit
	}
	now := s.d.Now().UTC()
	turns := append(slices.Clone(ss.Turns), Turn{Role: RoleLearner, Text: text, Spoken: spoken, At: now})
	last := said+1 == MaxTurns
	reply, err := s.d.AI.Talk(ctx, ai.TalkRequest{
		Topic: topicOf(ss), ScenarioVi: ss.ScenarioVi, Turns: aiTurns(turns), Last: last,
	})
	if err != nil {
		return Session{}, fmt.Errorf("%w: %w", errAI, err)
	}
	if !usable(reply) {
		return Session{}, ErrUnusableReply
	}
	ss.Turns = append(turns, aiTurn(reply, now))
	ss.Hints = hintsOf(reply)
	if last {
		ss.Hints = nil
	}
	ss.UpdatedAt = now
	if err := s.d.Repo.Save(ctx, ss); err != nil {
		return Session{}, fmt.Errorf("talk: save session: %w", err)
	}
	return ss, nil
}

// End finishes the conversation and asks the AI to review the learner's lines (none when they said
// nothing). Ending a session already ended returns it. When the AI fails the session stays open,
// so ending can be tried again.
func (s *Service) End(ctx context.Context, userID, id string) (Session, error) {
	ss, err := s.Get(ctx, userID, id)
	if err != nil || ss.Status == StatusEnded {
		return ss, err
	}
	if ss.LearnerTurns() > 0 {
		r, err := s.d.AI.ReviewTalk(ctx, ai.TalkReviewRequest{
			Topic: topicOf(ss), ScenarioVi: ss.ScenarioVi, Turns: aiTurns(ss.Turns),
		})
		if err != nil {
			return Session{}, fmt.Errorf("%w: %w", errAI, err)
		}
		review := checkReview(r, ss)
		ss.Review = &review
	}
	now := s.d.Now().UTC()
	ss.Status, ss.Hints = StatusEnded, nil
	ss.UpdatedAt, ss.EndedAt = now, now
	if err := s.d.Repo.Save(ctx, ss); err != nil {
		return Session{}, fmt.Errorf("talk: save session: %w", err)
	}
	return ss, nil
}

// IsAIError reports whether err came from the AI request (as opposed to the store or the input).
func IsAIError(err error) bool {
	return errors.Is(err, errAI) || errors.Is(err, ErrUnusableReply)
}

func topicOf(ss Session) ai.TalkTopic {
	return ai.TalkTopic{Name: ss.TopicName, Level: ss.Level, Words: ss.Words, Grammar: ss.Grammar}
}

func aiTurns(turns []Turn) []ai.TalkTurn {
	out := make([]ai.TalkTurn, len(turns))
	for i, t := range turns {
		out[i] = ai.TalkTurn{Role: string(t.Role), Text: t.Text}
	}
	return out
}

// aiTurn is the AI's reply as a turn, its reaction kept only when it is one of Reactions.
func aiTurn(r ai.TalkReply, at time.Time) Turn {
	t := Turn{Role: RoleAI, Text: clean(r.Reply), TextVi: clean(r.ReplyVi), At: at}
	if slices.Contains(Reactions, r.Reaction) {
		t.Reaction = r.Reaction
	}
	return t
}

// hintsOf keeps the AI's answers that have a text, each once, at most MaxHints.
func hintsOf(r ai.TalkReply) []Hint {
	out := []Hint{}
	seen := map[string]bool{}
	for _, h := range r.Hints {
		text := clean(h.Text)
		if text == "" || seen[fold(text)] {
			continue
		}
		seen[fold(text)] = true
		out = append(out, Hint{Text: text, Vi: clean(h.Vi)})
		if len(out) == MaxHints {
			break
		}
	}
	return out
}

func usable(r ai.TalkReply) bool {
	return clean(r.Reply) != ""
}

// checkReview keeps the parts of the AI's review that can be shown: fixes that change something,
// good lines that are not fixed, and only the topic's own words.
func checkReview(r ai.TalkReview, ss Session) Review {
	out := Review{OverallVi: clean(r.OverallVi), Good: []Good{}, Fixes: []Fix{}, WordsUsed: []string{}, WordsUnused: []string{}}
	fixed := map[string]bool{}
	for _, f := range r.Fixes {
		said, better := clean(f.Said), clean(f.Better)
		if said == "" || better == "" || fold(said) == fold(better) {
			continue
		}
		fixed[fold(said)] = true
		out.Fixes = append(out.Fixes, Fix{Said: said, Better: better, NoteVi: clean(f.NoteVi)})
	}
	for _, g := range r.Good {
		said := clean(g.Said)
		if said != "" && !fixed[fold(said)] {
			out.Good = append(out.Good, Good{Said: said, NoteVi: clean(g.NoteVi)})
		}
	}
	used := map[string]bool{}
	for _, w := range r.WordsUsed {
		used[strings.ToLower(clean(w))] = true
	}
	for _, w := range ss.Words {
		if used[strings.ToLower(w)] {
			out.WordsUsed = append(out.WordsUsed, w)
		} else {
			out.WordsUnused = append(out.WordsUnused, w)
		}
	}
	return out
}

// fold compares two lines without case, spacing and the closing punctuation.
func fold(s string) string {
	return strings.TrimRight(strings.ToLower(strings.Join(strings.Fields(s), " ")), ".!?")
}

// clean trims a text and joins its runs of spaces and line breaks.
func clean(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// firstUnique keeps the first occurrence of each text (case ignored), at most limit (0: all).
func firstUnique(texts []string, limit int) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, t := range texts {
		t = clean(t)
		key := strings.ToLower(t)
		if t == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, t)
		if limit > 0 && len(out) == limit {
			break
		}
	}
	return out
}
