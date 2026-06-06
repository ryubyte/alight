// cmd/icongen generates a 1024×1024 PNG application icon for AgLight.
//
// The icon is a dark rounded-rectangle housing containing three vertical
// traffic-light circles (red, yellow, green from top to bottom). The green
// light is shown in its bright "on" state with a glow halo — the most
// positive default for an application icon.
//
// Colors are derived from internal/icons/icons.go to keep visual consistency
// with the menu-bar status icon.
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
)

const size = 1024

// Colors derived from internal/icons/icons.go (float 0-1 → uint8 0-255).
var (
	housingFill = color.NRGBA{R: 30, G: 30, B: 36, A: 235}      // 0.12, 0.12, 0.14, 0.92
	housingEdge = color.NRGBA{R: 60, G: 60, B: 70, A: 153}       // subtle border
	lightDefs   = []struct {
		onR, onG, onB   uint8 // bright "on" color
		offR, offG, offB uint8 // dim "off" color
		glowR, glowG, glowB uint8
	}{
		{ // Red
			onR: 255, onG: 59, onB: 48,
			offR: 56, offG: 26, offB: 20,
			glowR: 255, glowG: 59, glowB: 48,
		},
		{ // Yellow
			onR: 255, onG: 204, onB: 0,
			offR: 61, offG: 51, offB: 13,
			glowR: 255, glowG: 204, glowB: 0,
		},
		{ // Green
			onR: 52, onG: 199, onB: 89,
			offR: 20, offG: 51, offB: 31,
			glowR: 52, glowG: 199, glowB: 89,
		},
	}
)

// fillRoundedRect fills a rounded rectangle on dst with c.
func fillRoundedRect(dst draw.Image, r image.Rectangle, radius float64, c color.Color) {
	cx := float64(r.Min.X+r.Max.X) / 2
	cy := float64(r.Min.Y+r.Max.Y) / 2
	hw := float64(r.Dx()) / 2
	hh := float64(r.Dy()) / 2
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			dx := math.Abs(float64(x) - cx)
			dy := math.Abs(float64(y) - cy)
			if dx <= hw-radius || dy <= hh-radius {
				dst.Set(x, y, c)
			} else if dx <= hw && dy <= hh {
				cornerDx := dx - (hw - radius)
				cornerDy := dy - (hh - radius)
				if cornerDx*cornerDx+cornerDy*cornerDy <= radius*radius {
					dst.Set(x, y, c)
				}
			}
		}
	}
}

// strokeRoundedRect draws a 2px stroke around a rounded rectangle.
func strokeRoundedRect(dst draw.Image, r image.Rectangle, radius, stroke float64, c color.Color) {
	outer := image.Rect(
		r.Min.X-int(stroke), r.Min.Y-int(stroke),
		r.Max.X+int(stroke), r.Max.Y+int(stroke),
	)
	// Draw pixels within <stroke> distance inside the rounded-rect edge
	cx := float64(r.Min.X+r.Max.X) / 2
	cy := float64(r.Min.Y+r.Max.Y) / 2
	hw := float64(r.Dx()) / 2
	hh := float64(r.Dy()) / 2
	for y := outer.Min.Y; y < outer.Max.Y; y++ {
		for x := outer.Min.X; x < outer.Max.X; x++ {
			dx := math.Abs(float64(x) - cx)
			dy := math.Abs(float64(y) - cy)
			// Signed distance to rounded rect
			var d float64
			if dx <= hw-radius && dy <= hh-radius {
				// Inside the flat region — distance is min inset
				d = -min(hw-radius-dx, hh-radius-dy)
			} else if dx > hw-radius && dy > hh-radius {
				// Corner region
				cdx := dx - (hw - radius)
				cdy := dy - (hh - radius)
				d = math.Sqrt(cdx*cdx+cdy*cdy) - radius
			} else if dx > hw-radius {
				d = dx - (hw - radius)
			} else {
				d = dy - (hh - radius)
			}
			// Convert to distance from the edge: negative inside, positive outside
			// We want pixels within [0, stroke) from the outer edge of the rect
			// The "edge" is at d=0 (on the rounded rect boundary)
			if d >= -stroke && d < 0 {
				dst.Set(x, y, c)
			}
		}
	}
}

// fillCircle fills a circle centered at (cx, cy) with given radius.
func fillCircle(dst draw.Image, cx, cy, radius float64, c color.Color) {
	x0 := int(cx - radius - 1)
	y0 := int(cy - radius - 1)
	x1 := int(cx + radius + 1)
	y1 := int(cy + radius + 1)
	r2 := radius * radius
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			dx := float64(x) - cx
			dy := float64(y) - cy
			if dx*dx+dy*dy <= r2 {
				dst.Set(x, y, c)
			}
		}
	}
}

// drawGlow draws a soft glow halo using concentric rings with decreasing alpha.
func drawGlow(dst draw.Image, cx, cy, baseRadius float64, c color.NRGBA, intensity float64) {
	rings := 8
	maxR := baseRadius + 50
	for i := rings; i >= 1; i-- {
		r := baseRadius + float64(i)*(maxR-baseRadius)/float64(rings)
		alpha := uint8(float64(c.A) * intensity * (1.0 - float64(i)/float64(rings)) * 0.6)
		fillCircle(dst, cx, cy, r, color.NRGBA{R: c.R, G: c.G, B: c.B, A: alpha})
	}
}

func main() {
	outPath := flag.String("o", "AppIcon.png", "output PNG path")
	flag.Parse()

	img := image.NewRGBA(image.Rect(0, 0, size, size))

	// 1. Draw housing — dark rounded rectangle
	housingRect := image.Rect(62, 62, size-62, size-62)
	housingRadius := 200.0
	fillRoundedRect(img, housingRect, housingRadius, housingFill)
	strokeRoundedRect(img, housingRect, housingRadius, 2, housingEdge)

	// 2. Draw three lights vertically (Red top, Yellow middle, Green bottom)
	lightRadius := 110.0
	lightCenters := []struct{ x, y float64 }{
		{size / 2, 230},  // Red — top
		{size / 2, 512},  // Yellow — middle
		{size / 2, 794},  // Green — bottom
	}

	activeIdx := 2 // Green is active (most positive default)

	for i, lc := range lightCenters {
		def := lightDefs[i]
		if i == activeIdx {
			// Active light: glow + bright circle + center highlight

			// Outer glow halo
			drawGlow(img, lc.x, lc.y, lightRadius, color.NRGBA{
				R: def.glowR, G: def.glowG, B: def.glowB, A: 64,
			}, 1.0)

			// Main bright circle
			fillCircle(img, lc.x, lc.y, lightRadius, color.NRGBA{
				R: def.onR, G: def.onG, B: def.onB, A: 217,
			})

			// Center highlight (brighter, smaller)
			highlightR := lightRadius * 0.35
			highlightOff := lightRadius * 0.12 // slight offset for "shine" effect
			fillCircle(img, lc.x-highlightOff, lc.y-highlightOff, highlightR, color.NRGBA{
				R: min255(uint16(def.onR) + 89),
				G: min255(uint16(def.onG) + 89),
				B: min255(uint16(def.onB) + 89),
				A: 140,
			})
		} else {
			// Dim/inactive light

			// Dim circle fill
			fillCircle(img, lc.x, lc.y, lightRadius, color.NRGBA{
				R: def.offR, G: def.offG, B: def.offB, A: 217,
			})

			// Subtle color-tinted edge stroke (draw a ring)
			fillCircle(img, lc.x, lc.y, lightRadius, color.NRGBA{
				R: def.onR, G: def.onG, B: def.onB, A: 38,
			})
			fillCircle(img, lc.x, lc.y, lightRadius-2, color.NRGBA{
				R: def.offR, G: def.offG, B: def.offB, A: 217,
			})
		}
	}

	f, err := os.Create(*outPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating %s: %v\n", *outPath, err)
		os.Exit(1)
	}
	defer f.Close()

	if err := png.Encode(f, img); err != nil {
		fmt.Fprintf(os.Stderr, "error encoding PNG: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Generated %s (%dx%d)\n", *outPath, size, size)
}

func min255(v uint16) uint8 {
	if v > 255 {
		return 255
	}
	return uint8(v)
}
