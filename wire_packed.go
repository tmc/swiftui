package swiftui

import (
	"encoding/binary"
	"errors"
	"runtime"
	"sync"
	"unsafe"
)

// Packed wire format for FFI payloads that would otherwise be serialized as
// JSON. P4 replaces marshalStringSlice/marshalChoiceOptions on the Go->Swift
// hot path with a length-prefix layout encoded onto a pooled scratch buffer
// and passed to Swift as a (pointer, length) pair.
//
// String slice layout (little-endian):
//
//	[count: uint32] [ [len: uint32] [bytes: len] ]*count
//
// ChoiceOption slice layout (little-endian):
//
//	[count: uint32] [ [label_len: uint32] [label bytes]
//	                  [value_len: uint32] [value bytes] ]*count
//
// Strings are UTF-8 with no NUL terminator. The Swift side reads the length
// explicitly.
//
// Structured single-payload envelope (ShareItem / DropPayload / PastePayload):
//
//	[version: uint8] [kind: uint8] [len: uint32] [bytes: len]
//
// The version byte lets the Swift decoder reject unknown encodings with a
// clean error instead of panicking or mis-parsing. ShareItem/DropPayload/
// PastePayload already flatten to a "kind + value" pair at the Go boundary
// (see share_drop.go), so the envelope exists as a reusable helper for
// future use. It is not on the hot path today.
//
// The JSON encoders remain available as marshalStringSliceJSON and
// marshalChoiceOptionsJSON for debug use and persistence paths. The hot
// call sites use the packed helpers below.

// Wire version constants. Bump when the layout changes; Swift must reject
// unknown versions.
const (
	wirePayloadV1 uint8 = 1
)

// Payload kinds for structured single-value payloads.
const (
	wirePayloadKindText uint8 = 1
	wirePayloadKindURL  uint8 = 2
	wirePayloadKindFile uint8 = 3
)

var errPackedShortBuffer = errors.New("swiftui: packed wire: short buffer")
var errPackedBadVersion = errors.New("swiftui: packed wire: unsupported version")
var errPackedBadKind = errors.New("swiftui: packed wire: unsupported kind")

// packedBufPool recycles []byte scratch buffers used to encode packed
// payloads. Mirrors the cstringBufPool pattern.
var packedBufPool = sync.Pool{
	New: func() any {
		buf := make([]byte, 0, 128)
		return &buf
	},
}

// appendUint32LE appends v in little-endian order to buf.
func appendUint32LE(buf []byte, v uint32) []byte {
	var tmp [4]byte
	binary.LittleEndian.PutUint32(tmp[:], v)
	return append(buf, tmp[:]...)
}

// encodePackedStringSlice writes v into dst in the string-slice layout and
// returns the updated slice.
func encodePackedStringSlice(dst []byte, v []string) []byte {
	dst = appendUint32LE(dst, uint32(len(v)))
	for _, s := range v {
		dst = appendUint32LE(dst, uint32(len(s)))
		dst = append(dst, s...)
	}
	return dst
}

// decodePackedStringSlice decodes a buffer written by encodePackedStringSlice.
// Used by tests and the fuzz harness; the Swift side has its own decoder.
func decodePackedStringSlice(buf []byte) ([]string, error) {
	if len(buf) < 4 {
		return nil, errPackedShortBuffer
	}
	n := binary.LittleEndian.Uint32(buf[:4])
	buf = buf[4:]
	// Bound the result size by the remaining buffer to avoid huge allocations
	// if a garbage count sneaks through.
	if uint64(n) > uint64(len(buf))+1 {
		return nil, errPackedShortBuffer
	}
	out := make([]string, 0, n)
	for i := uint32(0); i < n; i++ {
		if len(buf) < 4 {
			return nil, errPackedShortBuffer
		}
		l := binary.LittleEndian.Uint32(buf[:4])
		buf = buf[4:]
		if uint64(l) > uint64(len(buf)) {
			return nil, errPackedShortBuffer
		}
		out = append(out, string(buf[:l]))
		buf = buf[l:]
	}
	return out, nil
}

// withPackedStringSlice encodes v onto a pooled scratch buffer, invokes fn
// with a pointer and length, and returns the buffer to the pool. The pointer
// is only valid for the duration of fn; the caller must not retain it.
//
// For an empty slice the wire layout is still a 4-byte count=0 prefix, so fn
// always receives a non-nil pointer (provided the scratch buffer has non-zero
// length) and a length of at least 4.
func withPackedStringSlice(v []string, fn func(*byte, int)) {
	bp := packedBufPool.Get().(*[]byte)
	*bp = (*bp)[:0]
	*bp = encodePackedStringSlice(*bp, v)
	// Length of 0 is possible only if encodePackedStringSlice produced nothing,
	// which never happens (the count prefix is always 4 bytes). Defensive
	// handling of the degenerate case keeps the API predictable.
	var p *byte
	if len(*bp) > 0 {
		p = (*byte)(unsafe.Pointer(&(*bp)[0]))
	}
	fn(p, len(*bp))
	runtime.KeepAlive(*bp) // GC hazard: Swift holds a pointer into bp for the duration of fn.
	*bp = (*bp)[:0]
	packedBufPool.Put(bp)
}

// acquirePackedBuf / releasePackedBuf let a caller avoid the callback
// indirection of withPackedStringSlice when the closure would otherwise
// escape and allocate. The caller must hold the returned buffer alive (via
// runtime.KeepAlive) until the consumer of the pointer is done, and must
// hand the buffer back to releasePackedBuf.
func acquirePackedStringSliceBuf(v []string) (*[]byte, *byte, int) {
	bp := packedBufPool.Get().(*[]byte)
	*bp = (*bp)[:0]
	*bp = encodePackedStringSlice(*bp, v)
	var p *byte
	n := len(*bp)
	if n > 0 {
		p = (*byte)(unsafe.Pointer(&(*bp)[0]))
	}
	return bp, p, n
}

func releasePackedBuf(bp *[]byte) {
	runtime.KeepAlive(*bp)
	*bp = (*bp)[:0]
	packedBufPool.Put(bp)
}

// encodePackedChoiceOptions writes options into dst and returns the updated
// slice.
func encodePackedChoiceOptions(dst []byte, options []ChoiceOption) []byte {
	dst = appendUint32LE(dst, uint32(len(options)))
	for _, opt := range options {
		dst = appendUint32LE(dst, uint32(len(opt.Label)))
		dst = append(dst, opt.Label...)
		dst = appendUint32LE(dst, uint32(len(opt.Value)))
		dst = append(dst, opt.Value...)
	}
	return dst
}

// decodePackedChoiceOptions is the inverse of encodePackedChoiceOptions.
func decodePackedChoiceOptions(buf []byte) ([]ChoiceOption, error) {
	if len(buf) < 4 {
		return nil, errPackedShortBuffer
	}
	n := binary.LittleEndian.Uint32(buf[:4])
	buf = buf[4:]
	if uint64(n) > uint64(len(buf))+1 {
		return nil, errPackedShortBuffer
	}
	out := make([]ChoiceOption, 0, n)
	for i := uint32(0); i < n; i++ {
		label, rest, err := readPackedString(buf)
		if err != nil {
			return nil, err
		}
		value, rest, err := readPackedString(rest)
		if err != nil {
			return nil, err
		}
		out = append(out, ChoiceOption{Label: label, Value: value})
		buf = rest
	}
	return out, nil
}

// readPackedString peels one length-prefixed string off the head of buf.
func readPackedString(buf []byte) (string, []byte, error) {
	if len(buf) < 4 {
		return "", nil, errPackedShortBuffer
	}
	l := binary.LittleEndian.Uint32(buf[:4])
	buf = buf[4:]
	if uint64(l) > uint64(len(buf)) {
		return "", nil, errPackedShortBuffer
	}
	return string(buf[:l]), buf[l:], nil
}

// withPackedChoiceOptions mirrors withPackedStringSlice for ChoiceOption
// slices.
func withPackedChoiceOptions(options []ChoiceOption, fn func(*byte, int)) {
	bp := packedBufPool.Get().(*[]byte)
	*bp = (*bp)[:0]
	*bp = encodePackedChoiceOptions(*bp, options)
	var p *byte
	if len(*bp) > 0 {
		p = (*byte)(unsafe.Pointer(&(*bp)[0]))
	}
	fn(p, len(*bp))
	runtime.KeepAlive(*bp)
	*bp = (*bp)[:0]
	packedBufPool.Put(bp)
}

// encodePackedPayload writes a versioned single-payload envelope.
func encodePackedPayload(dst []byte, kind uint8, value string) []byte {
	dst = append(dst, wirePayloadV1, kind)
	dst = appendUint32LE(dst, uint32(len(value)))
	dst = append(dst, value...)
	return dst
}

// decodePackedPayload returns the kind and value of a versioned payload
// envelope.
func decodePackedPayload(buf []byte) (kind uint8, value string, err error) {
	if len(buf) < 6 {
		return 0, "", errPackedShortBuffer
	}
	version := buf[0]
	if version != wirePayloadV1 {
		return 0, "", errPackedBadVersion
	}
	kind = buf[1]
	switch kind {
	case wirePayloadKindText, wirePayloadKindURL, wirePayloadKindFile:
	default:
		return 0, "", errPackedBadKind
	}
	l := binary.LittleEndian.Uint32(buf[2:6])
	if uint64(l) > uint64(len(buf)-6) {
		return 0, "", errPackedShortBuffer
	}
	value = string(buf[6 : 6+l])
	return kind, value, nil
}

// decodePackedStringSliceFromPointer reads an encoded string slice from a raw
// pointer + length. Used by the Swift->Go path (e.g., for Get equivalents) so
// the Swift side can mirror the Go encoder's layout.
func decodePackedStringSliceFromPointer(p *byte, n int) ([]string, error) {
	if p == nil {
		if n == 0 {
			return nil, nil
		}
		return nil, errPackedShortBuffer
	}
	buf := unsafe.Slice(p, n)
	return decodePackedStringSlice(buf)
}
