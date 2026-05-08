package swiftui

import (
	"bytes"
	"encoding/binary"
	"sync"
	"testing"
)

// TestPackedModifierRoundTrip verifies that encoding a chain of modifiers and
// decoding it back yields the same kinds and payload bytes for chains of
// length 1, 5, and 10 (the sizes BenchmarkModifierChain exercises).
func TestPackedModifierRoundTrip(t *testing.T) {
	cases := []struct {
		name string
		mods []Modifier
	}{
		{
			name: "single",
			mods: []Modifier{ModPadding(8)},
		},
		{
			name: "five",
			mods: []Modifier{
				ModPadding(8),
				ModOpacity(0.9),
				ModForegroundRGBA(0.2, 0.4, 0.6, 1.0),
				ModCornerRadius(4),
				ModScaleEffect(1.05),
			},
		},
		{
			name: "ten",
			mods: []Modifier{
				ModPadding(8),
				ModOpacity(0.9),
				ModForegroundRGBA(0.2, 0.4, 0.6, 1.0),
				ModBackgroundRGBA(0.95, 0.95, 0.95, 1.0),
				ModCornerRadius(4),
				ModScaleEffect(1.05),
				ModRotationEffect(1.5),
				ModBlur(0.5),
				ModZIndex(2.0),
				ModLayoutPriority(1.0),
			},
		},
		{
			name: "mixed-payload-shapes",
			mods: []Modifier{
				ModPaddingEdge(EdgeTop|EdgeLeading, 12),
				ModDisabled(true),
				ModBold(),
				ModItalic(),
				ModFrame(100, 200),
				ModMaxFrame(-1, 0),
				ModFontWeight(WeightBold),
				ModTextAlignment(TextAlignmentCenter),
				ModFixedSize(),
			},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			buf := encodePackedModifiers(nil, tc.mods)
			got, err := decodePackedModifiers(buf)
			if err != nil {
				t.Fatalf("decodePackedModifiers: %v", err)
			}
			if len(got) != len(tc.mods) {
				t.Fatalf("decoded count = %d, want %d", len(got), len(tc.mods))
			}
			for i := range got {
				if got[i].Kind != tc.mods[i].kind {
					t.Errorf("mod[%d].kind = %d, want %d", i, got[i].Kind, tc.mods[i].kind)
				}
				// Re-encode just this modifier and compare payload bytes.
				// encodeModifier emits [kind u32][plen u32][payload]; strip
				// the 8-byte header before comparing.
				reenc := encodeModifier(nil, tc.mods[i])
				if len(reenc) < 8 {
					t.Fatalf("re-encoded header too short for mod[%d]: %d bytes", i, len(reenc))
				}
				wantPayload := reenc[8:]
				if !bytesEqual(got[i].Payload, wantPayload) {
					t.Errorf("mod[%d].payload = %x, want %x", i, got[i].Payload, wantPayload)
				}
			}
		})
	}
}

// TestPackedModifierUnknownKind verifies the decoder does not panic on an
// unknown kind value. The Go-side decoder is permissive (it accepts the kind
// and hands the payload bytes through); the Swift-side decoder is the one
// that rejects unknown kinds by returning the input ref unchanged. The
// behavioural guarantee we assert here is "no panic, no out-of-bounds read".
func TestPackedModifierUnknownKind(t *testing.T) {
	buf := encodePackedModifiers(nil, []Modifier{ModPadding(8)})
	// Buffer layout: [count u32=1][kind u32][plen u32][payload 8 bytes].
	// Overwrite the kind field with a value well outside the allocated range.
	if len(buf) < 8 {
		t.Fatalf("encoded buffer too short: %d", len(buf))
	}
	binary.LittleEndian.PutUint32(buf[4:8], 0xFFFFFFFF)
	got, err := decodePackedModifiers(buf)
	if err != nil {
		t.Fatalf("decodePackedModifiers rejected unknown kind: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("decoded count = %d, want 1", len(got))
	}
	if got[0].Kind != 0xFFFFFFFF {
		t.Errorf("decoded kind = %#x, want 0xFFFFFFFF", got[0].Kind)
	}
}

// TestPackedModifierTruncatedRejected verifies the decoder rejects truncated
// buffers cleanly (non-nil error, no panic).
func TestPackedModifierTruncatedRejected(t *testing.T) {
	cases := [][]byte{
		nil,
		{},
		{0x01},                 // too short for count
		{0x01, 0x00, 0x00, 0x00}, // count=1 but no entries follow
		{
			0x01, 0x00, 0x00, 0x00, // count=1
			0x01, 0x00, 0x00, 0x00, // kind=padding
			0x08, 0x00, 0x00, 0x00, // plen=8
			// payload missing
		},
	}
	for i, c := range cases {
		i, c := i, c
		t.Run("", func(t *testing.T) {
			if _, err := decodePackedModifiers(c); err == nil {
				t.Fatalf("case %d: expected error, got nil (buf=%x)", i, c)
			}
		})
	}
}

// TestApplyModifiersEmptyChain verifies the empty-chain fast path returns the
// input view unchanged without invoking the bridge symbol.
func TestApplyModifiersEmptyChain(t *testing.T) {
	v := View{ptr: 0xDEADBEEF, retained: nil}

	// Poison the bridge call target so any attempted invocation would be
	// observable (via SyscallN bad-address fault). Snapshot and restore.
	prev := _applyModifiersFn
	_applyModifiersFn = 0xBAD0CAFE
	defer func() { _applyModifiersFn = prev }()

	got := ApplyModifiers(v)
	if got.ptr != v.ptr {
		t.Fatalf("ApplyModifiers(v) with empty chain returned ptr=%#x, want %#x", got.ptr, v.ptr)
	}
}

// TestApplyModifiersNoBridgeFallback verifies that if SUIApplyModifiers is not
// resolved (e.g. dylib missing the symbol), ApplyModifiers falls back to
// applying modifiers one at a time via the legacy per-modifier @_cdecl path.
// Stubs observe that the expected per-modifier symbols are invoked.
func TestApplyModifiersNoBridgeFallback(t *testing.T) {
	// Force the Once to appear already run AND the fn pointer to be zero.
	// This exercises the fallback loop at the N>=2 site.
	prevFn := _applyModifiersFn
	prevOnce := packedModifierOnce
	packedModifierOnce = sync.Once{}
	packedModifierOnce.Do(func() {}) // consume the Once without resolving
	_applyModifiersFn = 0
	t.Cleanup(func() {
		_applyModifiersFn = prevFn
		packedModifierOnce = prevOnce
	})

	oldPadding := _SUIViewPadding
	oldOpacity := _SUIViewOpacity
	t.Cleanup(func() {
		_SUIViewPadding = oldPadding
		_SUIViewOpacity = oldOpacity
	})

	var paddingCalls, opacityCalls int
	_SUIViewPadding = func(ptr uintptr, amount float64) uintptr {
		paddingCalls++
		return ptr + 1
	}
	_SUIViewOpacity = func(ptr uintptr, opacity float64) uintptr {
		opacityCalls++
		return ptr + 1
	}

	base := View{ptr: 0x1000, retained: nil}
	out := ApplyModifiers(base, ModPadding(8), ModOpacity(0.5))
	if paddingCalls != 1 {
		t.Errorf("padding calls = %d, want 1", paddingCalls)
	}
	if opacityCalls != 1 {
		t.Errorf("opacity calls = %d, want 1", opacityCalls)
	}
	if out.ptr == 0 {
		t.Errorf("fallback returned zero ptr")
	}
}

// bytesEqual treats nil and empty slices as equal. bytes.Equal already does
// this; the helper name makes the intent explicit at the call sites.
func bytesEqual(a, b []byte) bool { return bytes.Equal(a, b) }

// FuzzPackedModifierList exercises the decoder with arbitrary byte slices.
// The decoder must never panic, no matter how malformed the input is.
func FuzzPackedModifierList(f *testing.F) {
	seeds := [][]byte{
		nil,
		{},
		{0x00, 0x00, 0x00, 0x00}, // count=0
		encodePackedModifiers(nil, []Modifier{ModPadding(8)}),
		encodePackedModifiers(nil, []Modifier{
			ModPadding(8),
			ModOpacity(0.9),
			ModForegroundRGBA(0.2, 0.4, 0.6, 1.0),
			ModCornerRadius(4),
			ModScaleEffect(1.05),
		}),
		encodePackedModifiers(nil, []Modifier{
			ModBold(),
			ModItalic(),
			ModUnderline(),
			ModStrikethrough(),
			ModClipped(),
			ModMonospacedDigit(),
			ModLabelsHidden(),
			ModFixedSize(),
		}),
		// Claims a huge count — decoder must bound-check.
		{0xFF, 0xFF, 0xFF, 0xFF},
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		mods, err := decodePackedModifiers(data)
		if err != nil {
			return
		}
		// If the decoder accepted, sanity-check the output.
		for _, m := range mods {
			if len(m.Payload) > len(data) {
				t.Fatalf("decoded payload len %d exceeds input len %d", len(m.Payload), len(data))
			}
		}
		if len(data) >= 4 {
			got := binary.LittleEndian.Uint32(data[:4])
			if uint64(len(mods)) != uint64(got) {
				t.Fatalf("decoded count %d != header count %d", len(mods), got)
			}
		}
	})
}
