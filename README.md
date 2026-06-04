# Codex Bar — Codex 状态红绿灯

macOS 菜单栏红绿灯，实时显示 [Codex CLI](https://github.com/openai/codex) 的工作状态。

![截图](docs/screenshot.png)

## 功能

- 🚦 菜单栏红绿灯：空闲（灰）、运行中（黄）、需要审批（红闪烁）、已完成（绿）
- 🔔 状态变化声音提醒（Sosumi / Glass）
- 🪝 自动注入 Codex Hooks，无需手动配置
- 📋 一键复制 Hooks TOML 配置到剪贴板
- 🌐 HTTP API + SSE 事件流，方便集成

## 安装

### go install

```bash
go install github.com/ryubyte/codex-bar@latest
```

### 从源码构建

```bash
git clone https://github.com/ryubyte/codex-bar.git
cd codex-bar
make build
```

构建产物为当前目录下的 `codex-bar` 二进制。

### 安装到 PATH

```bash
make install
```

## 使用方法

启动 Codex Bar：

```bash
codex-bar
```

启动后菜单栏出现灰色圆点图标，Codex Bar 会自动：

1. 检测可用端口（默认从 9876 开始）
2. 清理旧版 Hooks 配置
3. 注入新的 Hooks 到 `~/.codex/config.toml`
4. 启动 HTTP 服务监听状态变更

正常启动 `codex` 即可，状态灯自动切换。

菜单操作：

- **Hooks 配置**：点击复制 TOML 到剪贴板
- **重置为空闲**：手动重置状态
- **声音**：开关声音提醒
- **退出**：清理 Hooks 并退出

## 状态说明

| 颜色 | 状态 | 含义 |
|------|------|------|
| ⚫ 灰色 | idle（空闲） | Codex 未在运行 |
| 🟡 黄色 | running（运行中） | Codex 正在处理任务 |
| 🔴 红色闪烁 | approval_needed（需要审批） | Codex 等待用户批准操作 |
| 🟢 绿色 | completed（已完成） | Codex 完成当前任务 |

## 自动 Hooks 注入

Codex Bar 启动时会自动修改 `~/.codex/config.toml`，为以下 10 个事件注入 Hook：

| 事件 | 触发时机 | 灯色 |
|------|----------|------|
| SessionStart | 会话启动 | 黄 |
| PreToolUse | 工具执行前 | 黄 |
| PostToolUse | 工具执行后 | 黄 |
| UserPromptSubmit | 用户提交提示 | 黄 |
| PreCompact | 压缩前 | 黄 |
| PostCompact | 压缩后 | 黄 |
| SubagentStart | 子代理启动 | 黄 |
| PermissionRequest | 需要审批 | 红 |
| Stop | Agent 停止 | 绿 |
| SubagentStop | 子代理停止 | 绿 |

每个 Hook 通过 `curl` 向本地 HTTP 服务发送 POST 请求。退出时自动清理所有注入的 Hook，不影响用户原有配置。

## HTTP API

### POST /update

更新状态，由 Codex Hooks 自动调用。

```bash
# 指定状态
curl -X POST http://localhost:9876/update \
  -d '{"status":"running","event":"SessionStart"}'

# 从事件名推导状态
curl -X POST http://localhost:9876/update \
  -d '{"event":"PermissionRequest"}'
```

请求体字段：

| 字段 | 类型 | 说明 |
|------|------|------|
| status | string | 可选。直接指定状态：idle / running / approval_needed / completed |
| event | string | 事件名。status 为空时自动推导 |
| session_id | string | 可选。会话 ID |
| tool_name | string | 可选。工具名 |

### GET /status

查询当前状态。

```bash
curl http://localhost:9876/status
```

返回示例：

```json
{
  "status": "running",
  "updated_at": "2026-06-04T20:30:00+08:00",
  "history": [...]
}
```

### GET /events

SSE 事件流，实时推送状态变更。客户端断开后自动注销回调，无内存泄漏。

```bash
curl http://localhost:9876/events
```

## 配置说明

- **端口自动检测**：从 9876 开始扫描，自动选择可用端口（范围 9876-9975）
- **声音开关**：菜单栏点击「声音」切换，默认开启
- **Hooks 清理**：退出时（菜单退出 / SIGTERM / SIGINT）自动清理注入的 Hook

## 手动测试

不用 Codex 也可以手动测试状态灯：

```bash
# 黄灯 - 运行中
curl -X POST http://localhost:9876/update -d '{"event":"SessionStart"}'

# 红灯 - 需要审批
curl -X POST http://localhost:9876/update -d '{"event":"PermissionRequest"}'

# 绿灯 - 已完成
curl -X POST http://localhost:9876/update -d '{"event":"Stop"}'
```

## 开发指南

```bash
# 运行测试
go test ./... -v

# 构建
go build -o codex-bar .

# 清理
make clean
```

## 技术栈

- Go 1.23+
- [darwinkit](https://github.com/progrium/darwinkit) (AppKit) — macOS 原生菜单栏
- [go-toml/v2](https://github.com/pelletier/go-toml) — TOML 配置读写
- Go 标准库 `net/http` — HTTP 服务

## License

MIT
