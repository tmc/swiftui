import Charts
import Foundation
import SwiftUI

final class Box<T>: @unchecked Sendable {
    var value: T
    init(_ value: T) { self.value = value }
}

func decodeColor(_ r: Double, _ g: Double, _ b: Double, _ a: Double) -> Color {
    Color(red: r, green: g, blue: b, opacity: a)
}

func calendarUnit(_ unit: Int32) -> Calendar.Component {
    switch unit {
    case 1: return .weekOfYear
    case 2: return .month
    case 3: return .year
    case 4: return .hour
    case 5: return .minute
    default: return .day
    }
}

@_cdecl("CHRetain")
public func CHRetain(_ ref: UnsafeMutableRawPointer) -> UnsafeMutableRawPointer {
    _ = Unmanaged<AnyObject>.fromOpaque(ref).retain()
    return ref
}

@_cdecl("CHRelease")
public func CHRelease(_ ref: UnsafeMutableRawPointer) {
    Unmanaged<AnyObject>.fromOpaque(ref).release()
}
