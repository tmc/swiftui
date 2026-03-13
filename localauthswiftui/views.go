package localauthswiftui

import "unsafe"

// withCString calls fn with a null-terminated C string pointer.
func withCString(s string, fn func(*byte)) {
	b := append([]byte(s), 0)
	fn(&b[0])
}

// gostring copies a null-terminated C string to a Go string.
func gostring(p *byte) string {
	if p == nil {
		return ""
	}
	var buf []byte
	for ptr := p; *ptr != 0; ptr = (*byte)(unsafe.Add(unsafe.Pointer(ptr), 1)) {
		buf = append(buf, *ptr)
	}
	return string(buf)
}

// gostringFree copies a C string and frees it.
func gostringFree(p *byte) string {
	if p == nil {
		return ""
	}
	s := gostring(p)
	_LAS_FreeString(p)
	return s
}

// Retain increments the reference count of a bridge object.
func Retain(ptr uintptr) uintptr { return _LAS_Retain(ptr) }

// Release decrements the reference count of a bridge object.
func Release(ptr uintptr) { _LAS_Release(ptr) }

// NewLocalAuthView creates a LocalAuthenticationView with the given reason string.
func NewLocalAuthView(reason string) uintptr {
	var result uintptr
	withCString(reason, func(cReason *byte) {
		result = _LAS_LocalAuthViewCreate(cReason)
	})
	return result
}
