package arkit

//go:generate applegen swift-bridge ARKit --output . --module github.com/tmc/swiftui
//go:generate bash -lc "set -euo pipefail; rm -rf internal/swift; mv swift internal/swift"
//go:generate bash -lc "set -euo pipefail; cd internal/swift; rm -rf .build/universal-arm64 .build/universal-x86_64; swift build -c release --quiet --product ARKitSwiftUIBridge --triple arm64-apple-macosx --scratch-path .build/universal-arm64; swift build -c release --quiet --product ARKitSwiftUIBridge --triple x86_64-apple-macosx --scratch-path .build/universal-x86_64; mkdir -p .build/universal/release; lipo -create -output .build/universal/release/libARKitSwiftUIBridge.dylib .build/universal-arm64/arm64-apple-macosx/release/libARKitSwiftUIBridge.dylib .build/universal-x86_64/x86_64-apple-macosx/release/libARKitSwiftUIBridge.dylib; lipo -info .build/universal/release/libARKitSwiftUIBridge.dylib"
