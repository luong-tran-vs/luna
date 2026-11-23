package ai

import (
	"fmt"
	"strings"
)

// TopicBrief describes the picture of a topic of the course (added 2026-11-16): one banner per
// topic, shown on the "Luyện nói" topics and the topic summary.
type TopicBrief struct {
	// Name is the topic's name, in Vietnamese ("Gia đình"); Description may add what it covers.
	Name        string
	Description string
	// Words are some English words of the topic, so the picture shows the topic as the lessons see it.
	Words []string
}

// maxTopicWords bounds the words sent with a topic picture.
const maxTopicWords = 12

// topicPrompt describes the picture of a topic: a simple, welcoming everyday scene that sums the
// topic up at a glance, as a real photograph like the other pictures.
func topicPrompt(t TopicBrief, style string) string {
	var b strings.Builder
	fmt.Fprintf(&b, photoOpening+"that represents the topic %q of an English course for Vietnamese learners", strings.TrimSpace(t.Name))
	if d := strings.TrimSpace(t.Description); d != "" {
		fmt.Fprintf(&b, " (the topic covers: %q)", d)
	}
	b.WriteString(". The topic name is in Vietnamese; show what it means.\n")
	if len(t.Words) > 0 {
		words := t.Words
		if len(words) > maxTopicWords {
			words = words[:maxTopicWords]
		}
		fmt.Fprintf(&b, "Some English words the topic teaches: %s. Show a few of these things or actions, not all of them.\n", strings.Join(words, ", "))
	}
	b.WriteString("Show one simple, welcoming real everyday moment that sums the topic up at a glance, " +
		"with one to three ordinary people doing something typical of it, and only the objects that make the topic clear.\n")
	b.WriteString(photoDirection(style))
	b.WriteString("Composition: wide landscape 16:9 banner. The main people and objects in the middle, large enough to read " +
		"on a phone screen, in an uncluttered real setting softly out of focus; keep the outer edges calm, since the banner may be cropped slightly.\n")
	b.WriteString(photoAvoid)
	b.WriteString("Do not show any readable letters, words or numbers, no signs with writing, border, frame or watermark.")
	return b.String()
}
