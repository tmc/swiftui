package charts

import (
	"sync"

	"github.com/ebitengine/purego"
)

var (
	chartHoverMu       sync.Mutex
	chartHoverMap      = map[uintptr]*chartHoverMirror{}
	chartHoverNext     uintptr
	chartHoverObserver sync.Mutex
	chartHoverWatchers = map[uintptr]map[uintptr]func(ChartHoverEvent){}
	chartHoverWatchID  uintptr

	_CHSetHoverCallback func(uintptr)
	hoverLibOnce        sync.Once
)

type chartHoverMirror struct {
	mu    sync.RWMutex
	event ChartHoverEvent
}

func registerChartHoverState() uintptr {
	chartHoverMu.Lock()
	defer chartHoverMu.Unlock()
	chartHoverNext++
	id := chartHoverNext
	chartHoverMap[id] = &chartHoverMirror{}
	return id
}

func unregisterChartHoverState(id uintptr) {
	if id == 0 {
		return
	}
	chartHoverMu.Lock()
	delete(chartHoverMap, id)
	chartHoverMu.Unlock()
}

func loadChartHoverState(id uintptr) ChartHoverEvent {
	if id == 0 {
		return ChartHoverEvent{}
	}
	chartHoverMu.Lock()
	mirror := chartHoverMap[id]
	chartHoverMu.Unlock()
	if mirror == nil {
		return ChartHoverEvent{}
	}
	mirror.mu.RLock()
	event := mirror.event
	mirror.mu.RUnlock()
	return event
}

func storeChartHoverState(id uintptr, event ChartHoverEvent) {
	if id == 0 {
		return
	}
	chartHoverMu.Lock()
	mirror := chartHoverMap[id]
	chartHoverMu.Unlock()
	if mirror == nil {
		return
	}
	mirror.mu.Lock()
	mirror.event = event
	mirror.mu.Unlock()
}

func registerChartHoverObserver(id uintptr, fn func(ChartHoverEvent)) func() {
	if id == 0 || fn == nil {
		return noopCancel
	}
	chartHoverObserver.Lock()
	defer chartHoverObserver.Unlock()
	chartHoverWatchID++
	watchID := chartHoverWatchID
	observers := chartHoverWatchers[id]
	if observers == nil {
		observers = make(map[uintptr]func(ChartHoverEvent))
		chartHoverWatchers[id] = observers
	}
	observers[watchID] = fn
	return func() {
		chartHoverObserver.Lock()
		defer chartHoverObserver.Unlock()
		observers := chartHoverWatchers[id]
		if observers == nil {
			return
		}
		delete(observers, watchID)
		if len(observers) == 0 {
			delete(chartHoverWatchers, id)
		}
	}
}

func clearChartHoverObservers(id uintptr) {
	if id == 0 {
		return
	}
	chartHoverObserver.Lock()
	delete(chartHoverWatchers, id)
	chartHoverObserver.Unlock()
}

func chartHoverCallbackTrampoline(
	id uintptr,
	active int32,
	plotX, plotY, frameMinX, frameMinY, frameWidth, frameHeight float64,
	xKind int32,
	xNumber, xTime float64,
	yKind int32,
	yNumber, yTime float64,
) {
	event := ChartHoverEvent{
		Active:      active != 0,
		PlotX:       plotX,
		PlotY:       plotY,
		FrameMinX:   frameMinX,
		FrameMinY:   frameMinY,
		FrameWidth:  frameWidth,
		FrameHeight: frameHeight,
		XValue:      chartHoverValueFromBridge(xKind, xNumber, xTime),
		YValue:      chartHoverValueFromBridge(yKind, yNumber, yTime),
	}
	storeChartHoverState(id, event)

	chartHoverObserver.Lock()
	observers := chartHoverWatchers[id]
	callbacks := make([]func(ChartHoverEvent), 0, len(observers))
	for _, fn := range observers {
		callbacks = append(callbacks, fn)
	}
	chartHoverObserver.Unlock()
	for _, fn := range callbacks {
		fn(event)
	}
}

var chartHoverCallbackPtr = purego.NewCallback(chartHoverCallbackTrampoline)

func ensureHoverLibFuncs() {
	hoverLibOnce.Do(func() {
		if libHandle != 0 && tryRegisterLibFunc(&_CHSetHoverCallback, libHandle, "CHSetHoverCallback") {
			_CHSetHoverCallback(chartHoverCallbackPtr)
		}
		setHoverUnavailableStubs()
	})
}

func setHoverUnavailableStubs() {
	if _CHSetHoverCallback == nil {
		_CHSetHoverCallback = func(uintptr) {}
	}
}
