package icons

import (
	"bytes"
	"image"
	"image/color"
	"image/png"

	"sync"

	"github.com/ryubyte/codex-bar/internal/state"
)

const size = 22

var (
	idleColor  = color.NRGBA{R: 128, G: 128, B: 128, A: 255}
	runColor   = color.NRGBA{R: 255, G: 200, B: 0, A: 255}
	doneColor  = color.NRGBA{R: 0, G: 200, B: 80, A: 255}
	approvColor = color.NRGBA{R: 255, G: 50, B: 50, A: 255}
)

var (
	once     sync.Once
	cache    map[state.Status][]byte
)

// ForStatus returns the PNG byte data for the icon corresponding to the given status.
func ForStatus(s state.Status) []byte {
	once.Do(func() {
		cache = map[state.Status][]byte{
			state.StatusIdle:           generateCircle(idleColor),
			state.StatusRunning:        generateCircle(runColor),
			state.StatusCompleted:      generateCircle(doneColor),
			state.StatusApprovalNeeded: generateCircle(approvColor),
		}
	})
	return cache[s]
}

// generateCircle creates a 22x22 PNG image with a filled circle of the given color.
func generateCircle(c color.NRGBA) []byte {
	img := image.NewNRGBA(image.Rect(0, 0, size, size))

	cx := float64(size) / 2.0
	cy := float64(size) / 2.0
	r := float64(size) / 2.0

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx := float64(x) + 0.5 - cx
			dy := float64(y) + 0.5 - cy
			if dx*dx+dy*dy <= r*r {
				img.SetNRGBA(x, y, c)
			}
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		panic("icons: failed to encode PNG: " + err.Error())
	}
	return buf.Bytes()
}


