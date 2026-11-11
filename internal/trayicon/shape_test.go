package trayicon

import (
	"bytes"
	"image"
	"image/png"
	"testing"
)

func TestStatusShape(t *testing.T) {
	c := RGB{0x7f, 0xb5, 0x1a}
	tests := []struct {
		shape         Shape
		opaque, clear []image.Point
	}{
		// Circle: centre and edge filled, corner empty.
		{Circle, []image.Point{{12, 12}, {4, 12}}, []image.Point{{1, 1}}},
		// Ring: the first dash at 3 o'clock, hollow centre, gap after it.
		{Ring, []image.Point{{20, 12}}, []image.Point{{12, 12}, {16, 18}}},
		// Triangle: point up, flat base.
		{Triangle, []image.Point{{12, 14}, {4, 19}}, []image.Point{{4, 4}, {12, 22}}},
	}
	for _, tt := range tests {
		b := StatusShape(tt.shape, c)
		img, err := png.Decode(bytes.NewReader(b)) // also checks the chunk CRCs
		if err != nil {
			t.Fatal(err)
		}
		if got := img.Bounds().Size(); got != image.Pt(shapeSize, shapeSize) {
			t.Fatalf("shape %d: size %v", tt.shape, got)
		}
		for _, p := range tt.opaque {
			r, g, b, a := img.At(p.X, p.Y).RGBA()
			if a != 0xffff || uint8(r>>8) != c.R || uint8(g>>8) != c.G || uint8(b>>8) != c.B {
				t.Errorf("shape %d at %v = %v,%v,%v,%v, want opaque %v", tt.shape, p, r>>8, g>>8, b>>8, a>>8, c)
			}
		}
		for _, p := range tt.clear {
			if _, _, _, a := img.At(p.X, p.Y).RGBA(); a != 0 {
				t.Errorf("shape %d at %v: alpha %d, want 0", tt.shape, p, a>>8)
			}
		}
	}
}

func TestStatusShapeDPI(t *testing.T) {
	// pHYs right after IHDR: 5669 px per metre (144 dpi) both ways, unit metre.
	want := []byte{0, 0, 0, 9, 'p', 'H', 'Y', 's', 0, 0, 0x16, 0x25, 0, 0, 0x16, 0x25, 1}
	if got := StatusShape(Circle, RGB{})[33:50]; !bytes.Equal(got, want) {
		t.Errorf("chunk after IHDR = % x, want % x", got, want)
	}
}
