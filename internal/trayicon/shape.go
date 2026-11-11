package trayicon

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/color"
	"image/png"
	"math"
)

// Shape is a status shape, as in the dashboard sidebar.
type Shape int

const (
	Circle   Shape = iota // filled circle: ok
	Ring                  // dashed ring: busy
	Triangle              // filled triangle: error
)

// shapeSize is the edge of a status shape in pixels: 12 pt at 2x.
const shapeSize = 24

// shapeDPI makes macOS show a shapeSize canvas at 12 pt; NSImage takes its
// size from the PNG resolution.
const shapeDPI = 144

// StatusShape returns shape in colour c as a PNG for a menu item. The shape
// is drawn in the 16x16 view box of the dashboard's status shape.
func StatusShape(shape Shape, c RGB) []byte {
	img := image.NewNRGBA(image.Rect(0, 0, shapeSize, shapeSize))
	const n = 4 // samples per axis
	const scale = shapeSize / 16.0
	for py := 0; py < shapeSize; py++ {
		for px := 0; px < shapeSize; px++ {
			hits := 0
			for sy := 0; sy < n; sy++ {
				for sx := 0; sx < n; sx++ {
					x := (float64(px) + (float64(sx)+0.5)/n) / scale
					y := (float64(py) + (float64(sy)+0.5)/n) / scale
					if inShape(shape, x, y) {
						hits++
					}
				}
			}
			if hits > 0 {
				img.SetNRGBA(px, py, color.NRGBA{c.R, c.G, c.B, uint8(hits * 255 / (n * n))})
			}
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		panic(err) // encoding an in-memory NRGBA cannot fail
	}
	return withDPI(buf.Bytes(), shapeDPI)
}

// inShape reports whether view box point (x, y) is inside shape. The values
// are those of frontend/src/StatusShape.svelte.
func inShape(shape Shape, x, y float64) bool {
	switch shape {
	case Ring:
		// Circle r 5.5, stroke 2.4, dashes 4 on 2.6 off along the path,
		// which starts at 3 o'clock and runs clockwise.
		const r, stroke, on, period = 5.5, 2.4, 4.0, 6.6
		dx, dy := x-8, y-8
		if math.Abs(math.Hypot(dx, dy)-r) > stroke/2 {
			return false
		}
		a := math.Atan2(dy, dx) // y points down: clockwise
		if a < 0 {
			a += 2 * math.Pi
		}
		return math.Mod(a*r, period) < on
	case Triangle:
		// Path M8 1.8 L14.6 13.6 H1.4 Z.
		return side(x, y, 8, 1.8, 14.6, 13.6) >= 0 &&
			side(x, y, 14.6, 13.6, 1.4, 13.6) >= 0 &&
			side(x, y, 1.4, 13.6, 8, 1.8) >= 0
	default:
		return math.Hypot(x-8, y-8) <= 6
	}
}

// side is positive if (x, y) lies right of the edge (ax,ay)-(bx,by) in a
// y-down canvas, i.e. inside a clockwise polygon.
func side(x, y, ax, ay, bx, by float64) float64 {
	return (bx-ax)*(y-ay) - (by-ay)*(x-ax)
}

// withDPI inserts a pHYs chunk with dpi into png data after its IHDR
// chunk, which is always first.
func withDPI(data []byte, dpi float64) []byte {
	const ihdrEnd = 8 + 4 + 4 + 13 + 4 // signature, length, type, data, CRC
	ppm := uint32(math.Round(dpi / 0.0254))
	chunk := make([]byte, 0, 4+4+9+4)
	chunk = binary.BigEndian.AppendUint32(chunk, 9)
	chunk = append(chunk, "pHYs"...)
	chunk = binary.BigEndian.AppendUint32(chunk, ppm)
	chunk = binary.BigEndian.AppendUint32(chunk, ppm)
	chunk = append(chunk, 1) // unit: metre
	chunk = binary.BigEndian.AppendUint32(chunk, crc32.ChecksumIEEE(chunk[4:]))
	out := make([]byte, 0, len(data)+len(chunk))
	out = append(out, data[:ihdrEnd]...)
	out = append(out, chunk...)
	return append(out, data[ihdrEnd:]...)
}
