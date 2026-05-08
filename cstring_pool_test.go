package swiftui

import (
	"bytes"
	"testing"
	"unsafe"
)

func TestWithCStringPooledBasic(t *testing.T) {
	got := ""
	withCStringPooled("hello", func(p *byte) {
		got = gostringFast(p)
	})
	if got != "hello" {
		t.Fatalf("got %q, want %q", got, "hello")
	}
}

func TestWithCStringPooledNULTerminates(t *testing.T) {
	withCStringPooled("abc", func(p *byte) {
		base := uintptr(unsafe.Pointer(p))
		if *(*byte)(unsafe.Pointer(base + 3)) != 0 {
			t.Fatal("missing NUL terminator")
		}
	})
}

func TestWithCStringPooledReuseIsolation(t *testing.T) {
	var firstBuf []byte
	var firstPtr *byte
	withCStringPooled("first-payload", func(p *byte) {
		firstPtr = p
		n := 0
		for *(*byte)(unsafe.Pointer(uintptr(unsafe.Pointer(p)) + uintptr(n))) != 0 {
			n++
		}
		firstBuf = append([]byte(nil), unsafe.Slice(p, n)...)
	})

	withCStringPooled("second", func(p *byte) {
		if p == firstPtr {
			n := 0
			for *(*byte)(unsafe.Pointer(uintptr(unsafe.Pointer(p)) + uintptr(n))) != 0 {
				n++
			}
			inside := unsafe.Slice(p, n)
			if bytes.Contains(inside, []byte("first-payload")) {
				t.Fatalf("second call saw first call's bytes via reused buffer: %q", inside)
			}
		}
	})

	if string(firstBuf) != "first-payload" {
		t.Fatalf("first snapshot got %q, want %q", firstBuf, "first-payload")
	}
}

func TestGostringFastEmpty(t *testing.T) {
	if got := gostringFast(nil); got != "" {
		t.Fatalf("gostringFast(nil) = %q, want empty", got)
	}
	zero := byte(0)
	if got := gostringFast(&zero); got != "" {
		t.Fatalf("gostringFast(empty) = %q, want empty", got)
	}
}

func TestGostringFastRoundTrip(t *testing.T) {
	for _, in := range []string{"", "a", "hello", "a longer string with spaces"} {
		var got string
		withCStringPooled(in, func(p *byte) {
			got = gostringFast(p)
		})
		if got != in {
			t.Fatalf("round trip %q -> %q", in, got)
		}
	}
}
