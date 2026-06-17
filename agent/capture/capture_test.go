package capture

import (
	"image"
	"image/color"
	"testing"
)

func solidImage(w, h int, c color.RGBA) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, c)
		}
	}
	return img
}

func TestHashImage_sameImageSameHash(t *testing.T) {
	img := solidImage(100, 100, color.RGBA{255, 0, 0, 255})
	if hashImage(img) != hashImage(img) {
		t.Error("same image must produce same hash")
	}
}

func TestHashImage_differentImagesDifferentHash(t *testing.T) {
	red := solidImage(100, 100, color.RGBA{255, 0, 0, 255})
	blue := solidImage(100, 100, color.RGBA{0, 0, 255, 255})
	if hashImage(red) == hashImage(blue) {
		t.Error("different images should not hash equal")
	}
}

func TestHashImage_slightChangeDetected(t *testing.T) {
	img1 := solidImage(200, 200, color.RGBA{128, 128, 128, 255})
	img2 := image.NewRGBA(image.Rect(0, 0, 200, 200))
	// Copy img1 then change a small block in the center. The hash samples a
	// sparse grid (32x32), so a region — not a lone pixel — is what a realistic
	// screen update looks like and what change detection must catch.
	for y := 0; y < 200; y++ {
		for x := 0; x < 200; x++ {
			img2.Set(x, y, color.RGBA{128, 128, 128, 255})
		}
	}
	for y := 90; y < 110; y++ {
		for x := 90; x < 110; x++ {
			img2.Set(x, y, color.RGBA{0, 0, 0, 255})
		}
	}
	if hashImage(img1) == hashImage(img2) {
		t.Error("a visible block change should be detected")
	}
}

func TestCapture_nilOnUnchangedFrame(t *testing.T) {
	c := &Capture{}
	// Simulate two identical frames by hashing the same image twice
	img := solidImage(64, 64, color.RGBA{10, 20, 30, 255})
	h := hashImage(img)
	c.prevHash = h

	// Call processFrame with the same hash — should return nil
	result := c.processFrame(img, h)
	if result != nil {
		t.Error("unchanged frame should return nil")
	}
}

func TestCapture_frameReturnedOnChange(t *testing.T) {
	c := &Capture{}
	c.prevHash = 0 // different from any real image hash

	img := solidImage(64, 64, color.RGBA{10, 20, 30, 255})
	h := hashImage(img)

	result := c.processFrame(img, h)
	if result == nil {
		t.Error("changed frame should return JPEG bytes")
	}
	// JPEG magic bytes
	if len(result) < 2 || result[0] != 0xff || result[1] != 0xd8 {
		t.Error("result should be a JPEG (starts with FF D8)")
	}
}
