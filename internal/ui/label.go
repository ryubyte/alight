//go:build darwin

package ui

import (
	"log"
	"os/exec"
	"strings"

	"github.com/progrium/darwinkit/macos/appkit"
	"github.com/ryubyte/codex-bar/internal/state"
)

// StatusLabel returns a Chinese status label with emoji for the given status.
func StatusLabel(s state.Status) string {
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

// PlaySound plays a system sound by name.
func PlaySound(name appkit.SoundName) {
	snd := appkit.Sound_SoundNamed(name)
	snd.Play()
}

// CopyToClipboard copies text to the macOS clipboard via pbcopy.
func CopyToClipboard(text string) {
	pbcopy := exec.Command("pbcopy")
	pbcopy.Stdin = strings.NewReader(text)
	if err := pbcopy.Run(); err != nil {
		log.Printf("failed to copy to clipboard: %v", err)
	}
}
