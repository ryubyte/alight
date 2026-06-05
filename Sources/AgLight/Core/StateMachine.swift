import Foundation

public enum Status: String, Codable, Sendable {
    case idle
    case running
    case approvalNeeded = "approval_needed"
    case completed
}

public struct Event: Codable, Sendable {
    public let status: Status
    public let eventName: String
    public let sessionID: String
    public let toolName: String
    public let timestamp: Date

    public init(status: Status, eventName: String, sessionID: String, toolName: String, timestamp: Date) {
        self.status = status
        self.eventName = eventName
        self.sessionID = sessionID
        self.toolName = toolName
        self.timestamp = timestamp
    }

    enum CodingKeys: String, CodingKey {
        case status
        case eventName = "event_name"
        case sessionID = "session_id"
        case toolName = "tool_name"
        case timestamp
    }
}

public typealias StateChangeCallback = @Sendable (Status, Status, Event) -> Void

public final class StateMachine: @unchecked Sendable {
    private let queue = DispatchQueue(label: "aglight.statemachine")
    private var current: Status = .idle
    private var history: [Event] = []
    private var callbacks: [(id: UInt64, cb: StateChangeCallback)] = []
    private var nextID: UInt64 = 0

    public init() {}

    public func currentStatus() -> Status {
        queue.sync { current }
    }

    @discardableResult
    public func update(event: Event) -> Status {
        var cbs: [(id: UInt64, cb: StateChangeCallback)] = []
        var old: Status = .idle

        queue.sync {
            old = self.current
            self.current = event.status
            self.history.append(event)
            cbs = self.callbacks
        }

        if old != event.status {
            for entry in cbs {
                entry.cb(old, event.status, event)
            }
        }

        return event.status
    }

    public func onChange(_ cb: @escaping StateChangeCallback) -> () -> Void {
        let id = queue.sync { () -> UInt64 in
            let id = nextID
            nextID += 1
            callbacks.append((id: id, cb: cb))
            return id
        }
        return { [weak self] in
            self?.queue.sync {
                self?.callbacks.removeAll { $0.id == id }
            }
        }
    }

    public func getHistory() -> [Event] {
        queue.sync { history }
    }
}
