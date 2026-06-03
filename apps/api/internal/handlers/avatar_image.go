package handlers

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"

	"golang.org/x/image/draw"
	"golang.org/x/image/webp"
)

// avatarCanvasSize is the square edge (px) of the normalized avatar fed to the
// image model and stored as the public PNG. Keeping every avatar at one square
// size is what makes the crew's profile pictures visually unified, and it keeps
// the input within Nova Canvas's accepted dimension/pixel limits.
const avatarCanvasSize = 1024

// normalizeToSquarePNG decodes a png/jpeg/webp upload, center-crops it to a
// square, scales it to avatarCanvasSize, and re-encodes as PNG. This guarantees
// the image model receives a format and dimension it accepts (Nova Canvas takes
// PNG/JPEG only and is strict about size), regardless of what the user uploaded.
func normalizeToSquarePNG(data []byte, contentType string) ([]byte, error) {
	img, err := decodeImage(data, contentType)
	if err != nil {
		return nil, fmt.Errorf("decode upload: %w", err)
	}

	b := img.Bounds()
	side := b.Dx()
	if b.Dy() < side {
		side = b.Dy()
	}
	if side <= 0 {
		return nil, fmt.Errorf("upload has zero dimensions")
	}
	// Center-crop to the largest centered square.
	offX := b.Min.X + (b.Dx()-side)/2
	offY := b.Min.Y + (b.Dy()-side)/2
	srcRect := image.Rect(offX, offY, offX+side, offY+side)

	dst := image.NewRGBA(image.Rect(0, 0, avatarCanvasSize, avatarCanvasSize))
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, srcRect, draw.Over, nil)

	var buf bytes.Buffer
	if err := png.Encode(&buf, dst); err != nil {
		return nil, fmt.Errorf("encode png: %w", err)
	}
	return buf.Bytes(), nil
}

func decodeImage(data []byte, contentType string) (image.Image, error) {
	r := bytes.NewReader(data)
	switch contentType {
	case "image/png":
		return png.Decode(r)
	case "image/jpeg":
		return jpeg.Decode(r)
	case "image/webp":
		return webp.Decode(r)
	default:
		// The handler validates the content type before this is reached; fall
		// back to format sniffing so a mislabeled-but-valid image still decodes.
		img, _, err := image.Decode(r)
		return img, err
	}
}
