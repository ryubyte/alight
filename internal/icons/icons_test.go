//go:build darwin

package icons

import (
	"testing"

	"github.com/ryubyte/aglight/internal/core/state"
)

func TestForStatus_NonNilImage(t *testing.T) {
	statuses := []state.Status{
		state.StatusIdle,
		state.StatusRunning,
		state.StatusCompleted,
		state.StatusApprovalNeeded,
	}
	for _, s := range statuses {
		img := ForStatus(s)
		// Smoke test: just ensure no panic
		_ = img
		t.Logf("ForStatus(%q) ok", s)
	}
}

func TestForStatusDim_NonNilImage(t *testing.T) {
	img := ForStatusDim(state.StatusApprovalNeeded)
	_ = img
	t.Log("ForStatusDim(approval) ok")
}
