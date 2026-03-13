import SpriteKit
import SwiftUI


// MARK: - NewSpriteView

@_cdecl("SPS_SpriteViewCreate")
@MainActor
public func SPS_SpriteViewCreate(_ scene: UnsafeMutableRawPointer) -> UnsafeMutableRawPointer {
    let sc = Unmanaged<SKScene>.fromOpaque(scene).takeUnretainedValue()
    let view = AnyView(SpriteView(scene: sc))
    return Unmanaged.passRetained(Box(view)).toOpaque()
}

// MARK: - NewSpriteViewWithTransition

@_cdecl("SPS_SpriteViewCreateWithTransition")
@MainActor
public func SPS_SpriteViewCreateWithTransition(_ scene: UnsafeMutableRawPointer, _ transition: UnsafeMutableRawPointer) -> UnsafeMutableRawPointer {
    let sc = Unmanaged<SKScene>.fromOpaque(scene).takeUnretainedValue()
    let tr = Unmanaged<SKTransition>.fromOpaque(transition).takeUnretainedValue()
    let view = AnyView(SpriteView(scene: sc, transition: tr))
    return Unmanaged.passRetained(Box(view)).toOpaque()
}

// MARK: - NewSpriteViewWithOptions

@_cdecl("SPS_SpriteViewCreateWithOptions")
@MainActor
public func SPS_SpriteViewCreateWithOptions(_ scene: UnsafeMutableRawPointer, _ isPaused: Bool, _ preferredFramesPerSecond: Int32) -> UnsafeMutableRawPointer {
    let sc = Unmanaged<SKScene>.fromOpaque(scene).takeUnretainedValue()
    let view = AnyView(SpriteView(scene: sc, isPaused: isPaused, preferredFramesPerSecond: Int(preferredFramesPerSecond)))
    return Unmanaged.passRetained(Box(view)).toOpaque()
}
