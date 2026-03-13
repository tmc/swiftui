package swiftui

import "unsafe"

// IntState is a reactive integer state that bridges Go and SwiftUI.
// When the value changes via Set, any SwiftUI views observing this
// state update automatically.
type IntState struct {
	ptr      uintptr
	retained *retained
}

// NewIntState creates a new reactive integer state with the given initial value.
func NewIntState(initial int) *IntState {
	ptr := _SUIStateCreateInt(int32(initial))
	return &IntState{ptr: ptr, retained: newRetained(ptr)}
}

// Get returns the current value.
func (s *IntState) Get() int {
	return int(_SUIStateGetInt(s.ptr))
}

// Set updates the value, triggering SwiftUI view updates.
func (s *IntState) Set(v int) {
	_SUIStateSetInt(s.ptr, int32(v))
}

// SetAnimated updates the value inside withAnimation, triggering animated SwiftUI transitions.
func (s *IntState) SetAnimated(v int) {
	_SUIStateSetIntAnimated(s.ptr, int32(v))
}

// StringState is a reactive string state that bridges Go and SwiftUI.
type StringState struct {
	ptr      uintptr
	retained *retained
}

// NewStringState creates a new reactive string state with the given initial value.
func NewStringState(initial string) *StringState {
	var ptr uintptr
	withCString(initial, func(cs *byte) {
		ptr = _SUIStateCreateString(cs)
	})
	return &StringState{ptr: ptr, retained: newRetained(ptr)}
}

// Get returns the current string value.
func (s *StringState) Get() string {
	p := _SUIStateGetString(s.ptr)
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

// Set updates the string value, triggering SwiftUI view updates.
func (s *StringState) Set(v string) {
	withCString(v, func(cs *byte) {
		_SUIStateSetString(s.ptr, cs)
	})
}

// ColorState is a reactive RGBA color state that bridges Go and SwiftUI.
type ColorState struct {
	ptr      uintptr
	retained *retained
}

// NewColorState creates a new reactive color state from RGBA values (0.0-1.0).
func NewColorState(r, g, b, a float64) *ColorState {
	ptr := _SUIStateCreateColor(r, g, b, a)
	return &ColorState{ptr: ptr, retained: newRetained(ptr)}
}

// R returns the red component (0.0-1.0).
func (s *ColorState) R() float64 { return _SUIStateGetColorR(s.ptr) }

// G returns the green component (0.0-1.0).
func (s *ColorState) G() float64 { return _SUIStateGetColorG(s.ptr) }

// B returns the blue component (0.0-1.0).
func (s *ColorState) B() float64 { return _SUIStateGetColorB(s.ptr) }

// A returns the alpha component (0.0-1.0).
func (s *ColorState) A() float64 { return _SUIStateGetColorA(s.ptr) }

// Set updates the RGBA color values.
func (s *ColorState) Set(r, g, b, a float64) {
	_SUIStateSetColor(s.ptr, r, g, b, a)
}

// DateState is a reactive date state that bridges Go and SwiftUI.
// Dates are represented as Unix epoch seconds (float64).
type DateState struct {
	ptr      uintptr
	retained *retained
}

// NewDateState creates a new reactive date state from Unix epoch seconds.
func NewDateState(epochSeconds float64) *DateState {
	ptr := _SUIStateCreateDate(epochSeconds)
	return &DateState{ptr: ptr, retained: newRetained(ptr)}
}

// Get returns the current date as Unix epoch seconds.
func (s *DateState) Get() float64 {
	return _SUIStateGetDate(s.ptr)
}

// Set updates the date from Unix epoch seconds.
func (s *DateState) Set(epochSeconds float64) {
	_SUIStateSetDate(s.ptr, epochSeconds)
}

// FloatState is a reactive float64 state that bridges Go and SwiftUI.
type FloatState struct {
	ptr      uintptr
	retained *retained
}

// NewFloatState creates a new reactive float64 state with the given initial value.
func NewFloatState(initial float64) *FloatState {
	ptr := _SUIStateCreateFloat(initial)
	return &FloatState{ptr: ptr, retained: newRetained(ptr)}
}

// Get returns the current float64 value.
func (s *FloatState) Get() float64 {
	return _SUIStateGetFloat(s.ptr)
}

// Set updates the float64 value, triggering SwiftUI view updates.
func (s *FloatState) Set(v float64) {
	_SUIStateSetFloat(s.ptr, v)
}

// SetAnimated updates the float64 value inside withAnimation.
func (s *FloatState) SetAnimated(v float64) {
	_SUIStateSetFloatAnimated(s.ptr, v)
}

// BoolState is an observable boolean state value.
type BoolState struct {
	ptr      uintptr
	retained *retained
}

// NewBoolState creates a new observable boolean state.
func NewBoolState(initial bool) *BoolState {
	var v int32
	if initial {
		v = 1
	}
	ptr := _SUIStateCreateBool(v)
	return &BoolState{ptr: ptr, retained: newRetained(ptr)}
}

// Get returns the current boolean value.
func (s *BoolState) Get() bool {
	return _SUIStateGetBool(s.ptr) != 0
}

// Set updates the boolean value.
func (s *BoolState) Set(v bool) {
	var vv int32
	if v {
		vv = 1
	}
	_SUIStateSetBool(s.ptr, vv)
}

// Release decrements the underlying Swift retain count.
func (s *IntState) Release() {
	if s == nil || s.retained == nil {
		return
	}
	s.retained.release()
	s.retained = nil
	s.ptr = 0
}

// Release decrements the underlying Swift retain count.
func (s *StringState) Release() {
	if s == nil || s.retained == nil {
		return
	}
	s.retained.release()
	s.retained = nil
	s.ptr = 0
}

// Release decrements the underlying Swift retain count.
func (s *ColorState) Release() {
	if s == nil || s.retained == nil {
		return
	}
	s.retained.release()
	s.retained = nil
	s.ptr = 0
}

// Release decrements the underlying Swift retain count.
func (s *DateState) Release() {
	if s == nil || s.retained == nil {
		return
	}
	s.retained.release()
	s.retained = nil
	s.ptr = 0
}

// Release decrements the underlying Swift retain count.
func (s *FloatState) Release() {
	if s == nil || s.retained == nil {
		return
	}
	s.retained.release()
	s.retained = nil
	s.ptr = 0
}

// Release decrements the underlying Swift retain count.
func (s *BoolState) Release() {
	if s == nil || s.retained == nil {
		return
	}
	s.retained.release()
	s.retained = nil
	s.ptr = 0
}
