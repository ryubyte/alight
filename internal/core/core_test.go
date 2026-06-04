package core_test

import (
	"testing"

	"github.com/ryubyte/codex-bar/internal/adapter/claude"
	"github.com/ryubyte/codex-bar/internal/adapter/codex"
	"github.com/ryubyte/codex-bar/internal/core"
	"github.com/ryubyte/codex-bar/internal/core/state"
)

type mockAdapter struct {
	name    string
	inject  func(port string) error
	cleanup func() error
	mapping map[string]state.Status
}

func (m *mockAdapter) Name() string                          { return m.name }
func (m *mockAdapter) Inject(port string) error               { return m.inject(port) }
func (m *mockAdapter) Cleanup() error                         { return m.cleanup() }
func (m *mockAdapter) MapEvent(eventName string) state.Status {
	if s, ok := m.mapping[eventName]; ok {
		return s
	}
	return state.StatusIdle
}

func TestRegistry_RegisterAndMap(t *testing.T) {
	r := core.NewRegistry()
	r.Register(&mockAdapter{
		name: "test",
		mapping: map[string]state.Status{
			"Start": state.StatusRunning,
		},
	})

	s := r.MapEvent("test", "Start")
	if s != state.StatusRunning {
		t.Fatalf("expected running, got %s", s)
	}
}

func TestRegistry_FallbackMerge(t *testing.T) {
	r := core.NewRegistry()
	r.Register(&mockAdapter{
		name: "a",
		mapping: map[string]state.Status{
			"Start": state.StatusRunning,
		},
	})
	r.Register(&mockAdapter{
		name: "b",
		mapping: map[string]state.Status{
			"Stop": state.StatusCompleted,
		},
	})

	// Unknown source → try all adapters
	s := r.MapEvent("", "Start")
	if s != state.StatusRunning {
		t.Fatalf("expected running, got %s", s)
	}

	s = r.MapEvent("", "Stop")
	if s != state.StatusCompleted {
		t.Fatalf("expected completed, got %s", s)
	}

	s = r.MapEvent("", "Unknown")
	if s != state.StatusIdle {
		t.Fatalf("expected idle for unknown, got %s", s)
	}
}

func TestRegistry_InjectAll(t *testing.T) {
	injected := map[string]string{}
	r := core.NewRegistry()
	r.Register(&mockAdapter{
		name:   "a",
		inject: func(port string) error { injected["a"] = port; return nil },
		cleanup: func() error { return nil },
	})
	r.Register(&mockAdapter{
		name:   "b",
		inject: func(port string) error { injected["b"] = port; return nil },
		cleanup: func() error { return nil },
	})

	r.InjectAll("9876")
	if injected["a"] != "9876" || injected["b"] != "9876" {
		t.Fatalf("expected both injected on 9876, got %v", injected)
	}
}

func TestRegistry_CleanupAll(t *testing.T) {
	cleaned := map[string]bool{}
	r := core.NewRegistry()
	r.Register(&mockAdapter{
		name:    "a",
		inject:  func(port string) error { return nil },
		cleanup: func() error { cleaned["a"] = true; return nil },
	})
	r.Register(&mockAdapter{
		name:    "b",
		inject:  func(port string) error { return nil },
		cleanup: func() error { cleaned["b"] = true; return nil },
	})

	r.CleanupAll()
	if !cleaned["a"] || !cleaned["b"] {
		t.Fatalf("expected both cleaned, got %v", cleaned)
	}
}

func TestCodexAdapter_MapEvent(t *testing.T) {
	a := codex.New()
	tests := []struct {
		event    string
		expected state.Status
	}{
		{"SessionStart", state.StatusRunning},
		{"PreToolUse", state.StatusRunning},
		{"PermissionRequest", state.StatusApprovalNeeded},
		{"Stop", state.StatusCompleted},
		{"SubagentStop", state.StatusCompleted},
		{"UnknownEvent", state.StatusIdle},
	}
	for _, tt := range tests {
		got := a.MapEvent(tt.event)
		if got != tt.expected {
			t.Errorf("codex.MapEvent(%q) = %s, want %s", tt.event, got, tt.expected)
		}
	}
}

func TestClaudeAdapter_MapEvent(t *testing.T) {
	a := claude.New()
	tests := []struct {
		event    string
		expected state.Status
	}{
		{"SessionStart", state.StatusRunning},
		{"UserPromptSubmit", state.StatusRunning},
		{"PermissionRequest", state.StatusApprovalNeeded},
		{"Stop", state.StatusCompleted},
		{"StopFailure", state.StatusCompleted},
		{"UnknownEvent", state.StatusIdle},
	}
	for _, tt := range tests {
		got := a.MapEvent(tt.event)
		if got != tt.expected {
			t.Errorf("claude.MapEvent(%q) = %s, want %s", tt.event, got, tt.expected)
		}
	}
}

func TestStatusLabel(t *testing.T) {
	tests := []struct {
		status   state.Status
		expected string
	}{
		{state.StatusIdle, "空闲"},
		{state.StatusRunning, "运行中"},
		{state.StatusApprovalNeeded, "需要审批"},
		{state.StatusCompleted, "已完成"},
	}
	for _, tt := range tests {
		got := core.StatusLabel(tt.status)
		if got != tt.expected {
			t.Errorf("StatusLabel(%s) = %s, want %s", tt.status, got, tt.expected)
		}
	}
}
