// Package claude implements the Adapter interface for Anthropic Claude Code.
package claude

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ryubyte/aglight/internal/core/state"
)

// SettingsPathFn returns the path to Claude Code settings.json.
// Can be overridden in tests.
var SettingsPathFn = defaultSettingsPath

func defaultSettingsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("claude: get home dir: %w", err)
	}
	return filepath.Join(home, ".claude", "settings.json"), nil
}

// Adapter implements core.Adapter for Claude Code.
type Adapter struct{}

// New creates a new Claude Code adapter.
func New() *Adapter {
	return &Adapter{}
}

// Name returns the adapter name.
func (a *Adapter) Name() string {
	return "claude"
}

// IsInstalled returns true if the Claude Code settings file exists.
func (a *Adapter) IsInstalled() bool {
	path, err := SettingsPathFn()
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}

// Inject adds hook entries for Claude Code events that POST to the given port.
// If Claude Code is not installed (settings.json does not exist), it skips silently.
func (a *Adapter) Inject(port string) error {
	path, err := SettingsPathFn()
	if err != nil {
		return fmt.Errorf("claude: get settings path: %w", err)
	}

	// Skip if Claude Code is not installed
	if _, err := os.Stat(path); err != nil {
		return nil
	}

	settings, err := Read()
	if err != nil {
		return fmt.Errorf("claude: read settings: %w", err)
	}
	settings = injectHooks(settings, port)
	return Write(settings)
}

// Cleanup removes all aglight injected hooks from the settings.
// If Claude Code is not installed (settings.json does not exist), it skips silently.
func (a *Adapter) Cleanup() error {
	path, err := SettingsPathFn()
	if err != nil {
		return fmt.Errorf("claude: get settings path: %w", err)
	}

	// Skip if settings doesn't exist
	if _, err := os.Stat(path); err != nil {
		return nil
	}

	settings, err := Read()
	if err != nil {
		return fmt.Errorf("claude: read settings for cleanup: %w", err)
	}
	settings = cleanupHooks(settings)
	return Write(settings)
}

// MapEvent maps a Claude Code event name to a Status.
func (a *Adapter) MapEvent(eventName string) state.Status {
	switch eventName {
	case "SessionStart", "UserPromptSubmit":
		return state.StatusRunning
	case "PermissionRequest":
		return state.StatusApprovalNeeded
	case "Stop", "StopFailure":
		return state.StatusCompleted
	default:
		return state.StatusIdle
	}
}

// Read loads the Claude Code settings.json. Returns empty map if file doesn't exist.
func Read() (map[string]interface{}, error) {
	path, err := SettingsPathFn()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]interface{}{}, nil
		}
		return nil, fmt.Errorf("claude: read %s: %w", path, err)
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("claude: parse %s: %w", path, err)
	}
	return settings, nil
}

// Write saves the Claude Code settings.json.
func Write(settings map[string]interface{}) error {
	path, err := SettingsPathFn()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("claude: marshal: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("claude: mkdir: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// isAglightHook checks if a hook entry was injected by aglight.
func isAglightHook(hook map[string]interface{}) bool {
	cmd, _ := hook["command"].(string)
	return strings.Contains(cmd, "source=aglight")
}

func cleanupHooks(settings map[string]interface{}) map[string]interface{} {
	hooksRaw, ok := settings["hooks"]
	if !ok {
		return settings
	}
	hooksMap, ok := hooksRaw.(map[string]interface{})
	if !ok {
		return settings
	}

	for event, groups := range hooksMap {
		groupList, ok := groups.([]interface{})
		if !ok {
			continue
		}

		var cleanGroups []interface{}
		for _, g := range groupList {
			group, ok := g.(map[string]interface{})
			if !ok {
				cleanGroups = append(cleanGroups, g)
				continue
			}

			hooksList, ok := group["hooks"].([]interface{})
			if !ok {
				cleanGroups = append(cleanGroups, g)
				continue
			}

			var cleanHooks []interface{}
			for _, h := range hooksList {
				hm, ok := h.(map[string]interface{})
				if !ok || !isAglightHook(hm) {
					cleanHooks = append(cleanHooks, h)
				}
			}

			if len(cleanHooks) > 0 {
				group["hooks"] = cleanHooks
				cleanGroups = append(cleanGroups, group)
			}
		}

		if len(cleanGroups) > 0 {
			hooksMap[event] = cleanGroups
		} else {
			delete(hooksMap, event)
		}
	}

	if len(hooksMap) == 0 {
		delete(settings, "hooks")
	}

	return settings
}

var claudeEvents = []string{
	"SessionStart",
	"UserPromptSubmit",
	"Stop",
	"StopFailure",
	"PermissionRequest",
}

func injectHooks(settings map[string]interface{}, port string) map[string]interface{} {
	settings = cleanupHooks(settings)

	hooksRaw, ok := settings["hooks"]
	if !ok {
		hooksRaw = map[string]interface{}{}
		settings["hooks"] = hooksRaw
	}
	hooksMap, ok := hooksRaw.(map[string]interface{})
	if !ok {
		hooksMap = map[string]interface{}{}
		settings["hooks"] = hooksMap
	}

	for _, event := range claudeEvents {
		hook := map[string]interface{}{
			"type":    "command",
			"command": fmt.Sprintf("curl -s -X POST 'http://localhost:%s/update?source=aglight' -d '{\"event\":\"%s\"}'", port, event),
			"timeout": 5,
		}

		existing, ok := hooksMap[event].([]interface{})
		if ok && len(existing) > 0 {
			// Add to existing group's hooks array
			firstGroup, ok := existing[0].(map[string]interface{})
			if ok {
				hooksList, _ := firstGroup["hooks"].([]interface{})
				firstGroup["hooks"] = append(hooksList, hook)
				hooksMap[event] = existing
				continue
			}
		}

		// Create new group
		hooksMap[event] = []interface{}{
			map[string]interface{}{
				"hooks": []interface{}{hook},
			},
		}
	}

	return settings
}
