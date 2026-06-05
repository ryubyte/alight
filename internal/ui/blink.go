//go:build darwin

package ui

import (
	"time"

	"github.com/progrium/darwinkit/dispatch"
	"github.com/progrium/darwinkit/macos/appkit"
	"github.com/ryubyte/aglight/internal/core/state"
	"github.com/ryubyte/aglight/internal/icons"
)

const blinkInterval = 1200 * time.Millisecond

// BlinkController manages the blink animation for the approval-needed status.
type BlinkController struct {
	Btn     appkit.StatusBarButton
	machine *state.Machine
	timer   *time.Timer
	on      bool
}

// NewBlinkController creates a new BlinkController for the given status bar button.
func NewBlinkController(btn appkit.StatusBarButton, machine *state.Machine) *BlinkController {
	return &BlinkController{
		Btn:     btn,
		machine: machine,
	}
}

// Start begins the blink animation. It stops any existing blink first.
func (b *BlinkController) Start() {
	b.Stop()
	b.on = true
	b.timer = time.AfterFunc(blinkInterval, func() {
		b.on = !b.on
		if b.machine.Current() == state.StatusApprovalNeeded {
			dispatch.MainQueue().DispatchAsync(func() {
				if b.on {
					b.Btn.SetImage(icons.ForStatus(state.StatusApprovalNeeded))
				} else {
					b.Btn.SetImage(icons.ForStatusDim(state.StatusApprovalNeeded))
				}
			})
			b.timer.Reset(blinkInterval)
		}
	})
}

// Stop halts the blink animation.
func (b *BlinkController) Stop() {
	if b.timer != nil {
		b.timer.Stop()
		b.timer = nil
	}
}
