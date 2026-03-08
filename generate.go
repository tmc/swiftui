package swiftui

//go:generate applegen swift-bridge SwiftUI --output . --module github.com/tmc/swiftui
//go:generate bash -lc "set -euo pipefail; rm -rf internal/swift; mv swift internal/swift"
//go:generate bash -lc "set -euo pipefail; cd internal/swift; rm -rf .build/universal-arm64 .build/universal-x86_64; swift build -c release --quiet --product SwiftUIBridge --triple arm64-apple-macosx --scratch-path .build/universal-arm64; swift build -c release --quiet --product SwiftUIBridge --triple x86_64-apple-macosx --scratch-path .build/universal-x86_64; mkdir -p .build/universal/release; lipo -create -output .build/universal/release/libSwiftUIBridge.dylib .build/universal-arm64/arm64-apple-macosx/release/libSwiftUIBridge.dylib .build/universal-x86_64/x86_64-apple-macosx/release/libSwiftUIBridge.dylib; lipo -info .build/universal/release/libSwiftUIBridge.dylib"
//go:generate bash -lc "cp internal/swift/.build/universal/release/libSwiftUIBridge.dylib internal/embeddedbridge/libSwiftUIBridge.dylib"
