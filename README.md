# Codex Bar - Codex 状态展示灯

macOS 系统托盘状态灯，实时显示 [Codex CLI](https://github.com/openai/codex) 的工作状态。

## 状态说明

| 颜色 | 状态 | 含义 |
|------|------|------|
| 🔘 灰色 | 空闲 | Codex 未在运行 |
| 🟡 黄色 | 运行中 | Codex 正在处理任务 |
| 🔴 红色 | 需要审批 | Codex 等待用户批准操作 |
| 🟢 绿色 | 已完成 | Codex 完成当前任务 |

## 安装

### 从源码构建

```bash
git clone https://github.com/ryubyte/codex-bar.git
cd codex-bar
make build
```

### 安装到 PATH

```bash
make install
```

## 使用方法

### 1. 启动 Codex Bar

```bash
./codex-bar
```

启动后菜单栏会出现灰色圆点图标。

### 2. 配置 Codex Hooks

点击托盘图标 → **生成 Hooks 配置**，配置内容会自动复制到剪贴板。

然后将内容粘贴到 `~/.codex/config.toml`：

```bash
# 如果文件不存在
mkdir -p ~/.codex
cat >> ~/.codex/config.toml << 'EOF'
# 粘贴剪贴板中的配置
EOF
```

### 3. 启动 Codex

正常启动 `codex`，状态灯会自动切换。

## HTTP API

Codex Bar 在 `localhost:9876` 提供 HTTP 接口：

### POST /update

更新状态，由 Codex Hooks 调用。

```bash
# 指定状态
curl -X POST http://localhost:9876/update \
  -d '{"status":"running","event":"SessionStart"}'

# 从事件名推导状态
curl -X POST http://localhost:9876/update \
  -d '{"event":"PermissionRequest"}'
```

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

SSE 事件流，实时推送状态变更。

```bash
curl http://localhost:9876/events
```

## 支持的 Hook 事件

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

## 开发

```bash
# 运行测试
make test

# 构建
make build

# 清理
make clean
```

## 技术栈

- Go 1.22+
- [getlantern/systray](https://github.com/getlantern/systray) - macOS 系统托盘
- Go 标准库 `image/png` - 图标生成
- Go 标准库 `net/http` - HTTP 服务

## License

MIT
