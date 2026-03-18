import Foundation

@MainActor
final class NumberBindingState: ObservableObject {
    @Published var value: Double
    init(_ value: Double) { self.value = value }
}

@MainActor
final class DateBindingState: ObservableObject {
    @Published var value: Date
    init(_ value: Date) { self.value = value }
}

@MainActor
final class OptionalNumberBindingState: ObservableObject {
    @Published var value: Double?
    init(_ value: Double?) { self.value = value }
}

@MainActor
final class OptionalDateBindingState: ObservableObject {
    @Published var value: Date?
    init(_ value: Date?) { self.value = value }
}

@MainActor
final class NumberRangeBindingState: ObservableObject {
    @Published var value: ClosedRange<Double>?
    init(_ value: ClosedRange<Double>?) { self.value = value }
}

@MainActor
final class DateRangeBindingState: ObservableObject {
    @Published var value: ClosedRange<Date>?
    init(_ value: ClosedRange<Date>?) { self.value = value }
}

@_cdecl("CHStateCreateNumber")
@MainActor
public func CHStateCreateNumber(_ value: Double) -> UnsafeMutableRawPointer {
    Unmanaged.passRetained(NumberBindingState(value)).toOpaque()
}

@_cdecl("CHStateGetNumber")
@MainActor
public func CHStateGetNumber(_ ref: UnsafeMutableRawPointer) -> Double {
    Unmanaged<NumberBindingState>.fromOpaque(ref).takeUnretainedValue().value
}

@_cdecl("CHStateSetNumber")
@MainActor
public func CHStateSetNumber(_ ref: UnsafeMutableRawPointer, _ value: Double) {
    Unmanaged<NumberBindingState>.fromOpaque(ref).takeUnretainedValue().value = value
}

@_cdecl("CHStateCreateDate")
@MainActor
public func CHStateCreateDate(_ seconds: Double) -> UnsafeMutableRawPointer {
    Unmanaged.passRetained(DateBindingState(Date(timeIntervalSince1970: seconds))).toOpaque()
}

@_cdecl("CHStateGetDate")
@MainActor
public func CHStateGetDate(_ ref: UnsafeMutableRawPointer) -> Double {
    Unmanaged<DateBindingState>.fromOpaque(ref).takeUnretainedValue().value.timeIntervalSince1970
}

@_cdecl("CHStateSetDate")
@MainActor
public func CHStateSetDate(_ ref: UnsafeMutableRawPointer, _ seconds: Double) {
    Unmanaged<DateBindingState>.fromOpaque(ref).takeUnretainedValue().value = Date(timeIntervalSince1970: seconds)
}

@_cdecl("CHStateCreateOptionalNumber")
@MainActor
public func CHStateCreateOptionalNumber(_ value: Double, _ hasValue: Int32) -> UnsafeMutableRawPointer {
    let wrapped = hasValue != 0 ? value : nil
    return Unmanaged.passRetained(OptionalNumberBindingState(wrapped)).toOpaque()
}

@_cdecl("CHStateHasOptionalNumber")
@MainActor
public func CHStateHasOptionalNumber(_ ref: UnsafeMutableRawPointer) -> Int32 {
    Unmanaged<OptionalNumberBindingState>.fromOpaque(ref).takeUnretainedValue().value == nil ? 0 : 1
}

@_cdecl("CHStateGetOptionalNumber")
@MainActor
public func CHStateGetOptionalNumber(_ ref: UnsafeMutableRawPointer) -> Double {
    Unmanaged<OptionalNumberBindingState>.fromOpaque(ref).takeUnretainedValue().value ?? 0
}

@_cdecl("CHStateSetOptionalNumber")
@MainActor
public func CHStateSetOptionalNumber(_ ref: UnsafeMutableRawPointer, _ value: Double) {
    Unmanaged<OptionalNumberBindingState>.fromOpaque(ref).takeUnretainedValue().value = value
}

@_cdecl("CHStateClearOptionalNumber")
@MainActor
public func CHStateClearOptionalNumber(_ ref: UnsafeMutableRawPointer) {
    Unmanaged<OptionalNumberBindingState>.fromOpaque(ref).takeUnretainedValue().value = nil
}

@_cdecl("CHStateCreateOptionalDate")
@MainActor
public func CHStateCreateOptionalDate(_ seconds: Double, _ hasValue: Int32) -> UnsafeMutableRawPointer {
    let wrapped = hasValue != 0 ? Date(timeIntervalSince1970: seconds) : nil
    return Unmanaged.passRetained(OptionalDateBindingState(wrapped)).toOpaque()
}

@_cdecl("CHStateHasOptionalDate")
@MainActor
public func CHStateHasOptionalDate(_ ref: UnsafeMutableRawPointer) -> Int32 {
    Unmanaged<OptionalDateBindingState>.fromOpaque(ref).takeUnretainedValue().value == nil ? 0 : 1
}

@_cdecl("CHStateGetOptionalDate")
@MainActor
public func CHStateGetOptionalDate(_ ref: UnsafeMutableRawPointer) -> Double {
    Unmanaged<OptionalDateBindingState>.fromOpaque(ref).takeUnretainedValue().value?.timeIntervalSince1970 ?? 0
}

@_cdecl("CHStateSetOptionalDate")
@MainActor
public func CHStateSetOptionalDate(_ ref: UnsafeMutableRawPointer, _ seconds: Double) {
    Unmanaged<OptionalDateBindingState>.fromOpaque(ref).takeUnretainedValue().value = Date(timeIntervalSince1970: seconds)
}

@_cdecl("CHStateClearOptionalDate")
@MainActor
public func CHStateClearOptionalDate(_ ref: UnsafeMutableRawPointer) {
    Unmanaged<OptionalDateBindingState>.fromOpaque(ref).takeUnretainedValue().value = nil
}

@_cdecl("CHStateCreateNumberRange")
@MainActor
public func CHStateCreateNumberRange(_ start: Double, _ end: Double, _ hasValue: Int32) -> UnsafeMutableRawPointer {
    let wrapped = hasValue != 0 ? start ... end : nil
    return Unmanaged.passRetained(NumberRangeBindingState(wrapped)).toOpaque()
}

@_cdecl("CHStateHasNumberRange")
@MainActor
public func CHStateHasNumberRange(_ ref: UnsafeMutableRawPointer) -> Int32 {
    Unmanaged<NumberRangeBindingState>.fromOpaque(ref).takeUnretainedValue().value == nil ? 0 : 1
}

@_cdecl("CHStateGetNumberRangeStart")
@MainActor
public func CHStateGetNumberRangeStart(_ ref: UnsafeMutableRawPointer) -> Double {
    Unmanaged<NumberRangeBindingState>.fromOpaque(ref).takeUnretainedValue().value?.lowerBound ?? 0
}

@_cdecl("CHStateGetNumberRangeEnd")
@MainActor
public func CHStateGetNumberRangeEnd(_ ref: UnsafeMutableRawPointer) -> Double {
    Unmanaged<NumberRangeBindingState>.fromOpaque(ref).takeUnretainedValue().value?.upperBound ?? 0
}

@_cdecl("CHStateSetNumberRange")
@MainActor
public func CHStateSetNumberRange(_ ref: UnsafeMutableRawPointer, _ start: Double, _ end: Double) {
    Unmanaged<NumberRangeBindingState>.fromOpaque(ref).takeUnretainedValue().value = start ... end
}

@_cdecl("CHStateClearNumberRange")
@MainActor
public func CHStateClearNumberRange(_ ref: UnsafeMutableRawPointer) {
    Unmanaged<NumberRangeBindingState>.fromOpaque(ref).takeUnretainedValue().value = nil
}

@_cdecl("CHStateCreateDateRange")
@MainActor
public func CHStateCreateDateRange(_ start: Double, _ end: Double, _ hasValue: Int32) -> UnsafeMutableRawPointer {
    let wrapped = hasValue != 0 ? Date(timeIntervalSince1970: start) ... Date(timeIntervalSince1970: end) : nil
    return Unmanaged.passRetained(DateRangeBindingState(wrapped)).toOpaque()
}

@_cdecl("CHStateHasDateRange")
@MainActor
public func CHStateHasDateRange(_ ref: UnsafeMutableRawPointer) -> Int32 {
    Unmanaged<DateRangeBindingState>.fromOpaque(ref).takeUnretainedValue().value == nil ? 0 : 1
}

@_cdecl("CHStateGetDateRangeStart")
@MainActor
public func CHStateGetDateRangeStart(_ ref: UnsafeMutableRawPointer) -> Double {
    Unmanaged<DateRangeBindingState>.fromOpaque(ref).takeUnretainedValue().value?.lowerBound.timeIntervalSince1970 ?? 0
}

@_cdecl("CHStateGetDateRangeEnd")
@MainActor
public func CHStateGetDateRangeEnd(_ ref: UnsafeMutableRawPointer) -> Double {
    Unmanaged<DateRangeBindingState>.fromOpaque(ref).takeUnretainedValue().value?.upperBound.timeIntervalSince1970 ?? 0
}

@_cdecl("CHStateSetDateRange")
@MainActor
public func CHStateSetDateRange(_ ref: UnsafeMutableRawPointer, _ start: Double, _ end: Double) {
    Unmanaged<DateRangeBindingState>.fromOpaque(ref).takeUnretainedValue().value = Date(timeIntervalSince1970: start) ... Date(timeIntervalSince1970: end)
}

@_cdecl("CHStateClearDateRange")
@MainActor
public func CHStateClearDateRange(_ ref: UnsafeMutableRawPointer) {
    Unmanaged<DateRangeBindingState>.fromOpaque(ref).takeUnretainedValue().value = nil
}

func decodeStateRef(_ spec: StateRefSpec?) -> UnsafeMutableRawPointer? {
    decodePointer(spec?.ptr)
}
