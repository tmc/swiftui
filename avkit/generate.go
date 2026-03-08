package avkit

//go:generate applegen swift-bridge AVKit --output . --module github.com/tmc/swiftui
//go:generate bash -lc "set -euo pipefail; rm -rf internal/swift; mv swift internal/swift"
//go:generate bash -lc "set -euo pipefail; cd internal/swift; rm -rf .build/universal-arm64 .build/universal-x86_64; swift build -c release --quiet --product AVKitSwiftUIBridge --triple arm64-apple-macosx --scratch-path .build/universal-arm64; swift build -c release --quiet --product AVKitSwiftUIBridge --triple x86_64-apple-macosx --scratch-path .build/universal-x86_64; mkdir -p .build/universal/release; lipo -create -output .build/universal/release/libAVKitSwiftUIBridge.dylib .build/universal-arm64/arm64-apple-macosx/release/libAVKitSwiftUIBridge.dylib .build/universal-x86_64/x86_64-apple-macosx/release/libAVKitSwiftUIBridge.dylib; lipo -info .build/universal/release/libAVKitSwiftUIBridge.dylib"
