// Package state provides a concurrent-safe state machine for tracking AI tool status.
package state

import (
	"sync"
	"sync/atomic"
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
	Status    Status    `json:"status"`
	EventName string    `json:"event_name"`
	SessionID string    `json:"session_id"`
	ToolName  string    `json:"tool_name"`
	Timestamp time.Time `json:"timestamp"`
}

// StateChangeCallback is called when the machine transitions between states.
type StateChangeCallback func(old, new Status, event Event)

// nextCallbackID is used to assign unique IDs to registered callbacks.
var nextCallbackID atomic.Int64

// callbackEntry pairs a callback with its unique ID for safe removal.
type callbackEntry struct {
	id uint64
	cb StateChangeCallback
}

// Machine is a concurrent-safe state machine.
type Machine struct {
	mu        sync.RWMutex
	current   Status
	history   []Event
	callbacks []callbackEntry
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
	cbs := make([]callbackEntry, len(m.callbacks))
	copy(cbs, m.callbacks)
	m.mu.Unlock()

	if old != event.Status {
		for _, entry := range cbs {
			if entry.cb != nil {
				entry.cb(old, event.Status, event)
			}
		}
	}

	return event.Status
}

// OnChange registers a callback that fires on state changes.
// It returns an unregister function that removes the callback when called.
func (m *Machine) OnChange(cb StateChangeCallback) func() {
	id := uint64(nextCallbackID.Add(1))
	m.mu.Lock()
	m.callbacks = append(m.callbacks, callbackEntry{id: id, cb: cb})
	m.mu.Unlock()
	return func() {
		m.mu.Lock()
		defer m.mu.Unlock()
		for i, entry := range m.callbacks {
			if entry.id == id {
				m.callbacks = append(m.callbacks[:i], m.callbacks[i+1:]...)
				return
			}
		}
	}
}

// History returns a copy of the event history.
func (m *Machine) History() []Event {
	m.mu.RLock()
	defer m.mu.RUnlock()
	cp := make([]Event, len(m.history))
	copy(cp, m.history)
	return cp
}
