package icons

import (
	"testing"

	"github.com/ryubyte/codex-bar/internal/state"
)

func TestForStatus_ReturnsNonEmptyPNG(t *testing.T) {
	statuses := []state.Status{
		state.StatusIdle,
		state.StatusRunning,
		state.StatusCompleted,
		state.StatusApprovalNeeded,
	}
	for _, s := range statuses {
		data := ForStatus(s)
		if len(data) == 0 {
			t.Errorf("ForStatus(%q) returned empty data", s)
		}
	}
}

func TestForStatus_PNGMagicBytes(t *testing.T) {
	statuses := []state.Status{
		state.StatusIdle,
		state.StatusRunning,
		state.StatusCompleted,
		state.StatusApprovalNeeded,
	}
	for _, s := range statuses {
		data := ForStatus(s)
		if len(data) < 2 {
			t.Fatalf("ForStatus(%q) returned data shorter than 2 bytes", s)
		}
		if data[0] != 0x89 || data[1] != 0x50 {
			t.Errorf("ForStatus(%q) PNG magic bytes = %#x %#x, want 0x89 0x50", s, data[0], data[1])
		}
	}
}
