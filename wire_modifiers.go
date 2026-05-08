package swiftui

import (
	"encoding/binary"
	"math"
	"runtime"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

// Packed modifier chain wire format for the Go->Swift hot path.
//
// Pre-P6b, every chained view modifier (v.Padding(x).Opacity(y)...) crossed
// the FFI boundary as an independent @_cdecl call and allocated one *retained
// sentinel per intermediate View. P6b introduces a single
// SUIApplyModifiers(viewRef, bufPtr, bufLen) entry point that decodes a
// length-prefixed list of modifier ops and applies them in order, producing
// one retained view ref and paying one bridge crossing regardless of chain
// length.
//
// Layout (little-endian):
//
//	[count: uint32]
//	[ [kind: uint32] [payload_len: uint32] [payload: payload_len bytes] ]*count
//
// Payload encoding is per-kind and matches bridge_modifiers.gen.swift
// exactly. The Swift decoder rejects unknown kinds and truncated payloads by
// returning the input ref unchanged; this matches P4 discipline (fail loudly
// rather than silently apply a subset).
//
// Per-modifier @_cdecl entries are retained for backward compatibility and
// for single-modifier fast paths. Packing engages only for N>=2.

// Modifier kinds. Must stay in sync with suiModifierWireKind* in
// bridge_modifiers.gen.swift.
const (
	modKindPadding          uint32 = 1
	modKindPaddingEdge      uint32 = 2
	modKindOpacity          uint32 = 3
	modKindForegroundRGBA   uint32 = 4
	modKindBackgroundRGBA   uint32 = 5
	modKindTintRGBA         uint32 = 6
	modKindDisabled         uint32 = 7
	modKindBold             uint32 = 8
	modKindItalic           uint32 = 9
	modKindUnderline        uint32 = 10
	modKindStrikethrough    uint32 = 11
	modKindClipped          uint32 = 12
	modKindMonospacedDigit  uint32 = 13
	modKindLabelsHidden     uint32 = 14
	modKindBlur             uint32 = 15
	modKindCornerRadius     uint32 = 16
	modKindScaleEffect      uint32 = 17
	modKindRotationEffect   uint32 = 18
	modKindZIndex           uint32 = 19
	modKindLayoutPriority   uint32 = 20
	modKindFrame            uint32 = 21
	modKindMaxFrame         uint32 = 22
	modKindFontWeight       uint32 = 23
	modKindFontDesign       uint32 = 24
	modKindImageScale       uint32 = 25
	modKindControlSize      uint32 = 26
	modKindButtonStyle      uint32 = 27
	modKindToggleStyle      uint32 = 28
	modKindTextAlignment    uint32 = 29
	modKindTruncationMode   uint32 = 30
	modKindFixedSize        uint32 = 31
)

// Modifier is a single entry in a packed modifier chain. Construct via the
// Mod* helpers below (ModPadding, ModOpacity, ...). The zero value is not
// valid.
//
// Bridge surface.
type Modifier struct {
	kind uint32
	f0   float64 // primary scalar / r
	f1   float64 // height / g / edge-amount
	f2   float64 // b
	f3   float64 // a
	i0   int32   // enum / edges
	b0   uint8   // bool
}

// --- Modifier constructors ---

// ModPadding applies uniform padding.
func ModPadding(amount float64) Modifier {
	return Modifier{kind: modKindPadding, f0: amount}
}

// ModPaddingEdge applies padding to the given edge set.
func ModPaddingEdge(edges Edge, amount float64) Modifier {
	return Modifier{kind: modKindPaddingEdge, i0: int32(edges), f0: amount}
}

// ModOpacity sets view opacity (0.0-1.0).
func ModOpacity(opacity float64) Modifier {
	return Modifier{kind: modKindOpacity, f0: opacity}
}

// ModForegroundRGBA sets the foreground style to an RGBA color.
func ModForegroundRGBA(r, g, b, a float64) Modifier {
	return Modifier{kind: modKindForegroundRGBA, f0: r, f1: g, f2: b, f3: a}
}

// ModBackgroundRGBA sets the background to an RGBA color.
func ModBackgroundRGBA(r, g, b, a float64) Modifier {
	return Modifier{kind: modKindBackgroundRGBA, f0: r, f1: g, f2: b, f3: a}
}

// ModTintRGBA sets the tint color.
func ModTintRGBA(r, g, b, a float64) Modifier {
	return Modifier{kind: modKindTintRGBA, f0: r, f1: g, f2: b, f3: a}
}

// ModDisabled disables user interaction when v is true.
func ModDisabled(v bool) Modifier {
	var b uint8
	if v {
		b = 1
	}
	return Modifier{kind: modKindDisabled, b0: b}
}

// ModBold applies bold text styling.
func ModBold() Modifier { return Modifier{kind: modKindBold} }

// ModItalic applies italic text styling.
func ModItalic() Modifier { return Modifier{kind: modKindItalic} }

// ModUnderline applies an underline.
func ModUnderline() Modifier { return Modifier{kind: modKindUnderline} }

// ModStrikethrough applies a strikethrough.
func ModStrikethrough() Modifier { return Modifier{kind: modKindStrikethrough} }

// ModClipped clips subviews to the receiver's bounds.
func ModClipped() Modifier { return Modifier{kind: modKindClipped} }

// ModMonospacedDigit forces monospaced numeric digits.
func ModMonospacedDigit() Modifier { return Modifier{kind: modKindMonospacedDigit} }

// ModLabelsHidden hides labels in container controls.
func ModLabelsHidden() Modifier { return Modifier{kind: modKindLabelsHidden} }

// ModBlur applies a gaussian blur with the given radius.
func ModBlur(radius float64) Modifier {
	return Modifier{kind: modKindBlur, f0: radius}
}

// ModCornerRadius rounds the view corners.
func ModCornerRadius(r float64) Modifier {
	return Modifier{kind: modKindCornerRadius, f0: r}
}

// ModScaleEffect scales the view.
func ModScaleEffect(s float64) Modifier {
	return Modifier{kind: modKindScaleEffect, f0: s}
}

// ModRotationEffect rotates the view by the given angle in degrees.
func ModRotationEffect(degrees float64) Modifier {
	return Modifier{kind: modKindRotationEffect, f0: degrees}
}

// ModZIndex sets the view z-index.
func ModZIndex(z float64) Modifier {
	return Modifier{kind: modKindZIndex, f0: z}
}

// ModLayoutPriority sets the view layout priority.
func ModLayoutPriority(p float64) Modifier {
	return Modifier{kind: modKindLayoutPriority, f0: p}
}

// ModFrame fixes the view's width and height.
func ModFrame(width, height float64) Modifier {
	return Modifier{kind: modKindFrame, f0: width, f1: height}
}

// ModMaxFrame constrains the view's maximum width and height. Use -1 for
// .infinity and 0 for nil, matching the per-modifier ABI.
func ModMaxFrame(maxWidth, maxHeight float64) Modifier {
	return Modifier{kind: modKindMaxFrame, f0: maxWidth, f1: maxHeight}
}

// ModFontWeight sets the font weight.
func ModFontWeight(w Weight) Modifier {
	return Modifier{kind: modKindFontWeight, i0: int32(w)}
}

// ModFontDesign sets the font design.
func ModFontDesign(d Design) Modifier {
	return Modifier{kind: modKindFontDesign, i0: int32(d)}
}

// ModImageScale sets the image scale.
func ModImageScale(s ImageScale) Modifier {
	return Modifier{kind: modKindImageScale, i0: int32(s)}
}

// ModControlSize sets the control size.
func ModControlSize(s ControlSize) Modifier {
	return Modifier{kind: modKindControlSize, i0: int32(s)}
}

// ModButtonStyle sets the button style.
func ModButtonStyle(s ButtonStyleKind) Modifier {
	return Modifier{kind: modKindButtonStyle, i0: int32(s)}
}

// ModToggleStyle sets the toggle style.
func ModToggleStyle(s ToggleStyleKind) Modifier {
	return Modifier{kind: modKindToggleStyle, i0: int32(s)}
}

// ModTextAlignment sets the multiline text alignment.
func ModTextAlignment(a TextAlignment) Modifier {
	return Modifier{kind: modKindTextAlignment, i0: int32(a)}
}

// ModTruncationMode sets the text truncation mode.
func ModTruncationMode(m TruncationMode) Modifier {
	return Modifier{kind: modKindTruncationMode, i0: int32(m)}
}

// ModFixedSize makes the view take its ideal size.
func ModFixedSize() Modifier { return Modifier{kind: modKindFixedSize} }

// --- Encoding ---

// packedModifierBufPool recycles []byte scratch buffers used to encode packed
// modifier chains. Mirrors packedBufPool (wire_packed.go) but is kept
// separate so the two hot paths don't contend on the same pool.
var packedModifierBufPool = sync.Pool{
	New: func() any {
		buf := make([]byte, 0, 256)
		return &buf
	},
}

// appendUint32ModLE appends v in little-endian order.
//
// Named distinctly from appendUint32LE (wire_packed.go) to avoid cross-file
// duplication while letting future refactors share a common helper without a
// merge conflict.
func appendUint32ModLE(buf []byte, v uint32) []byte {
	var tmp [4]byte
	binary.LittleEndian.PutUint32(tmp[:], v)
	return append(buf, tmp[:]...)
}

// appendFloat64LE appends v as little-endian 8 bytes.
func appendFloat64LE(buf []byte, v float64) []byte {
	var tmp [8]byte
	binary.LittleEndian.PutUint64(tmp[:], math.Float64bits(v))
	return append(buf, tmp[:]...)
}

// encodeModifier writes a single modifier entry (kind + payload_len + payload)
// onto dst and returns the updated slice.
func encodeModifier(dst []byte, m Modifier) []byte {
	dst = appendUint32ModLE(dst, m.kind)
	// Reserve 4 bytes for payload length; we will overwrite once we know it.
	lenOffset := len(dst)
	dst = appendUint32ModLE(dst, 0)
	start := len(dst)
	switch m.kind {
	case modKindPadding, modKindOpacity, modKindBlur, modKindCornerRadius,
		modKindScaleEffect, modKindRotationEffect, modKindZIndex, modKindLayoutPriority:
		dst = appendFloat64LE(dst, m.f0)
	case modKindPaddingEdge:
		dst = appendUint32ModLE(dst, uint32(m.i0))
		dst = appendFloat64LE(dst, m.f0)
	case modKindForegroundRGBA, modKindBackgroundRGBA, modKindTintRGBA:
		dst = appendFloat64LE(dst, m.f0)
		dst = appendFloat64LE(dst, m.f1)
		dst = appendFloat64LE(dst, m.f2)
		dst = appendFloat64LE(dst, m.f3)
	case modKindDisabled:
		dst = append(dst, m.b0)
	case modKindBold, modKindItalic, modKindUnderline, modKindStrikethrough,
		modKindClipped, modKindMonospacedDigit, modKindLabelsHidden, modKindFixedSize:
		// empty payload
	case modKindFrame, modKindMaxFrame:
		dst = appendFloat64LE(dst, m.f0)
		dst = appendFloat64LE(dst, m.f1)
	case modKindFontWeight, modKindFontDesign, modKindImageScale,
		modKindControlSize, modKindButtonStyle, modKindToggleStyle,
		modKindTextAlignment, modKindTruncationMode:
		dst = appendUint32ModLE(dst, uint32(m.i0))
	default:
		// Unknown kind: emit the zero-length payload and let the Swift side
		// reject. Encoder does not validate kind so future additions on the
		// Swift side without a coordinated Go bump still flush cleanly.
	}
	payloadLen := uint32(len(dst) - start)
	binary.LittleEndian.PutUint32(dst[lenOffset:lenOffset+4], payloadLen)
	return dst
}

// encodePackedModifiers writes the full chain (count + entries) onto dst.
func encodePackedModifiers(dst []byte, mods []Modifier) []byte {
	dst = appendUint32ModLE(dst, uint32(len(mods)))
	for i := range mods {
		dst = encodeModifier(dst, mods[i])
	}
	return dst
}

// decodePackedModifiers peels the count and each (kind, payload_len, payload)
// tuple off buf. Used by the Go-side fuzz/test harness. Swift has its own
// decoder (SUIApplyModifiers in bridge_modifiers.gen.swift).
func decodePackedModifiers(buf []byte) ([]decodedModifier, error) {
	if len(buf) < 4 {
		return nil, errPackedShortBuffer
	}
	n := binary.LittleEndian.Uint32(buf[:4])
	buf = buf[4:]
	// Bound by remaining buffer to cap huge counts from fuzz input.
	if uint64(n)*8 > uint64(len(buf))+8 {
		return nil, errPackedShortBuffer
	}
	out := make([]decodedModifier, 0, n)
	for i := uint32(0); i < n; i++ {
		if len(buf) < 8 {
			return nil, errPackedShortBuffer
		}
		kind := binary.LittleEndian.Uint32(buf[:4])
		plen := binary.LittleEndian.Uint32(buf[4:8])
		buf = buf[8:]
		if uint64(plen) > uint64(len(buf)) {
			return nil, errPackedShortBuffer
		}
		payload := append([]byte(nil), buf[:plen]...)
		buf = buf[plen:]
		out = append(out, decodedModifier{Kind: kind, Payload: payload})
	}
	return out, nil
}

// decodedModifier is the test-only decoded view of one packed modifier entry.
type decodedModifier struct {
	Kind    uint32
	Payload []byte
}

// --- Registration / stub ---

var (
	packedModifierOnce sync.Once
	// _applyModifiersFn is the raw entrypoint to SUIApplyModifiers, obtained
	// via Dlsym. We call it through purego.SyscallN instead of
	// purego.RegisterLibFunc to bypass the reflect-based marshalling path
	// (RegisterLibFunc allocates ~4 extra objects per call via
	// reflect.MakeFunc). SyscallN is a thin syscall wrapper; all args fit in
	// uintptr registers here (pointer, pointer, int32), so no floating-point
	// marshalling is needed.
	_applyModifiersFn uintptr
)

// ensurePackedModifierFunc lazily resolves SUIApplyModifiers in the loaded
// dylib. Kept out of the generated lib.go init path so that adding or
// removing the symbol does not force a regen of lib.go. Matches the
// bridge_extra.go pattern for additive helpers.
func ensurePackedModifierFunc() {
	packedModifierOnce.Do(func() {
		if libHandle != 0 {
			if fn, err := purego.Dlsym(libHandle, "SUIApplyModifiers"); err == nil {
				_applyModifiersFn = fn
			}
		}
	})
}

// --- Public API ---

// ApplyModifiers applies a chain of modifiers to v in order and returns the
// resulting View.
//
// For len(mods) == 0 the call is a no-op and returns v unchanged (no bridge
// crossing, no allocation).
//
// For len(mods) == 1 the call falls through to the legacy per-modifier
// @_cdecl entry. This keeps the single-modifier fast path free of the
// encode/decode overhead.
//
// For len(mods) >= 2 the chain is encoded onto a pooled scratch buffer and
// flushed as a single SUIApplyModifiers bridge call, producing one retained
// view ref regardless of chain length.
func ApplyModifiers(v View, mods ...Modifier) View {
	if len(mods) == 0 {
		return v
	}
	if len(mods) == 1 {
		return applyOneModifier(v, mods[0])
	}
	ensurePackedModifierFunc()
	if _applyModifiersFn == 0 {
		// Fallback: apply modifiers one at a time (legacy path).
		current := v
		for i := range mods {
			current = applyOneModifier(current, mods[i])
		}
		return current
	}
	bp := packedModifierBufPool.Get().(*[]byte)
	*bp = (*bp)[:0]
	*bp = encodePackedModifiers(*bp, mods)
	var bufArg uintptr
	if len(*bp) > 0 {
		bufArg = uintptr(unsafe.Pointer(&(*bp)[0]))
	}
	ret, _, _ := purego.SyscallN(_applyModifiersFn, v.ptr, bufArg, uintptr(uint32(len(*bp))))
	runtime.KeepAlive(*bp) // Swift holds the buffer pointer for the duration of the call.
	runtime.KeepAlive(v.retained)
	*bp = (*bp)[:0]
	packedModifierBufPool.Put(bp)
	if ret == v.ptr {
		// Swift returned the input ref unchanged (decode error or no-op).
		// Do not construct a new retained sentinel; caller keeps v.
		return v
	}
	return View{ptr: ret, retained: newRetainedOwned(ret)}
}

// applyOneModifier is the fallback path that routes a single modifier entry
// through the legacy per-modifier @_cdecl call. Used by ApplyModifiers when
// len(mods)==1 and as the fallback when SUIApplyModifiers is unavailable.
//
// Text-only modifiers (bold/italic/underline/strikethrough/monospacedDigit/
// multilineTextAlignment/truncationMode) call the underlying @_cdecl symbols
// directly since the public Go API exposes them only on TextView. The Swift
// side accepts any AnyView, so calling with a plain View here is valid.
func applyOneModifier(v View, m Modifier) View {
	newView := func(ptr uintptr) View {
		runtime.KeepAlive(v.retained)
		return View{ptr: ptr, retained: newRetainedOwned(ptr)}
	}
	switch m.kind {
	case modKindPadding:
		return v.Padding(m.f0)
	case modKindPaddingEdge:
		return v.PaddingEdge(Edge(m.i0), m.f0)
	case modKindOpacity:
		return v.Opacity(m.f0)
	case modKindForegroundRGBA:
		return v.ForegroundStyle(m.f0, m.f1, m.f2, m.f3)
	case modKindBackgroundRGBA:
		return v.Background(m.f0, m.f1, m.f2, m.f3)
	case modKindTintRGBA:
		return v.Tint(m.f0, m.f1, m.f2, m.f3)
	case modKindDisabled:
		return v.Disabled(m.b0 != 0)
	case modKindBold:
		return newView(_SUIViewBold(v.ptr))
	case modKindItalic:
		return newView(_SUIViewItalic(v.ptr))
	case modKindUnderline:
		return newView(_SUIViewUnderline(v.ptr))
	case modKindStrikethrough:
		return newView(_SUIViewStrikethrough(v.ptr))
	case modKindClipped:
		return v.Clipped()
	case modKindMonospacedDigit:
		return newView(_SUIViewMonospacedDigit(v.ptr))
	case modKindLabelsHidden:
		return v.LabelsHidden()
	case modKindBlur:
		return v.Blur(m.f0)
	case modKindCornerRadius:
		return v.CornerRadius(m.f0)
	case modKindScaleEffect:
		return v.ScaleEffect(m.f0)
	case modKindRotationEffect:
		return v.RotationEffect(m.f0)
	case modKindZIndex:
		return v.ZIndex(m.f0)
	case modKindLayoutPriority:
		return v.LayoutPriority(m.f0)
	case modKindFrame:
		return v.Frame(m.f0, m.f1)
	case modKindMaxFrame:
		return v.MaxFrame(m.f0, m.f1)
	case modKindFontWeight:
		return v.FontWeight(Weight(m.i0))
	case modKindFontDesign:
		return v.FontDesign(Design(m.i0))
	case modKindImageScale:
		return v.ImageScale(ImageScale(m.i0))
	case modKindControlSize:
		return v.ControlSize(ControlSize(m.i0))
	case modKindButtonStyle:
		return v.ButtonStyle(ButtonStyleKind(m.i0))
	case modKindToggleStyle:
		return v.ToggleStyle(ToggleStyleKind(m.i0))
	case modKindTextAlignment:
		return newView(_SUIViewMultilineTextAlignment(v.ptr, m.i0))
	case modKindTruncationMode:
		return newView(_SUIViewTruncationMode(v.ptr, m.i0))
	case modKindFixedSize:
		return v.FixedSize()
	default:
		// Unknown kind: return v unchanged. Matches the Swift-side rejection.
		return v
	}
}
