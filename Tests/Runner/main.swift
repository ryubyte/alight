import Foundation
import AgLightLib

var passed = 0
var failed = 0

func check(_ condition: Bool, _ msg: String = "", file: String = #file, line: Int = #line) {
    if condition {
        passed += 1
    } else {
        failed += 1
        print("  FAIL [\(file):\(line)] \(msg)")
    }
}

func checkEqual<T: Equatable>(_ a: T, _ b: T, file: String = #file, line: Int = #line) {
    check(a == b, "expected \(b), got \(a)", file: file, line: line)
}

func test(_ name: String, _ body: () throws -> Void) {
    do {
        try body()
        print("  ✓ \(name)")
    } catch {
        failed += 1
        print("  ✗ \(name): \(error)")
    }
}

// MARK: - StateMachine Tests

print("StateMachine:")

test("initial status is idle") {
    let m = StateMachine()
    checkEqual(m.currentStatus(), .idle)
}

test("update changes status") {
    let m = StateMachine()
    let event = Event(status: .running, eventName: "SessionStart", sessionID: "", toolName: "", timestamp: .now)
    let result = m.update(event: event)
    checkEqual(result, .running)
    checkEqual(m.currentStatus(), .running)
}

test("callback fires on change") {
    let m = StateMachine()
    var oldStatus: Status?
    var newStatus: Status?
    _ = m.onChange { old, new, _ in
        oldStatus = old
        newStatus = new
    }
    let event = Event(status: .running, eventName: "SessionStart", sessionID: "", toolName: "", timestamp: .now)
    _ = m.update(event: event)
    checkEqual(oldStatus, .idle)
    checkEqual(newStatus, .running)
}

test("callback does not fire on same status") {
    let m = StateMachine()
    var callCount = 0
    _ = m.onChange { _, _, _ in callCount += 1 }
    let event = Event(status: .running, eventName: "SessionStart", sessionID: "", toolName: "", timestamp: .now)
    _ = m.update(event: event)
    _ = m.update(event: event)
    checkEqual(callCount, 1)
}

test("unregister removes callback") {
    let m = StateMachine()
    var callCount = 0
    let unreg = m.onChange { _, _, _ in callCount += 1 }
    _ = m.update(event: Event(status: .running, eventName: "X", sessionID: "", toolName: "", timestamp: .now))
    checkEqual(callCount, 1)
    unreg()
    _ = m.update(event: Event(status: .idle, eventName: "Y", sessionID: "", toolName: "", timestamp: .now))
    checkEqual(callCount, 1)
}

test("history records events") {
    let m = StateMachine()
    _ = m.update(event: Event(status: .running, eventName: "X", sessionID: "s1", toolName: "", timestamp: .now))
    let h = m.getHistory()
    checkEqual(h.count, 1)
    checkEqual(h[0].sessionID, "s1")
}

// MARK: - ClaudeAdapter Tests

print("\nClaudeAdapter:")

test("mapEvent") {
    let a = ClaudeAdapter()
    checkEqual(a.mapEvent("SessionStart"), .running)
    checkEqual(a.mapEvent("UserPromptSubmit"), .running)
    checkEqual(a.mapEvent("PermissionRequest"), .approvalNeeded)
    checkEqual(a.mapEvent("Stop"), .completed)
    checkEqual(a.mapEvent("Unknown"), .idle)
}

test("inject and cleanup") {
    let tempDir = FileManager.default.temporaryDirectory.appendingPathComponent(UUID().uuidString)
    try FileManager.default.createDirectory(at: tempDir, withIntermediateDirectories: true)
    let path = tempDir.appendingPathComponent("settings.json").path
    ClaudeAdapter.settingsPathOverride = path
    defer {
        ClaudeAdapter.settingsPathOverride = nil
        try? FileManager.default.removeItem(at: tempDir)
    }

    try "{}".write(toFile: path, atomically: true, encoding: .utf8)
    let a = ClaudeAdapter()
    try a.inject(port: "9876")

    let data = try Data(contentsOf: URL(fileURLWithPath: path))
    let settings = try JSONSerialization.jsonObject(with: data) as! [String: Any]
    let hooks = settings["hooks"] as! [String: Any]
    check(hooks.keys.contains("SessionStart"), "should have SessionStart hook")
    check(hooks.keys.contains("Stop"), "should have Stop hook")

    try a.cleanup()
    let data2 = try Data(contentsOf: URL(fileURLWithPath: path))
    let settings2 = try JSONSerialization.jsonObject(with: data2) as! [String: Any]
    check(settings2["hooks"] == nil, "hooks should be removed after cleanup")
}

test("inject preserves existing settings") {
    let tempDir = FileManager.default.temporaryDirectory.appendingPathComponent(UUID().uuidString)
    try FileManager.default.createDirectory(at: tempDir, withIntermediateDirectories: true)
    let path = tempDir.appendingPathComponent("settings.json").path
    ClaudeAdapter.settingsPathOverride = path
    defer {
        ClaudeAdapter.settingsPathOverride = nil
        try? FileManager.default.removeItem(at: tempDir)
    }

    let existing: [String: Any] = ["theme": "dark", "fontSize": 14]
    try JSONSerialization.data(withJSONObject: existing).write(to: URL(fileURLWithPath: path))

    let a = ClaudeAdapter()
    try a.inject(port: "9876")

    let data = try Data(contentsOf: URL(fileURLWithPath: path))
    let settings = try JSONSerialization.jsonObject(with: data) as! [String: Any]
    checkEqual(settings["theme"] as? String, "dark")
    checkEqual(settings["fontSize"] as? Int, 14)
    check(settings["hooks"] != nil, "hooks should exist")
}

// MARK: - CodexAdapter Tests

print("\nCodexAdapter:")

test("mapEvent") {
    let a = CodexAdapter()
    checkEqual(a.mapEvent("SessionStart"), .running)
    checkEqual(a.mapEvent("PreToolUse"), .running)
    checkEqual(a.mapEvent("PermissionRequest"), .approvalNeeded)
    checkEqual(a.mapEvent("Stop"), .completed)
    checkEqual(a.mapEvent("SubagentStop"), .completed)
    checkEqual(a.mapEvent("Unknown"), .idle)
}

test("inject and cleanup") {
    let tempDir = FileManager.default.temporaryDirectory.appendingPathComponent(UUID().uuidString)
    try FileManager.default.createDirectory(at: tempDir, withIntermediateDirectories: true)
    let path = tempDir.appendingPathComponent("config.toml").path
    CodexAdapter.configPathOverride = path
    defer {
        CodexAdapter.configPathOverride = nil
        try? FileManager.default.removeItem(at: tempDir)
    }

    try "".write(toFile: path, atomically: true, encoding: .utf8)
    let a = CodexAdapter()
    try a.inject(port: "9876")

    let content = try String(contentsOfFile: path, encoding: .utf8)
    check(content.contains("source=aglight"), "should contain aglight marker")
    check(content.contains("SessionStart"), "should contain SessionStart")

    try a.cleanup()
    let content2 = try String(contentsOfFile: path, encoding: .utf8)
    check(!content2.contains("source=aglight"), "should not contain aglight after cleanup")
}

// MARK: - Summary

print("\n---")
print("Results: \(passed) passed, \(failed) failed")
if failed > 0 {
    exit(1)
}
