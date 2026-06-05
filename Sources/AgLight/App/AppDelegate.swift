import AppKit
import Foundation

public final class AppDelegate: NSObject, NSApplicationDelegate {
    private var statusItem: NSStatusItem!
    private var statusMenuItem: NSMenuItem!
    private var machine: StateMachine!
    private var registry: Registry!
    private var server: HTTPServer!
    private var blink: BlinkController!
    private var soundEnabled = true

    private static let defaultStartPort = 9876

    public func applicationDidFinishLaunching(_ notification: Notification) {
        NSApp.setActivationPolicy(.accessory)

        machine = StateMachine()
        registry = Registry()
        registry.register(ClaudeAdapter())
        registry.register(CodexAdapter())

        // Find free port
        guard let port = HTTPServer.findFreePort(startPort: Self.defaultStartPort) else {
            print("error: no free port found")
            NSApp.terminate(nil)
            return
        }
        let portStr = "\(port)"

        // Inject hooks
        registry.injectAll(port: portStr)

        // Start HTTP server
        server = HTTPServer(machine: machine, registry: registry, host: "127.0.0.1", port: port)
        DispatchQueue.global().async { [weak self] in
            do {
                try self?.server.start()
            } catch {
                print("server error: \(error)")
            }
        }

        // Setup status bar
        statusItem = NSStatusBar.system.statusItem(withLength: NSStatusItem.variableLength)
        guard let button = statusItem.button else { return }
        button.image = SignalRenderer.forStatus(machine.currentStatus())
        button.toolTip = "AgLight idle"

        blink = BlinkController(button: button, machine: machine)

        // Build menu
        let menu = buildMenu()
        statusItem.menu = menu

        // Listen for state changes
        _ = machine.onChange { [weak self] oldStatus, newStatus, _ in
            DispatchQueue.main.async {
                guard let self else { return }

                if oldStatus == .approvalNeeded && newStatus != .approvalNeeded {
                    self.blink.stop()
                }

                self.statusItem.button?.image = SignalRenderer.forStatus(newStatus)
                self.statusItem.button?.toolTip = "AgLight \(statusLabel(newStatus))"
                self.statusMenuItem.title = statusLabel(newStatus)

                if self.soundEnabled {
                    switch newStatus {
                    case .approvalNeeded:
                        self.playSound("Sosumi")
                        self.blink.start()
                    case .completed:
                        self.playSound("Glass")
                    default:
                        break
                    }
                } else {
                    if newStatus == .approvalNeeded {
                        self.blink.start()
                    }
                }
            }
        }

        // Signal handling for cleanup
        let signalSource = DispatchSource.makeSignalSource(signal: SIGTERM, queue: .main)
        signalSource.setEventHandler { [weak self] in
            self?.registry.cleanupAll()
            NSApp.terminate(nil)
        }
        signalSource.resume()
        signal(SIGTERM, SIG_IGN)

        let intSource = DispatchSource.makeSignalSource(signal: SIGINT, queue: .main)
        intSource.setEventHandler { [weak self] in
            self?.registry.cleanupAll()
            NSApp.terminate(nil)
        }
        intSource.resume()
        signal(SIGINT, SIG_IGN)
    }

    public func applicationWillTerminate(_ notification: Notification) {
        registry?.cleanupAll()
        server?.stop()
    }

    private func buildMenu() -> NSMenu {
        let menu = NSMenu()

        statusMenuItem = NSMenuItem(title: statusLabel(machine.currentStatus()), action: nil, keyEquivalent: "")
        statusMenuItem.isEnabled = false
        menu.addItem(statusMenuItem)

        menu.addItem(.separator())

        let resetItem = NSMenuItem(title: "Reset", action: #selector(resetStatus), keyEquivalent: "")
        resetItem.target = self
        menu.addItem(resetItem)

        let soundItem = NSMenuItem(title: "Sound", action: #selector(toggleSound(_:)), keyEquivalent: "")
        soundItem.target = self
        soundItem.state = soundEnabled ? .on : .off
        menu.addItem(soundItem)

        menu.addItem(.separator())

        let quitItem = NSMenuItem(title: "Quit", action: #selector(quitApp), keyEquivalent: "q")
        quitItem.target = self
        menu.addItem(quitItem)

        return menu
    }

    @objc private func resetStatus() {
        let event = Event(
            status: .idle,
            eventName: "Reset",
            sessionID: "",
            toolName: "",
            timestamp: Date()
        )
        _ = machine.update(event: event)
    }

    @objc private func toggleSound(_ sender: NSMenuItem) {
        soundEnabled.toggle()
        sender.state = soundEnabled ? .on : .off
    }

    @objc private func quitApp() {
        registry.cleanupAll()
        NSApp.terminate(nil)
    }

    private func playSound(_ name: String) {
        NSSound(named: NSSound.Name(name))?.play()
    }
}
