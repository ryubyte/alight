package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/progrium/darwinkit/helper/action"
	"github.com/progrium/darwinkit/macos"
	"github.com/progrium/darwinkit/macos/appkit"
	"github.com/progrium/darwinkit/objc"

	"github.com/ryubyte/codex-bar/internal/codexcfg"
	"github.com/ryubyte/codex-bar/internal/hookgen"
	"github.com/ryubyte/codex-bar/internal/icons"
	"github.com/ryubyte/codex-bar/internal/server"
	"github.com/ryubyte/codex-bar/internal/state"
)

const defaultStartPort = 9876

const blinkInterval = 1200 * time.Millisecond

const (
	soundApproval = "Sosumi"
	soundComplete = "Glass"
)

func main() {
	macos.RunApp(didLaunch)
}

func didLaunch(app appkit.Application, delegate *appkit.ApplicationDelegate) {
	app.SetActivationPolicy(appkit.ApplicationActivationPolicyAccessory)

	machine := state.NewMachine()

	// Auto-detect free port
	port, err := server.FindFreePort(defaultStartPort)
	if err != nil {
		log.Fatalf("find free port: %v", err)
	}
	serverAddr := fmt.Sprintf("localhost:%d", port)

	// Cleanup stale hooks from previous runs, then inject fresh ones
	cfg, err := codexcfg.Read()
	if err != nil {
		log.Printf("warning: read codex config: %v", err)
	}
	cfg = codexcfg.Inject(cfg, serverAddr)
	if err := codexcfg.Write(cfg); err != nil {
		log.Printf("warning: write codex config: %v", err)
	}

	// Setup cleanup on exit (no defer — didLaunch returns immediately)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-sigCh
		cleanupHooks()
		app.Terminate(nil)
	}()

	// Start HTTP server
	srv := server.New(machine, serverAddr)
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Printf("server error: %v", err)
		}
	}()

	// Status bar item
	item := appkit.StatusBar_SystemStatusBar().StatusItemWithLength(appkit.VariableStatusItemLength)
	objc.Retain(&item)
	btn := item.Button()
	btn.SetImage(icons.ForStatus(machine.Current()))
	btn.SetToolTip("Codex Bar ⚫ 空闲")

	// Blink state
	blinkOn := true
	var blinkTimer *time.Timer

	stopBlink := func() {
		if blinkTimer != nil {
			blinkTimer.Stop()
			blinkTimer = nil
		}
	}

	startBlink := func() {
		stopBlink()
		blinkOn = true
		blinkTimer = time.AfterFunc(blinkInterval, func() {
			blinkOn = !blinkOn
			if machine.Current() == state.StatusApprovalNeeded {
				if blinkOn {
					btn.SetImage(icons.ForStatus(state.StatusApprovalNeeded))
				} else {
					btn.SetImage(icons.ForStatusDim(state.StatusApprovalNeeded))
				}
				blinkTimer.Reset(blinkInterval)
			}
		})
	}

	// Sound on/off
	soundOn := true

	// Build menu
	menu := appkit.NewMenuWithTitle("Codex Bar")

	mStatus := appkit.NewMenuItemWithTitleActionKeyEquivalent("⚫ 空闲", objc.Selector{}, "")
	mStatus.SetEnabled(false)
	menu.AddItem(mStatus)

	menu.AddItem(appkit.MenuItem_SeparatorItem())

	mHooks := appkit.NewMenuItemWithTitleActionKeyEquivalent(fmt.Sprintf("Hooks 配置 ✓ 已注入 :%d", port), objc.Selector{}, "")
	action.Set(mHooks, func(sender objc.Object) {
		// Copy the raw TOML to clipboard for reference
		cfg := hookgen.Config{ServerAddr: serverAddr}
		toml := hookgen.Generate(cfg)
		copyToClipboard(toml)
		mHooks.SetTitle("Hooks 配置 ✓ 已复制到剪贴板")
		go func() {
			time.Sleep(2 * time.Second)
			mHooks.SetTitle(fmt.Sprintf("Hooks 配置 ✓ 已注入 :%d", port))
		}()
	})
	menu.AddItem(mHooks)

	mReset := appkit.NewMenuItemWithTitleActionKeyEquivalent("重置为空闲", objc.Selector{}, "")
	action.Set(mReset, func(sender objc.Object) {
		machine.Update(state.Event{
			Status:    state.StatusIdle,
			EventName: "manual_reset",
		})
	})
	menu.AddItem(mReset)

	menu.AddItem(appkit.MenuItem_SeparatorItem())

	mSound := appkit.NewMenuItemWithTitleActionKeyEquivalent("声音", objc.Selector{}, "")
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

	mQuit := appkit.NewMenuItemWithTitleActionKeyEquivalent("退出", objc.Selector{}, "")
	action.Set(mQuit, func(sender objc.Object) {
		stopBlink()
		cleanupHooks()
		app.Terminate(nil)
	})
	menu.AddItem(mQuit)

	item.SetMenu(menu)

	// Status change callback
	machine.OnChange(func(old, newStatus state.Status, event state.Event) {
		if old == state.StatusApprovalNeeded && newStatus != state.StatusApprovalNeeded {
			stopBlink()
		}

		btn.SetImage(icons.ForStatus(newStatus))
		btn.SetToolTip("Codex Bar " + statusLabel(newStatus))
		mStatus.SetTitle(statusLabel(newStatus))

		if soundOn {
			switch newStatus {
			case state.StatusApprovalNeeded:
				playSound(soundApproval)
				startBlink()
			case state.StatusCompleted:
				playSound(soundComplete)
			}
		} else {
			if newStatus == state.StatusApprovalNeeded {
				startBlink()
			}
		}
	})
}

func cleanupHooks() {
	cfg, err := codexcfg.Read()
	if err != nil {
		log.Printf("warning: read codex config for cleanup: %v", err)
		return
	}
	cfg = codexcfg.Cleanup(cfg)
	if err := codexcfg.Write(cfg); err != nil {
		log.Printf("warning: write codex config for cleanup: %v", err)
	}
}

func playSound(name appkit.SoundName) {
	snd := appkit.Sound_SoundNamed(name)
	snd.Play()
}

func statusLabel(s state.Status) string {
	switch s {
	case state.StatusIdle:
		return "⚫ 空闲"
	case state.StatusRunning:
		return "🟡 运行中"
	case state.StatusCompleted:
		return "🟢 已完成"
	case state.StatusApprovalNeeded:
		return "🔴 需要审批"
	default:
		return string(s)
	}
}

func copyToClipboard(text string) {
	pbcopy := exec.Command("pbcopy")
	pbcopy.Stdin = strings.NewReader(text)
	if err := pbcopy.Run(); err != nil {
		log.Printf("failed to copy to clipboard: %v", err)
	}
}
