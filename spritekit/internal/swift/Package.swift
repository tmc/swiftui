// swift-tools-version: 6.2
import PackageDescription

let package = Package(
    name: "SpriteKitSwiftUIBridge",
    platforms: [.macOS(.v15)],
    products: [
        .library(name: "SpriteKitSwiftUIBridge", type: .dynamic, targets: ["SpriteKitSwiftUIBridge"]),
    ],
    targets: [
        .target(name: "SpriteKitSwiftUIBridge",
                path: "Sources",
                swiftSettings: [.unsafeFlags(["-parse-as-library"])])
    ]
)
