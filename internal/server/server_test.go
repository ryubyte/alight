package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ryubyte/codex-bar/internal/state"
)

func newTestServer() (*Server, *state.Machine) {
	m := state.NewMachine()
	s := New(m, ":0")
	return s, m
}

func TestPostUpdate_ChangesStatus(t *testing.T) {
	s, _ := newTestServer()

	body := `{"status":"running","event":"SessionStart","session_id":"sess-1"}`
	req := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.handleUpdate(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if w.Body.String() != "ok" {
		t.Errorf("body = %q, want %q", w.Body.String(), "ok")
	}
}

func TestPostUpdate_DeriveStatusFromEvent(t *testing.T) {
	s, m := newTestServer()

	// Don't pass status; TransitionFromHook("PermissionRequest") -> StatusApprovalNeeded
	body := `{"event":"PermissionRequest","session_id":"sess-2"}`
	req := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.handleUpdate(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if m.Current() != state.StatusApprovalNeeded {
		t.Errorf("current = %q, want %q", m.Current(), state.StatusApprovalNeeded)
	}
}

func TestGetStatus_ReturnsCurrentState(t *testing.T) {
	s, m := newTestServer()

	// First, update to running
	m.Update(state.Event{
		Status:    state.StatusRunning,
		EventName: "SessionStart",
		SessionID: "sess-1",
	})

	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	w := httptest.NewRecorder()

	s.handleStatus(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp StatusResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Status != string(state.StatusRunning) {
		t.Errorf("status = %q, want %q", resp.Status, state.StatusRunning)
	}
	if len(resp.History) != 1 {
		t.Fatalf("history length = %d, want 1", len(resp.History))
	}
	if resp.History[0].EventName != "SessionStart" {
		t.Errorf("history[0].EventName = %q, want %q", resp.History[0].EventName, "SessionStart")
	}
	if resp.UpdatedAt == "" {
		t.Error("updated_at is empty, want a timestamp")
	}
}

func TestGetUpdate_MethodNotAllowed(t *testing.T) {
	s, _ := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/update", nil)
	w := httptest.NewRecorder()

	s.handleUpdate(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}
