import Foundation

private let stateKindNumber: Int32 = 0
private let stateKindDate: Int32 = 1
private let stateKindOptionalNumber: Int32 = 2
private let stateKindOptionalDate: Int32 = 3
private let stateKindNumberRange: Int32 = 4
private let stateKindDateRange: Int32 = 5

nonisolated(unsafe) var _CHStateChangeCallback: (@convention(c) (UInt, Int32, Double, Double, Int32) -> Void)?

@_cdecl("CHSetStateChangeCallback")
public func CHSetStateChangeCallback(_ fn: @convention(c) (UInt, Int32, Double, Double, Int32) -> Void) {
    _CHStateChangeCallback = fn
}

private func notifyStateChange(
    _ object: AnyObject,
    kind: Int32,
    value0: Double,
    value1: Double = 0,
    hasValue: Bool = true
) {
    let ptr = UInt(bitPattern: Unmanaged.passUnretained(object).toOpaque())
    _CHStateChangeCallback?(ptr, kind, value0, value1, hasValue ? 1 : 0)
}

final class NumberBindingState: ObservableObject {
    @Published var value: Double {
        didSet { notifyStateChange(self, kind: stateKindNumber, value0: value) }
    }
    init(_ value: Double) { self.value = value }
}

final class DateBindingState: ObservableObject {
    @Published var value: Date {
        didSet { notifyStateChange(self, kind: stateKindDate, value0: value.timeIntervalSince1970) }
    }
    init(_ value: Date) { self.value = value }
}

final class OptionalNumberBindingState: ObservableObject {
    @Published var value: Double? {
        didSet { notifyStateChange(self, kind: stateKindOptionalNumber, value0: value ?? 0, hasValue: value != nil) }
    }
    init(_ value: Double?) { self.value = value }
}

final class OptionalDateBindingState: ObservableObject {
    @Published var value: Date? {
        didSet {
            notifyStateChange(
                self,
                kind: stateKindOptionalDate,
                value0: value?.timeIntervalSince1970 ?? 0,
                hasValue: value != nil
            )
        }
    }
    init(_ value: Date?) { self.value = value }
}

final class NumberRangeBindingState: ObservableObject {
    @Published var value: ClosedRange<Double>? {
        didSet {
            notifyStateChange(
                self,
                kind: stateKindNumberRange,
                value0: value?.lowerBound ?? 0,
                value1: value?.upperBound ?? 0,
                hasValue: value != nil
            )
        }
    }
    init(_ value: ClosedRange<Double>?) { self.value = value }
}

final class DateRangeBindingState: ObservableObject {
    @Published var value: ClosedRange<Date>? {
        didSet {
            notifyStateChange(
                self,
                kind: stateKindDateRange,
                value0: value?.lowerBound.timeIntervalSince1970 ?? 0,
                value1: value?.upperBound.timeIntervalSince1970 ?? 0,
                hasValue: value != nil
            )
        }
    }
    init(_ value: ClosedRange<Date>?) { self.value = value }
}

@_cdecl("CHStateCreateNumber")
public func CHStateCreateNumber(_ value: Double) -> UnsafeMutableRawPointer {
    Unmanaged.passRetained(NumberBindingState(value)).toOpaque()
}

@_cdecl("CHStateGetNumber")
public func CHStateGetNumber(_ ref: UnsafeMutableRawPointer) -> Double {
    Unmanaged<NumberBindingState>.fromOpaque(ref).takeUnretainedValue().value
}

@_cdecl("CHStateSetNumber")
public func CHStateSetNumber(_ ref: UnsafeMutableRawPointer, _ value: Double) {
    Unmanaged<NumberBindingState>.fromOpaque(ref).takeUnretainedValue().value = value
}

@_cdecl("CHStateCreateDate")
public func CHStateCreateDate(_ seconds: Double) -> UnsafeMutableRawPointer {
    Unmanaged.passRetained(DateBindingState(Date(timeIntervalSince1970: seconds))).toOpaque()
}

@_cdecl("CHStateGetDate")
public func CHStateGetDate(_ ref: UnsafeMutableRawPointer) -> Double {
    Unmanaged<DateBindingState>.fromOpaque(ref).takeUnretainedValue().value.timeIntervalSince1970
}

@_cdecl("CHStateSetDate")
public func CHStateSetDate(_ ref: UnsafeMutableRawPointer, _ seconds: Double) {
    Unmanaged<DateBindingState>.fromOpaque(ref).takeUnretainedValue().value = Date(timeIntervalSince1970: seconds)
}

@_cdecl("CHStateCreateOptionalNumber")
public func CHStateCreateOptionalNumber(_ value: Double, _ hasValue: Int32) -> UnsafeMutableRawPointer {
    let wrapped = hasValue != 0 ? value : nil
    return Unmanaged.passRetained(OptionalNumberBindingState(wrapped)).toOpaque()
}

@_cdecl("CHStateHasOptionalNumber")
public func CHStateHasOptionalNumber(_ ref: UnsafeMutableRawPointer) -> Int32 {
    Unmanaged<OptionalNumberBindingState>.fromOpaque(ref).takeUnretainedValue().value == nil ? 0 : 1
}

@_cdecl("CHStateGetOptionalNumber")
public func CHStateGetOptionalNumber(_ ref: UnsafeMutableRawPointer) -> Double {
    Unmanaged<OptionalNumberBindingState>.fromOpaque(ref).takeUnretainedValue().value ?? 0
}

@_cdecl("CHStateSetOptionalNumber")
public func CHStateSetOptionalNumber(_ ref: UnsafeMutableRawPointer, _ value: Double) {
    Unmanaged<OptionalNumberBindingState>.fromOpaque(ref).takeUnretainedValue().value = value
}

@_cdecl("CHStateClearOptionalNumber")
public func CHStateClearOptionalNumber(_ ref: UnsafeMutableRawPointer) {
    Unmanaged<OptionalNumberBindingState>.fromOpaque(ref).takeUnretainedValue().value = nil
}

@_cdecl("CHStateCreateOptionalDate")
public func CHStateCreateOptionalDate(_ seconds: Double, _ hasValue: Int32) -> UnsafeMutableRawPointer {
    let wrapped = hasValue != 0 ? Date(timeIntervalSince1970: seconds) : nil
    return Unmanaged.passRetained(OptionalDateBindingState(wrapped)).toOpaque()
}

@_cdecl("CHStateHasOptionalDate")
public func CHStateHasOptionalDate(_ ref: UnsafeMutableRawPointer) -> Int32 {
    Unmanaged<OptionalDateBindingState>.fromOpaque(ref).takeUnretainedValue().value == nil ? 0 : 1
}

@_cdecl("CHStateGetOptionalDate")
public func CHStateGetOptionalDate(_ ref: UnsafeMutableRawPointer) -> Double {
    Unmanaged<OptionalDateBindingState>.fromOpaque(ref).takeUnretainedValue().value?.timeIntervalSince1970 ?? 0
}

@_cdecl("CHStateSetOptionalDate")
public func CHStateSetOptionalDate(_ ref: UnsafeMutableRawPointer, _ seconds: Double) {
    Unmanaged<OptionalDateBindingState>.fromOpaque(ref).takeUnretainedValue().value = Date(timeIntervalSince1970: seconds)
}

@_cdecl("CHStateClearOptionalDate")
public func CHStateClearOptionalDate(_ ref: UnsafeMutableRawPointer) {
    Unmanaged<OptionalDateBindingState>.fromOpaque(ref).takeUnretainedValue().value = nil
}

@_cdecl("CHStateCreateNumberRange")
public func CHStateCreateNumberRange(_ start: Double, _ end: Double, _ hasValue: Int32) -> UnsafeMutableRawPointer {
    let wrapped = hasValue != 0 ? start ... end : nil
    return Unmanaged.passRetained(NumberRangeBindingState(wrapped)).toOpaque()
}

@_cdecl("CHStateHasNumberRange")
public func CHStateHasNumberRange(_ ref: UnsafeMutableRawPointer) -> Int32 {
    Unmanaged<NumberRangeBindingState>.fromOpaque(ref).takeUnretainedValue().value == nil ? 0 : 1
}

@_cdecl("CHStateGetNumberRangeStart")
public func CHStateGetNumberRangeStart(_ ref: UnsafeMutableRawPointer) -> Double {
    Unmanaged<NumberRangeBindingState>.fromOpaque(ref).takeUnretainedValue().value?.lowerBound ?? 0
}

@_cdecl("CHStateGetNumberRangeEnd")
public func CHStateGetNumberRangeEnd(_ ref: UnsafeMutableRawPointer) -> Double {
    Unmanaged<NumberRangeBindingState>.fromOpaque(ref).takeUnretainedValue().value?.upperBound ?? 0
}

@_cdecl("CHStateSetNumberRange")
public func CHStateSetNumberRange(_ ref: UnsafeMutableRawPointer, _ start: Double, _ end: Double) {
    Unmanaged<NumberRangeBindingState>.fromOpaque(ref).takeUnretainedValue().value = start ... end
}

@_cdecl("CHStateClearNumberRange")
public func CHStateClearNumberRange(_ ref: UnsafeMutableRawPointer) {
    Unmanaged<NumberRangeBindingState>.fromOpaque(ref).takeUnretainedValue().value = nil
}

@_cdecl("CHStateCreateDateRange")
public func CHStateCreateDateRange(_ start: Double, _ end: Double, _ hasValue: Int32) -> UnsafeMutableRawPointer {
    let wrapped = hasValue != 0 ? Date(timeIntervalSince1970: start) ... Date(timeIntervalSince1970: end) : nil
    return Unmanaged.passRetained(DateRangeBindingState(wrapped)).toOpaque()
}

@_cdecl("CHStateHasDateRange")
public func CHStateHasDateRange(_ ref: UnsafeMutableRawPointer) -> Int32 {
    Unmanaged<DateRangeBindingState>.fromOpaque(ref).takeUnretainedValue().value == nil ? 0 : 1
}

@_cdecl("CHStateGetDateRangeStart")
public func CHStateGetDateRangeStart(_ ref: UnsafeMutableRawPointer) -> Double {
    Unmanaged<DateRangeBindingState>.fromOpaque(ref).takeUnretainedValue().value?.lowerBound.timeIntervalSince1970 ?? 0
}

@_cdecl("CHStateGetDateRangeEnd")
public func CHStateGetDateRangeEnd(_ ref: UnsafeMutableRawPointer) -> Double {
    Unmanaged<DateRangeBindingState>.fromOpaque(ref).takeUnretainedValue().value?.upperBound.timeIntervalSince1970 ?? 0
}

@_cdecl("CHStateSetDateRange")
public func CHStateSetDateRange(_ ref: UnsafeMutableRawPointer, _ start: Double, _ end: Double) {
    Unmanaged<DateRangeBindingState>.fromOpaque(ref).takeUnretainedValue().value = Date(timeIntervalSince1970: start) ... Date(timeIntervalSince1970: end)
}

@_cdecl("CHStateClearDateRange")
public func CHStateClearDateRange(_ ref: UnsafeMutableRawPointer) {
    Unmanaged<DateRangeBindingState>.fromOpaque(ref).takeUnretainedValue().value = nil
}

func decodeStateRef(_ spec: StateRefSpec?) -> UnsafeMutableRawPointer? {
    decodePointer(spec?.ptr)
}
