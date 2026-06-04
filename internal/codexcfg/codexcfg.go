package codexcfg

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// CodexConfig represents the ~/.codex/config.toml structure.
// We only care about the hooks section; everything else is
// preserved as-is through round-trip serialization.
type CodexConfig map[string]interface{}

// ConfigPathFn returns the path to the Codex CLI config file.
// Can be overridden in tests.
var ConfigPathFn = defaultConfigPath

func defaultConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir: %w", err)
	}
	return filepath.Join(home, ".codex", "config.toml"), nil
}

// Read reads and parses the Codex config file.
// Returns an empty config if the file doesn't exist.
func Read() (CodexConfig, error) {
	path, err := ConfigPathFn()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return CodexConfig{}, nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg CodexConfig
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return cfg, nil
}

// Write serializes and writes the config back to disk.
func Write(cfg CodexConfig) error {
	path, err := ConfigPathFn()
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	data, err := toml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}

// Cleanup removes all hook entries previously injected by codex-bar.
// It detects our entries by checking if the command contains the codex-bar update endpoint.
func Cleanup(cfg CodexConfig) CodexConfig {
	hooksRaw, ok := cfg["hooks"]
	if !ok {
		return cfg
	}

	hooks, ok := hooksRaw.(map[string]interface{})
	if !ok {
		return cfg
	}

	for eventName, entriesRaw := range hooks {
		entries, ok := entriesRaw.([]interface{})
		if !ok {
			continue
		}

		// Filter out entries that contain our marker URL
		var filtered []interface{}
		for _, entryRaw := range entries {
			entry, ok := entryRaw.(map[string]interface{})
			if !ok {
				filtered = append(filtered, entryRaw)
				continue
			}

			cmd, _ := entry["command"].(string)
			hooksField, _ := entry["hooks"].([]interface{})

			// Check if any hook command contains our endpoint
			isOurs := false
			if containsCodexBarURL(cmd) {
				isOurs = true
			}
			for _, h := range hooksField {
				if hm, ok := h.(map[string]interface{}); ok {
					if c, ok := hm["command"].(string); ok && containsCodexBarURL(c) {
						isOurs = true
					}
				}
			}

			if !isOurs {
				filtered = append(filtered, entryRaw)
			}
		}

		if len(filtered) == 0 {
			delete(hooks, eventName)
		} else {
			hooks[eventName] = filtered
		}
	}

	if len(hooks) == 0 {
		delete(cfg, "hooks")
	} else {
		cfg["hooks"] = hooks
	}

	return cfg
}

// Inject adds hook entries for all Codex events that POST to the given serverAddr.
// It cleans up any existing codex-bar entries first, then adds fresh ones.
func Inject(cfg CodexConfig, serverAddr string) CodexConfig {
	// Clean up any stale entries first
	cfg = Cleanup(cfg)

	events := []string{
		"SessionStart",
		"PreToolUse",
		"PostToolUse",
		"PermissionRequest",
		"UserPromptSubmit",
		"PreCompact",
		"PostCompact",
		"Stop",
		"SubagentStart",
		"SubagentStop",
	}

	// Ensure hooks section exists
	hooksRaw, ok := cfg["hooks"]
	if !ok {
		hooksRaw = map[string]interface{}{}
		cfg["hooks"] = hooksRaw
	}
	hooks, ok := hooksRaw.(map[string]interface{})
	if !ok {
		hooks = map[string]interface{}{}
		cfg["hooks"] = hooks
	}

	for _, event := range events {
		entry := map[string]interface{}{
			"hooks": []interface{}{
				map[string]interface{}{
					"type":    "command",
					"command": fmt.Sprintf("curl -s -X POST http://%s/update -d '{\"event\":\"%s\"}'", serverAddr, event),
					"async":   true,
				},
			},
		}

		existing, ok := hooks[event].([]interface{})
		if !ok {
			existing = nil
		}
		hooks[event] = append(existing, entry)
	}

	return cfg
}

// containsCodexBarURL checks if a command string contains the codex-bar update endpoint.
// It matches commands that reference /update on localhost or 127.0.0.1.
func containsCodexBarURL(cmd string) bool {
	if !strings.Contains(cmd, "/update") {
		return false
	}
	return strings.Contains(cmd, "localhost") || strings.Contains(cmd, "127.0.0.1")
}
