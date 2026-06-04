package state_test

import (
	"testing"

	"github.com/ryubyte/codex-bar/internal/core/state"
)

func TestMachine_Initial(t *testing.T) {
	m := state.NewMachine()
	if m.Current() != state.StatusIdle {
		t.Fatalf("expected idle, got %s", m.Current())
	}
}

func TestMachine_Update(t *testing.T) {
	m := state.NewMachine()

	s := m.Update(state.Event{Status: state.StatusRunning})
	if s != state.StatusRunning {
		t.Fatalf("expected running, got %s", s)
	}
	if m.Current() != state.StatusRunning {
		t.Fatalf("expected running, got %s", m.Current())
	}
}

func TestMachine_Callback(t *testing.T) {
	m := state.NewMachine()

	var gotOld, gotNew state.Status
	m.OnChange(func(old, new state.Status, event state.Event) {
		gotOld = old
		gotNew = new
	})

	m.Update(state.Event{Status: state.StatusRunning})

	if gotOld != state.StatusIdle {
		t.Fatalf("expected old=idle, got %s", gotOld)
	}
	if gotNew != state.StatusRunning {
		t.Fatalf("expected new=running, got %s", gotNew)
	}
}

func TestMachine_Unregister(t *testing.T) {
	m := state.NewMachine()
	called := false

	unreg := m.OnChange(func(old, new state.Status, event state.Event) {
		called = true
	})

	unreg()
	m.Update(state.Event{Status: state.StatusRunning})

	if called {
		t.Fatal("callback should not be called after unregister")
	}
}

func TestMachine_History(t *testing.T) {
	m := state.NewMachine()
	m.Update(state.Event{Status: state.StatusRunning, EventName: "start"})
	m.Update(state.Event{Status: state.StatusCompleted, EventName: "stop"})

	h := m.History()
	if len(h) != 2 {
		t.Fatalf("expected 2 events, got %d", len(h))
	}
	if h[0].EventName != "start" {
		t.Fatalf("expected start, got %s", h[0].EventName)
	}
	if h[1].EventName != "stop" {
		t.Fatalf("expected stop, got %s", h[1].EventName)
	}
}
