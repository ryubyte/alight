import Foundation

/// Minimal TOML reader/writer that preserves file content while supporting
/// the subset needed for Codex config: key/value pairs and arrays of tables.
/// NOT a full TOML implementation.
enum TOMLParser {
    static func parse(_ content: String) -> [String: Any] {
        var result: [String: Any] = [:]
        var currentTable: String?
        var currentArrayTable: String?

        for line in content.components(separatedBy: .newlines) {
            let trimmed = line.trimmingCharacters(in: .whitespaces)
            if trimmed.isEmpty || trimmed.hasPrefix("#") { continue }

            // Array of tables: [[hooks.EventName]]
            if trimmed.hasPrefix("[[") && trimmed.hasSuffix("]]") {
                let key = String(trimmed.dropFirst(2).dropLast(2)).trimmingCharacters(in: .whitespaces)
                currentArrayTable = key
                currentTable = nil

                let parts = key.split(separator: ".", maxSplits: 1)
                if parts.count == 2 {
                    let section = String(parts[0])
                    let subKey = String(parts[1])

                    var sectionDict = result[section] as? [String: Any] ?? [:]
                    var arr = sectionDict[subKey] as? [[String: Any]] ?? []
                    arr.append([:])
                    sectionDict[subKey] = arr
                    result[section] = sectionDict
                }
                continue
            }

            // Table: [section]
            if trimmed.hasPrefix("[") && trimmed.hasSuffix("]") {
                currentTable = String(trimmed.dropFirst().dropLast()).trimmingCharacters(in: .whitespaces)
                currentArrayTable = nil
                if result[currentTable!] == nil {
                    result[currentTable!] = [String: Any]()
                }
                continue
            }

            // Key = value
            if let eqIdx = trimmed.firstIndex(of: "=") {
                let key = String(trimmed[trimmed.startIndex..<eqIdx]).trimmingCharacters(in: .whitespaces)
                let rawValue = String(trimmed[trimmed.index(after: eqIdx)...]).trimmingCharacters(in: .whitespaces)
                let value = parseTOMLValue(rawValue)

                if let arrayTable = currentArrayTable {
                    let parts = arrayTable.split(separator: ".", maxSplits: 1)
                    if parts.count == 2 {
                        let section = String(parts[0])
                        let subKey = String(parts[1])
                        var sectionDict = result[section] as? [String: Any] ?? [:]
                        var arr = sectionDict[subKey] as? [[String: Any]] ?? []
                        if !arr.isEmpty {
                            arr[arr.count - 1][key] = value
                        }
                        sectionDict[subKey] = arr
                        result[section] = sectionDict
                    }
                } else if let table = currentTable {
                    var tableDict = result[table] as? [String: Any] ?? [:]
                    tableDict[key] = value
                    result[table] = tableDict
                } else {
                    result[key] = value
                }
            }
        }

        return result
    }

    static func serialize(_ dict: [String: Any]) -> String {
        var lines: [String] = []

        // Top-level key/values first
        for (key, value) in dict.sorted(by: { $0.key < $1.key }) {
            if value is [String: Any] { continue }
            lines.append("\(key) = \(serializeTOMLValue(value))")
        }

        // Sections
        for (key, value) in dict.sorted(by: { $0.key < $1.key }) {
            guard let sectionDict = value as? [String: Any] else { continue }

            // Check if this section contains arrays of tables
            var hasArrayTables = false
            for (_, v) in sectionDict {
                if v is [[String: Any]] { hasArrayTables = true; break }
            }

            if hasArrayTables {
                for (subKey, subValue) in sectionDict.sorted(by: { $0.key < $1.key }) {
                    if let arr = subValue as? [[String: Any]] {
                        for entry in arr {
                            if !lines.isEmpty { lines.append("") }
                            lines.append("[[\(key).\(subKey)]]")
                            for (k, v) in entry.sorted(by: { $0.key < $1.key }) {
                                lines.append("\(k) = \(serializeTOMLValue(v))")
                            }
                        }
                    } else {
                        // Plain key in the section
                        if lines.isEmpty || !lines.last!.isEmpty { lines.append("") }
                        lines.append("[\(key)]")
                        lines.append("\(subKey) = \(serializeTOMLValue(subValue))")
                    }
                }
            } else {
                if !lines.isEmpty { lines.append("") }
                lines.append("[\(key)]")
                for (k, v) in sectionDict.sorted(by: { $0.key < $1.key }) {
                    lines.append("\(k) = \(serializeTOMLValue(v))")
                }
            }
        }

        return lines.joined(separator: "\n") + "\n"
    }

    private static func parseTOMLValue(_ raw: String) -> Any {
        // Boolean
        if raw == "true" { return true }
        if raw == "false" { return false }

        // Integer
        if let intVal = Int(raw) { return intVal }

        // Float
        if let dblVal = Double(raw) { return dblVal }

        // String (quoted)
        if raw.hasPrefix("\"") && raw.hasSuffix("\"") {
            return String(raw.dropFirst().dropLast())
                .replacingOccurrences(of: "\\\"", with: "\"")
                .replacingOccurrences(of: "\\\\", with: "\\")
        }

        // Inline array: [{ ... }, { ... }]
        if raw.hasPrefix("[") && raw.hasSuffix("]") {
            return parseInlineArray(raw)
        }

        return raw
    }

    private static func parseInlineArray(_ raw: String) -> Any {
        let inner = String(raw.dropFirst().dropLast()).trimmingCharacters(in: .whitespaces)
        if inner.isEmpty { return [[String: Any]]() }

        // Try to parse as array of inline tables: [{ k=v, k=v }, { k=v }]
        if inner.hasPrefix("{") {
            var tables: [[String: Any]] = []
            var depth = 0
            var current = ""
            for ch in inner {
                if ch == "{" {
                    depth += 1
                    if depth == 1 { current = ""; continue }
                }
                if ch == "}" {
                    depth -= 1
                    if depth == 0 {
                        tables.append(parseInlineTable(current))
                        current = ""
                        continue
                    }
                }
                if depth == 0 && ch == "," { continue }
                current.append(ch)
            }
            return tables
        }

        // Array of simple values
        return inner.split(separator: ",").map { parseTOMLValue(String($0).trimmingCharacters(in: .whitespaces)) }
    }

    private static func parseInlineTable(_ raw: String) -> [String: Any] {
        var result: [String: Any] = [:]
        let pairs = raw.split(separator: ",")
        for pair in pairs {
            let kv = pair.split(separator: "=", maxSplits: 1)
            if kv.count == 2 {
                let key = String(kv[0]).trimmingCharacters(in: .whitespaces)
                let value = String(kv[1]).trimmingCharacters(in: .whitespaces)
                result[key] = parseTOMLValue(value)
            }
        }
        return result
    }

    private static func serializeTOMLValue(_ value: Any) -> String {
        switch value {
        case let b as Bool:
            return b ? "true" : "false"
        case let i as Int:
            return "\(i)"
        case let d as Double:
            return "\(d)"
        case let s as String:
            let escaped = s.replacingOccurrences(of: "\\", with: "\\\\")
                .replacingOccurrences(of: "\"", with: "\\\"")
            return "\"\(escaped)\""
        case let arr as [[String: Any]]:
            let items = arr.map { table in
                let pairs = table.sorted(by: { $0.key < $1.key }).map { k, v in
                    "\(k) = \(serializeTOMLValue(v))"
                }
                return "{ \(pairs.joined(separator: ", ")) }"
            }
            return "[\(items.joined(separator: ", "))]"
        case let arr as [Any]:
            let items = arr.map { serializeTOMLValue($0) }
            return "[\(items.joined(separator: ", "))]"
        default:
            return "\"\(value)\""
        }
    }
}
