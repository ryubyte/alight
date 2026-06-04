//go:build darwin

package icons

import (
	"github.com/progrium/darwinkit/macos/appkit"
	"github.com/progrium/darwinkit/macos/foundation"
	"github.com/ryubyte/aglight/internal/core/state"
)

// Icon logical size: 56pt wide x 22pt tall (menu bar height)
// Wider than before to give more spacing between lights
const (
	iconW = 56.0
	iconH = 22.0
)

// Housing geometry
var housingRect = foundation.Rect{
	Origin: foundation.Point{X: 0, Y: 0},
	Size:   foundation.Size{Width: iconW, Height: iconH},
}

// Light center positions (left=red, center=yellow, right=green)
var lightCenters = []foundation.Point{
	{X: 10, Y: 11}, // Red
	{X: 28, Y: 11}, // Yellow
	{X: 46, Y: 11}, // Green
}

const lightRadius = 6.0
const housingRadius = 5.0

// statusToLight maps a state.Status to an active light index (-1 = none active).
var statusToLight = map[state.Status]int{
	state.StatusIdle:           -1,
	state.StatusRunning:        1,
	state.StatusCompleted:      2,
	state.StatusApprovalNeeded: 0,
}

// Light color definitions (RGBA as float64 0-1)
type lightColors struct {
	onR, onG, onB    float64 // bright "on" color
	offR, offG, offB float64 // dim "off" color
	glowR, glowG, glowB float64 // glow halo color
}

var lightDefs = []lightColors{
	{ // Red — approval needed
		onR: 1.0, onG: 0.231, onB: 0.188,
		offR: 0.22, offG: 0.10, offB: 0.08,
		glowR: 1.0, glowG: 0.231, glowB: 0.188,
	},
	{ // Yellow — running
		onR: 1.0, onG: 0.8, onB: 0.0,
		offR: 0.24, offG: 0.20, offB: 0.05,
		glowR: 1.0, glowG: 0.8, glowB: 0.0,
	},
	{ // Green — completed
		onR: 0.204, onG: 0.780, onB: 0.349,
		offR: 0.08, offG: 0.20, offB: 0.12,
		glowR: 0.204, glowG: 0.780, glowB: 0.349,
	},
}

// ForStatus returns an appkit.Image with a traffic-light icon for the given status.
func ForStatus(s state.Status) appkit.Image {
	return renderSignal(statusToLight[s], 1.0)
}

// ForStatusDim returns a dimmed version — used for the blink "off" phase
// of the approval (red) light. Only dims the active light; housing stays the same.
func ForStatusDim(s state.Status) appkit.Image {
	return renderSignal(statusToLight[s], 0.35)
}

// brightness: 1.0 = full bright, <1.0 = dimmed active light
func renderSignal(activeIdx int, brightness float64) appkit.Image {
	img := appkit.Image_ImageWithSizeFlippedDrawingHandler(foundation.Size{
		Width:  iconW,
		Height: iconH,
	}, false, func(rect foundation.Rect) bool {
		// 1. Draw dark rounded-rect housing
		housing := appkit.BezierPath_BezierPathWithRoundedRectXRadiusYRadius(housingRect, housingRadius, housingRadius)
		appkit.Color_ColorWithCalibratedRedGreenBlueAlpha(0.12, 0.12, 0.14, 0.92).SetFill()
		housing.Fill()

		// 2. Draw each light
		for i, colors := range lightDefs {
			center := lightCenters[i]
			circleRect := foundation.Rect{
				Origin: foundation.Point{X: center.X - lightRadius, Y: center.Y - lightRadius},
				Size:   foundation.Size{Width: lightRadius * 2, Height: lightRadius * 2},
			}

			if i == activeIdx {
				// Outer glow halo (scaled by brightness)
				glowRect := foundation.Rect{
					Origin: foundation.Point{X: center.X - lightRadius - 3, Y: center.Y - lightRadius - 3},
					Size:   foundation.Size{Width: (lightRadius + 3) * 2, Height: (lightRadius + 3) * 2},
				}
				glowPath := appkit.BezierPath_BezierPathWithOvalInRect(glowRect)
				appkit.Color_ColorWithCalibratedRedGreenBlueAlpha(colors.glowR, colors.glowG, colors.glowB, 0.25*brightness).SetFill()
				glowPath.Fill()

				// Main circle (brightness scaled)
				mainPath := appkit.BezierPath_BezierPathWithOvalInRect(circleRect)
				// Blend between on-color and off-color based on brightness
				r := colors.offR + (colors.onR-colors.offR)*brightness
				g := colors.offG + (colors.onG-colors.offG)*brightness
				b := colors.offB + (colors.onB-colors.offB)*brightness
				appkit.Color_ColorWithCalibratedRedGreenBlueAlpha(r, g, b, 0.85+0.15*brightness).SetFill()
				mainPath.Fill()

				// Bright center highlight (only visible at high brightness)
				if brightness > 0.7 {
					highlightR := lightRadius * 0.35
					highlightRect := foundation.Rect{
						Origin: foundation.Point{X: center.X - highlightR - 1, Y: center.Y - highlightR - 1},
						Size:   foundation.Size{Width: highlightR * 2, Height: highlightR * 2},
					}
					highlightPath := appkit.BezierPath_BezierPathWithOvalInRect(highlightRect)
					appkit.Color_ColorWithCalibratedRedGreenBlueAlpha(
						min1(colors.onR+0.35),
						min1(colors.onG+0.35),
						min1(colors.onB+0.35),
						0.55*brightness,
					).SetFill()
					highlightPath.Fill()
				}
			} else {
				// Dim circle
				dimPath := appkit.BezierPath_BezierPathWithOvalInRect(circleRect)
				appkit.Color_ColorWithCalibratedRedGreenBlueAlpha(colors.offR, colors.offG, colors.offB, 0.85).SetFill()
				dimPath.Fill()

				// Subtle color tint edge
				appkit.Color_ColorWithCalibratedRedGreenBlueAlpha(colors.onR, colors.onG, colors.onB, 0.15).SetStroke()
				dimPath.SetLineWidth(1.0)
				dimPath.Stroke()
			}
		}
		return true
	})

	img.SetTemplate(false)
	return img
}

func min1(v float64) float64 {
	if v > 1.0 {
		return 1.0
	}
	return v
}
