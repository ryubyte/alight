package hookgen

import "fmt"

// Config holds the configuration for generating Codex hooks TOML.
type Config struct {
	ServerAddr string
}

// DefaultServerAddr is the default address for the Codex Bar server.
const DefaultServerAddr = "localhost:9876"

// events is the list of all Codex hook events.
var events = []string{
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

// Generate produces a Codex hooks TOML configuration snippet from the given Config.
// If cfg.ServerAddr is empty, DefaultServerAddr is used.
func Generate(cfg Config) string {
	addr := cfg.ServerAddr
	if addr == "" {
		addr = DefaultServerAddr
	}

	result := "# Codex Bar - Status Light Hooks\n"
	result += "# Add the following to ~/.codex/config.toml\n"

	for _, event := range events {
		result += "\n"
		result += fmt.Sprintf("[[hooks.%s]]\n", event)
		result += fmt.Sprintf("hooks = [{ type = \"command\", command = \"curl -s -X POST http://%s/update -d '{\\\"event\\\":\\\"%s\\\"}'\", async = true }]\n", addr, event)
	}

	return result
}
