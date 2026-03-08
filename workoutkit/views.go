package workoutkit

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
	_WKS_FreeString(p)
	return s
}

// Retain increments the reference count of a bridge object.
func Retain(ptr uintptr) uintptr { return _WKS_Retain(ptr) }

// Release decrements the reference count of a bridge object.
func Release(ptr uintptr) { _WKS_Release(ptr) }

// NewWorkoutPreview creates a WorkoutPreview view for a workout composition.
func NewWorkoutPreview(workout uintptr) uintptr {
	return _WKS_WorkoutPreview(workout)
}
