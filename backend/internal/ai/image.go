package ai

import "context"

// ImageRequest asks for a picture that illustrates one vocabulary word (F23).
type ImageRequest struct {
	Word      string
	MeaningVi string
	// Sentence shows the word in use; may be empty.
	Sentence string
	// Style is the admin's description of how every picture of the lesson should look; may be empty.
	Style string
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

// DisabledImages is the image provider used when AI_PROVIDER=none or no key is set.
type DisabledImages struct{}

// GenerateImage always returns ErrNotConfigured.
func (DisabledImages) GenerateImage(context.Context, ImageRequest) (Image, error) {
	return Image{}, ErrNotConfigured
}
