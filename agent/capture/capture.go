package capture

import (
	"bytes"
	"hash/fnv"
	"image"
	"image/jpeg"

	"github.com/kbinani/screenshot"
)

type Capture struct {
	prevHash uint64
}

func New() *Capture { return &Capture{} }

// Frame captures the primary monitor. Returns nil, nil if the screen has not changed.
func (c *Capture) Frame() ([]byte, error) {
	bounds := screenshot.GetDisplayBounds(0)
	img, err := screenshot.CaptureRect(bounds)
	if err != nil {
		return nil, err
	}
	h := hashImage(img)
	return c.processFrame(img, h), nil
}

func (c *Capture) processFrame(img image.Image, h uint64) []byte {
	if h == c.prevHash {
		return nil
	}
	c.prevHash = h

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 70}); err != nil {
		return nil
	}
	return buf.Bytes()
}

func hashImage(img image.Image) uint64 {
	b := img.Bounds()
	w := b.Max.X - b.Min.X
	h := b.Max.Y - b.Min.Y

	const grid = 32
	stepX := w / grid
	if stepX < 1 {
		stepX = 1
	}
	stepY := h / grid
	if stepY < 1 {
		stepY = 1
	}

	fn := fnv.New64a()
	var pix [3]byte
	for y := b.Min.Y; y < b.Max.Y; y += stepY {
		for x := b.Min.X; x < b.Max.X; x += stepX {
			r, g, bv, _ := img.At(x, y).RGBA()
			pix[0] = byte(r >> 8)
			pix[1] = byte(g >> 8)
			pix[2] = byte(bv >> 8)
			fn.Write(pix[:])
		}
	}
	return fn.Sum64()
}
