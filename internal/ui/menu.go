//go:build darwin

package ui

import (
	"github.com/progrium/darwinkit/helper/action"
	"github.com/progrium/darwinkit/macos/appkit"
	"github.com/progrium/darwinkit/objc"
	"github.com/ryubyte/codex-bar/internal/icons"
	"github.com/ryubyte/codex-bar/internal/core/state"
)

const (
	soundApproval = "Sosumi"
	soundComplete = "Glass"
)

// MenuConfig holds the dependencies needed to build the menu.
type MenuConfig struct {
	Machine *state.Machine
	Blink   *BlinkController
	OnQuit  func()
}

// BuildMenu constructs the status bar menu and registers the OnChange callback.
func BuildMenu(cfg MenuConfig) appkit.Menu {
	soundOn := true

	menu := appkit.NewMenuWithTitle("Codex Bar")

	mStatus := appkit.NewMenuItemWithTitleActionKeyEquivalent(StatusLabel(state.StatusIdle), objc.Selector{}, "")
	mStatus.SetEnabled(false)
	menu.AddItem(mStatus)

	menu.AddItem(appkit.MenuItem_SeparatorItem())

	mReset := appkit.NewMenuItemWithTitleActionKeyEquivalent("Reset", objc.Selector{}, "")
	action.Set(mReset, func(sender objc.Object) {
		cfg.Machine.Update(state.Event{
			Status:    state.StatusIdle,
			EventName: "manual_reset",
		})
	})
	menu.AddItem(mReset)

	mSound := appkit.NewMenuItemWithTitleActionKeyEquivalent("Sound", objc.Selector{}, "")
	mSound.SetState(appkit.ControlStateValueOn)
	action.Set(mSound, func(sender objc.Object) {
		soundOn = !soundOn
		if soundOn {
			mSound.SetState(appkit.ControlStateValueOn)
		} else {
			mSound.SetState(appkit.ControlStateValueOff)
		}
	})
	menu.AddItem(mSound)

	mQuit := appkit.NewMenuItemWithTitleActionKeyEquivalent("Quit", objc.Selector{}, "")
	action.Set(mQuit, func(sender objc.Object) {
		cfg.Blink.Stop()
		cfg.OnQuit()
	})
	menu.AddItem(mQuit)

	// Register status change callback to update UI
	cfg.Machine.OnChange(func(old, newStatus state.Status, event state.Event) {
		if old == state.StatusApprovalNeeded && newStatus != state.StatusApprovalNeeded {
			cfg.Blink.Stop()
		}

		cfg.Blink.Btn.SetImage(icons.ForStatus(newStatus))
		cfg.Blink.Btn.SetToolTip("Codex Bar " + StatusLabel(newStatus))
		mStatus.SetTitle(StatusLabel(newStatus))

		if soundOn {
			switch newStatus {
			case state.StatusApprovalNeeded:
				PlaySound(soundApproval)
				cfg.Blink.Start()
			case state.StatusCompleted:
				PlaySound(soundComplete)
			}
		} else {
			if newStatus == state.StatusApprovalNeeded {
				cfg.Blink.Start()
			}
		}
	})

	return menu
}
