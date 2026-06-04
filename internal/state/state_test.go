package state

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestNewMachine_InitialStateIsIdle(t *testing.T) {
	m := NewMachine()
	if got := m.Current(); got != StatusIdle {
		t.Errorf("NewMachine() initial status = %q, want %q", got, StatusIdle)
	}
}

func TestUpdate_ChangesStatus(t *testing.T) {
	m := NewMachine()

	newStatus := m.Update(Event{
		Status:    StatusRunning,
		EventName: "SessionStart",
		SessionID: "sess-1",
		Timestamp: time.Now(),
	})

	if newStatus != StatusRunning {
		t.Errorf("Update() returned %q, want %q", newStatus, StatusRunning)
	}
	if m.Current() != StatusRunning {
		t.Errorf("Current() after Update = %q, want %q", m.Current(), StatusRunning)
	}
}

func TestUpdate_CallbackFiredOnStateChange(t *testing.T) {
	m := NewMachine()

	var called int32
	m.OnChange(func(old, new Status, event Event) {
		if old != StatusIdle {
			t.Errorf("callback old = %q, want %q", old, StatusIdle)
		}
		if new != StatusRunning {
			t.Errorf("callback new = %q, want %q", new, StatusRunning)
		}
		if event.EventName != "SessionStart" {
			t.Errorf("callback event.EventName = %q, want %q", event.EventName, "SessionStart")
		}
		atomic.AddInt32(&called, 1)
	})

	m.Update(Event{
		Status:    StatusRunning,
		EventName: "SessionStart",
		SessionID: "sess-1",
		Timestamp: time.Now(),
	})

	if atomic.LoadInt32(&called) != 1 {
		t.Errorf("callback called %d times, want 1", called)
	}
}

func TestUpdate_NoCallbackWhenSameStatus(t *testing.T) {
	m := NewMachine()

	// First transition to running
	m.Update(Event{
		Status:    StatusRunning,
		EventName: "SessionStart",
		Timestamp: time.Now(),
	})

	var called int32
	m.OnChange(func(old, new Status, event Event) {
		atomic.AddInt32(&called, 1)
	})

	// Update with same status — callback should NOT fire
	m.Update(Event{
		Status:    StatusRunning,
		EventName: "PreToolUse",
		Timestamp: time.Now(),
	})

	if atomic.LoadInt32(&called) != 0 {
		t.Errorf("callback called %d times on same-status update, want 0", called)
	}
}

func TestHistory_ReturnsCopy(t *testing.T) {
	m := NewMachine()
	ev1 := Event{Status: StatusRunning, EventName: "SessionStart", Timestamp: time.Now()}
	m.Update(ev1)

	h := m.History()
	if len(h) != 1 {
		t.Fatalf("History() length = %d, want 1", len(h))
	}
	if h[0].EventName != "SessionStart" {
		t.Errorf("History()[0].EventName = %q, want %q", h[0].EventName, "SessionStart")
	}

	// Mutating the copy should not affect the machine
	h[0].EventName = "tampered"
	if m.History()[0].EventName == "tampered" {
		t.Error("modifying History() copy affected internal state")
	}
}

func TestTransitionFromHook(t *testing.T) {
	tests := []struct {
		hook string
		want Status
	}{
		{"SessionStart", StatusRunning},
		{"PreToolUse", StatusRunning},
		{"PostToolUse", StatusRunning},
		{"UserPromptSubmit", StatusRunning},
		{"SubagentStart", StatusRunning},
		{"PreCompact", StatusRunning},
		{"PostCompact", StatusRunning},
		{"PermissionRequest", StatusApprovalNeeded},
		{"Stop", StatusCompleted},
		{"SubagentStop", StatusCompleted},
		{"UnknownEvent", StatusIdle},
	}

	for _, tt := range tests {
		t.Run(tt.hook, func(t *testing.T) {
			got := TransitionFromHook(tt.hook)
			if got != tt.want {
				t.Errorf("TransitionFromHook(%q) = %q, want %q", tt.hook, got, tt.want)
			}
		})
	}
}
