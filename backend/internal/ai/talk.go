package ai

import "context"

// Talk with the AI (added 2026-10-20): from the Ôn tập tab the learner picks a topic they studied
// and has a conversation with the AI about it. Each reply is one request with the whole
// conversation so far (the AI remembers nothing between requests); the review at the end is one
// more.

// TalkRoleAI and TalkRoleLearner are who said a turn.
const (
	TalkRoleAI      = "ai"
	TalkRoleLearner = "learner"
)

// TalkTurn is one line of the conversation.
type TalkTurn struct {
	Role string
	Text string
}

// TalkTopic is what the conversation is about: the topic and what the learner studied in it.
type TalkTopic struct {
	// Name is the topic's name, Level the CEFR level of the highest lesson studied in it.
	Name  string
	Level string
	// Words are the words and phrases of the lessons studied.
	Words []string
	// Grammar are the grammar points of those lessons ("can (S + can + V)").
	Grammar []string
}

// TalkRequest asks for the AI's next line. Without turns it starts the conversation: a new
// situation and the first line. Last asks it to close the conversation (the learner's last turn).
type TalkRequest struct {
	Topic TalkTopic
	// ScenarioVi is the situation chosen at the start (empty when starting).
	ScenarioVi string
	Turns      []TalkTurn
	Last       bool
}

// TalkReply is the AI's next line.
type TalkReply struct {
	// ScenarioVi describes the situation in Vietnamese; only when starting.
	ScenarioVi string `json:"scenarioVi"`
	Reply      string `json:"reply"`
	ReplyVi    string `json:"replyVi"`
	// Hint is an answer the learner could give, shown by Gợi ý; empty on the last line.
	Hint   string `json:"hint"`
	HintVi string `json:"hintVi"`
}

// TalkReviewRequest asks for the review of a finished conversation.
type TalkReviewRequest struct {
	Topic      TalkTopic
	ScenarioVi string
	Turns      []TalkTurn
}

// TalkGood is a learner's line that was well said.
type TalkGood struct {
	Said   string `json:"said"`
	NoteVi string `json:"noteVi"`
}

// TalkFix is a learner's line worth saying better.
type TalkFix struct {
	Said   string `json:"said"`
	Better string `json:"better"`
	NoteVi string `json:"noteVi"`
}

// TalkReview is the review of a conversation, for the learner's lines only.
type TalkReview struct {
	OverallVi string     `json:"overallVi"`
	Good      []TalkGood `json:"good"`
	Fixes     []TalkFix  `json:"fixes"`
	// WordsUsed are the topic's words the learner used, in any form.
	WordsUsed []string `json:"wordsUsed"`
}

// TalkProvider has the conversation and reviews it. It is separate from Provider so that the
// many fakes of Provider need not implement it.
type TalkProvider interface {
	Talk(ctx context.Context, req TalkRequest) (TalkReply, error)
	ReviewTalk(ctx context.Context, req TalkReviewRequest) (TalkReview, error)
}

// DisabledTalk is the talk provider used when AI_PROVIDER=none or no key is set.
type DisabledTalk struct{}

// Talk always returns ErrNotConfigured.
func (DisabledTalk) Talk(context.Context, TalkRequest) (TalkReply, error) {
	return TalkReply{}, ErrNotConfigured
}

// ReviewTalk always returns ErrNotConfigured.
func (DisabledTalk) ReviewTalk(context.Context, TalkReviewRequest) (TalkReview, error) {
	return TalkReview{}, ErrNotConfigured
}
