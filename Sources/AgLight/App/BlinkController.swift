import AppKit
import Foundation

final class BlinkController {
    private let button: NSStatusBarButton
    private let machine: StateMachine
    private var timer: Timer?
    private var isOn = true

    private static let blinkInterval: TimeInterval = 1.2

    init(button: NSStatusBarButton, machine: StateMachine) {
        self.button = button
        self.machine = machine
    }

    func start() {
        stop()
        isOn = true
        timer = Timer.scheduledTimer(withTimeInterval: Self.blinkInterval, repeats: true) { [weak self] _ in
            guard let self else { return }
            self.isOn.toggle()
            if self.machine.currentStatus() == .approvalNeeded {
                DispatchQueue.main.async {
                    if self.isOn {
                        self.button.image = SignalRenderer.forStatus(.approvalNeeded)
                    } else {
                        self.button.image = SignalRenderer.forStatusDim(.approvalNeeded)
                    }
                }
            }
        }
    }

    func stop() {
        timer?.invalidate()
        timer = nil
    }
}
