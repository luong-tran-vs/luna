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
	// Topic, when set, asks for the banner of a topic of the course (added 2026-11-16): Word, MeaningVi,
	// Sentence, Scene and Example are then ignored.
	Topic *TopicBrief
	// Photo, when set, asks for a TOEIC Part 1 photo (F27): every other field is then ignored.
	Photo *PhotoBrief
	// DocPicture, when set, asks for the picture of a TOEIC Part 6/7 text (F27): every other field is
	// then ignored.
	DocPicture *DocPictureBrief
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
// as a real photograph (changed 2026-11-23 from the illustration style of the word pictures).
func examplePrompt(ex ExampleBrief) string {
	if ex.Exercise {
		return exercisePicturePrompt(ex)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Create one photorealistic photograph, indistinguishable from a real photo taken by a professional photographer, "+
		"of this English example sentence for a Vietnamese learner of English: %q", strings.TrimSpace(ex.En))
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
		"such as the light of the day, the weather, a season or an action in progress. " +
		"For a negative sentence, show what is true instead (someone pushing away the food they do not like). " +
		"For a question, show the person asking with a curious expression and the thing asked about.\n")
	b.WriteString(photoStyle)
	b.WriteString("Composition: landscape 4:3. The main people and action centred and large enough to read as a small thumbnail, " +
		"in an ordinary real setting with only the objects the sentence needs, a friendly everyday mood.\n")
	b.WriteString(photoAvoid)
	b.WriteString("Do not show any readable letters, words or numbers, no speech bubbles, signs with writing, border, frame or watermark.")
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

// MaxImageBatch is the most pictures one background batch draws (word bank, irregular verbs,
// grammar examples), so a click cannot run up the bill; the rest are drawn by the next batch.
// Lesson scenes have their own bound (lesson.MaxSceneBatch).
const MaxImageBatch = 200

// maxSceneText bounds the story sent with a scene request.
const maxSceneText = 1500

// maxSceneAvoid bounds the questions listed in a scene request.
const maxSceneAvoid = 12

// ImageAspect is the aspect ratio of the picture: a wide banner for a scene, 4:3 for a word card.
func ImageAspect(req ImageRequest) string {
	if req.Scene != nil || req.Topic != nil {
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

// photoOpening starts every picture prompt (changed 2026-11-23: every picture of the app is a
// photorealistic photograph, the earlier illustration style is gone).
const photoOpening = "Create one photorealistic photograph, indistinguishable from a real photo taken by a professional photographer, "

// photoDirection is the camera and light direction of a picture, with the admin's style, when given,
// as an extra direction that keeps the picture a real photograph (the admin's style used to replace
// the illustration style; changed 2026-11-23).
func photoDirection(style string) string {
	style = strings.TrimRight(strings.TrimSpace(style), ". ")
	if style == "" {
		return photoStyle
	}
	return photoStyle + fmt.Sprintf("- Extra direction from the course author, followed only as far as the picture stays a real "+
		"photograph (never a drawing or render): %s.\n", style)
}

// ImagePrompt describes one picture: the word in one sense, as a real photograph, with no text
// shown, since a learner must recognise the thing, not read its name. It also gives the direction
// (what to show for things, actions and ideas, the composition) that keeps the pictures clear and
// alike. Every image provider sends it.
func ImagePrompt(req ImageRequest) string {
	if req.Photo != nil {
		return photoPrompt(*req.Photo)
	}
	if req.DocPicture != nil {
		return docPicturePrompt(*req.DocPicture)
	}
	if req.Topic != nil {
		return topicPrompt(*req.Topic, req.Style)
	}
	if req.Scene != nil {
		return scenePrompt(*req.Scene, req.Style)
	}
	if req.Example != nil {
		return examplePrompt(*req.Example)
	}
	if numeral, ok := NumberWord(req.Word); ok {
		return numberPrompt(req.Word, numeral, req.Style)
	}
	meaning := strings.TrimSpace(req.MeaningVi)
	var b strings.Builder
	fmt.Fprintf(&b, photoOpening+"that helps a Vietnamese learner of English remember the English word %q", req.Word)
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
	b.WriteString("What to show: if the word is a thing, photograph one real example of it in its most typical, instantly " +
		"recognisable form, like a clean product or stock photo. If it is an action, show a real person, an animal or an object " +
		"clearly doing it at its most telling moment. If it is a feeling, quality or idea, show a real everyday moment that makes " +
		"the meaning obvious at a glance.\n")
	b.WriteString(photoDirection(req.Style))
	b.WriteString("Composition: landscape 4:3. One main subject centred, filling about two thirds of the frame, clearly separated " +
		"from the background so it still reads as a small thumbnail. A plain, softly lit, light neutral background (a seamless " +
		"studio backdrop for a thing, a softly out-of-focus real setting for a person); at most one or two supporting objects, " +
		"only when they help the meaning.\n")
	b.WriteString(photoAvoid)
	b.WriteString("Do not show any readable letters, words or numbers, no border, frame or watermark.")
	return b.String()
}

// scenePrompt describes the scene picture of a lesson: the place and the people of its story, so a
// beginner can picture them before reading or listening, without giving away the answers to the
// questions on the story. Like the word pictures, it is a real photograph.
func scenePrompt(sc SceneBrief, style string) string {
	var b strings.Builder
	b.WriteString(photoOpening + "for the top of an English lesson for Vietnamese learners")
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
		fmt.Fprintf(&b, "Show this scene: %q.\n", d)
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
	b.WriteString(photoDirection(style))
	b.WriteString("Composition: wide landscape 16:9 banner. The main people in the middle ground, large enough to recognise " +
		"on a phone screen, in an uncluttered real setting with only a few objects that tell where the scene takes place. " +
		"Keep the outer edges calm, since the banner may be cropped slightly.\n")
	b.WriteString("Background: the full real setting of the scene (the room, street or place where it happens), slightly out of " +
		"focus so the people stand out; never an empty plain background.\n")
	b.WriteString(photoAvoid)
	b.WriteString("Do not show any readable letters, words or numbers, no speech bubbles, signs with writing, border, frame or watermark.")
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
func exercisePicturePrompt(ex ExampleBrief) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Create one photorealistic photograph, indistinguishable from a real photo taken by a professional "+
		"photographer, that shows exactly this: %q.\n", strings.TrimSpace(ex.En))
	if p := strings.TrimSpace(ex.Point); p != "" {
		fmt.Fprintf(&b, "It is the picture of an exercise on the English grammar point %q: ", p)
	} else {
		b.WriteString("It is the picture of an English grammar exercise: ")
	}
	b.WriteString("the learner looks at it and picks the word that completes a sentence about it, " +
		"so the detail the sentence asks about (where something is, what someone is doing, how many there are, " +
		"which one is bigger, how someone feels) must be shown clearly and unmistakably, and nothing in the picture may suggest another answer.\n")
	b.WriteString(photoStyle)
	b.WriteString("Composition: landscape 4:3. The people and things of the description large and centred, " +
		"in an ordinary real setting with only the objects it needs.\n")
	b.WriteString(photoAvoid)
	b.WriteString("Do not show any readable letters, words or numbers, no speech bubbles, signs with writing, border, frame or watermark.")
	return b.String()
}
