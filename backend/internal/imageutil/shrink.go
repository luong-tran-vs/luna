// Package imageutil scales pictures down for storage, shared by the lesson and grammar pictures.
package imageutil

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	_ "image/gif" // decodes uploaded GIF pictures
	"image/jpeg"
	_ "image/png" // decodes PNG pictures, drawn or uploaded
)

// jpegQuality is the JPEG quality of a stored picture (1–100): small files, no visible loss at card size.
const jpegQuality = 70

// Shrink scales a picture down so its longest side is at most side pixels and encodes it as JPEG
// on white (JPEG has no transparency).
func Shrink(data []byte, side int) ([]byte, error) {
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("imageutil: decode image: %w", err)
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w == 0 || h == 0 {
		return nil, errors.New("imageutil: empty image")
	}
	scale := min(1, float64(side)/float64(max(w, h)))
	dw, dh := max(1, int(float64(w)*scale)), max(1, int(float64(h)*scale))
	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))
	// Box filter: each target pixel is the mean of the source pixels it covers.
	for y := range dh {
		y0, y1 := b.Min.Y+y*h/dh, b.Min.Y+max((y+1)*h/dh, y*h/dh+1)
		for x := range dw {
			x0, x1 := b.Min.X+x*w/dw, b.Min.X+max((x+1)*w/dw, x*w/dw+1)
			var r, g, bl, a, n uint64
			for sy := y0; sy < y1; sy++ {
				for sx := x0; sx < x1; sx++ {
					cr, cg, cb, ca := src.At(sx, sy).RGBA()
					r, g, bl, a, n = r+uint64(cr), g+uint64(cg), bl+uint64(cb), a+uint64(ca), n+1
				}
			}
			// Premultiplied colors over a white background: JPEG has no transparency.
			white := 0xffff - a/n
			dst.Set(x, y, color.RGBA64{R: uint16(r/n + white), G: uint16(g/n + white), B: uint16(bl/n + white), A: 0xffff})
		}
	}
	var out bytes.Buffer
	if err := jpeg.Encode(&out, dst, &jpeg.Options{Quality: jpegQuality}); err != nil {
		return nil, fmt.Errorf("imageutil: encode image: %w", err)
	}
	return out.Bytes(), nil
}
