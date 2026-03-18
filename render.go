package swiftui

import (
	"fmt"
	"unsafe"
)

// RenderPNG renders a SwiftUI view hierarchy to a PNG file without opening a window.
func RenderPNG(path string, root View, width, height, scale float64) error {
	if root.ptr == 0 {
		return fmt.Errorf("swiftui: render png: nil root view")
	}
	if path == "" {
		return fmt.Errorf("swiftui: render png: empty path")
	}
	if width <= 0 || height <= 0 {
		return fmt.Errorf("swiftui: render png: width and height must be positive")
	}
	if scale <= 0 {
		scale = 1
	}
	if _SUIRenderPNG == nil {
		if loadErr != nil {
			return fmt.Errorf("swiftui: render png: %w", loadErr)
		}
		return fmt.Errorf("swiftui: render png: bridge not available")
	}

	var errPtr *byte
	withCString(path, func(pathC *byte) {
		errPtr = _SUIRenderPNG(root.ptr, pathC, width, height, scale)
	})
	if errPtr == nil {
		return nil
	}
	defer _SUIFreeString(errPtr)
	return fmt.Errorf("swiftui: render png: %s", cString(errPtr))
}

func cString(p *byte) string {
	if p == nil {
		return ""
	}
	var buf []byte
	for i := uintptr(0); ; i++ {
		b := *(*byte)(unsafe.Pointer(uintptr(unsafe.Pointer(p)) + i))
		if b == 0 {
			break
		}
		buf = append(buf, b)
	}
	return string(buf)
}
