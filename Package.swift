// swift-tools-version: 6.0

import PackageDescription

let package = Package(
    name: "AgLight",
    platforms: [.macOS(.v14)],
    dependencies: [
        .package(url: "https://github.com/apple/swift-nio.git", from: "2.65.0"),
    ],
    targets: [
        .target(
            name: "AgLightLib",
            dependencies: [
                .product(name: "NIOCore", package: "swift-nio"),
                .product(name: "NIOPosix", package: "swift-nio"),
                .product(name: "NIOHTTP1", package: "swift-nio"),
            ],
            path: "Sources/AgLight",
            swiftSettings: [.swiftLanguageMode(.v5)]
        ),
        .executableTarget(
            name: "AgLight",
            dependencies: ["AgLightLib"],
            path: "Sources/App",
            swiftSettings: [.swiftLanguageMode(.v5)]
        ),
        .executableTarget(
            name: "AgLightTestRunner",
            dependencies: ["AgLightLib"],
            path: "Tests/Runner",
            swiftSettings: [.swiftLanguageMode(.v5)]
        ),
    ]
)
