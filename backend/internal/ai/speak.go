package ai

import "context"

// SpeakAnswer is what a learner said aloud to one question of the Speaking step (added
// 2026-10-22), as the browser recognised it.
type SpeakAnswer struct {
	Question string
	Said     string
}

// SpeakCheckRequest asks for a light check of the learner's spoken answers, in one request.
type SpeakCheckRequest struct {
	Level   string
	Answers []SpeakAnswer
}

// SpeakFeedback is the check of one answer: Corrected is the answer with minimal corrections
// (equal to what was said when it is fine), NoteVi one Vietnamese sentence on the main mistake.
type SpeakFeedback struct {
	Index     int    `json:"index"`
	Corrected string `json:"corrected"`
	NoteVi    string `json:"noteVi"`
}

// SpeakChecker checks spoken answers. The Gemini client implements it; without it the Speaking
// step shows the answers without corrections.
type SpeakChecker interface {
	CheckSpeaking(ctx context.Context, req SpeakCheckRequest) ([]SpeakFeedback, error)
}
