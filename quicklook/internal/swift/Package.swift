// swift-tools-version: 6.2
import PackageDescription

let package = Package(
    name: "QuickLookSwiftUIBridge",
    platforms: [.macOS(.v15)],
    products: [
        .library(name: "QuickLookSwiftUIBridge", type: .dynamic, targets: ["QuickLookSwiftUIBridge"]),
    ],
    targets: [
        .target(name: "QuickLookSwiftUIBridge",
                path: "Sources",
                swiftSettings: [.unsafeFlags(["-parse-as-library"])])
    ]
)
