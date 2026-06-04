package claude_test

import (
	"os"
	"path/filepath"
	"testing"

	adapter "github.com/ryubyte/codex-bar/internal/adapter/claude"
	"github.com/ryubyte/codex-bar/internal/core/state"
)

func tmpSettingsPath(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	return filepath.Join(dir, "settings.json")
}

func TestInject(t *testing.T) {
	path := tmpSettingsPath(t)
	orig := adapter.SettingsPathFn
	adapter.SettingsPathFn = func() (string, error) { return path, nil }
	defer func() { adapter.SettingsPathFn = orig }()

	a := adapter.New()
	if err := a.Inject("9876"); err != nil {
		t.Fatalf("inject: %v", err)
	}

	settings, err := adapter.Read()
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	hooks, ok := settings["hooks"].(map[string]interface{})
	if !ok {
		t.Fatal("no hooks section")
	}
	if len(hooks) != 5 {
		t.Fatalf("expected 5 hook events, got %d", len(hooks))
	}
}

func TestCleanup(t *testing.T) {
	path := tmpSettingsPath(t)
	orig := adapter.SettingsPathFn
	adapter.SettingsPathFn = func() (string, error) { return path, nil }
	defer func() { adapter.SettingsPathFn = orig }()

	a := adapter.New()
	a.Inject("9876")
	a.Cleanup()

	settings, err := adapter.Read()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if _, exists := settings["hooks"]; exists {
		t.Fatal("hooks should be removed after cleanup")
	}
}

func TestInjectPreservesUserConfig(t *testing.T) {
	path := tmpSettingsPath(t)
	orig := adapter.SettingsPathFn
	adapter.SettingsPathFn = func() (string, error) { return path, nil }
	defer func() { adapter.SettingsPathFn = orig }()

	// Write user config first
	os.MkdirAll(filepath.Dir(path), 0755)
	settings := map[string]interface{}{
		"model": "claude-sonnet-4-20250514",
		"theme": "dark",
	}
	adapter.Write(settings)

	a := adapter.New()
	a.Inject("9876")

	read, _ := adapter.Read()
	if read["model"] != "claude-sonnet-4-20250514" {
		t.Fatal("user model should be preserved")
	}
	if read["theme"] != "dark" {
		t.Fatal("user theme should be preserved")
	}
}

func TestMapEvent(t *testing.T) {
	a := adapter.New()
	if a.MapEvent("SessionStart") != state.StatusRunning {
		t.Error("SessionStart should map to running")
	}
	if a.MapEvent("UserPromptSubmit") != state.StatusRunning {
		t.Error("UserPromptSubmit should map to running")
	}
	if a.MapEvent("PermissionRequest") != state.StatusApprovalNeeded {
		t.Error("PermissionRequest should map to approval_needed")
	}
	if a.MapEvent("Stop") != state.StatusCompleted {
		t.Error("Stop should map to completed")
	}
	if a.MapEvent("StopFailure") != state.StatusCompleted {
		t.Error("StopFailure should map to completed")
	}
	if a.MapEvent("Unknown") != state.StatusIdle {
		t.Error("Unknown should map to idle")
	}
}
