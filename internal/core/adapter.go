// Package core contains the framework-agnostic state machine, HTTP server,
// and adapter interface. It has zero knowledge of any specific AI tool.
package core

import (
	"fmt"
	"log"
	"sync"

	"github.com/ryubyte/aglight/internal/core/state"
)

// Adapter is the interface that each AI tool adapter must implement.
// Adapters are responsible for hook injection/cleanup and event-to-status mapping.
type Adapter interface {
	// Name returns a unique identifier for the adapter (e.g. "codex", "claude").
	Name() string

	// IsInstalled returns true if the tool's config file exists on this machine.
	IsInstalled() bool

	// Inject adds hooks for this tool, pointing to the given port.
	Inject(port string) error

	// Cleanup removes all hooks previously injected by this adapter.
	Cleanup() error

	// MapEvent translates a tool-specific event name to a Status.
	MapEvent(eventName string) state.Status
}

// Registry manages a collection of adapters.
type Registry struct {
	mu       sync.RWMutex
	adapters map[string]Adapter
}

// NewRegistry creates an empty Registry.
func NewRegistry() *Registry {
	return &Registry{
		adapters: make(map[string]Adapter),
	}
}

// Register adds an adapter to the registry.
func (r *Registry) Register(a Adapter) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.adapters[a.Name()] = a
}

// InjectAll calls Inject on every registered adapter.
// Errors are logged but do not stop other adapters from being injected.
func (r *Registry) InjectAll(port string) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, a := range r.adapters {
		if err := a.Inject(port); err != nil {
			log.Printf("warning: inject %s hooks: %v", a.Name(), err)
		}
	}
}

// CleanupAll calls Cleanup on every registered adapter.
// Errors are logged but do not stop other adapters from being cleaned up.
func (r *Registry) CleanupAll() {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, a := range r.adapters {
		if err := a.Cleanup(); err != nil {
			log.Printf("warning: cleanup %s hooks: %v", a.Name(), err)
		}
	}
}

// MapEvent finds the adapter by name and delegates event mapping.
// If the adapter is not found, it falls back to a merged mapping from all adapters.
func (r *Registry) MapEvent(source, eventName string) state.Status {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Try specific adapter first
	if source != "" {
		if a, ok := r.adapters[source]; ok {
			return a.MapEvent(eventName)
		}
	}

	// Fallback: try all adapters, first non-idle wins
	for _, a := range r.adapters {
		if s := a.MapEvent(eventName); s != state.StatusIdle {
			return s
		}
	}
	return state.StatusIdle
}

// AdapterNames returns the names of all registered adapters.
func (r *Registry) AdapterNames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.adapters))
	for name := range r.adapters {
		names = append(names, name)
	}
	return names
}

// InstalledAdapters returns the names of adapters whose tools are installed.
func (r *Registry) InstalledAdapters() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.adapters))
	for _, a := range r.adapters {
		if a.IsInstalled() {
			names = append(names, a.Name())
		}
	}
	return names
}

// StatusLabel returns a human-readable label for the given status.
func StatusLabel(s state.Status) string {
	switch s {
	case state.StatusIdle:
		return "空闲"
	case state.StatusRunning:
		return "运行中"
	case state.StatusApprovalNeeded:
		return "需要审批"
	case state.StatusCompleted:
		return "已完成"
	default:
		return fmt.Sprintf("未知(%s)", s)
	}
}
