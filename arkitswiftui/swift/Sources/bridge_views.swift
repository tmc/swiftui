import ARKit
import SwiftUI


// MARK: - NewARView

@_cdecl("ARS_ARViewCreate")
@MainActor
public func ARS_ARViewCreate() -> UnsafeMutableRawPointer {
    #if os(iOS) || os(visionOS)
    let arView = ARView(frame: .zero)
    let view = AnyView(ARViewContainer(arView: arView))
    return Unmanaged.passRetained(Box(view)).toOpaque()
    #else
    let view = AnyView(EmptyView())
    return Unmanaged.passRetained(Box(view)).toOpaque()
    #endif
}
