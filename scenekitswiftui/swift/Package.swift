// swift-tools-version: 6.2
import PackageDescription

let package = Package(
    name: "SceneKitSwiftUIBridge",
    platforms: [.macOS(.v15)],
    products: [
        .library(name: "SceneKitSwiftUIBridge", type: .dynamic, targets: ["SceneKitSwiftUIBridge"]),
    ],
    targets: [
        .target(name: "SceneKitSwiftUIBridge",
                path: "Sources",
                swiftSettings: [.unsafeFlags(["-parse-as-library"])])
    ]
)
