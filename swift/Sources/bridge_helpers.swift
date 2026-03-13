import Foundation


final class Box<T>: @unchecked Sendable {
    var value: T
    init(_ value: T) { self.value = value }
}

@_cdecl("SUIRelease")
public func SUIRelease(_ ref: UnsafeMutableRawPointer) {
    Unmanaged<AnyObject>.fromOpaque(ref).release()
}

@_cdecl("SUIFreeString")
public func SUIFreeString(_ s: UnsafeMutablePointer<CChar>) {
    free(s)
}

// Button callback function pointer, set by Go at init time.
nonisolated(unsafe) var _SUIButtonCallback: (@convention(c) (UInt) -> Void)?

@_cdecl("SUISetButtonCallback")
public func SUISetButtonCallback(_ fn: @convention(c) (UInt) -> Void) {
    _SUIButtonCallback = fn
}
