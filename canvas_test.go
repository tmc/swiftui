package swiftui

import (
	"encoding/binary"
	"math"
	"testing"
)

func TestCanvasOpsTextEncoding(t *testing.T) {
	ops := NewCanvasOps().Text("A", 3, 5, RGBA(0.1, 0.2, 0.3, 0.4), 12, TextAnchorBottomTrailing)
	got := ops.Bytes()
	if len(got) != 1+4+8*7+1+1 {
		t.Fatalf("encoded length = %d", len(got))
	}
	if got[0] != canvasOpText {
		t.Fatalf("opcode = %#x, want %#x", got[0], canvasOpText)
	}
	if n := binary.LittleEndian.Uint32(got[1:5]); n != 1 {
		t.Fatalf("text length = %d, want 1", n)
	}
	off := 5
	for _, tt := range []struct {
		name string
		want float64
	}{
		{"x", 3},
		{"y", 5},
		{"r", 0.1},
		{"g", 0.2},
		{"b", 0.3},
		{"a", 0.4},
		{"size", 12},
	} {
		got := math.Float64frombits(binary.LittleEndian.Uint64(got[off : off+8]))
		if got != tt.want {
			t.Fatalf("%s = %v, want %v", tt.name, got, tt.want)
		}
		off += 8
	}
	if got[off] != byte(TextAnchorBottomTrailing) {
		t.Fatalf("anchor = %d, want %d", got[off], TextAnchorBottomTrailing)
	}
	if string(got[off+1:]) != "A" {
		t.Fatalf("text = %q, want A", got[off+1:])
	}
}
