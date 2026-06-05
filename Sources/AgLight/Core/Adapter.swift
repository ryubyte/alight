import Foundation

public protocol Adapter {
    var name: String { get }
    func isInstalled() -> Bool
    func inject(port: String) throws
    func cleanup() throws
    func mapEvent(_ eventName: String) -> Status
}

final class Registry: @unchecked Sendable {
    private let queue = DispatchQueue(label: "aglight.registry")
    private var adapters: [String: Adapter] = [:]

    func register(_ adapter: Adapter) {
        queue.sync {
            adapters[adapter.name] = adapter
        }
    }

    func injectAll(port: String) {
        let all = queue.sync { Array(adapters.values) }
        for a in all {
            do {
                try a.inject(port: port)
            } catch {
                print("warning: inject \(a.name) hooks: \(error)")
            }
        }
    }

    func cleanupAll() {
        let all = queue.sync { Array(adapters.values) }
        for a in all {
            do {
                try a.cleanup()
            } catch {
                print("warning: cleanup \(a.name) hooks: \(error)")
            }
        }
    }

    func mapEvent(source: String, eventName: String) -> Status {
        let all = queue.sync { adapters }

        if !source.isEmpty, let a = all[source] {
            return a.mapEvent(eventName)
        }

        for a in all.values {
            let s = a.mapEvent(eventName)
            if s != .idle { return s }
        }
        return .idle
    }

    func adapterNames() -> [String] {
        queue.sync { Array(adapters.keys) }
    }

    func installedAdapters() -> [String] {
        let all = queue.sync { Array(adapters.values) }
        return all.filter { $0.isInstalled() }.map { $0.name }
    }
}

public func statusLabel(_ s: Status) -> String {
    switch s {
    case .idle: return "⚫ 空闲"
    case .running: return "🟡 运行中"
    case .approvalNeeded: return "🔴 需要审批"
    case .completed: return "🟢 已完成"
    }
}
