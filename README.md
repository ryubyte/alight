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

要求：macOS 14+ (Sonoma)，Swift 5.9+ 工具链（Xcode 或 Command Line Tools）。

## 使用

```bash
aglight
```

启动后菜单栏出现红绿灯图标，自动检测已安装的 AI 工具并注入 hooks。

## 项目结构

```
aglight/
├── Package.swift                      # Swift Package 配置
├── Sources/
│   ├── AgLight/                       # 库代码
│   │   ├── Core/
│   │   │   ├── StateMachine.swift     # 状态机 + 回调管理
│   │   │   ├── Adapter.swift          # Adapter 协议 + Registry
│   │   │   └── Server.swift           # HTTP 服务 (swift-nio) + SSE
│   │   ├── Adapters/
│   │   │   ├── ClaudeAdapter.swift    # Claude Code: settings.json hooks
│   │   │   └── CodexAdapter.swift     # Codex CLI: config.toml hooks
│   │   ├── Icons/
│   │   │   └── SignalRenderer.swift   # NSImage 红绿灯绘制
│   │   ├── App/
│   │   │   ├── AppDelegate.swift      # 菜单栏 UI + 生命周期
│   │   │   └── BlinkController.swift  # 红灯闪烁控制器
│   │   └── Util/
│   │       └── TOMLParser.swift       # 最小 TOML 解析器
│   └── App/
│       └── main.swift                 # 入口
├── Tests/
│   └── Runner/
│       └── main.swift                 # 测试运行器
└── Makefile
```

### 新增适配器

只需 2 步，核心代码零修改：

1. 在 `Sources/AgLight/Adapters/` 下新建文件，实现 `Adapter` 协议：

```swift
public protocol Adapter {
    var name: String { get }
    func isInstalled() -> Bool
    func inject(port: String) throws
    func cleanup() throws
    func mapEvent(_ eventName: String) -> Status
}
```

2. 在 `AppDelegate.swift` 中注册：

```swift
registry.register(YourAdapter())
```

## 技术栈

- Swift 5.9+ / macOS 14+
- [swift-nio](https://github.com/apple/swift-nio) — 轻量 HTTP 服务
- AppKit — macOS 原生菜单栏 UI

## License

MIT
