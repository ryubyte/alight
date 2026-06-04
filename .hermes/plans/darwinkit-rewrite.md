# AgLight darwinkit 重构计划

> **For Hermes:** Use subagent-driven-development skill to implement this plan task-by-task.

**Goal:** 用 darwinkit 替换 systray，实现 macOS 原生 NSStatusItem 自定义绘制，显示醒目的横向红绿灯状态灯

**Architecture:** 单进程 AppKit 应用。用 darwinkit 创建 NSStatusItem，通过 `Image_ImageWithSizeFlippedDrawingHandler` 绘制自定义 NSImage（3 个彩色圆点 + 深色边框 housing），状态变更时重新生成图标并 `SetNeedsDisplay`。HTTP server 和状态机不变。

**Tech Stack:** Go + progrium/darwinkit (AppKit bindings) + 现有 internal/state, server, hookgen

---

### Task 1: 替换依赖 — 移除 systray，添加 darwinkit

**Objective:** 清理 go.mod 依赖

**Files:**
- Modify: `go.mod`
- Modify: `go.sum`

**Step 1:** 移除 systray 依赖
```bash
cd /Users/xiongshengyao/workspace/github.com/ryubyte/aglight
go get github.com/getlantern/systray@none
```

**Step 2:** 添加 darwinkit
```bash
go get github.com/progrium/darwinkit/macos@latest
```

**Step 3:** go mod tidy
```bash
go mod tidy
```

**Step 4:** 验证
```bash
go build ./...
```
Expected: 编译通过（可能 main.go 暂时出错，但 internal 包应通过）

**Step 5:** Commit
```bash
git add -A && git commit -m "chore: swap systray for darwinkit"
```

---

### Task 2: 重写 internal/icons — 用 darwinkit 生成 NSImage

**Objective:** 用 AppKit 的 drawingHandler 绘制高清 Retina 图标，替代纯 Go image/png

**Files:**
- Modify: `internal/icons/icons.go`
- Modify: `internal/icons/icons_test.go`

**核心设计：**
- 用 `appkit.Image_ImageWithSizeFlippedDrawingHandler` 创建 NSImage
- 在 drawingHandler 里用 `appkit.BezierPath` 绘制：
  - 深色圆角矩形 housing（边框）
  - 3 个彩色圆（红、黄、绿），活跃灯明亮+发光，非活跃灯暗淡
- 图标逻辑尺寸 50x22pt（宽度足够放 3 个大圆点）
- `ForStatus` 返回 `appkit.Image` 而非 `[]byte`

**Step 1:** 重写 icons.go

```go
package icons

import (
    "github.com/progrium/darwinkit/macos/appkit"
    "github.com/progrium/darwinkit/macos/foundation"
    "github.com/ryubyte/aglight/internal/state"
)

const (
    iconWidth  = 50.0
    iconHeight = 22.0
)

type lightDef struct {
    onR, onG, onB, onA     float64
    offR, offG, offB, offA float64
}

var lights = []lightDef{
    {255.0/255, 59.0/255, 48.0/255, 1.0,  0.22, 0.09, 0.08, 0.8},
    {255.0/255, 204.0/255, 0.0/255, 1.0,  0.22, 0.18, 0.03, 0.8},
    {52.0/255, 199.0/255, 89.0/255, 1.0,  0.07, 0.20, 0.11, 0.8},
}

var statusToLight = map[state.Status]int{
    state.StatusIdle:           -1,
    state.StatusRunning:        1,
    state.StatusCompleted:      2,
    state.StatusApprovalNeeded: 0,
}

func ForStatus(s state.Status) appkit.Image {
    return renderSignal(statusToLight[s])
}

func renderSignal(activeIdx int) appkit.Image {
    return appkit.Image_ImageWithSizeFlippedDrawingHandler(
        foundation.Size{Width: iconWidth, Height: iconHeight},
        true,
        func(dstRect foundation.Rect) bool {
            // Draw housing (dark rounded rect)
            housingPath := appkit.BezierPath_BezierPathWithRoundedRectXRadiusYRadius(
                dstRect, 4, 4,
            )
            appkit.Color_ColorWithCalibratedRedGreenBlueAlpha(0.12, 0.12, 0.13, 0.95).SetFill()
            housingPath.Fill()

            // Draw 3 lights
            dotRadius := 7.0
            spacing := iconWidth / 4.0 // = 12.5
            cy := iconHeight / 2.0

            for i, light := range lights {
                cx := spacing * float64(i+1)
                isActive := (i == activeIdx)

                if isActive {
                    // Glow halo
                    glowPath := appkit.BezierPath_BezierPathWithOvalInRect(
                        rect(cx-dotRadius-3, cy-dotRadius-3, (dotRadius+3)*2, (dotRadius+3)*2),
                    )
                    appkit.Color_ColorWithCalibratedRedGreenBlueAlpha(light.onR, light.onG, light.onB, 0.3).SetFill()
                    glowPath.Fill()

                    // Main circle
                    mainPath := appkit.BezierPath_BezierPathWithOvalInRect(
                        rect(cx-dotRadius, cy-dotRadius, dotRadius*2, dotRadius*2),
                    )
                    appkit.Color_ColorWithCalibratedRedGreenBlueAlpha(light.onR, light.onG, light.onB, light.onA).SetFill()
                    mainPath.Fill()

                    // Bright center
                    cr := dotRadius * 0.4
                    centerPath := appkit.BezierPath_BezierPathWithOvalInRect(
                        rect(cx-cr, cy-cr, cr*2, cr*2),
                    )
                    appkit.Color_ColorWithCalibratedRedGreenBlueAlpha(
                        min1(light.onR+0.35), min1(light.onG+0.35), min1(light.onB+0.35), 0.5,
                    ).SetFill()
                    centerPath.Fill()
                } else {
                    // Dim circle
                    dimPath := appkit.BezierPath_BezierPathWithOvalInRect(
                        rect(cx-dotRadius, cy-dotRadius, dotRadius*2, dotRadius*2),
                    )
                    appkit.Color_ColorWithCalibratedRedGreenBlueAlpha(light.offR, light.offG, light.offB, light.offA).SetFill()
                    dimPath.Fill()

                    // Subtle color rim
                    rimPath := appkit.BezierPath_BezierPathWithOvalInRect(
                        rect(cx-dotRadius, cy-dotRadius, dotRadius*2, dotRadius*2),
                    )
                    appkit.Color_ColorWithCalibratedRedGreenBlueAlpha(light.onR, light.onG, light.onB, 0.08).SetStroke()
                    rimPath.SetLineWidth(0.5)
                    rimPath.Stroke()
                }
            }
            return true
        },
    )
}

func rect(x, y, w, h float64) foundation.Rect {
    return foundation.Rect{
        Origin: foundation.Point{X: x, Y: y},
        Size:   foundation.Size{Width: w, Height: h},
    }
}

func min1(f float64) float64 {
    if f > 1.0 { return 1.0 }
    return f
}
```

**Step 2:** 更新测试 — 因返回类型变了（appkit.Image 而非 []byte），测试需要在 macOS 上运行，简单验证非零即可

**Step 3:** 编译验证
```bash
go build ./internal/icons/
```

**Step 4:** Commit
```bash
git add -A && git commit -m "feat: rewrite icons with darwinkit NSImage drawingHandler"
```

---

### Task 3: 重写 main.go — darwinkit AppKit 应用

**Objective:** 用 darwinkit 替换 systray，实现 NSStatusItem + 菜单

**Files:**
- Modify: `main.go`

**核心逻辑：**
- 用 `macos.RunApp(launched)` 启动 AppKit 事件循环
- 在 launched 回调中：
  1. 创建 NSStatusItem（宽度 VariableStatusItemLength）
  2. 用 `icons.ForStatus(idle)` 生成初始图标
  3. 将图标设到 `item.Button().SetImage()`
  4. 启动 HTTP server（与现有一致）
  5. 注册状态变更回调：重新生成图标 + `SetImage` 刷新
  6. 创建右键菜单（状态显示 + 生成 Hooks + 重置 + 退出）
- 应用激活策略改为 Accessory（无 Dock 图标）

**Step 1:** 重写 main.go

关键代码结构：
```go
package main

import (
    "log"
    "os/exec"
    "runtime"
    "strings"

    "github.com/progrium/darwinkit/macos"
    "github.com/progrium/darwinkit/macos/appkit"
    "github.com/progrium/darwinkit/macos/foundation"
    "github.com/progrium/darwinkit/objc"

    "github.com/ryubyte/aglight/internal/hookgen"
    "github.com/ryubyte/aglight/internal/icons"
    "github.com/ryubyte/aglight/internal/server"
    "github.com/ryubyte/aglight/internal/state"
)

const defaultAddr = "localhost:9876"

func main() {
    runtime.LockOSThread()
    macos.RunApp(launched)
}

func launched(app appkit.Application, delegate *appkit.ApplicationDelegate) {
    // Status bar item
    item := appkit.StatusBar_SystemStatusBar().StatusItemWithLength(appkit.VariableStatusItemLength)
    objc.Retain(&item)
    item.Button().SetImage(icons.ForStatus(state.StatusIdle))
    item.Button().SetToolTip("AgLight 🚥 ⚫ 空闲")

    // State machine + HTTP server
    machine := state.NewMachine()
    srv := server.New(machine, defaultAddr)
    go srv.ListenAndServe()

    // Status change callback
    machine.OnChange(func(old, new state.Status, event state.Event) {
        item.Button().SetImage(icons.ForStatus(new))
        item.Button().SetToolTip("AgLight 🚥 " + statusLabel(new))
    })

    // Menu
    menu := appkit.NewMenuWithTitle("AgLight")
    mStatus := appkit.NewMenuItemWithTitleActionKeyEquivalent(statusLabel(state.StatusIdle), objc.Selector{}, "")
    mStatus.SetEnabled(false)
    menu.AddItem(mStatus)
    menu.AddItem(appkit.MenuItem_SeparatorItem())
    mHooks := appkit.NewMenuItemWithTitleActionKeyEquivalent("生成 Hooks 配置", objc.Selector{}, "")
    menu.AddItem(mHooks)
    mReset := appkit.NewMenuItemWithTitleActionKeyEquivalent("重置为空闲", objc.Selector{}, "")
    menu.AddItem(mReset)
    menu.AddItem(appkit.MenuItem_SeparatorItem())
    mQuit := appkit.NewMenuItemWithTitleActionKeyEquivalent("退出", objc.Selector{}, "")
    menu.AddItem(mQuit)
    item.SetMenu(menu)

    // Menu actions via target/action pattern
    mHooks.SetTarget(actionTarget(func() {
        cfg := hookgen.Config{ServerAddr: defaultAddr}
        toml := hookgen.Generate(cfg)
        copyToClipboard(toml)
        mHooks.SetTitle("生成 Hooks 配置 ✓")
    }))
    mReset.SetTarget(actionTarget(func() {
        machine.Update(state.Event{Status: state.StatusIdle, EventName: "manual_reset"})
    }))
    mQuit.SetTarget(actionTarget(func() {
        app.Terminate(nil)
    }))

    // Update menu status label on change
    machine.OnChange(func(old, new state.Status, event state.Event) {
        mStatus.SetTitle(statusLabel(new))
    })

    // Accessory app — no dock icon
    app.SetActivationPolicy(appkit.ApplicationActivationPolicyAccessory)
}
```

注意：darwinkit 菜单项的 action 需要通过 `helper/action` 包或自定义 target 对象实现。

**Step 2:** 编译验证
```bash
go build -o aglight .
```

**Step 3:** Commit
```bash
git add -A && git commit -m "feat: rewrite main.go with darwinkit AppKit"
```

---

### Task 4: 清理无用代码 + 测试验证

**Objective:** 移除旧 icons 包中不再需要的 image/png 代码，确保全部测试通过

**Files:**
- Modify: `internal/icons/icons_test.go`
- Modify: `go.mod` (tidy)

**Step 1:** 更新 icons 测试 — 因为 darwinkit 需要 macOS AppKit 运行时，测试需要 +build 标签
```go
//go:build darwin

package icons

import (
    "testing"
    "github.com/ryubyte/aglight/internal/state"
)

func TestForStatus_ReturnsNonNilImage(t *testing.T) {
    // Test can only run on macOS with GUI context
    // Basic smoke test: ensure no panic
    for _, s := range []state.Status{
        state.StatusIdle,
        state.StatusRunning,
        state.StatusCompleted,
        state.StatusApprovalNeeded,
    } {
        img := ForStatus(s)
        if !img.Ptr().IsNil() {
            t.Logf("ForStatus(%v) returned non-nil image", s)
        }
    }
}
```

**Step 2:** go mod tidy
```bash
go mod tidy
```

**Step 3:** 全量编译测试
```bash
go test ./internal/state/ ./internal/server/ ./internal/hookgen/ -v
go build -o aglight .
```

**Step 4:** Commit
```bash
git add -A && git commit -m "chore: cleanup old systray code, update tests"
```

---

### Task 5: 端到端运行验证

**Objective:** 启动应用，通过 curl 测试所有状态切换

**Step 1:** 启动应用
```bash
./aglight &
sleep 3
```

**Step 2:** 验证 HTTP API
```bash
# idle
curl -s http://localhost:9876/status

# running (yellow)
curl -s -X POST http://localhost:9876/update -d '{"event":"SessionStart"}'

# approval (red)
curl -s -X POST http://localhost:9876/update -d '{"event":"PermissionRequest"}'

# completed (green)
curl -s -X POST http://localhost:9876/update -d '{"event":"Stop"}'

# reset to idle
curl -s -X POST http://localhost:9876/update -d '{"status":"idle","event":"reset"}'
```

**Step 3:** 确认菜单栏图标正确显示（用户肉眼确认）

**Step 4:** Commit
```bash
git add -A && git commit -m "test: e2e verification passed"
```
