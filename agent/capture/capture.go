package capture

import (
	"bytes"
	"hash/fnv"
	"image"
	"image/jpeg"
	"math"

	"github.com/kbinani/screenshot"
	"golang.org/x/image/draw"
)

// Frames larger than this bounding box are downscaled before encoding. On
// high-resolution displays this reduces JPEG encode cost and bandwidth; frames
// within the box are encoded unchanged (no added work).
const (
	maxWidth  = 1920
	maxHeight = 1080
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

	img = downscale(img)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 70}); err != nil {
		return nil
	}
	return buf.Bytes()
}

// downscale returns img unchanged when it fits within maxWidth x maxHeight;
// otherwise it returns a resized copy that fits the box, preserving aspect ratio.
func downscale(img image.Image) image.Image {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= maxWidth && h <= maxHeight {
		return img
	}
	scale := math.Min(float64(maxWidth)/float64(w), float64(maxHeight)/float64(h))
	nw := int(float64(w) * scale)
	nh := int(float64(h) * scale)
	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	draw.ApproxBiLinear.Scale(dst, dst.Bounds(), img, b, draw.Over, nil)
	return dst
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
