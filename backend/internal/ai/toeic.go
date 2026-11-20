package ai

import "context"

// ToeicTag is a question type the AI may use, with what it means.
type ToeicTag struct {
	Code  string
	Label string
}

// ToeicGroupsRequest asks for new TOEIC question groups of one part (F27): the AI writes them in the
// import format of content/toeic (contracts/content-format.md); the service checks each one like an
// import and keeps the valid ones as drafts.
type ToeicGroupsRequest struct {
	Part int
	// Count is how many groups to write (Part 1-6).
	Count int
	// Questions is how many questions the groups add up to (Part 7, whose groups have 2-5).
	Questions int
	// Level is the target score of the groups (450, 600, 750 or 900).
	Level int
	// Topic is an optional hint from the admin (a setting, a grammar point...).
	Topic string
	// Tags are the question types allowed in the part.
	Tags []ToeicTag
	// Existing are short lines (titles, questions) of groups the part already has, not to be repeated.
	Existing []string
}

// ToeicGroups always returns ErrNotConfigured.
func (Disabled) ToeicGroups(context.Context, ToeicGroupsRequest) (string, error) {
	return "", ErrNotConfigured
}
