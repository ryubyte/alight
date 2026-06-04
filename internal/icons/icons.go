package icons

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"
	"sync"

	"github.com/ryubyte/codex-bar/internal/state"
)

// macOS menu bar icon dimensions (Retina: 44x44 for 22pt logical)
const size = 44

// Traffic signal 🚥 style: 3 horizontal lights in a dark housing
// [🔴 Red=Approval] [🟡 Yellow=Running] [🟢 Green=Completed]
// Active state is brightly lit, others are dim.

type lightDef struct {
	on   color.NRGBA
	off  color.NRGBA
	glow color.NRGBA
}

var lights = []lightDef{
	{ // Red — approval needed
		on:   color.NRGBA{R: 255, G: 59, B: 48, A: 255},
		off:  color.NRGBA{R: 60, G: 20, B: 18, A: 200},
		glow: color.NRGBA{R: 255, G: 59, B: 48, A: 50},
	},
	{ // Yellow — running
		on:   color.NRGBA{R: 255, G: 204, B: 0, A: 255},
		off:  color.NRGBA{R: 60, G: 50, B: 5, A: 200},
		glow: color.NRGBA{R: 255, G: 204, B: 0, A: 50},
	},
	{ // Green — completed
		on:   color.NRGBA{R: 52, G: 199, B: 89, A: 255},
		off:  color.NRGBA{R: 15, G: 50, B: 25, A: 200},
		glow: color.NRGBA{R: 52, G: 199, B: 89, A: 50},
	},
}

// statusToLight maps status to the active light index (0=red, 1=yellow, 2=green).
// -1 means no active light (idle).
var statusToLight = map[state.Status]int{
	state.StatusIdle:           -1,
	state.StatusRunning:        1, // yellow
	state.StatusCompleted:      2, // green
	state.StatusApprovalNeeded: 0, // red
}

// X positions for the 3 lights, evenly spaced in the 44px icon
var lightCX = []float64{9, 22, 35}

const lightCY = 22.0

var (
	once  sync.Once
	cache map[state.Status][]byte
)

// ForStatus returns PNG bytes for the traffic signal icon in the given status.
func ForStatus(s state.Status) []byte {
	once.Do(func() {
		cache = make(map[state.Status][]byte)
		for _, st := range []state.Status{
			state.StatusIdle,
			state.StatusRunning,
			state.StatusCompleted,
			state.StatusApprovalNeeded,
		} {
			cache[st] = renderSignal(statusToLight[st])
		}
	})
	return cache[s]
}

func renderSignal(activeIdx int) []byte {
	img := image.NewNRGBA(image.Rect(0, 0, size, size))

	// Dark rounded rect housing (the signal body)
	drawRoundedRect(img, 1, 6, 42, 32, 8, color.NRGBA{R: 70, G: 70, B: 75, A: 80})  // border
	drawRoundedRect(img, 2, 7, 40, 30, 7, color.NRGBA{R: 25, G: 25, B: 28, A: 240}) // fill

	for i, light := range lights {
		cx := lightCX[i]
		isActive := (i == activeIdx)

		if isActive {
			// Glow halo
			drawCircle(img, cx, lightCY, 9, light.glow)
			drawCircle(img, cx, lightCY, 7, color.NRGBA{R: light.on.R, G: light.on.G, B: light.on.B, A: 80})
			// Main lit circle
			drawCircle(img, cx, lightCY, 5.5, light.on)
			// Bright center highlight
			drawCircle(img, cx-0.5, lightCY-0.5, 2.5, color.NRGBA{
				R: minU8(light.on.R+90, 255),
				G: minU8(light.on.G+90, 255),
				B: minU8(light.on.B+90, 255),
				A: 150,
			})
		} else {
			// Dim/unlit circle
			drawCircle(img, cx, lightCY, 5.5, light.off)
			// Very subtle color rim so you can tell which light is which even when off
			drawCircle(img, cx, lightCY, 5.5, color.NRGBA{R: light.on.R, G: light.on.G, B: light.on.B, A: 20})
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		panic("icons: failed to encode PNG: " + err.Error())
	}
	return buf.Bytes()
}

// --- Drawing primitives ---

func drawRoundedRect(img *image.NRGBA, x, y, w, h, r float64, c color.NRGBA) {
	for dy := 0.0; dy < h; dy++ {
		for dx := 0.0; dx < w; dx++ {
			px := int(x + dx)
			py := int(y + dy)
			if px < 0 || py < 0 || px >= size || py >= size {
				continue
			}
			if isInRoundedRect(dx, dy, w, h, r) {
				blendPixel(img, px, py, c)
			}
		}
	}
}

func isInRoundedRect(x, y, w, h, r float64) bool {
	if x < r && y < r {
		return dist(x, y, r, r) <= r
	}
	if x > w-r && y < r {
		return dist(x, y, w-r, r) <= r
	}
	if x < r && y > h-r {
		return dist(x, y, r, h-r) <= r
	}
	if x > w-r && y > h-r {
		return dist(x, y, w-r, h-r) <= r
	}
	return true
}

func dist(x1, y1, x2, y2 float64) float64 {
	dx := x1 - x2
	dy := y1 - y2
	return math.Sqrt(dx*dx + dy*dy)
}

func drawCircle(img *image.NRGBA, cx, cy, r float64, c color.NRGBA) {
	for dy := -int(r) - 1; dy <= int(r)+1; dy++ {
		for dx := -int(r) - 1; dx <= int(r)+1; dx++ {
			px := int(cx) + dx
			py := int(cy) + dy
			if px < 0 || py < 0 || px >= size || py >= size {
				continue
			}
			d := dist(float64(dx), float64(dy), 0, 0)
			if d <= r {
				aSrc := float64(c.A) / 255.0
				// Anti-alias edge
				if d > r-1.5 {
					aSrc *= (r - d + 1.5) / 1.5
					if aSrc < 0 {
						aSrc = 0
					}
				}
				blendPixel(img, px, py, color.NRGBA{R: c.R, G: c.G, B: c.B, A: uint8(aSrc * 255)})
			}
		}
	}
}

func blendPixel(img *image.NRGBA, x, y int, c color.NRGBA) {
	if x < 0 || x >= size || y < 0 || y >= size {
		return
	}
	bg := img.NRGBAAt(x, y)
	aSrc := float64(c.A) / 255.0
	aDst := 1.0 - aSrc
	img.SetNRGBA(x, y, color.NRGBA{
		R: uint8(float64(c.R)*aSrc + float64(bg.R)*aDst),
		G: uint8(float64(c.G)*aSrc + float64(bg.G)*aDst),
		B: uint8(float64(c.B)*aSrc + float64(bg.B)*aDst),
		A: uint8((aSrc + float64(bg.A)/255.0*aDst) * 255),
	})
}

func minU8(a, b uint8) uint8 {
	if a < b {
		return a
	}
	return b
}
