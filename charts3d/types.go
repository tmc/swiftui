package charts3d

import (
	"strconv"
	"time"

	"github.com/tmc/swiftui"
)

type TimeUnit int32

const (
	TimeUnitDay TimeUnit = iota
	TimeUnitWeek
	TimeUnitMonth
	TimeUnitYear
	TimeUnitHour
	TimeUnitMinute
)

type valueKind int32

const (
	valueInteger valueKind = iota
	valueNumber
	valueTime
)

type Value struct {
	kind valueKind

	label     string
	integer   int
	number    float64
	timestamp time.Time
	timeUnit  TimeUnit
}

func IntegerValue(label string, value int) Value {
	return Value{kind: valueInteger, label: label, integer: value}
}

func NumberValue(label string, value float64) Value {
	return Value{kind: valueNumber, label: label, number: value}
}

func TimeValue(label string, value time.Time, unit TimeUnit) Value {
	return Value{kind: valueTime, label: label, timestamp: value, timeUnit: unit}
}

type dimRole int32

const (
	dimX dimRole = iota
	dimY
	dimZ
)

type Dim struct {
	role  dimRole
	value Value
}

func X(v Value) Dim { return Dim{role: dimX, value: v} }
func Y(v Value) Dim { return Dim{role: dimY, value: v} }
func Z(v Value) Dim { return Dim{role: dimZ, value: v} }

func XInt(label string, value int) Dim                       { return X(IntegerValue(label, value)) }
func XFloat(label string, value float64) Dim                 { return X(NumberValue(label, value)) }
func XDate(label string, value time.Time, unit TimeUnit) Dim { return X(TimeValue(label, value, unit)) }
func YInt(label string, value int) Dim                       { return Y(IntegerValue(label, value)) }
func YFloat(label string, value float64) Dim                 { return Y(NumberValue(label, value)) }
func ZInt(label string, value int) Dim                       { return Z(IntegerValue(label, value)) }
func ZFloat(label string, value float64) Dim                 { return Z(NumberValue(label, value)) }

type domainKind int32

const (
	domainInteger domainKind = iota
	domainNumber
	domainTime
)

type Domain struct {
	kind domainKind

	minInt, maxInt       int
	minNumber, maxNumber float64
	startTime, endTime   time.Time
}

func IntegerDomain(min, max int) Domain {
	return Domain{kind: domainInteger, minInt: min, maxInt: max}
}

func NumberDomain(min, max float64) Domain {
	return Domain{kind: domainNumber, minNumber: min, maxNumber: max}
}

func TimeDomain(start, end time.Time) Domain {
	return Domain{kind: domainTime, startTime: start, endTime: end}
}

type scaleTypeKind int32

const (
	scaleTypeAutomatic scaleTypeKind = iota
	scaleTypeLinear
	scaleTypeLog
	scaleTypeDate
	scaleTypePower
	scaleTypeSquareRoot
	scaleTypeSymmetricLog
)

type ScaleType struct {
	kind  scaleTypeKind
	value float64
}

var (
	ScaleTypeAutomatic  = ScaleType{kind: scaleTypeAutomatic}
	ScaleTypeLinear     = ScaleType{kind: scaleTypeLinear}
	ScaleTypeLog        = ScaleType{kind: scaleTypeLog}
	ScaleTypeDate       = ScaleType{kind: scaleTypeDate}
	ScaleTypeSquareRoot = ScaleType{kind: scaleTypeSquareRoot}
)

func ScaleTypePower(exponent float64) ScaleType {
	return ScaleType{kind: scaleTypePower, value: exponent}
}

func ScaleTypeSymmetricLog(slopeAtZero float64) ScaleType {
	return ScaleType{kind: scaleTypeSymmetricLog, value: slopeAtZero}
}

type AnnotationPosition int32

const (
	AnnotationTop AnnotationPosition = iota
	AnnotationBottom
	AnnotationLeading
	AnnotationTrailing
	AnnotationOverlay
	AnnotationTopLeading
	AnnotationTopTrailing
	AnnotationBottomLeading
	AnnotationBottomTrailing
	AnnotationAutomatic
)

type LabelAlignment int32

const (
	LabelAlignmentCenter LabelAlignment = iota
	LabelAlignmentLeading
	LabelAlignmentTrailing
	LabelAlignmentTop
	LabelAlignmentBottom
	LabelAlignmentTopLeading
	LabelAlignmentTopTrailing
	LabelAlignmentBottomLeading
	LabelAlignmentBottomTrailing
)

type axisLabel struct {
	text         string
	position     AnnotationPosition
	alignment    LabelAlignment
	hasPosition  bool
	hasAlignment bool
	hasSpacing   bool
	spacing      float64
}

type AxisLabelOption struct {
	kind axisLabelOptionKind

	position  AnnotationPosition
	alignment LabelAlignment
	spacing   float64
}

type axisLabelOptionKind int32

const (
	axisLabelOptionPosition axisLabelOptionKind = iota
	axisLabelOptionAlignment
	axisLabelOptionSpacing
)

func AxisLabelPosition(position AnnotationPosition) AxisLabelOption {
	return AxisLabelOption{kind: axisLabelOptionPosition, position: position}
}

func AxisLabelAlignment(alignment LabelAlignment) AxisLabelOption {
	return AxisLabelOption{kind: axisLabelOptionAlignment, alignment: alignment}
}

func AxisLabelSpacing(spacing float64) AxisLabelOption {
	return AxisLabelOption{kind: axisLabelOptionSpacing, spacing: spacing}
}

type markKind int32

const (
	markPoint markKind = iota
	markRectangle
	markRule
)

type markModKind int32

const (
	modForegroundStyle markModKind = iota
)

type markMod struct {
	kind  markModKind
	color swiftui.Color
}

type Mark struct {
	kind markKind
	dims []Dim
	mods []markMod
}

func PointMark(dims ...Dim) Mark {
	return Mark{kind: markPoint, dims: append([]Dim(nil), dims...)}
}

func RectangleMark(dims ...Dim) Mark {
	return Mark{kind: markRectangle, dims: append([]Dim(nil), dims...)}
}

func RuleMark(dims ...Dim) Mark {
	return Mark{kind: markRule, dims: append([]Dim(nil), dims...)}
}

func (m Mark) ForegroundStyle(color swiftui.Color) Mark {
	m.mods = append(append([]markMod(nil), m.mods...), markMod{kind: modForegroundStyle, color: color})
	return m
}

type valueSpec struct {
	Kind       int32   `json:"kind"`
	Label      string  `json:"label"`
	Integer    int     `json:"integer"`
	Number     float64 `json:"number"`
	TimeUnixMS int64   `json:"timeUnixMS"`
	TimeUnit   int32   `json:"timeUnit"`
}

type dimSpec struct {
	Role  int32     `json:"role"`
	Value valueSpec `json:"value"`
}

type markModSpec struct {
	Kind   int32   `json:"kind"`
	ColorR float64 `json:"colorR"`
	ColorG float64 `json:"colorG"`
	ColorB float64 `json:"colorB"`
	ColorA float64 `json:"colorA"`
}

type markSpec struct {
	Kind int32         `json:"kind"`
	Dims []dimSpec     `json:"dims"`
	Mods []markModSpec `json:"mods,omitempty"`
}

type domainSpec struct {
	Kind        int32   `json:"kind"`
	MinInt      int     `json:"minInt"`
	MaxInt      int     `json:"maxInt"`
	MinNumber   float64 `json:"minNumber"`
	MaxNumber   float64 `json:"maxNumber"`
	StartUnixMS int64   `json:"startUnixMS"`
	EndUnixMS   int64   `json:"endUnixMS"`
}

type scaleTypeSpec struct {
	Kind  int32   `json:"kind"`
	Value float64 `json:"value"`
}

type axisLabelSpec struct {
	Text         string  `json:"text"`
	Position     int32   `json:"position"`
	Alignment    int32   `json:"alignment"`
	HasPosition  bool    `json:"hasPosition"`
	HasAlignment bool    `json:"hasAlignment"`
	HasSpacing   bool    `json:"hasSpacing"`
	Spacing      float64 `json:"spacing"`
}

func pointerString(ptr uintptr) string {
	if ptr == 0 {
		return ""
	}
	return strconv.FormatUint(uint64(ptr), 10)
}

func colorParts(color swiftui.Color) (float64, float64, float64, float64) {
	return color.R, color.G, color.B, color.A
}

func (v Value) toSpec() valueSpec {
	return valueSpec{
		Kind:       int32(v.kind),
		Label:      v.label,
		Integer:    v.integer,
		Number:     v.number,
		TimeUnixMS: v.timestamp.UnixMilli(),
		TimeUnit:   int32(v.timeUnit),
	}
}

func (d Dim) toSpec() dimSpec {
	return dimSpec{Role: int32(d.role), Value: d.value.toSpec()}
}

func (m markMod) toSpec() markModSpec {
	r, g, b, a := colorParts(m.color)
	return markModSpec{
		Kind:   int32(m.kind),
		ColorR: r,
		ColorG: g,
		ColorB: b,
		ColorA: a,
	}
}

func (m Mark) toSpec() markSpec {
	spec := markSpec{Kind: int32(m.kind)}
	if len(m.dims) > 0 {
		spec.Dims = make([]dimSpec, len(m.dims))
		for i, dim := range m.dims {
			spec.Dims[i] = dim.toSpec()
		}
	}
	if len(m.mods) > 0 {
		spec.Mods = make([]markModSpec, len(m.mods))
		for i, mod := range m.mods {
			spec.Mods[i] = mod.toSpec()
		}
	}
	return spec
}

func (d Domain) toSpec() domainSpec {
	return domainSpec{
		Kind:        int32(d.kind),
		MinInt:      d.minInt,
		MaxInt:      d.maxInt,
		MinNumber:   d.minNumber,
		MaxNumber:   d.maxNumber,
		StartUnixMS: d.startTime.UnixMilli(),
		EndUnixMS:   d.endTime.UnixMilli(),
	}
}

func (t ScaleType) toSpec() scaleTypeSpec {
	return scaleTypeSpec{Kind: int32(t.kind), Value: t.value}
}

func (label axisLabel) toSpec() axisLabelSpec {
	return axisLabelSpec{
		Text:         label.text,
		Position:     int32(label.position),
		Alignment:    int32(label.alignment),
		HasPosition:  label.hasPosition,
		HasAlignment: label.hasAlignment,
		HasSpacing:   label.hasSpacing,
		Spacing:      label.spacing,
	}
}
