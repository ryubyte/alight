package codex_test

import (
	"os"
	"path/filepath"
	"testing"

	adapter "github.com/ryubyte/aglight/internal/adapter/codex"
	"github.com/ryubyte/aglight/internal/core/state"
)

func tmpConfigPath(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	return filepath.Join(dir, ".codex", "config.toml")
}

func TestInject_SkipsIfNotInstalled(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nonexistent", "config.toml")
	orig := adapter.ConfigPathFn
	adapter.ConfigPathFn = func() (string, error) { return path, nil }
	defer func() { adapter.ConfigPathFn = orig }()

	a := adapter.New()
	if err := a.Inject("9876"); err != nil {
		t.Fatalf("should skip silently, got: %v", err)
	}

	// File should not be created
	if _, err := os.Stat(path); err == nil {
		t.Fatal("config file should not be created when tool is not installed")
	}
}

func TestCleanup_SkipsIfNotInstalled(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nonexistent", "config.toml")
	orig := adapter.ConfigPathFn
	adapter.ConfigPathFn = func() (string, error) { return path, nil }
	defer func() { adapter.ConfigPathFn = orig }()

	a := adapter.New()
	if err := a.Cleanup(); err != nil {
		t.Fatalf("should skip silently, got: %v", err)
	}
}

func TestInject(t *testing.T) {
	path := tmpConfigPath(t)
	orig := adapter.ConfigPathFn
	adapter.ConfigPathFn = func() (string, error) { return path, nil }
	defer func() { adapter.ConfigPathFn = orig }()

	// Create an empty config file first (tool is installed)
	os.MkdirAll(filepath.Dir(path), 0755)
	os.WriteFile(path, []byte(""), 0644)

	a := adapter.New()
	if err := a.Inject("9876"); err != nil {
		t.Fatalf("inject: %v", err)
	}

	cfg, err := adapter.Read()
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	hooks, ok := cfg["hooks"].(map[string]interface{})
	if !ok {
		t.Fatal("no hooks section")
	}
	if len(hooks) != 10 {
		t.Fatalf("expected 10 hook events, got %d", len(hooks))
	}
}

func TestCleanup(t *testing.T) {
	path := tmpConfigPath(t)
	orig := adapter.ConfigPathFn
	adapter.ConfigPathFn = func() (string, error) { return path, nil }
	defer func() { adapter.ConfigPathFn = orig }()

	a := adapter.New()
	a.Inject("9876")
	a.Cleanup()

	cfg, err := adapter.Read()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if _, exists := cfg["hooks"]; exists {
		t.Fatal("hooks should be removed after cleanup")
	}
}

func TestInjectPreservesUserConfig(t *testing.T) {
	path := tmpConfigPath(t)
	orig := adapter.ConfigPathFn
	adapter.ConfigPathFn = func() (string, error) { return path, nil }
	defer func() { adapter.ConfigPathFn = orig }()

	// Write user config first
	os.MkdirAll(filepath.Dir(path), 0755)
	cfg := adapter.CodexConfig{
		"features": map[string]interface{}{
			"js_repl": false,
		},
	}
	adapter.Write(cfg)

	a := adapter.New()
	a.Inject("9876")

	read, _ := adapter.Read()
	features := read["features"].(map[string]interface{})
	if features["js_repl"] != false {
		t.Fatal("user config should be preserved")
	}
}

func TestMapEvent(t *testing.T) {
	a := adapter.New()
	if a.MapEvent("SessionStart") != state.StatusRunning {
		t.Error("SessionStart should map to running")
	}
	if a.MapEvent("PermissionRequest") != state.StatusApprovalNeeded {
		t.Error("PermissionRequest should map to approval_needed")
	}
	if a.MapEvent("Stop") != state.StatusCompleted {
		t.Error("Stop should map to completed")
	}
	if a.MapEvent("Unknown") != state.StatusIdle {
		t.Error("Unknown should map to idle")
	}
}
