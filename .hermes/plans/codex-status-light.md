# Codex 状态展示灯 实现计划

> **For Hermes:** Use subagent-driven-development skill to implement this plan task-by-task.

**Goal:** 用 Go 实现 Codex CLI 状态展示灯，通过 Codex Hooks 接收状态变更，在 macOS 系统托盘显示红黄绿灯。

**Architecture:** Go 程序由 3 层组成：(1) HTTP Server 接收 Codex Hooks 推送的状态变更；(2) 状态机管理 idle/running/approval_needed/completed 四种状态转换；(3) macOS 系统托盘展示当前状态颜色图标。Codex 通过 `~/.codex/config.toml` 配置 hooks，每个事件触发时 curl 推送 JSON 到 Go server。

**Tech Stack:** Go 1.22+, getlantern/systray (系统托盘), net/http (内置), encoding/json (内置)

---

## 项目结构

```
aglight/
├── main.go                  # 入口，启动 systray + HTTP server
├── internal/
│   ├── state/
│   │   └── state.go         # 状态机：4 种状态 + 转换逻辑
│   ├── server/
│   │   └── server.go        # HTTP server：/update, /status, /events (SSE)
│   ├── icons/
│   │   └── icons.go         # 嵌入红/黄/绿/灰 PNG 图标数据
│   └── hookgen/
│       └── hookgen.go       # 生成 Codex hooks 配置文件
├── icons/
│   ├── idle.png             # 灰灯 22x22
│   ├── running.png          # 黄灯 22x22
│   ├── completed.png        # 绿灯 22x22
│   └── approval.png         # 红灯 22x22
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

---

## Task 1: 初始化 Go 项目

**Objective:** 创建 go.mod 和基础项目结构

**Files:**
- Create: `go.mod`
- Create: `internal/state/state.go` (占位)
- Create: `internal/server/server.go` (占位)
- Create: `internal/icons/icons.go` (占位)
- Create: `internal/hookgen/hookgen.go` (占位)

**Step 1: 初始化 Go module**

```bash
cd /Users/xiongshengyao/workspace/github.com/ryubyte/aglight
go mod init github.com/ryubyte/aglight
```

**Step 2: 创建目录结构**

```bash
mkdir -p internal/state internal/server internal/icons internal/hookgen icons
```

**Step 3: 创建占位文件**

`internal/state/state.go`:
```go
package state
```

`internal/server/server.go`:
```go
package server
```

`internal/icons/icons.go`:
```go
package icons
```

`internal/hookgen/hookgen.go`:
```go
package hookgen
```

**Step 4: 验证**

```bash
go build ./...
```

Expected: 编译通过，无错误

**Step 5: 提交**

```bash
git add -A
git commit -m "chore: init go project with directory structure"
```

---

## Task 2: 实现状态机

**Objective:** 实现 4 种状态（idle/running/approval_needed/completed）的状态机，包含线程安全的状态转换和事件回调

**Files:**
- Create: `internal/state/state.go`

**Step 1: 实现状态机**

```go
package state

import (
	"sync"
	"time"
)

type Status string

const (
	StatusIdle           Status = "idle"
	StatusRunning        Status = "running"
	StatusApprovalNeeded Status = "approval_needed"
	StatusCompleted      Status = "completed"
)

func (s Status) String() string {
	return string(s)
}

type Event struct {
	Status    Status    `json:"status"`
	EventName string    `json:"event"`
	SessionID string    `json:"session_id,omitempty"`
	ToolName  string    `json:"tool_name,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

type StateChangeCallback func(old, new Status, event Event)

type Machine struct {
	mu        sync.RWMutex
	current   Status
	history   []Event
	callbacks []StateChangeCallback
}

func NewMachine() *Machine {
	return &Machine{
		current: StatusIdle,
		history: make([]Event, 0),
	}
}

func (m *Machine) Current() Status {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.current
}

func (m *Machine) Update(event Event) Status {
	m.mu.Lock()
	defer m.mu.Unlock()

	old := m.current
	event.Timestamp = time.Now()
	m.history = append(m.history, event)
	m.current = event.Status

	if old != event.Status {
		for _, cb := range m.callbacks {
			cb(old, event.Status, event)
		}
	}

	return old
}

func (m *Machine) OnChange(cb StateChangeCallback) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callbacks = append(m.callbacks, cb)
}

func (m *Machine) History() []Event {
	m.mu.RLock()
	defer m.mu.RUnlock()
	cp := make([]Event, len(m.history))
	copy(cp, m.history)
	return cp
}

// TransitionFromHook maps a Codex hook event name to the target status.
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
		return StatusRunning
	}
}
```

**Step 2: 写测试**

`internal/state/state_test.go`:
```go
package state

import (
	"sync/atomic"
	"testing"
)

func TestNewMachineStartsIdle(t *testing.T) {
	m := NewMachine()
	if m.Current() != StatusIdle {
		t.Errorf("expected idle, got %s", m.Current())
	}
}

func TestUpdateChangesStatus(t *testing.T) {
	m := NewMachine()
	m.Update(Event{Status: StatusRunning, EventName: "SessionStart"})
	if m.Current() != StatusRunning {
		t.Errorf("expected running, got %s", m.Current())
	}
}

func TestCallbackFiresOnStatusChange(t *testing.T) {
	m := NewMachine()
	var called atomic.Int32
	m.OnChange(func(old, new Status, event Event) {
		if old == StatusIdle && new == StatusRunning {
			called.Add(1)
		}
	})
	m.Update(Event{Status: StatusRunning, EventName: "SessionStart"})
	if called.Load() != 1 {
		t.Errorf("expected callback to fire once, got %d", called.Load())
	}
}

func TestCallbackNotFiredOnSameStatus(t *testing.T) {
	m := NewMachine()
	m.Update(Event{Status: StatusRunning, EventName: "SessionStart"})
	var called atomic.Int32
	m.OnChange(func(old, new Status, event Event) {
		called.Add(1)
	})
	m.Update(Event{Status: StatusRunning, EventName: "PreToolUse"})
	if called.Load() != 0 {
		t.Errorf("expected no callback on same status, got %d", called.Load())
	}
}

func TestTransitionFromHook(t *testing.T) {
	tests := []struct {
		hook  string
		want  Status
	}{
		{"SessionStart", StatusRunning},
		{"PreToolUse", StatusRunning},
		{"PostToolUse", StatusRunning},
		{"PermissionRequest", StatusApprovalNeeded},
		{"Stop", StatusCompleted},
		{"SubagentStop", StatusCompleted},
	}
	for _, tt := range tests {
		got := TransitionFromHook(tt.hook)
		if got != tt.want {
			t.Errorf("TransitionFromHook(%q) = %q, want %q", tt.hook, got, tt.want)
		}
	}
}
```

**Step 3: 运行测试**

```bash
go test ./internal/state/ -v
```

Expected: 全部 PASS

**Step 4: 提交**

```bash
git add internal/state/
git commit -m "feat: implement state machine with 4 statuses and callbacks"
```

---

## Task 3: 生成托盘图标

**Objective:** 用 Go 代码生成 4 个 22x22 PNG 图标（灰/黄/绿/红圆点），嵌入到二进制中

**Files:**
- Create: `internal/icons/icons.go`

**Step 1: 实现 icon 生成器**

用 Go 标准库 `image` + `image/color` + `image/png` 在内存中生成圆形图标，并 embed 到二进制。

```go
package icons

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	_ "embed"
	"sync"

	"github.com/ryubyte/aglight/internal/state"
)

const size = 22

var (
	idleColor  = color.NRGBA{R: 128, G: 128, B: 128, A: 255} // 灰
	runColor   = color.NRGBA{R: 255, G: 200, B: 0,   A: 255} // 黄
	doneColor  = color.NRGBA{R: 0,   G: 200, B: 80,  A: 255} // 绿
	approvColor = color.NRGBA{R: 255, G: 50,  B: 50,  A: 255} // 红
)

var cache struct {
	sync.Once
	data map[state.Status][]byte
}

func generateCircle(c color.NRGBA) []byte {
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	center := float64(size) / 2
	radius := center - 2
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx := float64(x) + 0.5 - center
			dy := float64(y) + 0.5 - center
			if dx*dx+dy*dy <= radius*radius {
				img.SetNRGBA(x, y, c)
			}
		}
	}
	var buf bytes.Buffer
	png.Encode(&buf, img)
	return buf.Bytes()
}

func initCache() {
	cache.data = map[state.Status][]byte{
		state.StatusIdle:           generateCircle(idleColor),
		state.StatusRunning:        generateCircle(runColor),
		state.StatusCompleted:      generateCircle(doneColor),
		state.StatusApprovalNeeded: generateCircle(approvColor),
	}
}

func ForStatus(s state.Status) []byte {
	cache.Do(initCache)
	return cache.data[s]
}
```

**Step 2: 写测试**

`internal/icons/icons_test.go`:
```go
package icons

import (
	"testing"

	"github.com/ryubyte/aglight/internal/state"
)

func TestForStatusReturnsPNGData(t *testing.T) {
	statuses := []state.Status{
		state.StatusIdle,
		state.StatusRunning,
		state.StatusCompleted,
		state.StatusApprovalNeeded,
	}
	for _, s := range statuses {
		data := ForStatus(s)
		if len(data) == 0 {
			t.Errorf("ForStatus(%s) returned empty data", s)
		}
		// PNG magic bytes
		if data[0] != 0x89 || data[1] != 'P' {
			t.Errorf("ForStatus(%s) does not look like PNG", s)
		}
	}
}
```

**Step 3: 运行测试**

```bash
go test ./internal/icons/ -v
```

Expected: 全部 PASS

**Step 4: 提交**

```bash
git add internal/icons/
git commit -m "feat: generate and cache PNG circle icons for each status"
```

---

## Task 4: 实现 HTTP Server

**Objective:** 实现 HTTP server 提供 3 个端点：POST /update（hooks 推送）、GET /status（查询当前状态）、GET /events（SSE 事件流）

**Files:**
- Create: `internal/server/server.go`

**Step 1: 实现 server**

```go
package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/ryubyte/aglight/internal/state"
)

type UpdateRequest struct {
	Status    string `json:"status"`
	EventName string `json:"event"`
	SessionID string `json:"session_id,omitempty"`
	ToolName  string `json:"tool_name,omitempty"`
}

type StatusResponse struct {
	Status    string     `json:"status"`
	UpdatedAt string     `json:"updated_at"`
	History   []state.Event `json:"history,omitempty"`
}

type Server struct {
	machine *state.Machine
	addr    string
}

func New(machine *state.Machine, addr string) *Server {
	return &Server{machine: machine, addr: addr}
}

func (s *Server) Addr() string {
	return s.addr
}

func (s *Server) ListenAndServe() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/update", s.handleUpdate)
	mux.HandleFunc("/status", s.handleStatus)
	mux.HandleFunc("/events", s.handleEvents)
	log.Printf("aglight server listening on %s", s.addr)
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

	// If status is explicit, use it; otherwise derive from event name
	var targetStatus state.Status
	if req.Status != "" {
		targetStatus = state.Status(req.Status)
	} else {
		targetStatus = state.TransitionFromHook(req.EventName)
	}

	event := state.Event{
		Status:    targetStatus,
		EventName: req.EventName,
		SessionID: req.SessionID,
		ToolName:  req.ToolName,
	}
	s.machine.Update(event)

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
		updatedAt = history[len(history)-1].Timestamp.Format("2006-01-02T15:04:05Z07:00")
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

	// Subscribe to state changes
	eventCh := make(chan state.Event, 8)
	s.machine.OnChange(func(old, new state.Status, event state.Event) {
		select {
		case eventCh <- event:
		default:
		}
	})

	for {
		select {
		case event := <-eventCh:
			data, _ := json.Marshal(event)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}
```

**Step 2: 写测试**

`internal/server/server_test.go`:
```go
package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ryubyte/aglight/internal/state"
)

func TestHandleUpdatePost(t *testing.T) {
	m := state.NewMachine()
	s := New(m, ":0")

	body, _ := json.Marshal(UpdateRequest{
		Status:    "running",
		EventName: "SessionStart",
	})
	req := httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.handleUpdate(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if m.Current() != state.StatusRunning {
		t.Errorf("expected running, got %s", m.Current())
	}
}

func TestHandleUpdateDerivesFromEvent(t *testing.T) {
	m := state.NewMachine()
	s := New(m, ":0")

	body, _ := json.Marshal(UpdateRequest{EventName: "Stop"})
	req := httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.handleUpdate(w, req)

	if m.Current() != state.StatusCompleted {
		t.Errorf("expected completed, got %s", m.Current())
	}
}

func TestHandleStatusGet(t *testing.T) {
	m := state.NewMachine()
	s := New(m, ":0")

	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	w := httptest.NewRecorder()

	s.handleStatus(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp StatusResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Status != "idle" {
		t.Errorf("expected idle, got %s", resp.Status)
	}
}

func TestHandleUpdateRejectsGet(t *testing.T) {
	m := state.NewMachine()
	s := New(m, ":0")

	req := httptest.NewRequest(http.MethodGet, "/update", nil)
	w := httptest.NewRecorder()

	s.handleUpdate(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}
```

**Step 3: 运行测试**

```bash
go test ./internal/server/ -v
```

Expected: 全部 PASS

**Step 4: 提交**

```bash
git add internal/server/
git commit -m "feat: implement HTTP server with /update, /status, /events endpoints"
```

---

## Task 5: 实现 Hooks 配置生成器

**Objective:** 生成 Codex 的 hooks 配置片段，用户可以复制到 `~/.codex/config.toml`

**Files:**
- Create: `internal/hookgen/hookgen.go`

**Step 1: 实现 hookgen**

```go
package hookgen

import (
	"fmt"
	"strings"
)

type Config struct {
	ServerAddr string // e.g. "localhost:9876"
}

// Generate produces the TOML hooks config for Codex.
// User should append this to ~/.codex/config.toml
func Generate(cfg Config) string {
	addr := cfg.ServerAddr
	if addr == "" {
		addr = "localhost:9876"
	}

	var b strings.Builder

	events := []struct {
		name     string
		hookName string
	}{
		{"SessionStart", "session_start"},
		{"PreToolUse", "pre_tool_use"},
		{"PostToolUse", "post_tool_use"},
		{"PermissionRequest", "permission_request"},
		{"UserPromptSubmit", "user_prompt_submit"},
		{"PreCompact", "pre_compact"},
		{"PostCompact", "post_compact"},
		{"Stop", "stop"},
		{"SubagentStart", "subagent_start"},
		{"SubagentStop", "subagent_stop"},
	}

	b.WriteString("# AgLight - Status Light Hooks\n")
	b.WriteString("# Add the following to ~/.codex/config.toml\n\n")

	for _, e := range events {
		cmd := fmt.Sprintf(
			"curl -s -X POST http://%s/update -d '{\"event\":\"%s\"}' &",
			addr, e.name,
		)
		fmt.Fprintf(&b, "[[hooks.%s]]\n", e.name)
		fmt.Fprintf(&b, "hooks = [{ type = \"command\", command = %q }]\n\n", cmd)
	}

	return b.String()
}
```

**Step 2: 写测试**

`internal/hookgen/hookgen_test.go`:
```go
package hookgen

import (
	"strings"
	"testing"
)

func TestGenerateContainsAllEvents(t *testing.T) {
	output := Generate(Config{ServerAddr: "localhost:9876"})

	events := []string{
		"SessionStart", "PreToolUse", "PostToolUse",
		"PermissionRequest", "Stop",
	}
	for _, e := range events {
		if !strings.Contains(output, e) {
			t.Errorf("generated config missing event %q", e)
		}
	}
}

func TestGenerateUsesServerAddr(t *testing.T) {
	output := Generate(Config{ServerAddr: "myhost:9999"})
	if !strings.Contains(output, "myhost:9999") {
		t.Error("generated config should use provided server addr")
	}
}

func TestGenerateDefaultsToLocalhost9876(t *testing.T) {
	output := Generate(Config{})
	if !strings.Contains(output, "localhost:9876") {
		t.Error("generated config should default to localhost:9876")
	}
}
```

**Step 3: 运行测试**

```bash
go test ./internal/hookgen/ -v
```

Expected: 全部 PASS

**Step 4: 提交**

```bash
git add internal/hookgen/
git commit -m "feat: implement hooks config generator for Codex"
```

---

## Task 6: 实现 main.go 入口 + 系统托盘

**Objective:** 将所有组件组合起来，实现 macOS 系统托盘应用

**Files:**
- Create: `main.go`

**Step 1: 安装 systray 依赖**

```bash
go get github.com/getlantern/systray
```

**Step 2: 实现 main.go**

```go
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/getlantern/systray"
	"github.com/ryubyte/aglight/internal/hookgen"
	"github.com/ryubyte/aglight/internal/icons"
	"github.com/ryubyte/aglight/internal/server"
	"github.com/ryubyte/aglight/internal/state"
)

const defaultAddr = "localhost:9876"

func main() {
	machine := state.NewMachine()

	// When status changes, update systray icon
	machine.OnChange(func(old, new state.Status, event state.Event) {
		systray.SetIcon(icons.ForStatus(new))
		systray.SetTooltip(fmt.Sprintf("Codex: %s", statusLabel(new)))
		log.Printf("status: %s -> %s (event: %s)", old, new, event.EventName)
	})

	srv := server.New(machine, defaultAddr)

	// Start HTTP server in background
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Fatalf("server error: %v", err)
		}
	}()

	systray.Run(func() {
		systray.SetIcon(icons.ForStatus(state.StatusIdle))
		systray.SetTooltip("Codex: 空闲")

		// Status display item (read-only)
		mStatus := systray.AddMenuItem("状态: 空闲", "当前 Codex 状态")
		mStatus.Disable()
		systray.AddSeparator()

		// Generate hooks config
		mHooks := systray.AddMenuItem("生成 Hooks 配置", "复制 Codex hooks 配置到剪贴板")

		// Reset to idle
		mReset := systray.AddMenuItem("重置为空闲", "手动重置为空闲状态")

		systray.AddSeparator()
		mQuit := systray.AddMenuItem("退出", "关闭 AgLight")

		// Update status menu item on state change
		machine.OnChange(func(old, new state.Status, event state.Event) {
			mStatus.SetTitle("状态: " + statusLabel(new))
		})

		go func() {
			for {
				select {
				case <-mHooks.ClickedCh:
					config := hookgen.Generate(hookgen.Config{ServerAddr: defaultAddr})
					copyToClipboard(config)
					log.Println("hooks config copied to clipboard")
				case <-mReset.ClickedCh:
					machine.Update(state.Event{
						Status:    state.StatusIdle,
						EventName: "manual_reset",
					})
				case <-mQuit.ClickedCh:
					systray.Quit()
				}
			}
		}()
	}, func() {
		log.Println("aglight exiting")
	})
}

func statusLabel(s state.Status) string {
	switch s {
	case state.StatusIdle:
		return "空闲"
	case state.StatusRunning:
		return "运行中"
	case state.StatusCompleted:
		return "已完成"
	case state.StatusApprovalNeeded:
		return "需要审批"
	default:
		return string(s)
	}
}

func copyToClipboard(text string) {
	// macOS pbcopy
	cmd := exec.Command("pbcopy")
	cmd.Stdin = strings.NewReader(text)
	cmd.Run()
}
```

注意: main.go 需要 import "os/exec" 和 "strings"，上面的代码中 copyToClipboard 需要补充这两个 import。

**Step 3: 编译验证**

```bash
go build -o aglight .
```

Expected: 编译成功，生成 aglight 可执行文件

**Step 4: 提交**

```bash
git add main.go go.mod go.sum
git commit -m "feat: implement main entry with systray integration"
```

---

## Task 7: 实现 Makefile

**Objective:** 添加常用构建命令

**Files:**
- Create: `Makefile`

**Step 1: 编写 Makefile**

```makefile
.PHONY: build run test clean hooks

build:
	go build -o aglight .

run: build
	./aglight

test:
	go test ./... -v

clean:
	rm -f aglight

hooks:
	@go run -mod=mod github.com/ryubyte/aglight/cmd/hookgen 2>/dev/null || \
	echo "Run ./aglight and click '生成 Hooks 配置' from the tray menu"
```

**Step 2: 提交**

```bash
git add Makefile
git commit -m "chore: add Makefile with build/run/test targets"
```

---

## Task 8: 编写 README.md

**Objective:** 编写中文 README 说明使用方式

**Files:**
- Create: `README.md`

**Step 1: 编写 README**

内容包含：
1. 项目简介
2. 安装方式 (`go install` / `make build`)
3. 使用步骤：启动 aglight → 托盘菜单点"生成 Hooks 配置" → 粘贴到 `~/.codex/config.toml` → 启动 codex
4. 状态说明（灰/黄/绿/红）
5. HTTP API 文档
6. 开发指南

**Step 2: 提交**

```bash
git add README.md
git commit -m "docs: add Chinese README"
```

---

## Task 9: 端到端测试

**Objective:** 启动 aglight，通过 curl 模拟 hooks 推送，验证托盘图标切换

**Step 1: 编译并启动**

```bash
make build
./aglight &
```

**Step 2: 模拟各事件**

```bash
# 模拟会话开始 -> 黄灯
curl -X POST http://localhost:9876/update -d '{"event":"SessionStart"}'

# 验证状态
curl http://localhost:9876/status

# 模拟需要审批 -> 红灯
curl -X POST http://localhost:9876/update -d '{"event":"PermissionRequest"}'

# 验证状态
curl http://localhost:9876/status

# 模拟完成 -> 绿灯
curl -X POST http://localhost:9876/update -d '{"event":"Stop"}'

# 验证状态
curl http://localhost:9876/status
```

**Step 3: 检查托盘图标**

目视确认：灰 → 黄 → 红 → 绿 切换正确

**Step 4: 清理**

```bash
kill %1
```

---

## 实现注意事项

1. **systray 在 macOS 上需要 cgo**：确保 `CGO_ENABLED=1`，需要 Xcode Command Line Tools
2. **Hooks 配置格式**：Codex hooks 配置在 `~/.codex/config.toml` 的 `[[hooks.EventName]]` 下，每个事件可以有多个 matcher group
3. **curl 后台执行**：hooks 命令末尾加 `&` 使其异步，不阻塞 Codex 执行
4. **图标尺寸**：macOS 系统托盘图标推荐 22x22 像素（Retina 下 44x44），用代码生成矢量圆点避免外部依赖
5. **hooks 配置去重**：如果用户已有 hooks 配置，生成时需提醒用户合并而非覆盖
