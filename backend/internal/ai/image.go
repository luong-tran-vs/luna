package ai

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// ImageRequest asks for a picture that illustrates one vocabulary word (F23).
type ImageRequest struct {
	Word      string
	MeaningVi string
	// Sentence shows the word in use; may be empty.
	Sentence string
	// Style is the admin's description of how every picture of the lesson should look; may be empty.
	Style string
	// Scene, when set, asks for the lesson's scene picture instead of a word picture (added
	// 2026-11-05): Word, MeaningVi and Sentence are then ignored.
	Scene *SceneBrief
	// Example, when set, asks for the picture of an example sentence of a grammar lesson (added
	// 2026-11-06): Word, MeaningVi, Sentence and Scene are then ignored.
	Example *ExampleBrief
}

// ExampleBrief describes the picture of one example sentence of a grammar lesson: the situation
// the sentence tells, so a learner understands it at a glance.
type ExampleBrief struct {
	// En is the English sentence, Vi its Vietnamese meaning.
	En string
	Vi string
	// Exercise says En is the description of the picture of a picture exercise, not a sentence to
	// illustrate: the learner reads the answer off the picture, so it must show it unmistakably.
	Exercise bool
	// Point is the grammar point the sentence shows, e.g. "Present continuous".
	Point string
}

// examplePrompt describes the picture of a grammar example: the people and action of the sentence,
// with what the sentence says about them (now or habitually, done or not, asked) shown by the scene,
// in the style and characters of the other pictures.
func examplePrompt(ex ExampleBrief, style string) string {
	if ex.Exercise {
		return exercisePicturePrompt(ex, style)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Create one illustration of this English example sentence for a Vietnamese learner of English: %q", strings.TrimSpace(ex.En))
	if vi := strings.TrimSpace(ex.Vi); vi != "" {
		fmt.Fprintf(&b, " (Vietnamese meaning: %q)", vi)
	}
	b.WriteString(".\n")
	if p := strings.TrimSpace(ex.Point); p != "" {
		fmt.Fprintf(&b, "The sentence teaches the grammar point %q. ", p)
	}
	b.WriteString("Show exactly the situation the sentence describes: the people or things it names, doing what it says, " +
		"so a learner who sees the picture understands the meaning of the sentence. " +
		"When the sentence is about a time (now, every day, yesterday, tomorrow), let the scene suggest it with simple visual cues " +
		"such as the light of the day, a season or an action in progress. " +
		"For a negative sentence, show what is true instead (someone pushing away the food they do not like). " +
		"For a question, show the person asking with a curious expression and the thing asked about.\n")
	fmt.Fprintf(&b, "Style: %s.\n", imageStyle(style))
	b.WriteString(characterStyle + "\n")
	b.WriteString("Composition: landscape 4:3. The main people and action centred and large enough to read as a small thumbnail, " +
		"in a simple setting with only the objects the sentence needs, on a soft, light background.\n")
	b.WriteString("Quality: neat, well-proportioned shapes, consistent line weight, colours that go well together, " +
		"a warm and cheerful mood suitable for learners of every age.\n")
	b.WriteString("Do not draw any letters, words or numbers, no speech bubbles, signs with writing, border or frame.")
	return b.String()
}

// SceneBrief describes the scene picture of a lesson: one picture of the lesson's story, shown at
// the top of the lesson and of its Reading and Listening steps.
type SceneBrief struct {
	// Title is the lesson's title.
	Title string
	// Text is the lesson's story (the reading text); it is cut to maxSceneText runes.
	Text string
	// Description is what to draw, in English, written by the admin or by whoever drafted the
	// lesson; "" lets the model choose a moment of the story.
	Description string
	// Avoid lists the questions the learner answers on the story: the picture must not show their answers.
	Avoid []string
}

// maxSceneText bounds the story sent with a scene request.
const maxSceneText = 1500

// maxSceneAvoid bounds the questions listed in a scene request.
const maxSceneAvoid = 12

// ImageAspect is the aspect ratio of the picture: a wide banner for a scene, 4:3 for a word card.
func ImageAspect(req ImageRequest) string {
	if req.Scene != nil {
		return "16:9"
	}
	return "4:3"
}

// Image is a picture returned by a provider: its media type and bytes.
type Image struct {
	MIME string
	Data []byte
}

// ImageProvider draws pictures. It is separate from Provider because it uses another model.
type ImageProvider interface {
	// GenerateImage draws one picture in one request.
	GenerateImage(ctx context.Context, req ImageRequest) (Image, error)
}

// ErrNoImageTool means the provider ran but could not draw: the Codex CLI has no image_gen tool
// in `codex exec` (added 2026-11-04).
var ErrNoImageTool = errors.New("ai: image tool not available")

// DisabledImages is the image provider used when AI_PROVIDER=none or no key is set.
type DisabledImages struct{}

// GenerateImage always returns ErrNotConfigured.
func (DisabledImages) GenerateImage(context.Context, ImageRequest) (Image, error) {
	return Image{}, ErrNotConfigured
}

// defaultImageStyle is used when the admin gave no style (F23; detailed 2026-11-04 so the pictures
// look polished rather than like clip art).
const defaultImageStyle = "a semi-flat illustration in a clean vector look: simple shapes with soft rounded edges, " +
	"gentle gradients and soft shading for depth, a harmonious pastel palette (soft lavender, peach, mint and warm cream) " +
	"with one or two richer accent colours, soft diffused lighting"

// imageStyle is the style line of a prompt: the admin's, else the default, without a final full stop
// (the prompt adds its own). The background is not part of the default: a word card has a plain one,
// the scene a setting (split 2026-11-05).
func imageStyle(style string) string {
	style = strings.TrimRight(strings.TrimSpace(style), ". ")
	if style == "" {
		return defaultImageStyle
	}
	return style
}

// mentionsBackground says an admin's style asks for a kind of background (in English or Vietnamese).
func mentionsBackground(style string) bool {
	s := strings.ToLower(style)
	return strings.Contains(s, "background") || strings.Contains(s, "nền")
}

// characterStyle keeps the people of every picture alike, so a lesson's pictures look like one set.
const characterStyle = "Any person is drawn in the same simple character style: a friendly, rounded, slightly chibi figure " +
	"with a simple face (dot eyes and a clear expression), simplified hands and plain solid-colour casual clothes."

// ImagePrompt describes one picture: the word in one sense, in the admin's style, with no text
// drawn, since a learner must recognise the thing, not read its name. It also gives the art
// direction (what to show for things, actions and ideas, the people, the composition) that keeps
// the pictures clear and alike. Instructions are stated positively where possible: image models
// tend to draw what a long list of "no ..." names. Every image provider sends it.
func ImagePrompt(req ImageRequest) string {
	if req.Scene != nil {
		return scenePrompt(*req.Scene, req.Style)
	}
	if req.Example != nil {
		return examplePrompt(*req.Example, req.Style)
	}
	style := imageStyle(req.Style)
	meaning := strings.TrimSpace(req.MeaningVi)
	var b strings.Builder
	fmt.Fprintf(&b, "Create one illustration that helps a Vietnamese learner of English remember the English word %q", req.Word)
	if meaning != "" {
		fmt.Fprintf(&b, " (Vietnamese meaning: %q)", meaning)
	}
	b.WriteString(".\n")
	switch first := FirstSense(meaning); {
	case req.Sentence != "":
		fmt.Fprintf(&b, "The word is used like this: %q. Show only this sense of the word.\n", req.Sentence)
	case first != meaning:
		fmt.Fprintf(&b, "Show only its first sense, %q.\n", first)
	}
	fmt.Fprintf(&b, "Style: %s.\n", style)
	b.WriteString("What to show: if the word is a thing, show that one thing in its most typical, instantly recognisable form. " +
		"If it is an action, show a person, an animal or an object clearly doing it at its most telling moment. " +
		"If it is a feeling, quality or idea, show a simple everyday scene that makes the meaning obvious at a glance.\n")
	b.WriteString(characterStyle + "\n")
	b.WriteString("Composition: landscape 4:3. One main subject centred, filling about two thirds of the frame, " +
		"with a bold, clear silhouette that still reads as a small thumbnail. Plain light pastel background with plenty of empty space;" +
		"at most one or two small supporting objects, only when they help the meaning.\n")
	b.WriteString("Quality: neat, well-proportioned shapes, consistent line weight, colours that go well together, " +
		"a warm and cheerful mood suitable for learners of every age.\n")
	b.WriteString("Do not draw any letters, words or numbers, and no border or frame.")
	return b.String()
}

// scenePrompt describes the scene picture of a lesson: the place and the people of its story, so a
// beginner can picture them before reading or listening, without giving away the answers to the
// questions on the story. It shares the style and characters of the word pictures.
func scenePrompt(sc SceneBrief, style string) string {
	custom := strings.TrimSpace(style) != ""
	style = imageStyle(style)
	var b strings.Builder
	b.WriteString("Create one illustration for the top of an English lesson for Vietnamese learners")
	if t := strings.TrimSpace(sc.Title); t != "" {
		fmt.Fprintf(&b, " titled %q", t)
	}
	b.WriteString(". It shows the setting of the lesson's story, so learners can picture where it happens and who is in it " +
		"before they read or listen to it.\n")
	text := strings.TrimSpace(sc.Text)
	if r := []rune(text); len(r) > maxSceneText {
		text = string(r[:maxSceneText]) + "…"
	}
	if d := strings.TrimSpace(sc.Description); d != "" {
		fmt.Fprintf(&b, "Draw this scene: %q.\n", d)
		if text != "" {
			fmt.Fprintf(&b, "The story, for context only: %q\n", text)
		}
	} else if text != "" {
		fmt.Fprintf(&b, "The story: %q\n", text)
		b.WriteString("Choose the one moment that best sums up the story and show its place and its main characters, " +
			"with their age and role in the story clear at a glance (a child looks like a child, a grandmother like a grandmother).\n")
	}
	if len(sc.Avoid) > 0 {
		b.WriteString("After looking at the picture, learners answer these questions about the story. " +
			"Keep the picture general so that it does not answer any of them: leave the details they ask about out of view, " +
			"or neutral (for example, an object whose colour is asked about appears in a plain neutral colour, or not at all).\n")
		for i, q := range sc.Avoid {
			if i == maxSceneAvoid {
				break
			}
			if q = strings.TrimSpace(q); q != "" {
				fmt.Fprintf(&b, "- %s\n", q)
			}
		}
	}
	fmt.Fprintf(&b, "Style: %s.\n", style)
	b.WriteString(characterStyle + "\n")
	b.WriteString("Composition: wide landscape 16:9 banner. The main characters in the middle ground, large enough to recognise " +
		"on a phone screen, in a simple, uncluttered setting with only a few objects that tell where the scene takes place. " +
		"Keep the outer edges calm, since the banner may be cropped slightly.\n")
	b.WriteString("Background: the full setting of the scene (the room, street or place where it happens), drawn simply " +
		"in soft, slightly muted colours so the characters stand out; never an empty plain background.")
	if custom && mentionsBackground(style) {
		b.WriteString(" Any background the style above asks for is meant for the word cards of the lesson, not for this scene.")
	}
	b.WriteString("\n")
	b.WriteString("Quality: neat, well-proportioned shapes, consistent line weight, colours that go well together, " +
		"a warm and cheerful mood suitable for learners of every age.\n")
	b.WriteString("Do not draw any letters, words or numbers, no speech bubbles, signs with writing, border or frame.")
	return b.String()
}

// FirstSense is the first of the senses in a Vietnamese meaning such as "ngôi nhà; nhà ở; gia
// đình": senses are separated by semicolons (commas join words of one sense).
func FirstSense(meaning string) string {
	first, _, _ := strings.Cut(meaning, ";")
	return strings.TrimSpace(first)
}

// exercisePicturePrompt describes the picture of a picture exercise of a grammar lesson: exactly
// what its description says, with the detail the learner reads the answer from shown unmistakably.
func exercisePicturePrompt(ex ExampleBrief, style string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Create one illustration that shows exactly this: %q.\n", strings.TrimSpace(ex.En))
	if p := strings.TrimSpace(ex.Point); p != "" {
		fmt.Fprintf(&b, "It is the picture of an exercise on the English grammar point %q: ", p)
	} else {
		b.WriteString("It is the picture of an English grammar exercise: ")
	}
	b.WriteString("the learner looks at it and picks the word that completes a sentence about it, " +
		"so the detail the sentence asks about (where something is, what someone is doing, how many there are, " +
		"which one is bigger, how someone feels) must be shown clearly and unmistakably, and nothing in the picture may suggest another answer.\n")
	fmt.Fprintf(&b, "Style: %s.\n", imageStyle(style))
	b.WriteString(characterStyle + "\n")
	b.WriteString("Composition: landscape 4:3. The people and things of the description large and centred, " +
		"in a simple setting with only the objects it needs, on a soft, light background.\n")
	b.WriteString("Quality: neat, well-proportioned shapes, consistent line weight, colours that go well together, " +
		"a warm and cheerful mood suitable for learners of every age.\n")
	b.WriteString("Do not draw any letters, words or numbers, no speech bubbles, signs with writing, border or frame.")
	return b.String()
}
