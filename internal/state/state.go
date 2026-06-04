package state

import (
	"sync"
	"time"
)

// Status represents the current state of the machine.
type Status string

const (
	StatusIdle           Status = "idle"
	StatusRunning        Status = "running"
	StatusApprovalNeeded Status = "approval_needed"
	StatusCompleted      Status = "completed"
)

// Event represents a state transition event.
type Event struct {
	Status     Status    `json:"status"`
	EventName  string    `json:"event_name"`
	SessionID  string    `json:"session_id"`
	ToolName   string    `json:"tool_name"`
	Timestamp  time.Time `json:"timestamp"`
}

// StateChangeCallback is called when the machine transitions between states.
type StateChangeCallback func(old, new Status, event Event)

// Machine is a concurrent-safe state machine.
type Machine struct {
	mu        sync.RWMutex
	current   Status
	history   []Event
	callbacks []StateChangeCallback
}

// NewMachine creates a new Machine with initial status StatusIdle.
func NewMachine() *Machine {
	return &Machine{
		current: StatusIdle,
		history: make([]Event, 0),
	}
}

// Current returns the current status.
func (m *Machine) Current() Status {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.current
}

// Update applies an event, transitions state, and triggers callbacks if the status changed.
// It returns the new status.
func (m *Machine) Update(event Event) Status {
	m.mu.Lock()
	old := m.current
	m.current = event.Status
	m.history = append(m.history, event)
	cbs := make([]StateChangeCallback, len(m.callbacks))
	copy(cbs, m.callbacks)
	m.mu.Unlock()

	if old != event.Status {
		for _, cb := range cbs {
			cb(old, event.Status, event)
		}
	}

	return event.Status
}

// OnChange registers a callback that fires on state changes.
func (m *Machine) OnChange(cb StateChangeCallback) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callbacks = append(m.callbacks, cb)
}

// History returns a copy of the event history.
func (m *Machine) History() []Event {
	m.mu.RLock()
	defer m.mu.RUnlock()
	cp := make([]Event, len(m.history))
	copy(cp, m.history)
	return cp
}

// TransitionFromHook maps a hook event name to the corresponding Status.
func TransitionFromHook(hookEvent string) Status {
	switch hookEvent {
	case "SessionStart", "PreToolUse", "PostToolUse", "UserPromptSubmit",
		"SubagentStart", "PreCompact", "PostCompact":
		return StatusRunning
	case "PermissionRequest":
		return StatusApprovalNeeded
	case "Stop", "SubagentStop":
		return StatusCompleted
	default:
		return StatusIdle
	}
}
