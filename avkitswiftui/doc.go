// Package avkitswiftui provides Go bindings for Apple's AVKit
// SwiftUI cross-import overlay (_AVKit_SwiftUI).
//
// This overlay provides SwiftUI views and modifiers that integrate
// AVKit with SwiftUI. All view constructors return uintptr values
// suitable for use with swiftui.ViewFromPointer.
//
// # Threading
//
// All bridge functions dispatch to the main thread internally.
// The Go API is safe to call from any goroutine.
//
// # Quick start
//
//	import "github.com/tmc/appledocs/swift-bridge/swiftui"
//	import "github.com/tmc/appledocs/swift-bridge/avkitswiftui"
//
//	ptr := avkitswiftui.NewVideoPlayer(...)
//	view := swiftui.ViewFromPointer(ptr)
package avkitswiftui
