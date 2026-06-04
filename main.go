package main

import (
	"log"
	"os/exec"
	"strings"

	"github.com/getlantern/systray"

	"github.com/ryubyte/codex-bar/internal/hookgen"
	"github.com/ryubyte/codex-bar/internal/icons"
	"github.com/ryubyte/codex-bar/internal/server"
	"github.com/ryubyte/codex-bar/internal/state"
)

const defaultAddr = "localhost:9876"

func main() {
	machine := state.NewMachine()

	// Register OnChange callback to update systray icon and tooltip
	machine.OnChange(func(old, new state.Status, event state.Event) {
		systray.SetIcon(icons.ForStatus(new))
		label := statusLabel(new)
		systray.SetTooltip("Codex Bar 🚥 " + label)
	})

	srv := server.New(machine, defaultAddr)

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Printf("server error: %v", err)
		}
	}()

	systray.Run(onReady(machine), onExit)
}

func onReady(machine *state.Machine) func() {
	return func() {
		systray.SetIcon(icons.ForStatus(state.StatusIdle))
		systray.SetTooltip("Codex Bar 🚥 ⚫ 空闲")

		mStatus := systray.AddMenuItem("🚥 ⚫ 空闲", "当前状态")
		mStatus.Disable()

		systray.AddSeparator()

		mHooks := systray.AddMenuItem("生成 Hooks 配置", "生成并复制 Codex hooks TOML")
		mReset := systray.AddMenuItem("重置为空闲", "将状态重置为空闲")

		systray.AddSeparator()

		mQuit := systray.AddMenuItem("退出", "退出应用")

		// Register OnChange to update mStatus title
		machine.OnChange(func(old, new state.Status, event state.Event) {
			mStatus.SetTitle("🚥 " + statusLabel(new))
		})

		// Handle menu clicks
		go func() {
			for {
				select {
				case <-mHooks.ClickedCh:
					cfg := hookgen.Config{ServerAddr: defaultAddr}
					toml := hookgen.Generate(cfg)
					copyToClipboard(toml)
					mHooks.SetTitle("生成 Hooks 配置 ✓")
					// Briefly show checkmark, then revert
					go func() {
						<-mHooks.ClickedCh // drain
					}()
				case <-mReset.ClickedCh:
					machine.Update(state.Event{
						Status:    state.StatusIdle,
						EventName: "manual_reset",
					})
				case <-mQuit.ClickedCh:
					systray.Quit()
				}
			}
		}()
	}
}

func onExit() {
	log.Println("codex-bar exiting")
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
