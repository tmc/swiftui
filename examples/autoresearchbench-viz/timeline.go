//go:build darwin

package main

import (
	"time"

	"github.com/tmc/swiftui"
	swcharts "github.com/tmc/swiftui/charts"
)

const (
	timelineUnit       = swcharts.TimeUnitMinute
	laneBandHeight     = 18.0
	visibleSeconds     = 3600.0
	maxEventGlyphPoint = 2000
)

var agentPalette = []swiftui.Color{
	swiftui.RGB(0.19, 0.43, 0.92),
	swiftui.RGB(0.93, 0.57, 0.19),
	swiftui.RGB(0.57, 0.44, 0.84),
	swiftui.RGB(0.16, 0.64, 0.73),
	swiftui.RGB(0.84, 0.33, 0.50),
	swiftui.RGB(0.17, 0.70, 0.42),
	swiftui.RGB(0.56, 0.62, 0.18),
}

func symbolForKind(kind string) swcharts.SymbolKind {
	switch kind {
	case "result":
		return swcharts.SymbolCircle
	case "insight":
		return swcharts.SymbolTriangle
	case "claim":
		return swcharts.SymbolDiamond
	case "best":
		return swcharts.SymbolPentagon
	case "baseline":
		return swcharts.SymbolSquare
	default:
		return swcharts.SymbolAsterisk
	}
}

func outcomeFillAlpha(outcome string) float64 {
	switch outcome {
	case "success", "keep":
		return 0.95
	case "failure", "crash":
		return 0.8
	case "discard":
		return 0.45
	default:
		return 0.7
	}
}

func timelineChart(events []Event, agents []string) swiftui.View {
	marks := make([]swcharts.Mark, 0, len(events)+len(agents))

	bandStart, bandEnd := timeBounds(events)
	bandStart = bandStart.Add(-5 * time.Minute)
	bandEnd = bandEnd.Add(5 * time.Minute)

	for _, agent := range agents {
		marks = append(marks, swcharts.RangeBarMark(
			swcharts.XDateRange("Time", bandStart, bandEnd, timelineUnit),
			swcharts.YString("Agent", agent),
			swcharts.HeightFixed(laneBandHeight),
		).
			ForegroundStyleBy("Agent", agent).
			Opacity(0.10).
			ExcludeFromLegend().
			ZIndex(1),
		)
	}

	step := 1
	if len(events) > maxEventGlyphPoint {
		step = (len(events) + maxEventGlyphPoint - 1) / maxEventGlyphPoint
	}

	for i, ev := range events {
		if i%step != 0 {
			continue
		}
		marks = append(marks, swcharts.PointMark(
			swcharts.XDate("Time", ev.T, timelineUnit),
			swcharts.YString("Agent", ev.Agent),
		).
			ForegroundStyleBy("Agent", ev.Agent).
			Symbol(symbolForKind(ev.Kind)).
			SymbolDiameter(6.5).
			Opacity(outcomeFillAlpha(ev.Outcome)).
			ZIndex(3),
		)
	}

	styles := make([]swcharts.StyleScaleEntry, 0, len(agents))
	for i, agent := range agents {
		styles = append(styles, swcharts.StyleScale(agent, agentPalette[i%len(agentPalette)]))
	}

	chart := swcharts.Chart(marks...).
		ChartForegroundStyleScale(styles...).
		ChartLegend(swcharts.LegendVisible(
			swcharts.LegendPositionTop,
			swcharts.LegendAlignmentLeading,
			8,
		)).
		ChartScrollableAxes(swcharts.ScrollAxisHorizontal).
		ChartXVisibleDomainLength(visibleSeconds).
		ChartScrollPositionInitialXDate(bandEnd.Add(-time.Duration(visibleSeconds) * time.Second)).
		ChartXAxis(swcharts.AxisMarks(
			swcharts.AxisValuesOption(swcharts.TimeStride(swcharts.TimeUnitHour)),
			swcharts.AxisGridLines(),
			swcharts.AxisLabels(),
		)).
		ChartYAxis(swcharts.AxisMarks(
			swcharts.AxisGridLinesStyled(swcharts.Stroke(1), false),
			swcharts.AxisLabels(),
		)).
		ChartPlotStyle(
			swcharts.PlotBackgroundColor(swiftui.RGB(1, 1, 1)),
			swcharts.PlotBorder(swiftui.RGBA(0.12, 0.18, 0.22, 0.08), 1),
		)
	return chart.View()
}
