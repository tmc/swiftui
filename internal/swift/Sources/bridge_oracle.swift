// Test-only SwiftUI oracle exports. This file is hand-owned: applegen never
// writes it. They are present only in a bridge compiled with
// -D SWIFTUI_ORACLE, which swiftgo's swiftuioracle-tagged tests build.

#if SWIFTUI_ORACLE
import Foundation
import Observation
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

    static func reset(to value: Int) {
        next = value
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

// SUIIdentityResetCounter makes the serial fixture deterministic for identity
// controls that compare a persistent tree with an independently built tree.
@_cdecl("SUIIdentityResetCounter")
public func SUIIdentityResetCounter(_ value: Int64) {
    SUIIdentityCounter.reset(to: Int(value))
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

// SUIIntSelfKeyPath returns a compiler-created KeyPath<Int, Int> for T29's
// direct-ABI control. The caller owns the returned reference.
@_cdecl("SUIIntSelfKeyPath")
public func SUIIntSelfKeyPath() -> UnsafeMutableRawPointer {
    let keyPath: KeyPath<Int, Int> = \.self
    return Unmanaged.passRetained(keyPath as AnyObject).toOpaque()
}

@_cdecl("SUIIdentityKeyedForEachNew")
public func SUIIdentityKeyedForEachNew(_ storage: UnsafeMutableRawPointer, _ reverse: Int32) {
    let values = reverse == 0 ? [0, 1] : [1, 0]
    storage.assumingMemoryBound(to: AnyView.self).initialize(to: AnyView(ForEach(values, id: \.self) { _ in
        SUIIdentityLeaf()
    }))
}

// SUIObservationModel and its leaf are a compiled-control for T31. The
// @Observable macro supplies the registrar and property access instrumentation
// that the direct probe must eventually reproduce without this fixture.
@MainActor @Observable
private final class SUIObservationModel {
    var value = 0

    func registrarAddress() -> UnsafeMutableRawPointer {
        withUnsafePointer(to: _$observationRegistrar) {
            UnsafeMutableRawPointer(mutating: $0)
        }
    }

    func valueAddress() -> UnsafeMutableRawPointer {
        withUnsafeMutablePointer(to: &_value) {
            UnsafeMutableRawPointer($0)
        }
    }
}

private struct SUIObservationLeaf: View {
    let model: SUIObservationModel

    var body: some View {
        Text("observation-\(model.value)")
    }
}

private struct SUIObservationBreakLeaf: View {
    let model: SUIObservationModel

    var body: some View {
        // Keep the model alive but do not read its observed property.
        Text("observation-0")
    }
}

@MainActor @_cdecl("SUIObservationNew")
public func SUIObservationNew(_ storage: UnsafeMutableRawPointer) -> UnsafeMutableRawPointer {
    let model = SUIObservationModel()
    storage.assumingMemoryBound(to: AnyView.self).initialize(to: AnyView(SUIObservationLeaf(model: model)))
    return Unmanaged.passRetained(model).toOpaque()
}

@MainActor @_cdecl("SUIObservationViewForExisting")
public func SUIObservationViewForExisting(_ storage: UnsafeMutableRawPointer, _ raw: UnsafeMutableRawPointer) {
    let model = Unmanaged<SUIObservationModel>.fromOpaque(raw).takeUnretainedValue()
    storage.assumingMemoryBound(to: AnyView.self).initialize(to: AnyView(SUIObservationLeaf(model: model)))
}

@MainActor @_cdecl("SUIObservationBreakNew")
public func SUIObservationBreakNew(_ storage: UnsafeMutableRawPointer) -> UnsafeMutableRawPointer {
    let model = SUIObservationModel()
    storage.assumingMemoryBound(to: AnyView.self).initialize(to: AnyView(SUIObservationBreakLeaf(model: model)))
    return Unmanaged.passRetained(model).toOpaque()
}

@MainActor @_cdecl("SUIObservationSet")
public func SUIObservationSet(_ raw: UnsafeMutableRawPointer, _ value: Int64) {
    Unmanaged<SUIObservationModel>.fromOpaque(raw).takeUnretainedValue().value = Int(value)
}

// The remaining exports are fixture-only ABI observables for T31. They expose
// the macro-generated layout without implementing any part of the Go path.
@MainActor @_cdecl("SUIObservationMetadata")
public func SUIObservationMetadata() -> UnsafeRawPointer {
    unsafeBitCast(SUIObservationModel.self, to: UnsafeRawPointer.self)
}

@MainActor @_cdecl("SUIObservationValueKeyPath")
public func SUIObservationValueKeyPath() -> UnsafeMutableRawPointer {
    Unmanaged.passRetained(\SUIObservationModel.value as AnyObject).toOpaque()
}

@MainActor @_cdecl("SUIObservationRegistrarAddress")
public func SUIObservationRegistrarAddress(_ raw: UnsafeMutableRawPointer) -> UnsafeMutableRawPointer {
    let model = Unmanaged<SUIObservationModel>.fromOpaque(raw).takeUnretainedValue()
    return model.registrarAddress()
}

@MainActor @_cdecl("SUIObservationValueAddress")
public func SUIObservationValueAddress(_ raw: UnsafeMutableRawPointer) -> UnsafeMutableRawPointer {
    let model = Unmanaged<SUIObservationModel>.fromOpaque(raw).takeUnretainedValue()
    return model.valueAddress()
}

@MainActor @_cdecl("SUIObservationRegistrarSize")
public func SUIObservationRegistrarSize() -> Int {
    MemoryLayout<ObservationRegistrar>.stride
}
#endif
