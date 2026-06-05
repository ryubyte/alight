import Foundation

public struct CodexAdapter: Adapter {
    public var name: String { "codex" }

    public nonisolated(unsafe) static var configPathOverride: String?

    public init() {}

    private var configPath: String {
        if let override = Self.configPathOverride { return override }
        let home = FileManager.default.homeDirectoryForCurrentUser.path
        return "\(home)/.codex/config.toml"
    }

    public func isInstalled() -> Bool {
        FileManager.default.fileExists(atPath: configPath)
    }

    public func inject(port: String) throws {
        guard FileManager.default.fileExists(atPath: configPath) else { return }
        var cfg = try readConfig()
        let serverAddr = "localhost:\(port)"
        cfg = injectHooks(cfg, serverAddr: serverAddr)
        try writeConfig(cfg)
    }

    public func cleanup() throws {
        guard FileManager.default.fileExists(atPath: configPath) else { return }
        var cfg = try readConfig()
        cfg = cleanupHooks(cfg)
        try writeConfig(cfg)
    }

    public func mapEvent(_ eventName: String) -> Status {
        switch eventName {
        case "SessionStart", "PreToolUse", "PostToolUse", "UserPromptSubmit",
             "SubagentStart", "PreCompact", "PostCompact":
            return .running
        case "PermissionRequest":
            return .approvalNeeded
        case "Stop", "StopFailure", "SubagentStop":
            return .completed
        default:
            return .idle
        }
    }

    // MARK: - Config I/O

    private func readConfig() throws -> [String: Any] {
        let content = try String(contentsOfFile: configPath, encoding: .utf8)
        return TOMLParser.parse(content)
    }

    private func writeConfig(_ cfg: [String: Any]) throws {
        let content = TOMLParser.serialize(cfg)
        let dir = (configPath as NSString).deletingLastPathComponent
        try FileManager.default.createDirectory(atPath: dir, withIntermediateDirectories: true)
        try content.write(toFile: configPath, atomically: true, encoding: .utf8)
    }

    // MARK: - Hook injection

    private static let codexEvents = [
        "SessionStart",
        "PreToolUse",
        "PostToolUse",
        "PermissionRequest",
        "UserPromptSubmit",
        "PreCompact",
        "PostCompact",
        "Stop",
        "SubagentStart",
        "SubagentStop",
    ]

    private func containsAglightURL(_ cmd: String) -> Bool {
        cmd.contains("source=aglight")
    }

    private func cleanupHooks(_ cfg: [String: Any]) -> [String: Any] {
        var cfg = cfg
        guard var hooks = cfg["hooks"] as? [String: Any] else { return cfg }

        for (event, entries) in hooks {
            guard var entriesList = entries as? [[String: Any]] else { continue }

            entriesList = entriesList.compactMap { entry -> [String: Any]? in
                guard var hooksList = entry["hooks"] as? [[String: Any]] else { return entry }
                hooksList.removeAll { hook in
                    guard let cmd = hook["command"] as? String else { return false }
                    return containsAglightURL(cmd)
                }
                if hooksList.isEmpty { return nil }
                var e = entry
                e["hooks"] = hooksList
                return e
            }

            if entriesList.isEmpty {
                hooks.removeValue(forKey: event)
            } else {
                hooks[event] = entriesList
            }
        }

        if hooks.isEmpty {
            cfg.removeValue(forKey: "hooks")
        } else {
            cfg["hooks"] = hooks
        }
        return cfg
    }

    private func injectHooks(_ cfg: [String: Any], serverAddr: String) -> [String: Any] {
        var cfg = cleanupHooks(cfg)
        var hooks = cfg["hooks"] as? [String: Any] ?? [:]

        for event in Self.codexEvents {
            let entry: [String: Any] = [
                "hooks": [[
                    "type": "command",
                    "command": "curl -s -X POST 'http://\(serverAddr)/update?source=aglight' -d '{\"event\":\"\(event)\"}'",
                    "async": true,
                ] as [String: Any]]
            ]

            var existing = hooks[event] as? [[String: Any]] ?? []
            existing.append(entry)
            hooks[event] = existing
        }

        cfg["hooks"] = hooks
        return cfg
    }
}
