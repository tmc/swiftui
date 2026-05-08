package swiftui

import (
	"runtime"
	"sync"
	"unsafe"
)

var cstringBufPool = sync.Pool{
	New: func() any {
		buf := make([]byte, 0, 64)
		return &buf
	},
}

// withCStringPooled hands fn a NUL-terminated byte pointer sourced from a
// sync.Pool scratch buffer. The buffer is reused after fn returns.
func withCStringPooled(s string, fn func(*byte)) {
	bp := cstringBufPool.Get().(*[]byte)
	need := len(s) + 1
	if cap(*bp) < need {
		*bp = make([]byte, need)
	} else {
		*bp = (*bp)[:need]
	}
	copy(*bp, s)
	(*bp)[len(s)] = 0
	fn((*byte)(unsafe.Pointer(&(*bp)[0])))
	runtime.KeepAlive(*bp) // GC hazard: Swift holds a pointer into bp for the duration of fn.
	*bp = (*bp)[:0]
	cstringBufPool.Put(bp)
}

// gostringFast copies a NUL-terminated C string into a Go string with one
// length pass and one allocation for the result. p must remain valid for the
// duration of the call.
func gostringFast(p *byte) string {
	if p == nil {
		return ""
	}
	base := uintptr(unsafe.Pointer(p))
	n := uintptr(0)
	for {
		if *(*byte)(unsafe.Pointer(base + n)) == 0 {
			break
		}
		n++
	}
	if n == 0 {
		return ""
	}
	return string(unsafe.Slice(p, n))
}
