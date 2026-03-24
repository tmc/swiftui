package charts

import "time"

func noopCancel() {}

type stateBinding interface {
	statePointer() uintptr
	stateKind() int32
}

const (
	stateKindNumber int32 = iota
	stateKindDate
	stateKindOptionalNumber
	stateKindOptionalDate
	stateKindNumberRange
	stateKindDateRange
)

// NumberState is a bindable numeric chart state.
type NumberState struct {
	ptr      uintptr
	retained *retained
}

func NewNumberState(initial float64) *NumberState {
	ensureExtraLibFuncs()
	ptr := _CHStateCreateNumber(initial)
	return &NumberState{ptr: ptr, retained: newRetained(ptr)}
}

func (s *NumberState) Get() float64 {
	if s == nil {
		return 0
	}
	return _CHStateGetNumber(s.ptr)
}

func (s *NumberState) Set(v float64) {
	if s == nil {
		return
	}
	_CHStateSetNumber(s.ptr, v)
}

// OnChange registers a callback for changes to the state value.
// The returned function removes the observer.
func (s *NumberState) OnChange(fn func(float64)) func() {
	if s == nil || s.ptr == 0 || fn == nil {
		return noopCancel
	}
	return registerStateObserver(s.ptr, func(snapshot stateSnapshot) {
		fn(snapshot.value0)
	})
}

func (s *NumberState) Release() {
	if s == nil || s.retained == nil {
		return
	}
	clearStateObservers(s.ptr)
	s.retained.release()
	s.retained = nil
	s.ptr = 0
}

func (s *NumberState) statePointer() uintptr {
	if s == nil {
		return 0
	}
	return s.ptr
}
func (s *NumberState) stateKind() int32 { return stateKindNumber }

// DateState is a bindable time-based chart state.
type DateState struct {
	ptr      uintptr
	retained *retained
}

func NewDateState(initial time.Time) *DateState {
	ensureExtraLibFuncs()
	ptr := _CHStateCreateDate(float64(initial.UnixMilli()) / 1000.0)
	return &DateState{ptr: ptr, retained: newRetained(ptr)}
}

func (s *DateState) Get() time.Time {
	if s == nil {
		return time.Unix(0, 0)
	}
	seconds := _CHStateGetDate(s.ptr)
	return time.UnixMilli(int64(seconds * 1000))
}

func (s *DateState) Set(v time.Time) {
	if s == nil {
		return
	}
	_CHStateSetDate(s.ptr, float64(v.UnixMilli())/1000.0)
}

// OnChange registers a callback for changes to the state value.
// The returned function removes the observer.
func (s *DateState) OnChange(fn func(time.Time)) func() {
	if s == nil || s.ptr == 0 || fn == nil {
		return noopCancel
	}
	return registerStateObserver(s.ptr, func(snapshot stateSnapshot) {
		fn(time.UnixMilli(int64(snapshot.value0 * 1000)))
	})
}

func (s *DateState) Release() {
	if s == nil || s.retained == nil {
		return
	}
	clearStateObservers(s.ptr)
	s.retained.release()
	s.retained = nil
	s.ptr = 0
}

func (s *DateState) statePointer() uintptr {
	if s == nil {
		return 0
	}
	return s.ptr
}
func (s *DateState) stateKind() int32 { return stateKindDate }

// OptionalNumberState is a bindable optional numeric selection state.
type OptionalNumberState struct {
	ptr      uintptr
	retained *retained
}

func NewOptionalNumberState(initial float64, ok bool) *OptionalNumberState {
	ensureExtraLibFuncs()
	var has int32
	if ok {
		has = 1
	}
	ptr := _CHStateCreateOptionalNumber(initial, has)
	return &OptionalNumberState{ptr: ptr, retained: newRetained(ptr)}
}

func (s *OptionalNumberState) Get() (float64, bool) {
	if s == nil {
		return 0, false
	}
	if _CHStateHasOptionalNumber(s.ptr) == 0 {
		return 0, false
	}
	return _CHStateGetOptionalNumber(s.ptr), true
}

func (s *OptionalNumberState) Set(v float64) {
	if s == nil {
		return
	}
	_CHStateSetOptionalNumber(s.ptr, v)
}

func (s *OptionalNumberState) Clear() {
	if s == nil {
		return
	}
	_CHStateClearOptionalNumber(s.ptr)
}

// OnChange registers a callback for changes to the selection state.
// The returned function removes the observer.
func (s *OptionalNumberState) OnChange(fn func(float64, bool)) func() {
	if s == nil || s.ptr == 0 || fn == nil {
		return noopCancel
	}
	return registerStateObserver(s.ptr, func(snapshot stateSnapshot) {
		fn(snapshot.value0, snapshot.hasValue)
	})
}

func (s *OptionalNumberState) Release() {
	if s == nil || s.retained == nil {
		return
	}
	clearStateObservers(s.ptr)
	s.retained.release()
	s.retained = nil
	s.ptr = 0
}

func (s *OptionalNumberState) statePointer() uintptr {
	if s == nil {
		return 0
	}
	return s.ptr
}
func (s *OptionalNumberState) stateKind() int32 { return stateKindOptionalNumber }

// OptionalDateState is a bindable optional time-based selection state.
type OptionalDateState struct {
	ptr      uintptr
	retained *retained
}

func NewOptionalDateState(initial time.Time, ok bool) *OptionalDateState {
	ensureExtraLibFuncs()
	var has int32
	if ok {
		has = 1
	}
	ptr := _CHStateCreateOptionalDate(float64(initial.UnixMilli())/1000.0, has)
	return &OptionalDateState{ptr: ptr, retained: newRetained(ptr)}
}

func (s *OptionalDateState) Get() (time.Time, bool) {
	if s == nil {
		return time.Unix(0, 0), false
	}
	if _CHStateHasOptionalDate(s.ptr) == 0 {
		return time.Unix(0, 0), false
	}
	seconds := _CHStateGetOptionalDate(s.ptr)
	return time.UnixMilli(int64(seconds * 1000)), true
}

func (s *OptionalDateState) Set(v time.Time) {
	if s == nil {
		return
	}
	_CHStateSetOptionalDate(s.ptr, float64(v.UnixMilli())/1000.0)
}

func (s *OptionalDateState) Clear() {
	if s == nil {
		return
	}
	_CHStateClearOptionalDate(s.ptr)
}

// OnChange registers a callback for changes to the selection state.
// The returned function removes the observer.
func (s *OptionalDateState) OnChange(fn func(time.Time, bool)) func() {
	if s == nil || s.ptr == 0 || fn == nil {
		return noopCancel
	}
	return registerStateObserver(s.ptr, func(snapshot stateSnapshot) {
		fn(time.UnixMilli(int64(snapshot.value0*1000)), snapshot.hasValue)
	})
}

func (s *OptionalDateState) Release() {
	if s == nil || s.retained == nil {
		return
	}
	clearStateObservers(s.ptr)
	s.retained.release()
	s.retained = nil
	s.ptr = 0
}

func (s *OptionalDateState) statePointer() uintptr {
	if s == nil {
		return 0
	}
	return s.ptr
}
func (s *OptionalDateState) stateKind() int32 { return stateKindOptionalDate }

// NumberRangeState is a bindable numeric range selection state.
type NumberRangeState struct {
	ptr      uintptr
	retained *retained
}

func NewNumberRangeState(start, end float64, ok bool) *NumberRangeState {
	ensureExtraLibFuncs()
	var has int32
	if ok {
		has = 1
	}
	ptr := _CHStateCreateNumberRange(start, end, has)
	return &NumberRangeState{ptr: ptr, retained: newRetained(ptr)}
}

func (s *NumberRangeState) Get() (float64, float64, bool) {
	if s == nil {
		return 0, 0, false
	}
	if _CHStateHasNumberRange(s.ptr) == 0 {
		return 0, 0, false
	}
	return _CHStateGetNumberRangeStart(s.ptr), _CHStateGetNumberRangeEnd(s.ptr), true
}

func (s *NumberRangeState) Set(start, end float64) {
	if s == nil {
		return
	}
	_CHStateSetNumberRange(s.ptr, start, end)
}

func (s *NumberRangeState) Clear() {
	if s == nil {
		return
	}
	_CHStateClearNumberRange(s.ptr)
}

// OnChange registers a callback for changes to the range state.
// The returned function removes the observer.
func (s *NumberRangeState) OnChange(fn func(float64, float64, bool)) func() {
	if s == nil || s.ptr == 0 || fn == nil {
		return noopCancel
	}
	return registerStateObserver(s.ptr, func(snapshot stateSnapshot) {
		fn(snapshot.value0, snapshot.value1, snapshot.hasValue)
	})
}

func (s *NumberRangeState) Release() {
	if s == nil || s.retained == nil {
		return
	}
	clearStateObservers(s.ptr)
	s.retained.release()
	s.retained = nil
	s.ptr = 0
}

func (s *NumberRangeState) statePointer() uintptr {
	if s == nil {
		return 0
	}
	return s.ptr
}
func (s *NumberRangeState) stateKind() int32 { return stateKindNumberRange }

// DateRangeState is a bindable time-based range selection state.
type DateRangeState struct {
	ptr      uintptr
	retained *retained
}

func NewDateRangeState(start, end time.Time, ok bool) *DateRangeState {
	ensureExtraLibFuncs()
	var has int32
	if ok {
		has = 1
	}
	ptr := _CHStateCreateDateRange(
		float64(start.UnixMilli())/1000.0,
		float64(end.UnixMilli())/1000.0,
		has,
	)
	return &DateRangeState{ptr: ptr, retained: newRetained(ptr)}
}

func (s *DateRangeState) Get() (time.Time, time.Time, bool) {
	if s == nil {
		return time.Unix(0, 0), time.Unix(0, 0), false
	}
	if _CHStateHasDateRange(s.ptr) == 0 {
		return time.Unix(0, 0), time.Unix(0, 0), false
	}
	start := time.UnixMilli(int64(_CHStateGetDateRangeStart(s.ptr) * 1000))
	end := time.UnixMilli(int64(_CHStateGetDateRangeEnd(s.ptr) * 1000))
	return start, end, true
}

func (s *DateRangeState) Set(start, end time.Time) {
	if s == nil {
		return
	}
	_CHStateSetDateRange(
		s.ptr,
		float64(start.UnixMilli())/1000.0,
		float64(end.UnixMilli())/1000.0,
	)
}

func (s *DateRangeState) Clear() {
	if s == nil {
		return
	}
	_CHStateClearDateRange(s.ptr)
}

// OnChange registers a callback for changes to the range state.
// The returned function removes the observer.
func (s *DateRangeState) OnChange(fn func(time.Time, time.Time, bool)) func() {
	if s == nil || s.ptr == 0 || fn == nil {
		return noopCancel
	}
	return registerStateObserver(s.ptr, func(snapshot stateSnapshot) {
		fn(
			time.UnixMilli(int64(snapshot.value0*1000)),
			time.UnixMilli(int64(snapshot.value1*1000)),
			snapshot.hasValue,
		)
	})
}

func (s *DateRangeState) Release() {
	if s == nil || s.retained == nil {
		return
	}
	clearStateObservers(s.ptr)
	s.retained.release()
	s.retained = nil
	s.ptr = 0
}

func (s *DateRangeState) statePointer() uintptr {
	if s == nil {
		return 0
	}
	return s.ptr
}
func (s *DateRangeState) stateKind() int32 { return stateKindDateRange }
