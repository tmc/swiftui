package avkit

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
	_AVS_FreeString(p)
	return s
}

// Retain increments the reference count of a bridge object.
func Retain(ptr uintptr) uintptr { return _AVS_Retain(ptr) }

// Release decrements the reference count of a bridge object.
func Release(ptr uintptr) { _AVS_Release(ptr) }

// NewVideoPlayer creates a VideoPlayer view for an AVPlayer pointer.
func NewVideoPlayer(player uintptr) uintptr {
	return _AVS_VideoPlayerCreate(player)
}

// NewVideoPlayerWithOverlay creates a VideoPlayer with a SwiftUI overlay view.
func NewVideoPlayerWithOverlay(player uintptr, overlay uintptr) uintptr {
	return _AVS_VideoPlayerWithOverlayCreate(player, overlay)
}
