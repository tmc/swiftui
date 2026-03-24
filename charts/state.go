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
	mirror   *stateMirror
}

func NewNumberState(initial float64) *NumberState {
	ensureExtraLibFuncs()
	ptr := _CHStateCreateNumber(initial)
	mirror := newStateMirror(stateKindNumber, initial, 0, true)
	registerStateMirror(ptr, mirror)
	return &NumberState{ptr: ptr, retained: newRetained(ptr), mirror: mirror}
}

func (s *NumberState) Get() float64 {
	if s == nil {
		return 0
	}
	return s.mirror.load().value0
}

func (s *NumberState) Set(v float64) {
	if s == nil {
		return
	}
	_CHStateSetNumber(s.ptr, v)
	s.mirror.store(stateSnapshot{kind: stateKindNumber, value0: v, hasValue: true})
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
	clearStateMirror(s.ptr)
	s.retained.release()
	s.retained = nil
	s.mirror = nil
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
	mirror   *stateMirror
}

func NewDateState(initial time.Time) *DateState {
	ensureExtraLibFuncs()
	ptr := _CHStateCreateDate(float64(initial.UnixMilli()) / 1000.0)
	seconds := float64(initial.UnixMilli()) / 1000.0
	mirror := newStateMirror(stateKindDate, seconds, 0, true)
	registerStateMirror(ptr, mirror)
	return &DateState{ptr: ptr, retained: newRetained(ptr), mirror: mirror}
}

func (s *DateState) Get() time.Time {
	if s == nil {
		return time.Unix(0, 0)
	}
	seconds := s.mirror.load().value0
	return time.UnixMilli(int64(seconds * 1000))
}

func (s *DateState) Set(v time.Time) {
	if s == nil {
		return
	}
	seconds := float64(v.UnixMilli()) / 1000.0
	_CHStateSetDate(s.ptr, seconds)
	s.mirror.store(stateSnapshot{kind: stateKindDate, value0: seconds, hasValue: true})
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
	clearStateMirror(s.ptr)
	s.retained.release()
	s.retained = nil
	s.mirror = nil
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
	mirror   *stateMirror
}

func NewOptionalNumberState(initial float64, ok bool) *OptionalNumberState {
	ensureExtraLibFuncs()
	var has int32
	if ok {
		has = 1
	}
	ptr := _CHStateCreateOptionalNumber(initial, has)
	mirror := newStateMirror(stateKindOptionalNumber, initial, 0, ok)
	registerStateMirror(ptr, mirror)
	return &OptionalNumberState{ptr: ptr, retained: newRetained(ptr), mirror: mirror}
}

func (s *OptionalNumberState) Get() (float64, bool) {
	if s == nil {
		return 0, false
	}
	snapshot := s.mirror.load()
	if !snapshot.hasValue {
		return 0, false
	}
	return snapshot.value0, true
}

func (s *OptionalNumberState) Set(v float64) {
	if s == nil {
		return
	}
	_CHStateSetOptionalNumber(s.ptr, v)
	s.mirror.store(stateSnapshot{kind: stateKindOptionalNumber, value0: v, hasValue: true})
}

func (s *OptionalNumberState) Clear() {
	if s == nil {
		return
	}
	_CHStateClearOptionalNumber(s.ptr)
	s.mirror.store(stateSnapshot{kind: stateKindOptionalNumber})
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
	clearStateMirror(s.ptr)
	s.retained.release()
	s.retained = nil
	s.mirror = nil
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
	mirror   *stateMirror
}

func NewOptionalDateState(initial time.Time, ok bool) *OptionalDateState {
	ensureExtraLibFuncs()
	var has int32
	if ok {
		has = 1
	}
	seconds := float64(initial.UnixMilli()) / 1000.0
	ptr := _CHStateCreateOptionalDate(seconds, has)
	mirror := newStateMirror(stateKindOptionalDate, seconds, 0, ok)
	registerStateMirror(ptr, mirror)
	return &OptionalDateState{ptr: ptr, retained: newRetained(ptr), mirror: mirror}
}

func (s *OptionalDateState) Get() (time.Time, bool) {
	if s == nil {
		return time.Unix(0, 0), false
	}
	snapshot := s.mirror.load()
	if !snapshot.hasValue {
		return time.Unix(0, 0), false
	}
	seconds := snapshot.value0
	return time.UnixMilli(int64(seconds * 1000)), true
}

func (s *OptionalDateState) Set(v time.Time) {
	if s == nil {
		return
	}
	seconds := float64(v.UnixMilli()) / 1000.0
	_CHStateSetOptionalDate(s.ptr, seconds)
	s.mirror.store(stateSnapshot{kind: stateKindOptionalDate, value0: seconds, hasValue: true})
}

func (s *OptionalDateState) Clear() {
	if s == nil {
		return
	}
	_CHStateClearOptionalDate(s.ptr)
	s.mirror.store(stateSnapshot{kind: stateKindOptionalDate})
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
	clearStateMirror(s.ptr)
	s.retained.release()
	s.retained = nil
	s.mirror = nil
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
	mirror   *stateMirror
}

func NewNumberRangeState(start, end float64, ok bool) *NumberRangeState {
	ensureExtraLibFuncs()
	var has int32
	if ok {
		has = 1
	}
	ptr := _CHStateCreateNumberRange(start, end, has)
	mirror := newStateMirror(stateKindNumberRange, start, end, ok)
	registerStateMirror(ptr, mirror)
	return &NumberRangeState{ptr: ptr, retained: newRetained(ptr), mirror: mirror}
}

func (s *NumberRangeState) Get() (float64, float64, bool) {
	if s == nil {
		return 0, 0, false
	}
	snapshot := s.mirror.load()
	if !snapshot.hasValue {
		return 0, 0, false
	}
	return snapshot.value0, snapshot.value1, true
}

func (s *NumberRangeState) Set(start, end float64) {
	if s == nil {
		return
	}
	_CHStateSetNumberRange(s.ptr, start, end)
	s.mirror.store(stateSnapshot{kind: stateKindNumberRange, value0: start, value1: end, hasValue: true})
}

func (s *NumberRangeState) Clear() {
	if s == nil {
		return
	}
	_CHStateClearNumberRange(s.ptr)
	s.mirror.store(stateSnapshot{kind: stateKindNumberRange})
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
	clearStateMirror(s.ptr)
	s.retained.release()
	s.retained = nil
	s.mirror = nil
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
	mirror   *stateMirror
}

func NewDateRangeState(start, end time.Time, ok bool) *DateRangeState {
	ensureExtraLibFuncs()
	var has int32
	if ok {
		has = 1
	}
	startSeconds := float64(start.UnixMilli()) / 1000.0
	endSeconds := float64(end.UnixMilli()) / 1000.0
	ptr := _CHStateCreateDateRange(startSeconds, endSeconds, has)
	mirror := newStateMirror(stateKindDateRange, startSeconds, endSeconds, ok)
	registerStateMirror(ptr, mirror)
	return &DateRangeState{ptr: ptr, retained: newRetained(ptr), mirror: mirror}
}

func (s *DateRangeState) Get() (time.Time, time.Time, bool) {
	if s == nil {
		return time.Unix(0, 0), time.Unix(0, 0), false
	}
	snapshot := s.mirror.load()
	if !snapshot.hasValue {
		return time.Unix(0, 0), time.Unix(0, 0), false
	}
	start := time.UnixMilli(int64(snapshot.value0 * 1000))
	end := time.UnixMilli(int64(snapshot.value1 * 1000))
	return start, end, true
}

func (s *DateRangeState) Set(start, end time.Time) {
	if s == nil {
		return
	}
	startSeconds := float64(start.UnixMilli()) / 1000.0
	endSeconds := float64(end.UnixMilli()) / 1000.0
	_CHStateSetDateRange(s.ptr, startSeconds, endSeconds)
	s.mirror.store(stateSnapshot{kind: stateKindDateRange, value0: startSeconds, value1: endSeconds, hasValue: true})
}

func (s *DateRangeState) Clear() {
	if s == nil {
		return
	}
	_CHStateClearDateRange(s.ptr)
	s.mirror.store(stateSnapshot{kind: stateKindDateRange})
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
	clearStateMirror(s.ptr)
	s.retained.release()
	s.retained = nil
	s.mirror = nil
	s.ptr = 0
}

func (s *DateRangeState) statePointer() uintptr {
	if s == nil {
		return 0
	}
	return s.ptr
}
func (s *DateRangeState) stateKind() int32 { return stateKindDateRange }
