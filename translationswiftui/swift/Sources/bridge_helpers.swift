import Foundation
import Translation
import SwiftUI

// Box wraps a value type so it can be stored in Unmanaged.
final class Box<T>: @unchecked Sendable {
    var value: T
    init(_ value: T) { self.value = value }
}

@_cdecl("TRS_Retain")
public func TRS_Retain(_ ptr: UnsafeMutableRawPointer) -> UnsafeMutableRawPointer {
    _ = Unmanaged<AnyObject>.fromOpaque(ptr).retain()
    return ptr
}

@_cdecl("TRS_Release")
public func TRS_Release(_ ptr: UnsafeMutableRawPointer) {
    Unmanaged<AnyObject>.fromOpaque(ptr).release()
}

@_cdecl("TRS_FreeString")
public func TRS_FreeString(_ ptr: UnsafeMutablePointer<CChar>) {
    ptr.deallocate()
}
