// swift-tools-version: 6.2
import PackageDescription

let package = Package(
    name: "Charts3DBridge",
    platforms: [.macOS(.v26)],
    products: [
        .library(name: "Charts3DBridge", type: .dynamic, targets: ["Charts3DBridge"]),
    ],
    targets: [
        .target(
            name: "Charts3DBridge",
            path: "Sources",
            swiftSettings: [.unsafeFlags(["-parse-as-library"])],
        ),
    ]
)
