import Foundation

public struct ClaudeAdapter: Adapter {
    public var name: String { "claude" }

    public nonisolated(unsafe) static var settingsPathOverride: String?

    public init() {}

    private var settingsPath: String {
        if let override = Self.settingsPathOverride { return override }
        let home = FileManager.default.homeDirectoryForCurrentUser.path
        return "\(home)/.claude/settings.json"
    }

    public func isInstalled() -> Bool {
        FileManager.default.fileExists(atPath: settingsPath)
    }

    public func inject(port: String) throws {
        guard FileManager.default.fileExists(atPath: settingsPath) else { return }
        var settings = try readSettings()
        settings = injectHooks(settings, port: port)
        try writeSettings(settings)
    }

    public func cleanup() throws {
        guard FileManager.default.fileExists(atPath: settingsPath) else { return }
        var settings = try readSettings()
        settings = cleanupHooks(settings)
        try writeSettings(settings)
    }

    public func mapEvent(_ eventName: String) -> Status {
        switch eventName {
        case "SessionStart", "UserPromptSubmit":
            return .running
        case "PermissionRequest":
            return .approvalNeeded
        case "Stop", "StopFailure":
            return .completed
        default:
            return .idle
        }
    }

    // MARK: - Settings I/O

    private func readSettings() throws -> [String: Any] {
        let data = try Data(contentsOf: URL(fileURLWithPath: settingsPath))
        guard let dict = try JSONSerialization.jsonObject(with: data) as? [String: Any] else {
            return [:]
        }
        return dict
    }

    private func writeSettings(_ settings: [String: Any]) throws {
        let data = try JSONSerialization.data(withJSONObject: settings, options: [.prettyPrinted, .sortedKeys])
        let dir = (settingsPath as NSString).deletingLastPathComponent
        try FileManager.default.createDirectory(atPath: dir, withIntermediateDirectories: true)
        try data.write(to: URL(fileURLWithPath: settingsPath))
    }

    // MARK: - Hook injection

    private static let claudeEvents = [
        "SessionStart",
        "UserPromptSubmit",
        "Stop",
        "StopFailure",
        "PermissionRequest",
    ]

    private func isAglightHook(_ hook: [String: Any]) -> Bool {
        guard let cmd = hook["command"] as? String else { return false }
        return cmd.contains("source=aglight")
    }

    private func cleanupHooks(_ settings: [String: Any]) -> [String: Any] {
        var settings = settings
        guard var hooksMap = settings["hooks"] as? [String: Any] else { return settings }

        for (event, groups) in hooksMap {
            guard var groupList = groups as? [[String: Any]] else { continue }

            groupList = groupList.compactMap { group -> [String: Any]? in
                guard var hooksList = group["hooks"] as? [[String: Any]] else { return group }
                hooksList.removeAll { isAglightHook($0) }
                if hooksList.isEmpty { return nil }
                var g = group
                g["hooks"] = hooksList
                return g
            }

            if groupList.isEmpty {
                hooksMap.removeValue(forKey: event)
            } else {
                hooksMap[event] = groupList
            }
        }

        if hooksMap.isEmpty {
            settings.removeValue(forKey: "hooks")
        } else {
            settings["hooks"] = hooksMap
        }
        return settings
    }

    private func injectHooks(_ settings: [String: Any], port: String) -> [String: Any] {
        var settings = cleanupHooks(settings)
        var hooksMap = settings["hooks"] as? [String: Any] ?? [:]

        for event in Self.claudeEvents {
            let hook: [String: Any] = [
                "type": "command",
                "command": "curl -s -X POST 'http://localhost:\(port)/update?source=aglight' -d '{\"event\":\"\(event)\"}'",
                "timeout": 5,
            ]

            if var existing = hooksMap[event] as? [[String: Any]], !existing.isEmpty {
                var firstGroup = existing[0]
                var hooksList = firstGroup["hooks"] as? [[String: Any]] ?? []
                hooksList.append(hook)
                firstGroup["hooks"] = hooksList
                existing[0] = firstGroup
                hooksMap[event] = existing
            } else {
                hooksMap[event] = [["hooks": [hook]]]
            }
        }

        settings["hooks"] = hooksMap
        return settings
    }
}
