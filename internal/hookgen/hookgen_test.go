package hookgen

import (
	"strings"
	"testing"
)

func TestGenerateContainsAllEvents(t *testing.T) {
	output := Generate(Config{ServerAddr: "localhost:9876"})

	expectedEvents := []string{
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

	for _, event := range expectedEvents {
		if !strings.Contains(output, event) {
			t.Errorf("generated config missing event: %s", event)
		}
		// Also verify the hook section header
		header := "[[hooks." + event + "]]"
		if !strings.Contains(output, header) {
			t.Errorf("generated config missing hook header: %s", header)
		}
	}
}

func TestGenerateUsesProvidedServerAddr(t *testing.T) {
	addr := "myhost:1234"
	output := Generate(Config{ServerAddr: addr})

	if !strings.Contains(output, "http://"+addr+"/update") {
		t.Errorf("generated config does not contain provided ServerAddr %q", addr)
	}

	// Make sure the command line references the custom addr and source marker
	expectedCmd := "curl -s -X POST 'http://" + addr + "/update?source=codex-bar'"
	if !strings.Contains(output, expectedCmd) {
		t.Errorf("generated config does not contain expected command with addr %q", addr)
	}
}

func TestGenerateDefaultServerAddr(t *testing.T) {
	output := Generate(Config{})

	if !strings.Contains(output, "http://"+DefaultServerAddr+"/update") {
		t.Errorf("generated config does not contain default ServerAddr %q", DefaultServerAddr)
	}

	// Also test with empty string explicitly
	output2 := Generate(Config{ServerAddr: ""})
	if !strings.Contains(output2, "http://"+DefaultServerAddr+"/update") {
		t.Errorf("generated config does not contain default ServerAddr when empty string provided")
	}
}
