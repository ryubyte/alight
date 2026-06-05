import AppKit
import Foundation

enum SignalRenderer {
    private static let iconW: CGFloat = 56.0
    private static let iconH: CGFloat = 22.0
    private static let lightRadius: CGFloat = 6.0
    private static let housingRadius: CGFloat = 5.0

    private static let lightCenters: [CGPoint] = [
        CGPoint(x: 10, y: 11), // Red
        CGPoint(x: 28, y: 11), // Yellow
        CGPoint(x: 46, y: 11), // Green
    ]

    private static let statusToLight: [Status: Int] = [
        .idle: -1,
        .running: 1,
        .completed: 2,
        .approvalNeeded: 0,
    ]

    private struct LightColors {
        let onR, onG, onB: CGFloat
        let offR, offG, offB: CGFloat
        let glowR, glowG, glowB: CGFloat
    }

    private static let lightDefs: [LightColors] = [
        // Red — approval needed
        LightColors(onR: 1.0, onG: 0.231, onB: 0.188,
                    offR: 0.22, offG: 0.10, offB: 0.08,
                    glowR: 1.0, glowG: 0.231, glowB: 0.188),
        // Yellow — running
        LightColors(onR: 1.0, onG: 0.8, onB: 0.0,
                    offR: 0.24, offG: 0.20, offB: 0.05,
                    glowR: 1.0, glowG: 0.8, glowB: 0.0),
        // Green — completed
        LightColors(onR: 0.204, onG: 0.780, onB: 0.349,
                    offR: 0.08, offG: 0.20, offB: 0.12,
                    glowR: 0.204, glowG: 0.780, glowB: 0.349),
    ]

    static func forStatus(_ status: Status) -> NSImage {
        renderSignal(activeIdx: statusToLight[status] ?? -1, brightness: 1.0)
    }

    static func forStatusDim(_ status: Status) -> NSImage {
        renderSignal(activeIdx: statusToLight[status] ?? -1, brightness: 0.35)
    }

    private static func renderSignal(activeIdx: Int, brightness: CGFloat) -> NSImage {
        let size = NSSize(width: iconW, height: iconH)
        let image = NSImage(size: size, flipped: false) { _ in
            let housingRect = NSRect(x: 0, y: 0, width: iconW, height: iconH)
            let housing = NSBezierPath(roundedRect: housingRect, xRadius: housingRadius, yRadius: housingRadius)
            NSColor(calibratedRed: 0.12, green: 0.12, blue: 0.14, alpha: 0.92).setFill()
            housing.fill()

            for (i, colors) in lightDefs.enumerated() {
                let center = lightCenters[i]
                let circleRect = NSRect(
                    x: center.x - lightRadius,
                    y: center.y - lightRadius,
                    width: lightRadius * 2,
                    height: lightRadius * 2
                )

                if i == activeIdx {
                    // Outer glow
                    let glowRect = NSRect(
                        x: center.x - lightRadius - 3,
                        y: center.y - lightRadius - 3,
                        width: (lightRadius + 3) * 2,
                        height: (lightRadius + 3) * 2
                    )
                    let glowPath = NSBezierPath(ovalIn: glowRect)
                    NSColor(calibratedRed: colors.glowR, green: colors.glowG, blue: colors.glowB, alpha: 0.25 * brightness).setFill()
                    glowPath.fill()

                    // Main circle
                    let mainPath = NSBezierPath(ovalIn: circleRect)
                    let r = colors.offR + (colors.onR - colors.offR) * brightness
                    let g = colors.offG + (colors.onG - colors.offG) * brightness
                    let b = colors.offB + (colors.onB - colors.offB) * brightness
                    NSColor(calibratedRed: r, green: g, blue: b, alpha: 0.85 + 0.15 * brightness).setFill()
                    mainPath.fill()

                    // Bright center highlight
                    if brightness > 0.7 {
                        let highlightR = lightRadius * 0.35
                        let highlightRect = NSRect(
                            x: center.x - highlightR - 1,
                            y: center.y - highlightR - 1,
                            width: highlightR * 2,
                            height: highlightR * 2
                        )
                        let highlightPath = NSBezierPath(ovalIn: highlightRect)
                        NSColor(calibratedRed: min(colors.onR + 0.35, 1.0),
                                green: min(colors.onG + 0.35, 1.0),
                                blue: min(colors.onB + 0.35, 1.0),
                                alpha: 0.55 * brightness).setFill()
                        highlightPath.fill()
                    }
                } else {
                    // Dim circle
                    let dimPath = NSBezierPath(ovalIn: circleRect)
                    NSColor(calibratedRed: colors.offR, green: colors.offG, blue: colors.offB, alpha: 0.85).setFill()
                    dimPath.fill()

                    // Subtle tint edge
                    NSColor(calibratedRed: colors.onR, green: colors.onG, blue: colors.onB, alpha: 0.15).setStroke()
                    dimPath.lineWidth = 1.0
                    dimPath.stroke()
                }
            }
            return true
        }
        image.isTemplate = false
        return image
    }
}
