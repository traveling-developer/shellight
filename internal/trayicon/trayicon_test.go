package trayicon

import (
	"bytes"
	"image"
	"image/png"
	"testing"
)

func decode(t *testing.T, b []byte) image.Image {
	t.Helper()
	img, err := png.Decode(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	if got := img.Bounds().Size(); got != image.Pt(Size, Size) {
		t.Fatalf("size %v, want %dx%d", got, Size, Size)
	}
	return img
}

func TestBadgedDotAndGap(t *testing.T) {
	dot := RGB{0xff, 0x3b, 0x30}
	img := decode(t, Badged(RGB{255, 255, 255}, dot))

	r, g, b, a := img.At(37, 33).RGBA()
	if a != 0xffff || uint8(r>>8) != dot.R || uint8(g>>8) != dot.G || uint8(b>>8) != dot.B {
		t.Errorf("dot centre = %v,%v,%v,%v, want %v", r>>8, g>>8, b>>8, a, dot)
	}
	// The frame's right edge passes through the gap ring and must be cut out.
	if _, _, _, a := img.At(40, 25).RGBA(); a != 0 {
		t.Errorf("gap pixel alpha %d, want 0", a)
	}
	// Away from the dot the frame is white.
	r, g, b, a = img.At(24, 11).RGBA()
	if a != 0xffff || r != 0xffff || g != 0xffff || b != 0xffff {
		t.Errorf("frame pixel = %v,%v,%v,%v, want opaque white", r, g, b, a)
	}
}
