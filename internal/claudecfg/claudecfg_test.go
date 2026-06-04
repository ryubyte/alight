package claudecfg

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func tmpSettingsPath(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	return filepath.Join(dir, "settings.json")
}

func setupConfigPath(t *testing.T) {
	t.Helper()
	path := tmpSettingsPath(t)
	ConfigPathFn = func() string { return path }
	t.Cleanup(func() { ConfigPathFn = defaultConfigPath })
}

func writeTestSettings(t *testing.T, s Settings) {
	t.Helper()
	data, _ := json.MarshalIndent(s, "", "  ")
	data = append(data, '\n')
	os.MkdirAll(filepath.Dir(ConfigPathFn()), 0755)
	os.WriteFile(ConfigPathFn(), data, 0644)
}

func readTestSettings(t *testing.T) Settings {
	t.Helper()
	s, err := Read()
	if err != nil {
		t.Fatalf("read test settings: %v", err)
	}
	return s
}

func TestRead_NonExistent(t *testing.T) {
	setupConfigPath(t)
	s, err := Read()
	if err != nil {
		t.Fatalf("Read() error: %v", err)
	}
	if len(s) != 0 {
		t.Fatalf("expected empty, got %v", s)
	}
}

func TestWriteRead_RoundTrip(t *testing.T) {
	setupConfigPath(t)
	original := Settings{
		"model": "claude-sonnet-4",
		"hooks": map[string]interface{}{
			"SessionStart": []interface{}{
				map[string]interface{}{
					"hooks": []interface{}{
						map[string]interface{}{
							"type":    "command",
							"command": "echo hello",
							"timeout": 5,
						},
					},
				},
			},
		},
	}
	if err := Write(original); err != nil {
		t.Fatalf("Write() error: %v", err)
	}

	s := readTestSettings(t)
	if s["model"] != "claude-sonnet-4" {
		t.Fatalf("expected model=claude-sonnet-4, got %v", s["model"])
	}
}

func TestInject_ThenRead(t *testing.T) {
	setupConfigPath(t)
	writeTestSettings(t, Settings{"model": "claude-sonnet-4"})

	if err := Inject("9876"); err != nil {
		t.Fatalf("Inject() error: %v", err)
	}

	s := readTestSettings(t)
	hooks, ok := s["hooks"].(map[string]interface{})
	if !ok {
		t.Fatal("no hooks in settings")
	}

	// Verify SessionStart was injected
	groups, ok := hooks["SessionStart"].([]interface{})
	if !ok || len(groups) == 0 {
		t.Fatal("no SessionStart hooks")
	}

	group := groups[0].(map[string]interface{})
	hookList := group["hooks"].([]interface{})
	entry := hookList[0].(map[string]interface{})

	cmd, _ := entry["command"].(string)
	if cmd != "curl -s -X POST 'http://localhost:9876/update?source=codex-bar' -d '{\"event\":\"SessionStart\"}'" {
		t.Fatalf("unexpected command: %s", cmd)
	}
	if entry["timeout"] != float64(5) {
		t.Fatalf("expected timeout=5, got %v", entry["timeout"])
	}

	// Model should be preserved
	if s["model"] != "claude-sonnet-4" {
		t.Fatalf("model should be preserved, got %v", s["model"])
	}
}

func TestCleanup_RemovesOnlyCodexBarHooks(t *testing.T) {
	setupConfigPath(t)
	writeTestSettings(t, Settings{
		"model": "claude-sonnet-4",
		"hooks": map[string]interface{}{
			"SessionStart": []interface{}{
				map[string]interface{}{
					"hooks": []interface{}{
						map[string]interface{}{
							"type":    "command",
							"command": "echo hello",
							"timeout": 5,
						},
						map[string]interface{}{
							"type":    "command",
							"command": "curl -s -X POST 'http://localhost:9876/update?source=codex-bar' -d '{\"event\":\"SessionStart\"}'",
							"timeout": 5,
						},
					},
				},
			},
		},
	})

	if err := Cleanup(); err != nil {
		t.Fatalf("Cleanup() error: %v", err)
	}

	s := readTestSettings(t)
	hooks := s["hooks"].(map[string]interface{})
	groups := hooks["SessionStart"].([]interface{})
	group := groups[0].(map[string]interface{})
	hookList := group["hooks"].([]interface{})

	if len(hookList) != 1 {
		t.Fatalf("expected 1 hook after cleanup, got %d", len(hookList))
	}
	entry := hookList[0].(map[string]interface{})
	cmd, _ := entry["command"].(string)
	if cmd != "echo hello" {
		t.Fatalf("should keep non-codex-bar hook, got: %s", cmd)
	}
}

func TestInject_CleansUpFirst(t *testing.T) {
	setupConfigPath(t)

	// Inject twice — should not duplicate
	if err := Inject("9876"); err != nil {
		t.Fatalf("first Inject() error: %v", err)
	}
	if err := Inject("9999"); err != nil {
		t.Fatalf("second Inject() error: %v", err)
	}

	s := readTestSettings(t)
	hooks := s["hooks"].(map[string]interface{})
	groups := hooks["SessionStart"].([]interface{})

	// Should have only one codex-bar group (with port 9999)
	codexBarCount := 0
	for _, g := range groups {
		gm := g.(map[string]interface{})
		for _, h := range gm["hooks"].([]interface{}) {
			hm := h.(map[string]interface{})
			cmd, _ := hm["command"].(string)
			if cmd == "curl -s -X POST 'http://localhost:9999/update?source=codex-bar' -d '{\"event\":\"SessionStart\"}'" {
				codexBarCount++
			}
		}
	}
	if codexBarCount != 1 {
		t.Fatalf("expected 1 codex-bar hook for port 9999, got %d", codexBarCount)
	}
}

func TestCleanup_EmptyHooksRemovesHooksKey(t *testing.T) {
	setupConfigPath(t)

	// Only codex-bar hooks
	if err := Inject("9876"); err != nil {
		t.Fatalf("Inject() error: %v", err)
	}
	if err := Cleanup(); err != nil {
		t.Fatalf("Cleanup() error: %v", err)
	}

	s := readTestSettings(t)
	if _, ok := s["hooks"]; ok {
		t.Fatal("hooks key should be removed when empty")
	}
}
