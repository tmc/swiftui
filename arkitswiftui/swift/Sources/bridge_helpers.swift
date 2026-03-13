import Foundation
import ARKit
import SwiftUI

// Box wraps a value type so it can be stored in Unmanaged.
final class Box<T>: @unchecked Sendable {
    var value: T
    init(_ value: T) { self.value = value }
}

@_cdecl("ARS_Retain")
public func ARS_Retain(_ ptr: UnsafeMutableRawPointer) -> UnsafeMutableRawPointer {
    _ = Unmanaged<AnyObject>.fromOpaque(ptr).retain()
    return ptr
}

@_cdecl("ARS_Release")
public func ARS_Release(_ ptr: UnsafeMutableRawPointer) {
    Unmanaged<AnyObject>.fromOpaque(ptr).release()
}

@_cdecl("ARS_FreeString")
public func ARS_FreeString(_ ptr: UnsafeMutablePointer<CChar>) {
    ptr.deallocate()
}
