package charts

import (
	"sync"

	"github.com/ebitengine/purego"
)

type retained struct {
	ptr  uintptr
	once sync.Once
}

var (
	stateObserverMu   sync.Mutex
	stateObserverMap  = map[uintptr]map[uintptr]func(stateSnapshot){}
	stateObserverNext uintptr
)

type stateSnapshot struct {
	kind     int32
	value0   float64
	value1   float64
	hasValue bool
}

func registerStateObserver(ptr uintptr, fn func(stateSnapshot)) func() {
	if ptr == 0 || fn == nil {
		return func() {}
	}
	stateObserverMu.Lock()
	defer stateObserverMu.Unlock()
	stateObserverNext++
	id := stateObserverNext
	observers := stateObserverMap[ptr]
	if observers == nil {
		observers = make(map[uintptr]func(stateSnapshot))
		stateObserverMap[ptr] = observers
	}
	observers[id] = fn
	return func() {
		stateObserverMu.Lock()
		defer stateObserverMu.Unlock()
		observers := stateObserverMap[ptr]
		if observers == nil {
			return
		}
		delete(observers, id)
		if len(observers) == 0 {
			delete(stateObserverMap, ptr)
		}
	}
}

func clearStateObservers(ptr uintptr) {
	if ptr == 0 {
		return
	}
	stateObserverMu.Lock()
	delete(stateObserverMap, ptr)
	stateObserverMu.Unlock()
}

func stateChangeCallbackTrampoline(ptr uintptr, kind int32, value0, value1 float64, hasValue int32) {
	stateObserverMu.Lock()
	observers := stateObserverMap[ptr]
	callbacks := make([]func(stateSnapshot), 0, len(observers))
	for _, fn := range observers {
		callbacks = append(callbacks, fn)
	}
	stateObserverMu.Unlock()
	snapshot := stateSnapshot{
		kind:     kind,
		value0:   value0,
		value1:   value1,
		hasValue: hasValue != 0,
	}
	for _, fn := range callbacks {
		fn(snapshot)
	}
}

var stateChangeCallbackPtr = purego.NewCallback(stateChangeCallbackTrampoline)

func newRetained(ptr uintptr) *retained {
	if ptr == 0 {
		return nil
	}
	return &retained{ptr: ptr}
}

func (r *retained) release() {
	if r == nil {
		return
	}
	r.once.Do(func() {
		if r.ptr == 0 || _CHRelease == nil {
			return
		}
		_CHRelease(r.ptr)
		r.ptr = 0
	})
}

var (
	_CHStateCreateNumber         func(float64) uintptr
	_CHStateGetNumber            func(uintptr) float64
	_CHStateSetNumber            func(uintptr, float64)
	_CHStateCreateDate           func(float64) uintptr
	_CHStateGetDate              func(uintptr) float64
	_CHStateSetDate              func(uintptr, float64)
	_CHStateCreateOptionalNumber func(float64, int32) uintptr
	_CHStateHasOptionalNumber    func(uintptr) int32
	_CHStateGetOptionalNumber    func(uintptr) float64
	_CHStateSetOptionalNumber    func(uintptr, float64)
	_CHStateClearOptionalNumber  func(uintptr)
	_CHStateCreateOptionalDate   func(float64, int32) uintptr
	_CHStateHasOptionalDate      func(uintptr) int32
	_CHStateGetOptionalDate      func(uintptr) float64
	_CHStateSetOptionalDate      func(uintptr, float64)
	_CHStateClearOptionalDate    func(uintptr)
	_CHStateCreateNumberRange    func(float64, float64, int32) uintptr
	_CHStateHasNumberRange       func(uintptr) int32
	_CHStateGetNumberRangeStart  func(uintptr) float64
	_CHStateGetNumberRangeEnd    func(uintptr) float64
	_CHStateSetNumberRange       func(uintptr, float64, float64)
	_CHStateClearNumberRange     func(uintptr)
	_CHStateCreateDateRange      func(float64, float64, int32) uintptr
	_CHStateHasDateRange         func(uintptr) int32
	_CHStateGetDateRangeStart    func(uintptr) float64
	_CHStateGetDateRangeEnd      func(uintptr) float64
	_CHStateSetDateRange         func(uintptr, float64, float64)
	_CHStateClearDateRange       func(uintptr)
	_CHSetStateChangeCallback    func(uintptr)
)

var extraLibOnce sync.Once

func ensureExtraLibFuncs() {
	extraLibOnce.Do(func() {
		if libHandle != 0 {
			tryRegisterLibFunc(&_CHStateCreateNumber, libHandle, "CHStateCreateNumber")
			tryRegisterLibFunc(&_CHStateGetNumber, libHandle, "CHStateGetNumber")
			tryRegisterLibFunc(&_CHStateSetNumber, libHandle, "CHStateSetNumber")
			tryRegisterLibFunc(&_CHStateCreateDate, libHandle, "CHStateCreateDate")
			tryRegisterLibFunc(&_CHStateGetDate, libHandle, "CHStateGetDate")
			tryRegisterLibFunc(&_CHStateSetDate, libHandle, "CHStateSetDate")
			tryRegisterLibFunc(&_CHStateCreateOptionalNumber, libHandle, "CHStateCreateOptionalNumber")
			tryRegisterLibFunc(&_CHStateHasOptionalNumber, libHandle, "CHStateHasOptionalNumber")
			tryRegisterLibFunc(&_CHStateGetOptionalNumber, libHandle, "CHStateGetOptionalNumber")
			tryRegisterLibFunc(&_CHStateSetOptionalNumber, libHandle, "CHStateSetOptionalNumber")
			tryRegisterLibFunc(&_CHStateClearOptionalNumber, libHandle, "CHStateClearOptionalNumber")
			tryRegisterLibFunc(&_CHStateCreateOptionalDate, libHandle, "CHStateCreateOptionalDate")
			tryRegisterLibFunc(&_CHStateHasOptionalDate, libHandle, "CHStateHasOptionalDate")
			tryRegisterLibFunc(&_CHStateGetOptionalDate, libHandle, "CHStateGetOptionalDate")
			tryRegisterLibFunc(&_CHStateSetOptionalDate, libHandle, "CHStateSetOptionalDate")
			tryRegisterLibFunc(&_CHStateClearOptionalDate, libHandle, "CHStateClearOptionalDate")
			tryRegisterLibFunc(&_CHStateCreateNumberRange, libHandle, "CHStateCreateNumberRange")
			tryRegisterLibFunc(&_CHStateHasNumberRange, libHandle, "CHStateHasNumberRange")
			tryRegisterLibFunc(&_CHStateGetNumberRangeStart, libHandle, "CHStateGetNumberRangeStart")
			tryRegisterLibFunc(&_CHStateGetNumberRangeEnd, libHandle, "CHStateGetNumberRangeEnd")
			tryRegisterLibFunc(&_CHStateSetNumberRange, libHandle, "CHStateSetNumberRange")
			tryRegisterLibFunc(&_CHStateClearNumberRange, libHandle, "CHStateClearNumberRange")
			tryRegisterLibFunc(&_CHStateCreateDateRange, libHandle, "CHStateCreateDateRange")
			tryRegisterLibFunc(&_CHStateHasDateRange, libHandle, "CHStateHasDateRange")
			tryRegisterLibFunc(&_CHStateGetDateRangeStart, libHandle, "CHStateGetDateRangeStart")
			tryRegisterLibFunc(&_CHStateGetDateRangeEnd, libHandle, "CHStateGetDateRangeEnd")
			tryRegisterLibFunc(&_CHStateSetDateRange, libHandle, "CHStateSetDateRange")
			tryRegisterLibFunc(&_CHStateClearDateRange, libHandle, "CHStateClearDateRange")
			if tryRegisterLibFunc(&_CHSetStateChangeCallback, libHandle, "CHSetStateChangeCallback") {
				_CHSetStateChangeCallback(stateChangeCallbackPtr)
			}
		}
		setExtraUnavailableStubs()
	})
}

func setExtraUnavailableStubs() {
	stub := func(name string) {
		if loadErr != nil {
			panic("charts: " + name + ": " + loadErr.Error())
		}
		panic("charts: " + name + ": dylib not loaded")
	}
	if _CHStateCreateNumber == nil {
		_CHStateCreateNumber = func(float64) uintptr { stub("CHStateCreateNumber"); return 0 }
	}
	if _CHStateGetNumber == nil {
		_CHStateGetNumber = func(uintptr) float64 { stub("CHStateGetNumber"); return 0 }
	}
	if _CHStateSetNumber == nil {
		_CHStateSetNumber = func(uintptr, float64) { stub("CHStateSetNumber") }
	}
	if _CHStateCreateDate == nil {
		_CHStateCreateDate = func(float64) uintptr { stub("CHStateCreateDate"); return 0 }
	}
	if _CHStateGetDate == nil {
		_CHStateGetDate = func(uintptr) float64 { stub("CHStateGetDate"); return 0 }
	}
	if _CHStateSetDate == nil {
		_CHStateSetDate = func(uintptr, float64) { stub("CHStateSetDate") }
	}
	if _CHStateCreateOptionalNumber == nil {
		_CHStateCreateOptionalNumber = func(float64, int32) uintptr { stub("CHStateCreateOptionalNumber"); return 0 }
	}
	if _CHStateHasOptionalNumber == nil {
		_CHStateHasOptionalNumber = func(uintptr) int32 { stub("CHStateHasOptionalNumber"); return 0 }
	}
	if _CHStateGetOptionalNumber == nil {
		_CHStateGetOptionalNumber = func(uintptr) float64 { stub("CHStateGetOptionalNumber"); return 0 }
	}
	if _CHStateSetOptionalNumber == nil {
		_CHStateSetOptionalNumber = func(uintptr, float64) { stub("CHStateSetOptionalNumber") }
	}
	if _CHStateClearOptionalNumber == nil {
		_CHStateClearOptionalNumber = func(uintptr) { stub("CHStateClearOptionalNumber") }
	}
	if _CHStateCreateOptionalDate == nil {
		_CHStateCreateOptionalDate = func(float64, int32) uintptr { stub("CHStateCreateOptionalDate"); return 0 }
	}
	if _CHStateHasOptionalDate == nil {
		_CHStateHasOptionalDate = func(uintptr) int32 { stub("CHStateHasOptionalDate"); return 0 }
	}
	if _CHStateGetOptionalDate == nil {
		_CHStateGetOptionalDate = func(uintptr) float64 { stub("CHStateGetOptionalDate"); return 0 }
	}
	if _CHStateSetOptionalDate == nil {
		_CHStateSetOptionalDate = func(uintptr, float64) { stub("CHStateSetOptionalDate") }
	}
	if _CHStateClearOptionalDate == nil {
		_CHStateClearOptionalDate = func(uintptr) { stub("CHStateClearOptionalDate") }
	}
	if _CHStateCreateNumberRange == nil {
		_CHStateCreateNumberRange = func(float64, float64, int32) uintptr { stub("CHStateCreateNumberRange"); return 0 }
	}
	if _CHStateHasNumberRange == nil {
		_CHStateHasNumberRange = func(uintptr) int32 { stub("CHStateHasNumberRange"); return 0 }
	}
	if _CHStateGetNumberRangeStart == nil {
		_CHStateGetNumberRangeStart = func(uintptr) float64 { stub("CHStateGetNumberRangeStart"); return 0 }
	}
	if _CHStateGetNumberRangeEnd == nil {
		_CHStateGetNumberRangeEnd = func(uintptr) float64 { stub("CHStateGetNumberRangeEnd"); return 0 }
	}
	if _CHStateSetNumberRange == nil {
		_CHStateSetNumberRange = func(uintptr, float64, float64) { stub("CHStateSetNumberRange") }
	}
	if _CHStateClearNumberRange == nil {
		_CHStateClearNumberRange = func(uintptr) { stub("CHStateClearNumberRange") }
	}
	if _CHStateCreateDateRange == nil {
		_CHStateCreateDateRange = func(float64, float64, int32) uintptr { stub("CHStateCreateDateRange"); return 0 }
	}
	if _CHStateHasDateRange == nil {
		_CHStateHasDateRange = func(uintptr) int32 { stub("CHStateHasDateRange"); return 0 }
	}
	if _CHStateGetDateRangeStart == nil {
		_CHStateGetDateRangeStart = func(uintptr) float64 { stub("CHStateGetDateRangeStart"); return 0 }
	}
	if _CHStateGetDateRangeEnd == nil {
		_CHStateGetDateRangeEnd = func(uintptr) float64 { stub("CHStateGetDateRangeEnd"); return 0 }
	}
	if _CHStateSetDateRange == nil {
		_CHStateSetDateRange = func(uintptr, float64, float64) { stub("CHStateSetDateRange") }
	}
	if _CHStateClearDateRange == nil {
		_CHStateClearDateRange = func(uintptr) { stub("CHStateClearDateRange") }
	}
	if _CHSetStateChangeCallback == nil {
		_CHSetStateChangeCallback = func(uintptr) {}
	}
}
