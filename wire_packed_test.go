package swiftui

import (
	"reflect"
	"strings"
	"sync"
	"testing"
	"unsafe"
)

func TestPackedStringSliceRoundTrip(t *testing.T) {
	cases := [][]string{
		nil,
		{},
		{""},
		{"alpha"},
		{"alpha", "beta", "gamma"},
		{"", "nonempty", ""},
		{"emoji-🙂", "CJK-漢字", "mixed-\x00-NUL-is-fine-inside"},
		{strings.Repeat("x", 4096), strings.Repeat("y", 0)},
	}
	for _, tc := range cases {
		tc := tc
		t.Run("", func(t *testing.T) {
			buf := encodePackedStringSlice(nil, tc)
			got, err := decodePackedStringSlice(buf)
			if err != nil {
				t.Fatalf("decodePackedStringSlice: %v", err)
			}
			want := tc
			if want == nil {
				want = []string{}
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("round trip = %#v, want %#v", got, want)
			}
		})
	}
}

func TestWithPackedStringSlicePassesPointerAndLength(t *testing.T) {
	values := []string{"a", "bb", "ccc"}
	var gotBytes []byte
	withPackedStringSlice(values, func(p *byte, n int) {
		if p == nil {
			t.Fatal("withPackedStringSlice: got nil pointer")
		}
		gotBytes = append(gotBytes, append([]byte(nil), unsafeSlice(p, n)...)...)
	})
	decoded, err := decodePackedStringSlice(gotBytes)
	if err != nil {
		t.Fatalf("decodePackedStringSlice: %v", err)
	}
	if !reflect.DeepEqual(decoded, values) {
		t.Fatalf("decoded = %#v, want %#v", decoded, values)
	}
}

func TestPackedChoiceOptionsRoundTrip(t *testing.T) {
	options := []ChoiceOption{
		{Label: "Apple", Value: "🍎"},
		{Label: "", Value: "empty-label"},
		{Label: "no-value", Value: ""},
	}
	buf := encodePackedChoiceOptions(nil, options)
	got, err := decodePackedChoiceOptions(buf)
	if err != nil {
		t.Fatalf("decodePackedChoiceOptions: %v", err)
	}
	if !reflect.DeepEqual(got, options) {
		t.Fatalf("round trip = %#v, want %#v", got, options)
	}
}

func TestPackedPayloadVersion(t *testing.T) {
	buf := encodePackedPayload(nil, wirePayloadKindURL, "https://example.test/")
	kind, value, err := decodePackedPayload(buf)
	if err != nil {
		t.Fatalf("decodePackedPayload: %v", err)
	}
	if kind != wirePayloadKindURL || value != "https://example.test/" {
		t.Fatalf("decode = (%d, %q), want (%d, %q)", kind, value, wirePayloadKindURL, "https://example.test/")
	}

	// Flip the version byte. Decoder must reject cleanly, not panic.
	buf[0] = 0xFF
	if _, _, err := decodePackedPayload(buf); err == nil {
		t.Fatal("decodePackedPayload with bad version: got nil error")
	}
}

func TestPackedPayloadBadKind(t *testing.T) {
	buf := encodePackedPayload(nil, wirePayloadKindText, "hi")
	buf[1] = 99
	if _, _, err := decodePackedPayload(buf); err == nil {
		t.Fatal("decodePackedPayload with bad kind: got nil error")
	}
}

func TestPackedStringSliceDecoderRejectsTruncated(t *testing.T) {
	buf := encodePackedStringSlice(nil, []string{"hello", "world"})
	for i := 0; i < len(buf); i++ {
		if _, err := decodePackedStringSlice(buf[:i]); err == nil {
			t.Errorf("decodePackedStringSlice(buf[:%d]) = nil error, want short-buffer", i)
		}
	}
}

func TestPackedStringSlicePoolReuseRace(t *testing.T) {
	// -race is explicitly out of scope per the P4 charter, but the pool must
	// still be safe when multiple goroutines share it. Exercise that here so
	// a future -race run would catch accidental sharing across scratch
	// buffers.
	values := []string{"one", "two", "three", "four", "five"}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 256; j++ {
				withPackedStringSlice(values, func(p *byte, n int) {
					got, err := decodePackedStringSliceFromPointer(p, n)
					if err != nil {
						t.Errorf("decode: %v", err)
						return
					}
					if !reflect.DeepEqual(got, values) {
						t.Errorf("pool contamination: got %#v", got)
					}
				})
			}
		}()
	}
	wg.Wait()
}

func TestMarshalStringSlicePackedHotPath(t *testing.T) {
	// Live integration: NewStringListState/Set/Get round-trip goes through the
	// packed path end-to-end. Exercising it here protects against Swift side
	// regressions.
	state := NewStringListState([]string{"alpha", "beta"})
	defer state.Release()

	if got := state.Get(); !reflect.DeepEqual(got, []string{"alpha", "beta"}) {
		t.Fatalf("Get() = %v, want [alpha beta]", got)
	}
	state.Set([]string{"emoji-🙂", "漢字"})
	if got := state.Get(); !reflect.DeepEqual(got, []string{"emoji-🙂", "漢字"}) {
		t.Fatalf("Get() after multi-byte Set = %v", got)
	}
	state.Set(nil)
	if got := state.Get(); got != nil && len(got) != 0 {
		t.Fatalf("Get() after Set(nil) = %v, want empty", got)
	}
}

func FuzzPackedStringSlice(f *testing.F) {
	seeds := [][]byte{
		encodePackedStringSlice(nil, nil),
		encodePackedStringSlice(nil, []string{""}),
		encodePackedStringSlice(nil, []string{"hello"}),
		encodePackedStringSlice(nil, []string{"a", "b", "c"}),
		{0x00, 0x00, 0x00, 0x00},
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		// Must never panic on arbitrary input.
		if got, err := decodePackedStringSlice(data); err == nil {
			// If the decoder accepts, re-encoding should produce the same
			// bytes or at least not crash.
			_ = encodePackedStringSlice(nil, got)
		}
	})
}

// unsafeSlice returns a []byte that aliases the region [p, p+n). The caller
// must ensure the underlying memory lives for the duration of use.
func unsafeSlice(p *byte, n int) []byte {
	if p == nil || n == 0 {
		return nil
	}
	return unsafe.Slice(p, n)
}
