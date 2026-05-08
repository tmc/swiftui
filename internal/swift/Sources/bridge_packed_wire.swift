import Foundation

// Packed wire format decoder/encoder.
//
// Go side writes the following little-endian layout for string slices:
//
//     [count: UInt32] [ [len: UInt32] [bytes: len] ]*count
//
// and for ChoiceOption slices:
//
//     [count: UInt32] [ [label_len: UInt32] [label]
//                       [value_len: UInt32] [value] ]*count
//
// Strings are UTF-8 with no NUL terminator; the length is explicit.
//
// The decoder performs bounds checks and returns empty results on malformed
// input rather than trapping, which matches the Go-side behavior for a
// versioned FFI surface.

private let wirePayloadV1: UInt8 = 1

private struct PackedReader {
    let buffer: UnsafeBufferPointer<UInt8>
    var offset: Int = 0

    init(_ ptr: UnsafePointer<UInt8>?, _ length: Int) {
        if let ptr, length > 0 {
            self.buffer = UnsafeBufferPointer(start: ptr, count: length)
        } else {
            self.buffer = UnsafeBufferPointer(start: nil, count: 0)
        }
    }

    var remaining: Int { buffer.count - offset }

    mutating func readU32() -> UInt32? {
        guard remaining >= 4 else {
            return nil
        }
        let b0 = UInt32(buffer[offset])
        let b1 = UInt32(buffer[offset + 1])
        let b2 = UInt32(buffer[offset + 2])
        let b3 = UInt32(buffer[offset + 3])
        offset += 4
        return (b3 << 24) | (b2 << 16) | (b1 << 8) | b0
    }

    mutating func readString(_ length: Int) -> String? {
        guard length >= 0, remaining >= length else {
            return nil
        }
        let start = buffer.baseAddress!.advanced(by: offset)
        offset += length
        if length == 0 {
            return ""
        }
        let slice = UnsafeBufferPointer(start: start, count: length)
        return String(decoding: slice, as: UTF8.self)
    }
}

/// Decode a packed string slice from a pointer + length pair. Returns an empty
/// array on decode failure so callers can treat the operation as best-effort.
func suiDecodePackedStringSlice(_ ptr: UnsafePointer<UInt8>?, _ length: Int) -> [String] {
    var reader = PackedReader(ptr, length)
    guard let count = reader.readU32() else {
        return []
    }
    // Bound against remaining length to avoid large pre-allocations on a
    // malformed input.
    let cappedCount = min(Int(count), reader.remaining + 1)
    var out: [String] = []
    out.reserveCapacity(cappedCount)
    for _ in 0..<Int(count) {
        guard let length = reader.readU32(), let value = reader.readString(Int(length)) else {
            return []
        }
        out.append(value)
    }
    return out
}

/// Encode a Swift string array into the packed wire format. The returned
/// pointer is malloc-allocated and must be freed by the Go side via
/// SUIFreePackedBuffer (see bridge_packed_wire.swift).
func suiEncodePackedStringSlice(_ values: [String]) -> (UnsafeMutablePointer<UInt8>?, Int) {
    // First pass: compute the total byte length.
    var total = 4
    for s in values {
        total += 4
        total += s.utf8.count
    }
    let raw = malloc(total)
    guard let raw else {
        return (nil, 0)
    }
    let buf = raw.assumingMemoryBound(to: UInt8.self)
    var offset = 0
    // count
    writeU32(buf, offset, UInt32(values.count))
    offset += 4
    for s in values {
        let utf8 = Array(s.utf8)
        writeU32(buf, offset, UInt32(utf8.count))
        offset += 4
        if !utf8.isEmpty {
            _ = utf8.withUnsafeBufferPointer { src in
                memcpy(buf.advanced(by: offset), src.baseAddress, utf8.count)
            }
            offset += utf8.count
        }
    }
    return (buf, total)
}

private func writeU32(_ buf: UnsafeMutablePointer<UInt8>, _ offset: Int, _ value: UInt32) {
    buf[offset] = UInt8(value & 0xFF)
    buf[offset + 1] = UInt8((value >> 8) & 0xFF)
    buf[offset + 2] = UInt8((value >> 16) & 0xFF)
    buf[offset + 3] = UInt8((value >> 24) & 0xFF)
}

struct SUIDecodedChoiceOption: Hashable {
    let label: String
    let value: String
}

/// Decode a packed ChoiceOption slice.
func suiDecodePackedChoiceOptions(_ ptr: UnsafePointer<UInt8>?, _ length: Int) -> [SUIDecodedChoiceOption] {
    var reader = PackedReader(ptr, length)
    guard let count = reader.readU32() else {
        return []
    }
    var out: [SUIDecodedChoiceOption] = []
    out.reserveCapacity(min(Int(count), reader.remaining + 1))
    for _ in 0..<Int(count) {
        guard
            let labelLen = reader.readU32(),
            let label = reader.readString(Int(labelLen)),
            let valueLen = reader.readU32(),
            let value = reader.readString(Int(valueLen))
        else {
            return []
        }
        out.append(SUIDecodedChoiceOption(label: label, value: value))
    }
    return out
}

/// Decode a versioned single-payload envelope.
///
///     [version: UInt8] [kind: UInt8] [len: UInt32] [bytes: len]
///
/// Returns nil on unknown version, unknown kind, or truncation.
func suiDecodePackedPayload(_ ptr: UnsafePointer<UInt8>?, _ length: Int) -> (kind: UInt8, value: String)? {
    guard let ptr, length >= 6 else {
        return nil
    }
    let buf = UnsafeBufferPointer(start: ptr, count: length)
    let version = buf[0]
    guard version == wirePayloadV1 else {
        return nil
    }
    let kind = buf[1]
    // Accept only kinds 1..3. Keep in sync with wirePayloadKind* in Go.
    guard kind >= 1 && kind <= 3 else {
        return nil
    }
    let len = UInt32(buf[2]) | (UInt32(buf[3]) << 8) | (UInt32(buf[4]) << 16) | (UInt32(buf[5]) << 24)
    guard Int(len) <= length - 6 else {
        return nil
    }
    let start = ptr.advanced(by: 6)
    let slice = UnsafeBufferPointer(start: start, count: Int(len))
    let value = String(decoding: slice, as: UTF8.self)
    return (kind, value)
}

/// Free a buffer allocated by suiEncodePackedStringSlice. Exposed to Go via
/// SUIFreePackedBuffer.
@_cdecl("SUIFreePackedBuffer")
public func SUIFreePackedBuffer(_ ptr: UnsafeMutablePointer<UInt8>?) {
    guard let ptr else {
        return
    }
    free(UnsafeMutableRawPointer(ptr))
}
