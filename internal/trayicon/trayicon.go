// Package trayicon renders the shellight menu bar icon as PNG: a black or
// white glyph with a coloured status dot. It also renders the status shapes
// of the menu's sandbox lines.
//
// The glyph is a terminal frame with a ">_" prompt. Wails scales the image to
// the menu bar thickness (about 24 pt), so the canvas is 48x48 px (@2x).
// Shapes are drawn as distance fields with 4x4 supersampling, so no image
// library beyond the standard library is needed.
package trayicon

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"
)

// Size is the canvas edge in pixels.
const Size = 48

// RGB is an opaque colour.
type RGB struct{ R, G, B uint8 }

// Dot geometry (canvas pixels), bottom right of the frame.
const (
	dotX, dotY = 37.5, 33.5
	dotR       = 7.0
	dotGap     = 2.5 // transparent ring between dot and glyph
)

// Badged returns the glyph in glyph colour with a status dot in dot colour.
// The glyph is cut out around the dot. The result is not a template image.
func Badged(glyph, dot RGB) []byte {
	img := image.NewNRGBA(image.Rect(0, 0, Size, Size))
	const n = 4 // samples per axis
	for py := 0; py < Size; py++ {
		for px := 0; px < Size; px++ {
			var g, d int
			for sy := 0; sy < n; sy++ {
				for sx := 0; sx < n; sx++ {
					x := float64(px) + (float64(sx)+0.5)/n
					y := float64(py) + (float64(sy)+0.5)/n
					dd := math.Hypot(x-dotX, y-dotY)
					if dd <= dotR {
						d++
						continue
					}
					if dd <= dotR+dotGap {
						continue
					}
					if inGlyph(x, y) {
						g++
					}
				}
			}
			switch {
			case d > 0:
				// Dot pixel; glyph coverage at the edge is cut out anyway.
				img.SetNRGBA(px, py, color.NRGBA{dot.R, dot.G, dot.B, uint8(d * 255 / (n * n))})
			case g > 0:
				img.SetNRGBA(px, py, color.NRGBA{glyph.R, glyph.G, glyph.B, uint8(g * 255 / (n * n))})
			}
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		panic(err) // encoding an in-memory NRGBA cannot fail
	}
	return buf.Bytes()
}

// inGlyph reports whether canvas point (x, y) is inside the glyph.
func inGlyph(x, y float64) bool {
	const stroke = 3.0
	// Terminal frame: rounded rectangle outline.
	outer := roundRect(x, y, 6, 10, 42, 38, 6)
	if outer <= 0 && outer > -stroke {
		return true
	}
	// Prompt ">" and "_".
	h := stroke / 2
	return segment(x, y, 13, 18, 19, 24) <= h ||
		segment(x, y, 19, 24, 13, 30) <= h ||
		segment(x, y, 22.5, 30, 29, 30) <= h
}

// roundRect is the signed distance to a rounded rectangle (x0,y0)-(x1,y1).
func roundRect(x, y, x0, y0, x1, y1, r float64) float64 {
	cx, cy := (x0+x1)/2, (y0+y1)/2
	hx, hy := (x1-x0)/2-r, (y1-y0)/2-r
	qx, qy := math.Abs(x-cx)-hx, math.Abs(y-cy)-hy
	return math.Hypot(math.Max(qx, 0), math.Max(qy, 0)) + math.Min(math.Max(qx, qy), 0) - r
}

// segment is the distance from (x, y) to the segment (ax,ay)-(bx,by).
func segment(x, y, ax, ay, bx, by float64) float64 {
	px, py := x-ax, y-ay
	dx, dy := bx-ax, by-ay
	t := math.Max(0, math.Min(1, (px*dx+py*dy)/(dx*dx+dy*dy)))
	return math.Hypot(px-t*dx, py-t*dy)
}
