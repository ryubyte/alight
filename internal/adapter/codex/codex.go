// Package codex implements the Adapter interface for OpenAI Codex CLI.
package codex

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"github.com/ryubyte/aglight/internal/core/state"
)

// ConfigPathFn returns the path to Codex CLI config.toml.
// Can be overridden in tests.
var ConfigPathFn = defaultConfigPath

func defaultConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("codex: get home dir: %w", err)
	}
	return filepath.Join(home, ".codex", "config.toml"), nil
}

// CodexConfig is a map representing the TOML config file.
type CodexConfig map[string]interface{}

// Adapter implements core.Adapter for Codex CLI.
type Adapter struct{}

// New creates a new Codex CLI adapter.
func New() *Adapter {
	return &Adapter{}
}

// Name returns the adapter name.
func (a *Adapter) Name() string {
	return "codex"
}

// IsInstalled returns true if the Codex CLI config file exists.
func (a *Adapter) IsInstalled() bool {
	path, err := ConfigPathFn()
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}

// Inject adds hook entries for all Codex events that POST to the given serverAddr.
// If Codex CLI is not installed (config.toml does not exist), it skips silently.
func (a *Adapter) Inject(port string) error {
	path, err := ConfigPathFn()
	if err != nil {
		return fmt.Errorf("codex: get config path: %w", err)
	}

	// Skip if Codex CLI is not installed
	if _, err := os.Stat(path); err != nil {
		return nil
	}

	cfg, err := Read()
	if err != nil {
		return fmt.Errorf("codex: read config: %w", err)
	}
	serverAddr := fmt.Sprintf("localhost:%s", port)
	cfg = injectHooks(cfg, serverAddr)
	return Write(cfg)
}

// Cleanup removes all aglight injected hooks from the config.
// If Codex CLI is not installed (config.toml does not exist), it skips silently.
func (a *Adapter) Cleanup() error {
	path, err := ConfigPathFn()
	if err != nil {
		return fmt.Errorf("codex: get config path: %w", err)
	}

	// Skip if config doesn't exist
	if _, err := os.Stat(path); err != nil {
		return nil
	}

	cfg, err := Read()
	if err != nil {
		return fmt.Errorf("codex: read config for cleanup: %w", err)
	}
	cfg = cleanupHooks(cfg)
	return Write(cfg)
}

// MapEvent maps a Codex CLI event name to a Status.
func (a *Adapter) MapEvent(eventName string) state.Status {
	switch eventName {
	case "SessionStart", "PreToolUse", "PostToolUse", "UserPromptSubmit",
		"SubagentStart", "PreCompact", "PostCompact":
		return state.StatusRunning
	case "PermissionRequest":
		return state.StatusApprovalNeeded
	case "Stop", "StopFailure", "SubagentStop":
		return state.StatusCompleted
	default:
		return state.StatusIdle
	}
}

// Read loads the Codex CLI config.toml. Returns empty CodexConfig if file doesn't exist.
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
		return nil, fmt.Errorf("codex: read %s: %w", path, err)
	}

	var cfg CodexConfig
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("codex: parse %s: %w", path, err)
	}
	return cfg, nil
}

// Write saves the Codex CLI config.toml.
func Write(cfg CodexConfig) error {
	path, err := ConfigPathFn()
	if err != nil {
		return err
	}

	data, err := toml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("codex: marshal: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("codex: mkdir: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// containsAglightURL checks if a command string was injected by aglight.
func containsAglightURL(cmd string) bool {
	return strings.Contains(cmd, "source=aglight")
}

func cleanupHooks(cfg CodexConfig) CodexConfig {
	if cfg == nil {
		return cfg
	}
	hooksRaw, ok := cfg["hooks"]
	if !ok {
		return cfg
	}
	hooks, ok := hooksRaw.(map[string]interface{})
	if !ok {
		return cfg
	}

	for event, entries := range hooks {
		entriesList, ok := entries.([]interface{})
		if !ok {
			continue
		}

		var filtered []interface{}
		for _, entry := range entriesList {
			entryMap, ok := entry.(map[string]interface{})
			if !ok {
				filtered = append(filtered, entry)
				continue
			}

			hooksList, ok := entryMap["hooks"].([]interface{})
			if !ok {
				filtered = append(filtered, entry)
				continue
			}

			var cleanHooks []interface{}
			for _, h := range hooksList {
				hm, ok := h.(map[string]interface{})
				if !ok {
					cleanHooks = append(cleanHooks, h)
					continue
				}
				cmd, _ := hm["command"].(string)
				if !containsAglightURL(cmd) {
					cleanHooks = append(cleanHooks, h)
				}
			}

			if len(cleanHooks) > 0 {
				entryMap["hooks"] = cleanHooks
				filtered = append(filtered, entry)
			}
		}

		if len(filtered) > 0 {
			hooks[event] = filtered
		} else {
			delete(hooks, event)
		}
	}

	if len(hooks) == 0 {
		delete(cfg, "hooks")
	}

	return cfg
}

var codexEvents = []string{
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

func injectHooks(cfg CodexConfig, serverAddr string) CodexConfig {
	if cfg == nil {
		cfg = CodexConfig{}
	}
	cfg = cleanupHooks(cfg)

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

	for _, event := range codexEvents {
		entry := map[string]interface{}{
			"hooks": []interface{}{
				map[string]interface{}{
					"type":    "command",
					"command": fmt.Sprintf("curl -s -X POST 'http://%s/update?source=aglight' -d '{\"event\":\"%s\"}'", serverAddr, event),
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
