package swiftui

import (
	"runtime"
	"sync"
)

var (
	surfaceExtraLibOnce sync.Once

	_SUIViewOnTapGestureCount             func(uintptr, int32, uintptr) uintptr
	_SUIViewScaleEffectAnchor             func(uintptr, float64, float64, float64) uintptr
	_SUIViewAnimationWithDuration         func(uintptr, int32, float64) uintptr
	_SUIStateSetIntAnimatedWithDuration   func(uintptr, int32, int32, float64)
	_SUIStateSetFloatAnimatedWithDuration func(uintptr, float64, int32, float64)
	_SUIStateSetBoolAnimatedWithDuration  func(uintptr, int32, int32, float64)
)

func ensureSurfaceExtraLibFuncs() {
	surfaceExtraLibOnce.Do(func() {
		if libHandle != 0 {
			tryRegisterLibFunc(&_SUIViewOnTapGestureCount, libHandle, "SUIViewOnTapGestureCount")
			tryRegisterLibFunc(&_SUIViewScaleEffectAnchor, libHandle, "SUIViewScaleEffectAnchor")
			tryRegisterLibFunc(&_SUIViewAnimationWithDuration, libHandle, "SUIViewAnimationWithDuration")
			tryRegisterLibFunc(&_SUIStateSetIntAnimatedWithDuration, libHandle, "SUIStateSetIntAnimatedWithDuration")
			tryRegisterLibFunc(&_SUIStateSetFloatAnimatedWithDuration, libHandle, "SUIStateSetFloatAnimatedWithDuration")
			tryRegisterLibFunc(&_SUIStateSetBoolAnimatedWithDuration, libHandle, "SUIStateSetBoolAnimatedWithDuration")
		}
		setSurfaceExtraUnavailableStubs()
	})
}

func setSurfaceExtraUnavailableStubs() {
	if _SUIViewOnTapGestureCount == nil {
		_SUIViewOnTapGestureCount = func(ptr uintptr, _ int32, callbackID uintptr) uintptr {
			return _SUIViewOnTapGesture(ptr, callbackID)
		}
	}
	if _SUIViewScaleEffectAnchor == nil {
		_SUIViewScaleEffectAnchor = func(ptr uintptr, scale float64, _ float64, _ float64) uintptr {
			return _SUIViewScaleEffect(ptr, scale)
		}
	}
	if _SUIViewAnimationWithDuration == nil {
		_SUIViewAnimationWithDuration = func(ptr uintptr, kind int32, _ float64) uintptr {
			return _SUIViewAnimation(ptr, kind)
		}
	}
	if _SUIStateSetIntAnimatedWithDuration == nil {
		_SUIStateSetIntAnimatedWithDuration = func(ptr uintptr, value int32, kind int32, _ float64) {
			_SUIStateSetIntAnimatedWith(ptr, value, kind)
		}
	}
	if _SUIStateSetFloatAnimatedWithDuration == nil {
		_SUIStateSetFloatAnimatedWithDuration = func(ptr uintptr, value float64, kind int32, _ float64) {
			_SUIStateSetFloatAnimatedWith(ptr, value, kind)
		}
	}
	if _SUIStateSetBoolAnimatedWithDuration == nil {
		_SUIStateSetBoolAnimatedWithDuration = func(ptr uintptr, value int32, kind int32, _ float64) {
			_SUIStateSetBoolAnimatedWith(ptr, value, kind)
		}
	}
}

// OnTapGestureCount adds a tap gesture handler that fires after count taps.
func (v View) OnTapGestureCount(count int, action func()) View {
	if count <= 1 {
		return v.OnTapGesture(action)
	}
	ensureSurfaceExtraLibFuncs()
	actionID := registerCallback(action)
	ptr := _SUIViewOnTapGestureCount(v.ptr, int32(count), actionID)
	ret := View{ptr: ptr, retained: newRetainedOwned(ptr)}
	ret.retained.addCallbackID(actionID)
	runtime.KeepAlive(v.retained)
	return ret
}

// ScaleEffectAnchor scales the view around the provided anchor point.
func (v View) ScaleEffectAnchor(scale float64, anchor UnitPoint) View {
	ensureSurfaceExtraLibFuncs()
	ptr := _SUIViewScaleEffectAnchor(v.ptr, scale, anchor.X, anchor.Y)
	runtime.KeepAlive(v.retained)
	return View{ptr: ptr, retained: newRetainedOwned(ptr)}
}

// AnimationWithDuration applies an animation curve with an explicit duration.
//
// Duration is honored by the easing families. Spring and bouncy curves keep
// their platform defaults today.
func (v View) AnimationWithDuration(kind AnimationKind, duration float64) View {
	if duration <= 0 {
		return v.Animation(kind)
	}
	ensureSurfaceExtraLibFuncs()
	ptr := _SUIViewAnimationWithDuration(v.ptr, int32(kind), duration)
	runtime.KeepAlive(v.retained)
	return View{ptr: ptr, retained: newRetainedOwned(ptr)}
}

// SetAnimatedWithDuration updates the value using the provided animation curve and duration.
func (s *IntState) SetAnimatedWithDuration(v int, kind AnimationKind, duration float64) {
	if duration <= 0 {
		s.SetAnimatedWith(v, kind)
		return
	}
	ensureSurfaceExtraLibFuncs()
	s.cache.observe(int64(v))
	_SUIStateSetIntAnimatedWithDuration(s.ptr, int32(v), int32(kind), duration)
}

// SetAnimatedWithDuration updates the float64 value using the provided animation curve and duration.
func (s *FloatState) SetAnimatedWithDuration(v float64, kind AnimationKind, duration float64) {
	if duration <= 0 {
		s.SetAnimatedWith(v, kind)
		return
	}
	ensureSurfaceExtraLibFuncs()
	s.cache.checkSet(v)
	_SUIStateSetFloatAnimatedWithDuration(s.ptr, v, int32(kind), duration)
}

// SetAnimatedWithDuration updates the boolean value using the provided animation curve and duration.
func (s *BoolState) SetAnimatedWithDuration(v bool, kind AnimationKind, duration float64) {
	if duration <= 0 {
		s.SetAnimatedWith(v, kind)
		return
	}
	ensureSurfaceExtraLibFuncs()
	s.cache.checkSet(v)
	var vv int32
	if v {
		vv = 1
	}
	_SUIStateSetBoolAnimatedWithDuration(s.ptr, vv, int32(kind), duration)
}
