package tray

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"math"
	"runtime"

	"fyne.io/systray"
)

// iconPNG draws a 32x32 Wi-Fi glyph (three arcs and a dot) in black on transparent; macOS
// uses it as a template image (recoloured for light/dark menu bars).
func iconPNG() []byte {
	const n = 32
	img := image.NewNRGBA(image.Rect(0, 0, n, n))
	cx, cy := 15.5, 25.0
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			dx, dy := float64(x)-cx, float64(y)-cy
			r := math.Hypot(dx, dy)
			ang := math.Atan2(-dy, dx) // 0 = right, pi/2 = up
			in := r < 2.6
			if ang > math.Pi/4 && ang < 3*math.Pi/4 {
				for _, rr := range []float64{8, 14, 20} {
					if math.Abs(r-rr) < 1.6 {
						in = true
					}
				}
			}
			if in {
				img.Set(x, y, color.NRGBA{0, 0, 0, 255})
			}
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

// icoFromPNG wraps a PNG in a single-image .ico container (Windows Vista+ reads PNG entries).
func icoFromPNG(p []byte, size int) []byte {
	var b bytes.Buffer
	w := func(v any) { _ = binary.Write(&b, binary.LittleEndian, v) }
	w(uint16(0))      // reserved
	w(uint16(1))      // type: icon
	w(uint16(1))      // count
	w(uint8(size))    // width
	w(uint8(size))    // height
	w(uint8(0))       // palette
	w(uint8(0))       // reserved
	w(uint16(1))      // planes
	w(uint16(32))     // bpp
	w(uint32(len(p))) // size of image data
	w(uint32(6 + 16)) // offset of image data
	b.Write(p)
	return b.Bytes()
}

func setIcon() {
	p := iconPNG()
	switch runtime.GOOS {
	case "windows":
		systray.SetIcon(icoFromPNG(p, 32))
	case "darwin":
		systray.SetTemplateIcon(p, p)
	default:
		systray.SetIcon(p)
	}
}
