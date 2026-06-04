package core

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/ryubyte/codex-bar/internal/core/state"
)

// UpdateRequest represents the JSON body for POST /update.
type UpdateRequest struct {
	Status    string `json:"status"`
	EventName string `json:"event"`
	Source    string `json:"source"`
	SessionID string `json:"session_id"`
	ToolName  string `json:"tool_name"`
}

// StatusResponse represents the JSON response for GET /status.
type StatusResponse struct {
	Status    string        `json:"status"`
	UpdatedAt string        `json:"updated_at"`
	History   []state.Event `json:"history"`
}

// Server is an HTTP server that exposes the state machine.
type Server struct {
	machine  *state.Machine
	registry *Registry
	addr     string
}

// NewServer creates a new Server.
func NewServer(machine *state.Machine, registry *Registry, addr string) *Server {
	return &Server{
		machine:  machine,
		registry: registry,
		addr:     addr,
	}
}

// Addr returns the server address.
func (s *Server) Addr() string {
	return s.addr
}

// Port returns the server port.
func (s *Server) Port() string {
	_, port, _ := net.SplitHostPort(s.addr)
	return port
}

// FindFreePort returns an available port on localhost starting from the given port.
func FindFreePort(startPort int) (int, error) {
	for port := startPort; port < startPort+100; port++ {
		addr := fmt.Sprintf("localhost:%d", port)
		ln, err := net.Listen("tcp", addr)
		if err == nil {
			ln.Close()
			return port, nil
		}
	}
	return 0, fmt.Errorf("no free port found in range %d-%d", startPort, startPort+99)
}

// ListenAndServe registers routes and starts the HTTP server.
func (s *Server) ListenAndServe() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/update", s.handleUpdate)
	mux.HandleFunc("/status", s.handleStatus)
	mux.HandleFunc("/events", s.handleEvents)
	return http.ListenAndServe(s.addr, mux)
}

func (s *Server) handleUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	var status state.Status
	if req.Status != "" {
		status = state.Status(req.Status)
	} else {
		status = s.registry.MapEvent(req.Source, req.EventName)
	}

	s.machine.Update(state.Event{
		Status:    status,
		EventName: req.EventName,
		SessionID: req.SessionID,
		ToolName:  req.ToolName,
		Timestamp: time.Now(),
	})

	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "ok")
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	current := s.machine.Current()
	history := s.machine.History()

	var updatedAt string
	if len(history) > 0 {
		updatedAt = history[len(history)-1].Timestamp.Format(time.RFC3339)
	}

	resp := StatusResponse{
		Status:    string(current),
		UpdatedAt: updatedAt,
		History:   history,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	unregister := s.machine.OnChange(func(old, new state.Status, event state.Event) {
		data, _ := json.Marshal(event)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	})

	fmt.Fprint(w, ": connected\n\n")
	flusher.Flush()

	<-r.Context().Done()
	unregister()
}
