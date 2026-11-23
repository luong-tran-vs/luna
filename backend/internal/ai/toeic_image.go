package ai

import (
	"fmt"
	"strings"
)

// PhotoBrief describes a TOEIC Part 1 photo (F27): the scene and the one statement that must be true
// of it. The wrong statements are never sent, so the photo cannot match one of them by chance.
type PhotoBrief struct {
	Scene     string
	Statement string
}

// DocPictureBrief describes the picture printed with a TOEIC Part 6/7 text (an ad, a web page, a
// notice...): what the text is about, so the picture fits it without giving an answer away.
type DocPictureBrief struct {
	// Kind is the kind of text (ad, web, notice, article...), Title its title.
	Kind  string
	Title string
	// Text is the text itself; it is cut to maxDocPictureText runes.
	Text string
}

// maxDocPictureText bounds the text sent with a Part 6/7 picture.
const maxDocPictureText = 1200

// ImageOp is the op recorded for the usage of a picture request: TOEIC pictures are counted apart.
func ImageOp(req ImageRequest) string {
	if req.Photo != nil || req.DocPicture != nil {
		return "toeic_photo"
	}
	return "image"
}

// photoStyle is the camera, light and detail direction of every TOEIC picture: without it image
// models tend to give smooth, illustration-like pictures even when asked for a photo.
const photoStyle = "Photography style:\n" +
	"- Shot on a full-frame DSLR, 35mm lens, f/5.6, 1/250s, ISO 200, at eye level.\n" +
	"- Natural light, soft realistic shadows, true-to-life colours, no colour grading, no HDR look.\n" +
	"- Real human anatomy and proportions: natural skin texture with pores and slight imperfections, realistic hair, " +
	"correct hands with five fingers, natural relaxed facial expressions, not looking at the camera.\n" +
	"- Ordinary people of varied ages and backgrounds (not models), in ordinary work or everyday clothes with real fabric " +
	"folds and wrinkles.\n" +
	"- Real materials: worn surfaces, small everyday details, slight dust and wear, as in an unstaged documentary or stock photo.\n" +
	"- Sharp focus on the main subject, light natural depth of field.\n"

// photoAvoid names the looks that give an AI picture away.
const photoAvoid = "Avoid: illustration, cartoon, anime, 3D render, CGI, digital painting, plastic or waxy skin, airbrushed " +
	"faces, over-smooth surfaces, oversaturated colours, glowing light, fantasy look, distorted hands or faces.\n"

// photoPrompt describes a realistic photograph of a workplace or everyday scene, as printed in the
// TOEIC test book: the statement must be plainly true of it.
func photoPrompt(p PhotoBrief) string {
	var b strings.Builder
	b.WriteString("Create one photorealistic photograph, indistinguishable from a real photo taken by a professional " +
		"photographer, for an English listening test (TOEIC Part 1).\n")
	fmt.Fprintf(&b, "Scene: %s\n", strings.TrimSpace(p.Scene))
	if s := strings.TrimSpace(p.Statement); s != "" {
		fmt.Fprintf(&b, "This statement must be clearly and unmistakably true of the photo: %q. "+
			"Show every person, action and object it names, in the state it describes.\n", s)
	}
	b.WriteString(photoStyle)
	b.WriteString("Composition: landscape 4:3, the people or objects of the scene large and easy to see, an ordinary setting " +
		"with only a few other objects that fit it.\n")
	b.WriteString(photoAvoid)
	b.WriteString("Do not show any readable letters, words, numbers, logos or signs with writing; no border, frame or watermark.")
	return b.String()
}

// productKinds are the texts a company prints with a photo of what they offer; the other texts
// (emails, letters, memos, forms, invoices, schedules, chats) get a photo of their situation.
var productKinds = map[string]bool{"ad": true, "web": true, "article": true, "notice": true, "review": true}

// docPicturePrompt describes the small picture that goes with a Part 6/7 text: a realistic photo of
// the product, place or event of an ad, web page, article, notice or review, or of the workplace
// situation of an email, a memo, a chat... (never the document itself).
func docPicturePrompt(d DocPictureBrief) string {
	text := []rune(strings.TrimSpace(d.Text))
	if len(text) > maxDocPictureText {
		text = text[:maxDocPictureText]
	}
	kind := strings.TrimSpace(d.Kind)
	var b strings.Builder
	fmt.Fprintf(&b, "Create one photorealistic photo, like a real stock photo, to be printed next to a short English %s "+
		"in a reading test", kind)
	if t := strings.TrimSpace(d.Title); t != "" {
		fmt.Fprintf(&b, " titled %q", t)
	}
	if productKinds[kind] {
		b.WriteString(". It shows the main product, place or event of the text, as a company would put on its advertisement, " +
			"web page or notice.\n")
	} else {
		b.WriteString(". It illustrates the situation the text is about: the place, the people at work, the goods or the event " +
			"it concerns (for an order, the delivered goods; for an office move, an office being packed). Do not show the " +
			"document, the message, a screen or a phone with the text on it.\n")
	}
	if len(text) > 0 {
		fmt.Fprintf(&b, "The text: %q\n", string(text))
	}
	b.WriteString(photoStyle)
	b.WriteString("Composition: landscape 4:3, one clear subject, a simple background.\n")
	b.WriteString(photoAvoid)
	b.WriteString("Do not show any readable letters, words, numbers, prices, dates or logos, and nothing that answers a question " +
		"about the text; no border, frame or watermark.")
	return b.String()
}
