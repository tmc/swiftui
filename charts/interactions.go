package charts

import (
	"time"

	"github.com/tmc/swiftui"
)

type SelectionValueBinding interface {
	stateBinding
	isSelectionValueBinding()
}

type SelectionRangeBinding interface {
	stateBinding
	isSelectionRangeBinding()
}

type ScrollPositionBinding interface {
	stateBinding
	isScrollPositionBinding()
}

func (*OptionalNumberState) isSelectionValueBinding() {}
func (*OptionalDateState) isSelectionValueBinding()   {}
func (*NumberRangeState) isSelectionRangeBinding()    {}
func (*DateRangeState) isSelectionRangeBinding()      {}
func (*NumberState) isScrollPositionBinding()         {}
func (*DateState) isScrollPositionBinding()           {}

type stateRefSpec struct {
	Kind int32  `json:"kind"`
	Ptr  string `json:"ptr"`
}

func stateSpec(binding stateBinding) *stateRefSpec {
	if binding == nil || binding.statePointer() == 0 {
		return nil
	}
	return &stateRefSpec{
		Kind: binding.stateKind(),
		Ptr:  pointerString(binding.statePointer()),
	}
}

type visibleDomainLength struct {
	kind   int32
	number float64
}

type visibleDomainLengthSpec struct {
	Kind   int32   `json:"kind"`
	Number float64 `json:"number"`
}

const (
	visibleDomainNumeric int32 = iota
	visibleDomainTime
)

// VisibleDomainNumeric uses a numeric visible-domain length.
func VisibleDomainNumeric(length float64) visibleDomainLength {
	return visibleDomainLength{kind: visibleDomainNumeric, number: length}
}

// VisibleDomainTime uses a time-domain visible length.
func VisibleDomainTime(length time.Duration) visibleDomainLength {
	return visibleDomainLength{kind: visibleDomainTime, number: length.Seconds()}
}

func (v visibleDomainLength) toSpec() visibleDomainLengthSpec {
	return visibleDomainLengthSpec{Kind: v.kind, Number: v.number}
}

type ValueAlignedLimitBehavior int32

const (
	ValueAlignedLimitAutomatic ValueAlignedLimitBehavior = iota
	ValueAlignedLimitAlways
	ValueAlignedLimitNever
)

type majorValueAlignmentKind int32

const (
	majorValueAlignmentNone majorValueAlignmentKind = iota
	majorValueAlignmentUnit
	majorValueAlignmentMatching
	majorValueAlignmentPage
)

// DateComponents is a narrow bridge for date-aligned scrolling.
type DateComponents struct {
	Year   int
	Month  int
	Day    int
	Hour   int
	Minute int
	Second int
}

// MatchTimeUnit returns date components for common chart time strides.
func MatchTimeUnit(unit TimeUnit, count int) DateComponents {
	if count < 1 {
		count = 1
	}
	switch unit {
	case TimeUnitYear:
		return DateComponents{Year: count}
	case TimeUnitMonth:
		return DateComponents{Month: count}
	case TimeUnitWeek:
		return DateComponents{Day: 7 * count}
	case TimeUnitHour:
		return DateComponents{Hour: count}
	case TimeUnitMinute:
		return DateComponents{Minute: count}
	default:
		return DateComponents{Day: count}
	}
}

type dateComponentsSpec struct {
	Year   int `json:"year"`
	Month  int `json:"month"`
	Day    int `json:"day"`
	Hour   int `json:"hour"`
	Minute int `json:"minute"`
	Second int `json:"second"`
}

func (d DateComponents) toSpec() dateComponentsSpec {
	return dateComponentsSpec{
		Year:   d.Year,
		Month:  d.Month,
		Day:    d.Day,
		Hour:   d.Hour,
		Minute: d.Minute,
		Second: d.Second,
	}
}

// MajorValueAlignment controls how value-aligned scrolling snaps to larger boundaries.
type MajorValueAlignment struct {
	kind       majorValueAlignmentKind
	numberUnit float64
	dateUnit   DateComponents
}

var MajorValueAlignmentPage = MajorValueAlignment{kind: majorValueAlignmentPage}

func MajorValueAlignmentUnit(unit float64) MajorValueAlignment {
	return MajorValueAlignment{kind: majorValueAlignmentUnit, numberUnit: unit}
}

func MajorValueAlignmentMatching(components DateComponents) MajorValueAlignment {
	return MajorValueAlignment{kind: majorValueAlignmentMatching, dateUnit: components}
}

type majorValueAlignmentSpec struct {
	Kind       int32              `json:"kind"`
	NumberUnit float64            `json:"numberUnit"`
	DateUnit   dateComponentsSpec `json:"dateUnit"`
}

func (m MajorValueAlignment) toSpec() majorValueAlignmentSpec {
	return majorValueAlignmentSpec{
		Kind:       int32(m.kind),
		NumberUnit: m.numberUnit,
		DateUnit:   m.dateUnit.toSpec(),
	}
}

type scrollTargetBehaviorKind int32

const (
	scrollTargetBehaviorPaging scrollTargetBehaviorKind = iota
	scrollTargetBehaviorValueAlignedUnit
	scrollTargetBehaviorValueAlignedDate
	scrollTargetBehaviorValueAlignedXYUnit
	scrollTargetBehaviorValueAlignedXYDate
)

// ScrollTargetBehavior configures snapping for scrollable charts.
type ScrollTargetBehavior struct {
	kind scrollTargetBehaviorKind

	xUnit  float64
	yUnit  float64
	xDate  DateComponents
	yDate  DateComponents
	xMajor *MajorValueAlignment
	yMajor *MajorValueAlignment
	limit  ValueAlignedLimitBehavior
}

// PagingScrollTarget snaps by pages.
var PagingScrollTarget = ScrollTargetBehavior{kind: scrollTargetBehaviorPaging}

// ValueAlignedScrollTarget aligns scrolling to a numeric unit.
func ValueAlignedScrollTarget(unit float64, major *MajorValueAlignment, limit ValueAlignedLimitBehavior) ScrollTargetBehavior {
	return ScrollTargetBehavior{
		kind:   scrollTargetBehaviorValueAlignedUnit,
		xUnit:  unit,
		xMajor: major,
		limit:  limit,
	}
}

// DateAlignedScrollTarget aligns scrolling to matching date components.
func DateAlignedScrollTarget(components DateComponents, major *MajorValueAlignment, limit ValueAlignedLimitBehavior) ScrollTargetBehavior {
	return ScrollTargetBehavior{
		kind:   scrollTargetBehaviorValueAlignedDate,
		xDate:  components,
		xMajor: major,
		limit:  limit,
	}
}

// ValueAlignedXYScrollTarget aligns both axes to numeric units.
func ValueAlignedXYScrollTarget(xUnit, yUnit float64, xMajor, yMajor *MajorValueAlignment, limit ValueAlignedLimitBehavior) ScrollTargetBehavior {
	return ScrollTargetBehavior{
		kind:   scrollTargetBehaviorValueAlignedXYUnit,
		xUnit:  xUnit,
		yUnit:  yUnit,
		xMajor: xMajor,
		yMajor: yMajor,
		limit:  limit,
	}
}

// DateAlignedXYScrollTarget aligns both axes to date components.
func DateAlignedXYScrollTarget(xDate, yDate DateComponents, xMajor, yMajor *MajorValueAlignment, limit ValueAlignedLimitBehavior) ScrollTargetBehavior {
	return ScrollTargetBehavior{
		kind:   scrollTargetBehaviorValueAlignedXYDate,
		xDate:  xDate,
		yDate:  yDate,
		xMajor: xMajor,
		yMajor: yMajor,
		limit:  limit,
	}
}

type scrollTargetBehaviorSpec struct {
	Kind   int32                    `json:"kind"`
	XUnit  float64                  `json:"xUnit"`
	YUnit  float64                  `json:"yUnit"`
	XDate  dateComponentsSpec       `json:"xDate"`
	YDate  dateComponentsSpec       `json:"yDate"`
	XMajor *majorValueAlignmentSpec `json:"xMajor,omitempty"`
	YMajor *majorValueAlignmentSpec `json:"yMajor,omitempty"`
	Limit  int32                    `json:"limit"`
}

func (b ScrollTargetBehavior) toSpec() scrollTargetBehaviorSpec {
	spec := scrollTargetBehaviorSpec{
		Kind:  int32(b.kind),
		XUnit: b.xUnit,
		YUnit: b.yUnit,
		XDate: b.xDate.toSpec(),
		YDate: b.yDate.toSpec(),
		Limit: int32(b.limit),
	}
	if b.xMajor != nil {
		m := b.xMajor.toSpec()
		spec.XMajor = &m
	}
	if b.yMajor != nil {
		m := b.yMajor.toSpec()
		spec.YMajor = &m
	}
	return spec
}

type proxyLayerKind int32

const (
	proxyLayerCrosshair proxyLayerKind = iota
	proxyLayerReadout
	proxyLayerSelectionBandX
	proxyLayerSelectionBandY
)

type proxyLayer struct {
	kind       proxyLayerKind
	alignment  LabelAlignment
	xState     stateBinding
	yState     stateBinding
	rangeState stateBinding
	color      swiftui.Color
	width      float64
	xFormat    ValueFormat
	yFormat    ValueFormat
}

type proxyLayerSpec struct {
	Kind      int32           `json:"kind"`
	Alignment int32           `json:"alignment"`
	XState    *stateRefSpec   `json:"xState,omitempty"`
	YState    *stateRefSpec   `json:"yState,omitempty"`
	Range     *stateRefSpec   `json:"range,omitempty"`
	ColorR    float64         `json:"colorR"`
	ColorG    float64         `json:"colorG"`
	ColorB    float64         `json:"colorB"`
	ColorA    float64         `json:"colorA"`
	Width     float64         `json:"width"`
	XFormat   valueFormatSpec `json:"xFormat"`
	YFormat   valueFormatSpec `json:"yFormat"`
}

func (l proxyLayer) toSpec() proxyLayerSpec {
	r, g, b, a := colorParts(l.color)
	return proxyLayerSpec{
		Kind:      int32(l.kind),
		Alignment: int32(l.alignment),
		XState:    stateSpec(l.xState),
		YState:    stateSpec(l.yState),
		Range:     stateSpec(l.rangeState),
		ColorR:    r,
		ColorG:    g,
		ColorB:    b,
		ColorA:    a,
		Width:     l.width,
		XFormat:   l.xFormat.toSpec(),
		YFormat:   l.yFormat.toSpec(),
	}
}

// ProxyOverlay is a practical chart overlay driven by ChartProxy data.
type ProxyOverlay struct{ layer proxyLayer }

// ProxyBackground is a practical chart background driven by ChartProxy data.
type ProxyBackground struct{ layer proxyLayer }

// CrosshairOverlay draws x/y crosshair guides from selection state.
func CrosshairOverlay(x SelectionValueBinding, y SelectionValueBinding, color swiftui.Color, width float64) ProxyOverlay {
	return ProxyOverlay{layer: proxyLayer{
		kind:   proxyLayerCrosshair,
		xState: x,
		yState: y,
		color:  color,
		width:  width,
	}}
}

// ReadoutOverlay shows a compact hover/readout box from selection state.
func ReadoutOverlay(x SelectionValueBinding, y SelectionValueBinding, alignment LabelAlignment, xFormat, yFormat ValueFormat) ProxyOverlay {
	return ProxyOverlay{layer: proxyLayer{
		kind:      proxyLayerReadout,
		xState:    x,
		yState:    y,
		alignment: alignment,
		xFormat:   xFormat,
		yFormat:   yFormat,
	}}
}

// SelectionBandBackgroundX shows the selected x-range behind the plot.
func SelectionBandBackgroundX(r SelectionRangeBinding, color swiftui.Color) ProxyBackground {
	return ProxyBackground{layer: proxyLayer{
		kind:       proxyLayerSelectionBandX,
		rangeState: r,
		color:      color,
	}}
}

// SelectionBandBackgroundY shows the selected y-range behind the plot.
func SelectionBandBackgroundY(r SelectionRangeBinding, color swiftui.Color) ProxyBackground {
	return ProxyBackground{layer: proxyLayer{
		kind:       proxyLayerSelectionBandY,
		rangeState: r,
		color:      color,
	}}
}

type proxyGestureKind int32

const (
	proxyGestureDragXValue proxyGestureKind = iota
	proxyGestureDragXRange
	proxyGestureDragYValue
	proxyGestureDragYRange
	proxyGestureDragAngleValue
)

// ProxyGesture is a practical drag-selection gesture driven by ChartProxy.
type ProxyGesture struct {
	kind        proxyGestureKind
	state       stateBinding
	minDistance float64
}

type proxyGestureSpec struct {
	Kind        int32         `json:"kind"`
	State       *stateRefSpec `json:"state,omitempty"`
	MinDistance float64       `json:"minDistance"`
}

func (g ProxyGesture) toSpec() proxyGestureSpec {
	return proxyGestureSpec{
		Kind:        int32(g.kind),
		State:       stateSpec(g.state),
		MinDistance: g.minDistance,
	}
}

func DragXSelectionGesture(state SelectionValueBinding, minDistance float64) ProxyGesture {
	return ProxyGesture{kind: proxyGestureDragXValue, state: state, minDistance: minDistance}
}

func DragXRangeSelectionGesture(state SelectionRangeBinding, minDistance float64) ProxyGesture {
	return ProxyGesture{kind: proxyGestureDragXRange, state: state, minDistance: minDistance}
}

func DragYSelectionGesture(state SelectionValueBinding, minDistance float64) ProxyGesture {
	return ProxyGesture{kind: proxyGestureDragYValue, state: state, minDistance: minDistance}
}

func DragYRangeSelectionGesture(state SelectionRangeBinding, minDistance float64) ProxyGesture {
	return ProxyGesture{kind: proxyGestureDragYRange, state: state, minDistance: minDistance}
}

func DragAngleSelectionGesture(state SelectionValueBinding, minDistance float64) ProxyGesture {
	return ProxyGesture{kind: proxyGestureDragAngleValue, state: state, minDistance: minDistance}
}

func (c ChartView) ChartXSelection(binding SelectionValueBinding) ChartView {
	return c.with(func(b *chartBuilder) { b.xSelection = binding })
}

func (c ChartView) ChartXSelectionRange(binding SelectionRangeBinding) ChartView {
	return c.with(func(b *chartBuilder) { b.xSelectionRange = binding })
}

func (c ChartView) ChartYSelection(binding SelectionValueBinding) ChartView {
	return c.with(func(b *chartBuilder) { b.ySelection = binding })
}

func (c ChartView) ChartYSelectionRange(binding SelectionRangeBinding) ChartView {
	return c.with(func(b *chartBuilder) { b.ySelectionRange = binding })
}

func (c ChartView) ChartAngleSelection(binding SelectionValueBinding) ChartView {
	return c.with(func(b *chartBuilder) { b.angleSelection = binding })
}

func (c ChartView) ChartScrollTargetBehavior(behavior ScrollTargetBehavior) ChartView {
	return c.with(func(b *chartBuilder) {
		value := behavior
		b.scrollTarget = &value
	})
}

func (c ChartView) ChartScrollPositionX(binding ScrollPositionBinding) ChartView {
	return c.with(func(b *chartBuilder) { b.xScrollBinding = binding })
}

func (c ChartView) ChartScrollPositionY(binding ScrollPositionBinding) ChartView {
	return c.with(func(b *chartBuilder) { b.yScrollBinding = binding })
}

func (c ChartView) ChartXVisibleDomain(length visibleDomainLength) ChartView {
	return c.with(func(b *chartBuilder) {
		value := length
		b.xVisibleLength = &value
	})
}

func (c ChartView) ChartYVisibleDomain(length visibleDomainLength) ChartView {
	return c.with(func(b *chartBuilder) {
		value := length
		b.yVisibleLength = &value
	})
}

func (c ChartView) ChartOverlay(overlay ProxyOverlay) ChartView {
	return c.with(func(b *chartBuilder) {
		b.overlays = append(append([]proxyLayer(nil), b.overlays...), overlay.layer)
	})
}

func (c ChartView) ChartBackground(background ProxyBackground) ChartView {
	return c.with(func(b *chartBuilder) {
		b.backgrounds = append(append([]proxyLayer(nil), b.backgrounds...), background.layer)
	})
}

func (c ChartView) ChartGesture(gesture ProxyGesture) ChartView {
	return c.with(func(b *chartBuilder) {
		value := gesture
		b.gesture = &value
	})
}
