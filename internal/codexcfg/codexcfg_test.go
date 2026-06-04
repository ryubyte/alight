package codexcfg

import (
	"path/filepath"
	"testing"
)

func TestCleanup_RemovesCodexBarEntries(t *testing.T) {
	cfg := CodexConfig{
		"hooks": map[string]interface{}{
			"SessionStart": []interface{}{
				map[string]interface{}{
					"hooks": []interface{}{
						map[string]interface{}{
							"type":    "command",
							"command": "curl -s -X POST http://localhost:9876/update -d '{\"event\":\"SessionStart\"}' &",
						},
					},
				},
				map[string]interface{}{
					"hooks": []interface{}{
						map[string]interface{}{
							"type":    "command",
							"command": "echo user-script",
						},
					},
				},
			},
		},
	}

	cleaned := Cleanup(cfg)

	hooks := cleaned["hooks"].(map[string]interface{})
	entries := hooks["SessionStart"].([]interface{})
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry after cleanup, got %d", len(entries))
	}

	entry := entries[0].(map[string]interface{})
	hooksField := entry["hooks"].([]interface{})
	cmd := hooksField[0].(map[string]interface{})["command"].(string)
	if cmd != "echo user-script" {
		t.Fatalf("expected user script to remain, got %s", cmd)
	}
}

func TestCleanup_EmptyHooksSectionRemoved(t *testing.T) {
	cfg := CodexConfig{
		"hooks": map[string]interface{}{
			"SessionStart": []interface{}{
				map[string]interface{}{
					"hooks": []interface{}{
						map[string]interface{}{
							"type":    "command",
							"command": "curl -s -X POST http://localhost:9876/update -d '{\"event\":\"SessionStart\"}' &",
						},
					},
				},
			},
		},
	}

	cleaned := Cleanup(cfg)

	if _, exists := cleaned["hooks"]; exists {
		t.Fatal("expected hooks section to be removed when empty")
	}
}

func TestInject_AddsEntries(t *testing.T) {
	cfg := CodexConfig{}
	cfg = Inject(cfg, "localhost:9876")

	hooks := cfg["hooks"].(map[string]interface{})
	if len(hooks) != 10 {
		t.Fatalf("expected 10 hook events, got %d", len(hooks))
	}

	entries := hooks["SessionStart"].([]interface{})
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	entry := entries[0].(map[string]interface{})
	hooksField := entry["hooks"].([]interface{})
	cmd := hooksField[0].(map[string]interface{})["command"].(string)
	if cmd != "curl -s -X POST http://localhost:9876/update -d '{\"event\":\"SessionStart\"}' &" {
		t.Fatalf("unexpected command: %s", cmd)
	}
}

func TestInject_CleansUpOldEntriesFirst(t *testing.T) {
	cfg := CodexConfig{}

	// First inject on port 9876
	cfg = Inject(cfg, "localhost:9876")

	// Now inject on port 9999 (simulating restart with different port)
	cfg = Inject(cfg, "localhost:9999")

	hooks := cfg["hooks"].(map[string]interface{})
	entries := hooks["SessionStart"].([]interface{})

	// Should only have 1 entry (old 9876 cleaned up, new 9999 added)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry after re-inject, got %d", len(entries))
	}

	entry := entries[0].(map[string]interface{})
	hooksField := entry["hooks"].([]interface{})
	cmd := hooksField[0].(map[string]interface{})["command"].(string)
	if cmd != "curl -s -X POST http://localhost:9999/update -d '{\"event\":\"SessionStart\"}' &" {
		t.Fatalf("unexpected command: %s", cmd)
	}
}

func TestInject_PreservesUserEntries(t *testing.T) {
	cfg := CodexConfig{
		"hooks": map[string]interface{}{
			"SessionStart": []interface{}{
				map[string]interface{}{
					"hooks": []interface{}{
						map[string]interface{}{
							"type":    "command",
							"command": "echo user-hook",
						},
					},
				},
			},
		},
	}

	cfg = Inject(cfg, "localhost:9876")

	hooks := cfg["hooks"].(map[string]interface{})
	entries := hooks["SessionStart"].([]interface{})

	// Should have 2 entries: user's + ours
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
}

func TestReadWriteRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	orig := ConfigPathFn
	ConfigPathFn = func() (string, error) { return filepath.Join(tmpDir, ".codex", "config.toml"), nil }
	defer func() { ConfigPathFn = orig }()

	cfg := CodexConfig{
		"features": map[string]interface{}{
			"js_repl": false,
		},
	}

	if err := Write(cfg); err != nil {
		t.Fatalf("write: %v", err)
	}

	read, err := Read()
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	features := read["features"].(map[string]interface{})
	if features["js_repl"] != false {
		t.Fatalf("round trip failed: %v", features["js_repl"])
	}
}

func TestRead_NonexistentFile(t *testing.T) {
	tmpDir := t.TempDir()
	orig := ConfigPathFn
	ConfigPathFn = func() (string, error) { return filepath.Join(tmpDir, ".codex", "config.toml"), nil }
	defer func() { ConfigPathFn = orig }()

	cfg, err := Read()
	if err != nil {
		t.Fatalf("expected no error for missing file, got: %v", err)
	}
	if len(cfg) != 0 {
		t.Fatalf("expected empty config, got %v", cfg)
	}
}

func TestCleanup_NoHooksSection(t *testing.T) {
	cfg := CodexConfig{
		"features": map[string]interface{}{
			"js_repl": false,
		},
	}

	cleaned := Cleanup(cfg)
	if _, exists := cleaned["hooks"]; exists {
		t.Fatal("expected no hooks section")
	}
	features := cleaned["features"].(map[string]interface{})
	if features["js_repl"] != false {
		t.Fatal("features section should be preserved")
	}
}

func TestContainsCodexBarURL(t *testing.T) {
	tests := []struct {
		cmd      string
		expected bool
	}{
		{"curl -s -X POST http://localhost:9876/update -d '{\"event\":\"SessionStart\"}' &", true},
		{"curl -s -X POST http://127.0.0.1:9876/update -d '{\"event\":\"SessionStart\"}' &", true},
		{"echo hello", false},
		{"curl -s -X POST https://example.com/api", false},
	}

	for _, tt := range tests {
		got := containsCodexBarURL(tt.cmd)
		if got != tt.expected {
			t.Errorf("containsCodexBarURL(%q) = %v, want %v", tt.cmd, got, tt.expected)
		}
	}
}
