// Package claudecfg manages Claude Code hooks configuration in ~/.claude/settings.json.
package claudecfg

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ConfigPathFn returns the path to Claude Code settings.json.
// Can be overridden in tests.
var ConfigPathFn = defaultConfigPath

func defaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".claude", "settings.json")
}

// hookEntry represents a single hook command in settings.json.
type hookEntry struct {
	Type    string `json:"type"`
	Command string `json:"command"`
	Timeout int    `json:"timeout"`
}

// hookGroup represents a group of hooks for one event, with an optional matcher.
type hookGroup struct {
	Matcher string      `json:"matcher,omitempty"`
	Hooks   []hookEntry `json:"hooks"`
}

// Settings represents the Claude Code settings.json structure.
type Settings map[string]interface{}

// Read loads settings.json. Returns empty Settings if file doesn't exist.
func Read() (Settings, error) {
	path := ConfigPathFn()
	if path == "" {
		return nil, fmt.Errorf("claudecfg: config path is empty")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Settings{}, nil
		}
		return nil, fmt.Errorf("claudecfg: read %s: %w", path, err)
	}

	var s Settings
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("claudecfg: parse %s: %w", path, err)
	}
	return s, nil
}

// Write saves settings.json with pretty-printed JSON.
func Write(s Settings) error {
	path := ConfigPathFn()
	if path == "" {
		return fmt.Errorf("claudecfg: config path is empty")
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("claudecfg: marshal: %w", err)
	}
	data = append(data, '\n')

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("claudecfg: mkdir: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// isCodexBarHook checks if a hook entry was injected by codex-bar.
func isCodexBarHook(h hookEntry) bool {
	return strings.Contains(h.Command, "codex-bar")
}

// containsCodexBarURL checks if any hook group contains codex-bar hooks.
func containsCodexBarURL(groups []interface{}) bool {
	for _, g := range groups {
		m, ok := g.(map[string]interface{})
		if !ok {
			continue
		}
		hooks, ok := m["hooks"].([]interface{})
		if !ok {
			continue
		}
		for _, h := range hooks {
			hm, ok := h.(map[string]interface{})
			if !ok {
				continue
			}
			cmd, _ := hm["command"].(string)
			if strings.Contains(cmd, "codex-bar") {
				return true
			}
		}
	}
	return false
}

// Cleanup removes all codex-bar injected hooks from settings.
func Cleanup() error {
	s, err := Read()
	if err != nil {
		return err
	}

	hooks, ok := s["hooks"]
	if !ok {
		return nil
	}
	hooksMap, ok := hooks.(map[string]interface{})
	if !ok {
		return nil
	}

	modified := false
	for event, groups := range hooksMap {
		groupsList, ok := groups.([]interface{})
		if !ok {
			continue
		}

		var filtered []interface{}
		for _, g := range groupsList {
			gm, ok := g.(map[string]interface{})
			if !ok {
				filtered = append(filtered, g)
				continue
			}

			hooksList, ok := gm["hooks"].([]interface{})
			if !ok {
				filtered = append(filtered, g)
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
				if !strings.Contains(cmd, "codex-bar") {
					cleanHooks = append(cleanHooks, h)
				} else {
					modified = true
				}
			}

			if len(cleanHooks) > 0 {
				gm["hooks"] = cleanHooks
				filtered = append(filtered, g)
			} else {
				modified = true
			}
		}

		if len(filtered) > 0 {
			hooksMap[event] = filtered
		} else {
			delete(hooksMap, event)
			modified = true
		}
	}

	if len(hooksMap) == 0 {
		delete(s, "hooks")
	}

	if modified {
		return Write(s)
	}
	return nil
}

// events to inject hooks for.
var events = []string{
	"SessionStart",
	"UserPromptSubmit",
	"Stop",
	"StopFailure",
	"PermissionRequest",
}

// Inject adds codex-bar hooks to Claude Code settings.
// It first cleans up any existing codex-bar hooks, then injects new ones.
func Inject(port string) error {
	// Cleanup first to avoid duplicates
	if err := Cleanup(); err != nil {
		return fmt.Errorf("claudecfg: cleanup before inject: %w", err)
	}

	s, err := Read()
	if err != nil {
		return err
	}

	hooks, ok := s["hooks"]
	if !ok {
		hooks = map[string]interface{}{}
		s["hooks"] = hooks
	}
	hooksMap, ok := hooks.(map[string]interface{})
	if !ok {
		hooksMap = map[string]interface{}{}
		s["hooks"] = hooksMap
	}

	for _, event := range events {
		group := map[string]interface{}{
			"hooks": []interface{}{
				map[string]interface{}{
					"type":    "command",
					"command": fmt.Sprintf("curl -s -X POST 'http://localhost:%s/update?source=codex-bar' -d '{\"event\":\"%s\"}'", port, event),
					"timeout": 5,
				},
			},
		}

		existing, ok := hooksMap[event]
		if !ok {
			hooksMap[event] = []interface{}{group}
		} else {
			existingList, ok := existing.([]interface{})
			if !ok {
				existingList = []interface{}{}
			}
			hooksMap[event] = append(existingList, group)
		}
	}

	return Write(s)
}
