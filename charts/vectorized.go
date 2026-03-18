package charts

type markDimensionKind int32

const (
	markDimensionAutomatic markDimensionKind = iota
	markDimensionFixed
	markDimensionRatio
	markDimensionInset
)

// MarkDimension configures vectorized rectangle sizing.
type MarkDimension struct {
	kind  markDimensionKind
	value float64
}

// MarkDimensionAutomatic uses the framework default sizing.
var MarkDimensionAutomatic = MarkDimension{kind: markDimensionAutomatic}

// MarkDimensionFixed uses a fixed size in points.
func MarkDimensionFixed(points float64) MarkDimension {
	return MarkDimension{kind: markDimensionFixed, value: points}
}

// MarkDimensionRatio uses a slot-relative size ratio.
func MarkDimensionRatio(ratio float64) MarkDimension {
	return MarkDimension{kind: markDimensionRatio, value: ratio}
}

// MarkDimensionInset uses an inset relative to the slot.
func MarkDimensionInset(inset float64) MarkDimension {
	return MarkDimension{kind: markDimensionInset, value: inset}
}

// PointPlotDatum is one point in a vectorized point plot.
type PointPlotDatum struct {
	X Value
	Y Value
}

// PlotPoint creates a vectorized point plot datum.
func PlotPoint(x, y Value) PointPlotDatum {
	return PointPlotDatum{X: x, Y: y}
}

// RectanglePlotDatum is one rectangle in a vectorized rectangle plot.
type RectanglePlotDatum struct {
	X      *Value
	Y      *Value
	XStart *Value
	XEnd   *Value
	YStart *Value
	YEnd   *Value
	Width  MarkDimension
	Height MarkDimension
}

// PlotRectangleXY creates a centered rectangle datum.
func PlotRectangleXY(x, y Value, width, height MarkDimension) RectanglePlotDatum {
	return RectanglePlotDatum{
		X:      &x,
		Y:      &y,
		Width:  width,
		Height: height,
	}
}

// PlotRectangleXRange creates a ranged-x rectangle datum.
func PlotRectangleXRange(xStart, xEnd, y Value, height MarkDimension) RectanglePlotDatum {
	return RectanglePlotDatum{
		XStart: &xStart,
		XEnd:   &xEnd,
		Y:      &y,
		Height: height,
	}
}

// PlotRectangleYRange creates a ranged-y rectangle datum.
func PlotRectangleYRange(x, yStart, yEnd Value, width MarkDimension) RectanglePlotDatum {
	return RectanglePlotDatum{
		X:      &x,
		YStart: &yStart,
		YEnd:   &yEnd,
		Width:  width,
	}
}

// PlotRectangleRange creates a fully-ranged rectangle datum.
func PlotRectangleRange(xStart, xEnd, yStart, yEnd Value) RectanglePlotDatum {
	return RectanglePlotDatum{
		XStart: &xStart,
		XEnd:   &xEnd,
		YStart: &yStart,
		YEnd:   &yEnd,
	}
}

// RulePlotDatum is one rule in a vectorized rule plot.
type RulePlotDatum struct {
	X      *Value
	Y      *Value
	XStart *Value
	XEnd   *Value
	YStart *Value
	YEnd   *Value
}

// PlotRuleXRange creates an x-ranged rule datum.
func PlotRuleXRange(xStart, xEnd, y Value) RulePlotDatum {
	return RulePlotDatum{XStart: &xStart, XEnd: &xEnd, Y: &y}
}

// PlotRuleYRange creates a y-ranged rule datum.
func PlotRuleYRange(x, yStart, yEnd Value) RulePlotDatum {
	return RulePlotDatum{X: &x, YStart: &yStart, YEnd: &yEnd}
}

// PointPlot creates a vectorized point plot.
func PointPlot(data ...PointPlotDatum) Mark {
	return Mark{
		kind:      markPointPlot,
		pointData: append([]PointPlotDatum(nil), data...),
	}
}

// RectanglePlot creates a vectorized rectangle plot.
func RectanglePlot(data ...RectanglePlotDatum) Mark {
	return Mark{
		kind:          markRectanglePlot,
		rectangleData: append([]RectanglePlotDatum(nil), data...),
	}
}

// RulePlot creates a vectorized rule plot.
func RulePlot(data ...RulePlotDatum) Mark {
	return Mark{
		kind:     markRulePlot,
		ruleData: append([]RulePlotDatum(nil), data...),
	}
}

// MapPointPlot maps a Go slice into a vectorized point plot.
func MapPointPlot[T any](data []T, x func(T) Value, y func(T) Value) Mark {
	points := make([]PointPlotDatum, len(data))
	for i, item := range data {
		points[i] = PlotPoint(x(item), y(item))
	}
	return PointPlot(points...)
}

// MapRectanglePlot maps a Go slice into a vectorized centered rectangle plot.
func MapRectanglePlot[T any](data []T, x func(T) Value, y func(T) Value, width, height MarkDimension) Mark {
	rects := make([]RectanglePlotDatum, len(data))
	for i, item := range data {
		rects[i] = PlotRectangleXY(x(item), y(item), width, height)
	}
	return RectanglePlot(rects...)
}

// MapRulePlot maps a Go slice into a vectorized ranged rule plot.
func MapRulePlot[T any](data []T, xStart func(T) Value, xEnd func(T) Value, y func(T) Value) Mark {
	rules := make([]RulePlotDatum, len(data))
	for i, item := range data {
		rules[i] = PlotRuleXRange(xStart(item), xEnd(item), y(item))
	}
	return RulePlot(rules...)
}
