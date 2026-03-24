package charts

import "time"

type ChartHoverValueKind int32

const (
	ChartHoverValueNone ChartHoverValueKind = iota
	ChartHoverValueNumber
	ChartHoverValueDate
)

// ChartHoverValue is a ChartProxy lookup result for a hover location.
type ChartHoverValue struct {
	kind         ChartHoverValueKind
	number       float64
	epochSeconds float64
}

// Kind reports the underlying value kind.
func (v ChartHoverValue) Kind() ChartHoverValueKind { return v.kind }

// Number returns the numeric value when the proxy lookup resolved to a number.
func (v ChartHoverValue) Number() (float64, bool) {
	if v.kind != ChartHoverValueNumber {
		return 0, false
	}
	return v.number, true
}

// Date returns the date value when the proxy lookup resolved to a date.
func (v ChartHoverValue) Date() (time.Time, bool) {
	if v.kind != ChartHoverValueDate {
		return time.Time{}, false
	}
	return time.UnixMilli(int64(v.epochSeconds * 1000)), true
}

// ChartHoverEvent describes a plot-local hover update sourced from ChartProxy.
type ChartHoverEvent struct {
	Active      bool
	PlotX       float64
	PlotY       float64
	FrameMinX   float64
	FrameMinY   float64
	FrameWidth  float64
	FrameHeight float64
	XValue      ChartHoverValue
	YValue      ChartHoverValue
}

// ChartHoverState mirrors chart-local hover updates and allows observing them from Go.
type ChartHoverState struct {
	id uintptr
}

func NewChartHoverState() *ChartHoverState {
	ensureHoverLibFuncs()
	return &ChartHoverState{id: registerChartHoverState()}
}

func (s *ChartHoverState) Get() ChartHoverEvent {
	if s == nil || s.id == 0 {
		return ChartHoverEvent{}
	}
	return loadChartHoverState(s.id)
}

// OnChange registers a callback for hover updates.
// The returned function removes the observer.
func (s *ChartHoverState) OnChange(fn func(ChartHoverEvent)) func() {
	if s == nil || s.id == 0 || fn == nil {
		return noopCancel
	}
	return registerChartHoverObserver(s.id, fn)
}

func (s *ChartHoverState) Release() {
	if s == nil || s.id == 0 {
		return
	}
	clearChartHoverObservers(s.id)
	unregisterChartHoverState(s.id)
	s.id = 0
}

func (s *ChartHoverState) callbackID() uint64 {
	if s == nil {
		return 0
	}
	return uint64(s.id)
}

func chartHoverValueFromBridge(kind int32, number, epochSeconds float64) ChartHoverValue {
	return ChartHoverValue{
		kind:         ChartHoverValueKind(kind),
		number:       number,
		epochSeconds: epochSeconds,
	}
}
