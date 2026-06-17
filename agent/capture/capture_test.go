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

func TestDownscale_largeImageResizedToFit(t *testing.T) {
	src := solidImage(3840, 2160, color.RGBA{10, 20, 30, 255})
	out := downscale(src)
	b := out.Bounds()
	if b.Dx() > maxWidth || b.Dy() > maxHeight {
		t.Errorf("downscaled image %dx%d exceeds %dx%d box", b.Dx(), b.Dy(), maxWidth, maxHeight)
	}
	// 16:9 source fills the 1920x1080 box exactly.
	if b.Dx() != 1920 || b.Dy() != 1080 {
		t.Errorf("expected 1920x1080, got %dx%d", b.Dx(), b.Dy())
	}
}

func TestDownscale_smallImageUntouched(t *testing.T) {
	src := solidImage(1280, 720, color.RGBA{1, 2, 3, 255})
	out := downscale(src)
	if out.Bounds() != src.Bounds() {
		t.Errorf("in-bounds image should be untouched, got %v", out.Bounds())
	}
}

func TestDownscale_aspectRatioPreserved(t *testing.T) {
	// Ultrawide 3440x1440 -> limited by width: scale 1920/3440 -> 1920x803
	// (1440 * 1920/3440 = 803.72, truncated).
	src := solidImage(3440, 1440, color.RGBA{9, 9, 9, 255})
	out := downscale(src)
	b := out.Bounds()
	if b.Dx() != 1920 {
		t.Errorf("expected width 1920, got %d", b.Dx())
	}
	if b.Dy() != 803 {
		t.Errorf("expected height 803, got %d", b.Dy())
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
