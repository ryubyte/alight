import Darwin.C
import Foundation
import NIOCore
import NIOPosix
import NIOHTTP1

struct UpdateRequest: Decodable {
    let status: String?
    let event: String?
    let source: String?
    let sessionID: String?
    let toolName: String?

    enum CodingKeys: String, CodingKey {
        case status, event, source
        case sessionID = "session_id"
        case toolName = "tool_name"
    }
}

struct StatusResponse: Encodable {
    let status: String
    let updatedAt: String
    let history: [Event]

    enum CodingKeys: String, CodingKey {
        case status
        case updatedAt = "updated_at"
        case history
    }
}

final class HTTPServer {
    private let machine: StateMachine
    private let registry: Registry
    private let host: String
    private let port: Int
    private var group: MultiThreadedEventLoopGroup?
    private var channel: Channel?

    init(machine: StateMachine, registry: Registry, host: String, port: Int) {
        self.machine = machine
        self.registry = registry
        self.host = host
        self.port = port
    }

    func start() throws {
        let group = MultiThreadedEventLoopGroup(numberOfThreads: 1)
        self.group = group

        let bootstrap = ServerBootstrap(group: group)
            .serverChannelOption(.backlog, value: 256)
            .childChannelInitializer { channel in
                channel.pipeline.configureHTTPServerPipeline().flatMap {
                    channel.pipeline.addHandler(HTTPHandler(machine: self.machine, registry: self.registry))
                }
            }
            .childChannelOption(.maxMessagesPerRead, value: 1)

        channel = try bootstrap.bind(host: host, port: port).wait()
    }

    func stop() {
        try? channel?.close().wait()
        try? group?.syncShutdownGracefully()
    }

    static func findFreePort(startPort: Int) -> Int? {
        for port in startPort..<(startPort + 100) {
            let fd = socket(AF_INET, SOCK_STREAM, 0)
            guard fd >= 0 else { continue }
            var addr = sockaddr_in()
            addr.sin_family = sa_family_t(AF_INET)
            addr.sin_port = in_port_t(port).bigEndian
            addr.sin_addr.s_addr = inet_addr("127.0.0.1")
            let result = withUnsafePointer(to: &addr) { ptr in
                ptr.withMemoryRebound(to: sockaddr.self, capacity: 1) { sockPtr in
                    Darwin.bind(fd, sockPtr, socklen_t(MemoryLayout<sockaddr_in>.size))
                }
            }
            close(fd)
            if result == 0 { return port }
        }
        return nil
    }
}

private final class HTTPHandler: ChannelInboundHandler, RemovableChannelHandler {
    typealias InboundIn = HTTPServerRequestPart
    typealias OutboundOut = HTTPServerResponsePart

    private let machine: StateMachine
    private let registry: Registry
    private var requestHead: HTTPRequestHead?
    private var body = ByteBuffer()
    private var sseChannels: [ObjectIdentifier: Channel] = [:]

    init(machine: StateMachine, registry: Registry) {
        self.machine = machine
        self.registry = registry
    }

    func channelRead(context: ChannelHandlerContext, data: NIOAny) {
        let part = unwrapInboundIn(data)
        switch part {
        case .head(let head):
            requestHead = head
            body.clear()
        case .body(var buf):
            body.writeBuffer(&buf)
        case .end:
            guard let head = requestHead else { return }
            handleRequest(context: context, head: head, body: body)
            requestHead = nil
        }
    }

    private func handleRequest(context: ChannelHandlerContext, head: HTTPRequestHead, body: ByteBuffer) {
        let uri = head.uri.split(separator: "?").first.map(String.init) ?? head.uri

        switch (head.method, uri) {
        case (.POST, "/update"):
            handleUpdate(context: context, head: head, body: body)
        case (.GET, "/status"):
            handleStatus(context: context)
        case (.GET, "/events"):
            handleEvents(context: context)
        default:
            sendResponse(context: context, status: .notFound, body: "not found")
        }
    }

    private func handleUpdate(context: ChannelHandlerContext, head: HTTPRequestHead, body: ByteBuffer) {
        var bodyMutable = body
        guard let bytes = bodyMutable.readBytes(length: bodyMutable.readableBytes) else {
            sendResponse(context: context, status: .badRequest, body: "empty body")
            return
        }
        let data = Data(bytes)
        guard let req = try? JSONDecoder().decode(UpdateRequest.self, from: data) else {
            sendResponse(context: context, status: .badRequest, body: "invalid json")
            return
        }

        // Also check query string for source param
        let querySource: String? = {
            guard let query = head.uri.split(separator: "?").dropFirst().first else { return nil }
            for param in query.split(separator: "&") {
                let kv = param.split(separator: "=", maxSplits: 1)
                if kv.count == 2, kv[0] == "source" { return String(kv[1]) }
            }
            return nil
        }()

        let source = req.source ?? querySource ?? ""

        let status: Status
        if let s = req.status, let parsed = Status(rawValue: s) {
            status = parsed
        } else {
            status = registry.mapEvent(source: source, eventName: req.event ?? "")
        }

        let event = Event(
            status: status,
            eventName: req.event ?? "",
            sessionID: req.sessionID ?? "",
            toolName: req.toolName ?? "",
            timestamp: Date()
        )
        _ = machine.update(event: event)

        sendResponse(context: context, status: .ok, body: "ok")
    }

    private func handleStatus(context: ChannelHandlerContext) {
        let current = machine.currentStatus()
        let history = machine.getHistory()

        let updatedAt: String
        if let last = history.last {
            let formatter = ISO8601DateFormatter()
            updatedAt = formatter.string(from: last.timestamp)
        } else {
            updatedAt = ""
        }

        let resp = StatusResponse(status: current.rawValue, updatedAt: updatedAt, history: history)
        if let data = try? JSONEncoder().encode(resp),
           let json = String(data: data, encoding: .utf8) {
            sendResponse(context: context, status: .ok, body: json, contentType: "application/json")
        } else {
            sendResponse(context: context, status: .internalServerError, body: "encode error")
        }
    }

    private func handleEvents(context: ChannelHandlerContext) {
        let head = HTTPResponseHead(version: .http1_1, status: .ok, headers: [
            "Content-Type": "text/event-stream",
            "Cache-Control": "no-cache",
            "Connection": "keep-alive",
        ])
        context.write(wrapOutboundOut(.head(head)), promise: nil)

        var buf = context.channel.allocator.buffer(capacity: 64)
        buf.writeString(": connected\n\n")
        context.writeAndFlush(wrapOutboundOut(.body(.byteBuffer(buf))), promise: nil)

        let channel = context.channel
        let unregister = machine.onChange { [weak channel] _, _, event in
            guard let channel = channel, channel.isActive else { return }
            if let data = try? JSONEncoder().encode(event),
               let json = String(data: data, encoding: .utf8) {
                var buf = channel.allocator.buffer(capacity: json.count + 16)
                buf.writeString("data: \(json)\n\n")
                channel.write(HTTPServerResponsePart.body(.byteBuffer(buf)), promise: nil)
                channel.flush()
            }
        }

        context.channel.closeFuture.whenComplete { _ in
            unregister()
        }
    }

    private func sendResponse(context: ChannelHandlerContext, status: HTTPResponseStatus, body: String, contentType: String = "text/plain") {
        let head = HTTPResponseHead(version: .http1_1, status: status, headers: [
            "Content-Type": contentType,
            "Content-Length": "\(body.utf8.count)",
        ])
        context.write(wrapOutboundOut(.head(head)), promise: nil)
        var buf = context.channel.allocator.buffer(capacity: body.utf8.count)
        buf.writeString(body)
        context.write(wrapOutboundOut(.body(.byteBuffer(buf))), promise: nil)
        context.writeAndFlush(wrapOutboundOut(.end(nil)), promise: nil)
    }
}
