# AgLight

macOS 菜单栏红绿灯，实时显示 AI 编码助手的工作状态。

支持 [Claude Code](https://docs.anthropic.com/en/docs/claude-code) 和 [Codex CLI](https://github.com/openai/codex)。

## 功能

- 🟡 **运行中** — Agent 正在工作
- 🟢 **已完成** — Agent 完成任务
- 🔴 **需要审批** — Agent 等待确认（闪烁 + 音效提醒）
- ⚫ **空闲** — 无活动

## 特性

- **零配置** — 启动后自动检测已安装的 AI 工具并注入 Hooks，无需手动设置
- **无残留** — 退出时自动清理所有注入的配置，不影响原有设置
- **非侵入** — 仅追加自己的 Hooks，与你已有的配置互不干扰
- **状态提醒** — 需要审批时红灯闪烁 + 音效提示，完成后绿灯提示，即使不看屏幕也能感知
- **可扩展** — 支持接入更多 AI 工具

## 安装

### 一键安装（推荐）

```bash
curl -fsSL https://raw.githubusercontent.com/ryubyte/aglight/master/install.sh | sh
```

自动完成：下载二进制 → 安装到 `~/.local/bin` → 启动

### 卸载

```bash
rm ~/.local/bin/aglight
```

### 从源码构建

**前置条件**：Go 1.23+、macOS 12+

```bash
git clone https://github.com/ryubyte/aglight.git
cd aglight
make build
make install
```

## 使用

运行 `aglight` 启动，菜单栏出现红绿灯图标。打开 Claude Code 或 Codex CLI 开始工作，红绿灯自动跟随状态变化。

点击菜单栏图标可：
- **重置** — 手动回到空闲状态
- **声音** — 开关状态变化提示音（默认开启）
- **退出** — 退出 AgLight

如需开机自启，可在系统设置 → 通用 → 登录项中添加 AgLight。

## 支持的工具

| 工具 | 状态 |
|------|------|
| [Claude Code](https://docs.anthropic.com/en/docs/claude-code) | ✅ 已支持 |
| [Codex CLI](https://github.com/openai/codex) | ✅ 已支持 |

## License

MIT
