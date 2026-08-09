// Test-only SwiftUI oracle exports. This file is hand-owned: applegen never
// writes it. They are present only in a bridge compiled with
// -D SWIFTUI_ORACLE, which swiftgo's swiftuioracle-tagged tests build.

#if SWIFTUI_ORACLE
import Foundation
import SwiftUI

// SUIBoxAnyView packages an AnyView value constructed outside the bridge for
// the bridge's opaque-handle APIs. The input is borrowed; this copies it into
// the retained Box used by SUIRenderPNG and the view modifiers.
@_cdecl("SUIBoxAnyView")
public func SUIBoxAnyView(_ view: UnsafeRawPointer) -> UnsafeMutableRawPointer {
    if ProcessInfo.processInfo.environment["SWIFTUI_E6_NO_RETAIN"] == "1" {
        fputs("swiftgo E6 no-retain mutation active\n", stderr)
        let box = Box(view.assumingMemoryBound(to: AnyView.self).pointee)
        return Unmanaged.passUnretained(box).toOpaque()
    }
    return retainView(view.assumingMemoryBound(to: AnyView.self).pointee)
}

// SUIConditionalVStack is a test oracle for a static ViewBuilder branch. It
// keeps the builder construction on the bridge side so the direct probe can
// compare each branch against Swift's own lowering.
@_cdecl("SUIConditionalVStack")
public func SUIConditionalVStack(_ first: UnsafeMutableRawPointer, _ second: UnsafeMutableRawPointer, _ branch: Int32) -> UnsafeMutableRawPointer {
    let firstView = Unmanaged<Box<AnyView>>.fromOpaque(first).takeUnretainedValue().value
    let secondView = Unmanaged<Box<AnyView>>.fromOpaque(second).takeUnretainedValue().value
    let content: _ConditionalContent<AnyView, AnyView>
    if branch != 0 {
        content = ViewBuilder.buildEither(first: firstView)
    } else {
        content = ViewBuilder.buildEither(second: secondView)
    }
    return retainView(AnyView(VStack { content }))
}

@_cdecl("SUIAccessibilityIdentifier")
public func SUIAccessibilityIdentifier(_ viewRef: UnsafeMutableRawPointer, _ identifierPtr: UnsafePointer<CChar>) -> UnsafeMutableRawPointer {
    let base = Unmanaged<Box<AnyView>>.fromOpaque(viewRef).takeUnretainedValue().value
    let identifier = String(cString: identifierPtr)
    let view = AnyView(base.accessibilityIdentifier(identifier))
    return Unmanaged.passRetained(Box(view)).toOpaque()
}

// SUIOracleHostingView adapts a bridge-owned AnyView to AppKit's host for an
// interaction-test control. It is available only in the tagged test bridge.
@_cdecl("SUIOracleHostingView")
public func SUIOracleHostingView(_ viewRef: UnsafeMutableRawPointer) -> UnsafeMutableRawPointer {
    let view = Unmanaged<Box<AnyView>>.fromOpaque(viewRef).takeUnretainedValue().value
    return Unmanaged.passRetained(NSHostingView(rootView: view)).toOpaque()
}

// SUIIdentityNew initializes an AnyView storage slot with a stateful leaf.
// It is a test fixture for observing SwiftUI identity across direct rootView
// replacement. Consumer builds never contain it.
private enum SUIIdentityCounter {
    nonisolated(unsafe) static var next = 0

    static func take() -> Int {
        next += 1
        return next
    }
}

private struct SUIIdentityLeaf: View {
    @State private var serial = SUIIdentityCounter.take()

    var body: some View {
        Text("identity-\(serial)")
    }
}

private struct SUIIdentityBreakLeaf: View {
    @State private var serial = SUIIdentityCounter.take()

    var body: some View {
        Text("identity-\(serial)")
    }
}

@_cdecl("SUIIdentityNew")
public func SUIIdentityNew(_ storage: UnsafeMutableRawPointer) {
    storage.assumingMemoryBound(to: AnyView.self).initialize(to: AnyView(SUIIdentityLeaf()))
}

@_cdecl("SUIIdentityBreakNew")
public func SUIIdentityBreakNew(_ storage: UnsafeMutableRawPointer) {
    storage.assumingMemoryBound(to: AnyView.self).initialize(to: AnyView(SUIIdentityBreakLeaf()))
}

// SUIIdentityForEachNew is a compiled-Swift control for T23. It isolates the
// test harness from the hand-built ForEach content closure by constructing the
// same stateful leaf beneath ForEach entirely in Swift.
@_cdecl("SUIIdentityForEachNew")
public func SUIIdentityForEachNew(_ storage: UnsafeMutableRawPointer, _ count: Int64) {
    storage.assumingMemoryBound(to: AnyView.self).initialize(to: AnyView(ForEach(0..<Int(count)) { _ in
        SUIIdentityLeaf()
    }))
}
#endif
