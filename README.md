# AgLight — AI Agent 状态红绿灯

macOS 菜单栏红绿灯，实时显示 AI 编码助手的工作状态。

支持 [Codex CLI](https://github.com/openai/codex) 和 [Claude Code](https://docs.anthropic.com/en/docs/claude-code)，可扩展。

## 功能

- 🟡 运行中 — Agent 正在工作
- 🟢 已完成 — Agent 完成任务
- 🔴 需要审批 — Agent 等待用户确认（闪烁 + 音效提醒）
- ⚫ 空闲 — 无活动

## 工作原理

AgLight 启动时自动向各 AI 工具注入 Hooks 配置，工具的状态事件通过 HTTP 推送到本地服务，驱动菜单栏红绿灯切换。

退出时自动清理注入的配置，不留残留。

## 安装

```bash
git clone https://github.com/ryubyte/aglight.git
cd aglight
make build
make install   # 可选，安装到 /usr/local/bin
```

## 使用

```bash
aglight
```

启动后菜单栏出现红绿灯图标，自动检测已安装的 AI 工具并注入 hooks。

## 项目结构

```
aglight/
├── main.go                        # 入口，注册适配器 + AppKit 启动
├── internal/
│   ├── core/                      # 核心层（零依赖 AI 工具）
│   │   ├── adapter.go             # Adapter 接口 + Registry
│   │   ├── server.go              # HTTP 服务 + SSE + 端口检测
│   │   └── state/                 # 状态机 + 回调管理
│   │       └── state.go
│   ├── adapter/                   # 适配器层（每个 AI 工具一个子包）
│   │   ├── codex/                 # Codex CLI: config.toml + TOML hooks
│   │   │   └── codex.go
│   │   └── claude/                # Claude Code: settings.json + JSON hooks
│   │       └── claude.go
│   ├── hookgen/                   # Hooks TOML 生成（复制到剪贴板用）
│   │   └── hookgen.go
│   ├── icons/                     # darwinkit NSImage 红绿灯绘制
│   │   └── icons.go
│   └── ui/                        # 菜单栏 UI
│       ├── blink.go               # 红灯闪烁控制器
│       ├── label.go               # 状态标签 + 声音 + 剪贴板
│       └── menu.go                # 菜单构建
├── Makefile
└── README.md
```

### 新增适配器

只需 2 步，核心代码零修改：

1. 在 `internal/adapter/` 下新建子包，实现 `core.Adapter` 接口：

```go
type Adapter interface {
    Name() string                    // 唯一标识
    IsInstalled() bool               // 工具是否已安装
    Inject(port string) error        // 注入 hooks
    Cleanup() error                  // 清理 hooks
    MapEvent(eventName string) state.Status  // 事件映射
}
```

2. 在 `main.go` 中注册：

```go
registry.Register(your_adapter.New())
```

## 技术栈

- Go 1.23+
- [darwinkit](https://github.com/progrium/darwinkit) — macOS 原生 AppKit 绑定
- [go-toml](https://github.com/pelletier/go-toml) — TOML 读写

## License

MIT
