// Package tts turns text into speech audio.
package tts

import "context"

// Synthesizer returns mp3 audio for one sentence.
type Synthesizer interface {
	Synthesize(ctx context.Context, text string) ([]byte, error)
}
