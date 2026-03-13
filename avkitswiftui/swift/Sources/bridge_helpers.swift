import Foundation
import AVKit
import SwiftUI
import AVFoundation

// Box wraps a value type so it can be stored in Unmanaged.
final class Box<T>: @unchecked Sendable {
    var value: T
    init(_ value: T) { self.value = value }
}

@_cdecl("AVS_Retain")
public func AVS_Retain(_ ptr: UnsafeMutableRawPointer) -> UnsafeMutableRawPointer {
    _ = Unmanaged<AnyObject>.fromOpaque(ptr).retain()
    return ptr
}

@_cdecl("AVS_Release")
public func AVS_Release(_ ptr: UnsafeMutableRawPointer) {
    Unmanaged<AnyObject>.fromOpaque(ptr).release()
}

@_cdecl("AVS_FreeString")
public func AVS_FreeString(_ ptr: UnsafeMutablePointer<CChar>) {
    ptr.deallocate()
}
