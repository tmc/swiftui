//go:build darwin
// +build darwin

// Command charts showcases native Swift Charts bindings generated for Go.
//
// It focuses on practical ML, benchmarking, evaluation, and SRE examples
// using only the generated charts package and the existing swiftui bridge.
//
// Usage:
//
//	go run .
package main

import (
	"fmt"
	"math"
	"runtime"
	"time"

	"github.com/tmc/swiftui"
	"github.com/tmc/swiftui/charts"
)

func init() { runtime.LockOSThread() }

const (
	windowWidth  = 960
	windowHeight = 980
	chartWidth   = 760
	tallChart    = 240
	shortChart   = 168
)

var (
	muted  = swiftui.RGB(0.69, 0.73, 0.79)
	blue   = swiftui.RGB(0.23, 0.46, 0.94)
	cyan   = swiftui.RGB(0.11, 0.70, 0.84)
	teal   = swiftui.RGB(0.13, 0.63, 0.57)
	green  = swiftui.RGB(0.23, 0.71, 0.37)
	amber  = swiftui.RGB(0.92, 0.62, 0.21)
	orange = swiftui.RGB(0.87, 0.46, 0.21)
	red    = swiftui.RGB(0.86, 0.29, 0.28)
	plum   = swiftui.RGB(0.55, 0.40, 0.86)
)

type regionError struct {
	Region string
	Errors int
}

type lossPoint struct {
	At    time.Time
	Train float64
	Valid float64
}

type throughputPoint struct {
	Stage      string
	Throughput float64
	LossX100   float64
}

type benchCase struct {
	Name string
	Old  int
	New  int
}

type benchInterval struct {
	Name string
	Run  string
	Mean float64
	Low  float64
	High float64
}

type latencyPoint struct {
	At  time.Time
	P50 float64
	P95 float64
	P99 float64
}

type latencySpread struct {
	Service string
	Min     float64
	Q1      float64
	Median  float64
	Q3      float64
	Max     float64
}

type evalScore struct {
	Task  string
	Model string
	Score float64
}

type bucketCount struct {
	Label string
	Count int
}

type matrixCell struct {
	Model string
	Task  string
	X     float64
	Y     float64
	Score float64
}

type sectorShare struct {
	Label string
	Value float64
}

type backlogPoint struct {
	Load    int
	Backlog float64
}

func main() {
	section := swiftui.NewIntState(0)
	swiftui.Run(swiftui.AppConfig{
		Title:  "Charts Showcase",
		Width:  windowWidth,
		Height: windowHeight,
	}, swiftui.VStackSpaced(0,
		swiftui.HStack(
			swiftui.PickerSegmented("Section", section, segmentedOptions("Highlights", "Research", "Ops"), func() {}).
				MaxFrame(360, 0),
			swiftui.Spacer(),
		).
			Padding(16).
			BackgroundRoundedRect(0.10, 0.13, 0.18, 0.94, 18).
			Border(0.82, 0.86, 0.92, 0.08, 1),
		swiftui.DynamicView(section, func(v int) swiftui.View {
			switch v {
			case 1:
				return researchScreen()
			case 2:
				return opsScreen()
			default:
				return highlightsScreen()
			}
		}).MaxFrame(-1, -1),
	).
		Padding(14).
		BackgroundStyle("windowBackground"))
}

func segmentedOptions(labels ...string) swiftui.View {
	children := make([]swiftui.Viewable, 0, len(labels))
	for i, label := range labels {
		children = append(children, swiftui.Text(label).AsView().Tag(int32(i)))
	}
	return swiftui.VStack(children...)
}

func highlightsScreen() swiftui.View {
	return screen(
		"Highlights",
		"Chart-first native Swift Charts demos for ML training, benchmarking, evaluation, and SRE reviews.",
		[]string{"marks", "axes", "scales", "legends", "annotations", "scroll"},
		section(
			"Featured Views",
			"The strongest product-style examples sit first so the package reads like a real dashboard surface, not an API checklist.",
			featuredCard(
				"Training Loss Rollup",
				"Training and validation loss stay on one date-based review chart with an area envelope, scoped visible domain, and highlighted validation endpoint.",
				"AreaMark, LineMark, PointMark, date scales, interpolation, scroll position, annotations.",
				chartView(trainingLossChart(), tallChart),
			),
			compactCard(
				"Training Throughput And Deploy Gate",
				"Mixed marks compose throughput bars, smoothed loss, and a labeled threshold line on one review card.",
				"BarMark, LineMark, PointMark, RuleMark, interpolation, plot overlay hooks.",
				chartView(mixedTrainingChart(), tallChart),
			),
			compactCard(
				"Token Share By Workload",
				"Sector marks summarize how training, serving, and evaluation tokens are distributed across the stack.",
				"SectorMark, ForegroundStyleScale, AngularInset, CornerRadius, Annotation, legend placement.",
				chartView(sectorShareChart(), shortChart),
			),
		),
		section(
			"Foundation",
			"Compact cards keep the basics visible without crowding out the stronger mixed and time-series examples.",
			compactCard(
				"Regional Error Hotspots",
				"Grouped category/value bars show where service errors are accumulating, with explicit category order and compact axis formatting.",
				"BarMark, string categories, ForegroundStyleBy, CornerRadius, explicit axis values, CompactFormat.",
				chartView(serviceErrorsChart(), tallChart),
			),
			coverageBand(
				"Coverage In This Screen",
				"Core marks: bar, line, area, point, rule, sector, mixed composition.",
				"Plottables: strings, integers, floats, dates.",
				"Chart-wide features: legends, plot chrome, axis labels, scroll position, visible domain length.",
			),
		),
	)
}

func researchScreen() swiftui.View {
	return screen(
		"Research",
		"Benchmarking and evaluation views that feel at home in model-review and experiment-analysis workflows.",
		[]string{"intervals", "benchmarks", "evaluation", "matrix", "formatting", "thresholds"},
		section(
			"Benchmarking",
			"Explicit uncertainty and compact formatting make the research surface read like a benchstat review rather than a toy bar chart.",
			featuredCard(
				"Benchstat Comparison",
				"Grouped bars compare old and new runs, the interval view adds uncertainty bands, and the delta strip isolates regressions and wins.",
				"Grouped BarMark, PositionBy, ErrorBarMark, PointMark, symmetric-log.",
				swiftui.VStackSpaced(12,
					chartView(benchmarkBarsChart(), tallChart),
					chartView(benchmarkUncertaintyChart(), tallChart),
					chartView(benchmarkDeltaChart(), shortChart),
				),
			),
		),
		section(
			"Evaluation",
			"Grouped comparisons and matrix views show how the bindings hold up under common model-evaluation presentation patterns.",
			compactCard(
				"Model Evaluation Across Tasks",
				"Grouped bars compare models across benchmark tasks with styled legends, ordered axes, and intentional percent formatting.",
				"BarMark, ForegroundStyleScale, category scales, legend configuration, plot border, AxisValueLabels.",
				chartView(evaluationChart(), tallChart),
			),
			compactCard(
				"Model Score Matrix",
				"Rectangle marks render a heatmap-like score matrix while deterministic numeric axes keep the bridge native and predictable.",
				"RectangleMark, explicit numeric axes, Opacity, CornerRadius, Shadow, Annotation.",
				swiftui.VStackSpaced(10,
					chartView(heatmapChart(), 260),
					indexLegend(),
				),
			),
			compactCard(
				"Training Throughput And Deploy Gate",
				"Mixed marks compose throughput bars, smoothed loss, and a labeled threshold line on one review card.",
				"BarMark, LineMark, PointMark, RuleMark, interpolation, plot overlay hooks.",
				chartView(mixedTrainingChart(), tallChart),
			),
			coverageBand(
				"Coverage In This Screen",
				"Statistical review: grouped bars, error bars, grouped evaluations, matrix rectangles.",
				"Formatting: percent labels, suffix labels, legends, explicit axes.",
				"Scale coverage: category axes and symmetric-log response deltas.",
			),
		),
	)
}

func opsScreen() swiftui.View {
	return screen(
		"Operations",
		"SRE-focused views for latency, capacity, thresholds, and error distribution, all kept native to Swift Charts.",
		[]string{"latency", "slos", "distributions", "power scale", "reference lines", "box plots"},
		section(
			"Latency Reviews",
			"Latency percentiles and service spread are presented as first-class operational views instead of generic chart samples.",
			featuredCard(
				"Latency Percentile Review",
				"p50, p95, and p99 share one scrollable review chart with explicit SLO markers and scale-aware symbols that stay legible on dark plot chrome.",
				"LineMark, PointMark, RuleMark, log scale, date axes, scroll position.",
				chartView(latencyChart(), tallChart),
			),
		),
		section(
			"Counts And Capacity",
			"Counts, percentiles, and queue growth demonstrate that the package supports common operational review patterns with readable scale choices.",
			compactCard(
				"Regional Error Hotspots",
				"Grouped category/value bars show where service errors are accumulating, with explicit category order and compact axis formatting.",
				"BarMark, string categories, ForegroundStyleBy, CornerRadius, explicit axis values, CompactFormat.",
				chartView(serviceErrorsChart(), tallChart),
			),
			compactCard(
				"Latency Distribution Summary",
				"Pre-bucketed bars model a histogram-like latency distribution while native reference lines call out p50, p95, and p99 cut points.",
				"BarMark, RuleMark, string buckets, square-root scale, AxisValueLabels, Annotation.",
				chartView(distributionChart(), tallChart),
			),
			compactCard(
				"Queue Backlog Under Burst Load",
				"Queue backlog rides a power-scale axis so burst growth stays readable without flattening early movement.",
				"Integer plottables, PointMark, LineMark, RuleMark, ScaleTypePower, desired-count and minimum-stride axes.",
				chartView(backlogChart(), tallChart),
			),
			compactCard(
				"Latency Envelope By Service",
				"Native box plots summarize spread per service with a reference line for the p95 SLO.",
				"BoxPlotMark, quartiles, category y-axis, DurationFormat, ReferenceLineX.",
				chartView(latencySpreadChart(), tallChart),
			),
			coverageBand(
				"Coverage In This Screen",
				"Operational scales: log, square-root, and power.",
				"Threshold tools: rule marks, reference lines, styled ticks and grid lines.",
				"Newer marks: box plots and distribution-oriented bars.",
			),
		),
	)
}

func screen(title, subtitle string, badges []string, sections ...swiftui.View) swiftui.View {
	children := []swiftui.Viewable{screenHeader(title, subtitle, badges...)}
	for _, section := range sections {
		children = append(children, section)
	}
	return swiftui.ScrollView(
		swiftui.VStackSpaced(18, children...).Padding(20),
	).BackgroundStyle("windowBackground")
}

func screenHeader(title, subtitle string, badges ...string) swiftui.View {
	items := []swiftui.Viewable{
		swiftui.HStack(
			swiftui.Text(title).
				Font(swiftui.FontTitle2).
				FontWeight(swiftui.WeightBold),
			swiftui.Spacer(),
		),
		swiftui.HStack(
			swiftui.Text(subtitle).
				Font(swiftui.FontCallout).
				ForegroundStyleNamed("secondary").
				LineLimit(0),
			swiftui.Spacer(),
		),
		capabilityRows(badges...),
		swiftui.HStackSpaced(10,
			headerStat("Marks", "native composition"),
			headerStat("Scales", "typed and formatted"),
			headerStat("Use Cases", "ML + benchmarking + SRE"),
		),
	}
	return swiftui.VStackSpaced(10, items...).
		Padding(16).
		BackgroundRoundedRect(0.10, 0.13, 0.18, 0.92, 20).
		Border(0.30, 0.34, 0.40, 0.25, 1).
		Shadow(0, 0, 0, 0.18, 16, 0, 8)
}

func capabilityRows(labels ...string) swiftui.View {
	var rows []swiftui.Viewable
	for len(labels) > 0 {
		n := 3
		if len(labels) < n {
			n = len(labels)
		}
		rowLabels := labels[:n]
		labels = labels[n:]
		row := make([]swiftui.Viewable, 0, len(rowLabels)+1)
		for _, label := range rowLabels {
			row = append(row, capabilityPill(label))
		}
		row = append(row, swiftui.Spacer())
		rows = append(rows, swiftui.HStackSpaced(8, row...))
	}
	return swiftui.VStackSpaced(8, rows...)
}

func capabilityPill(text string) swiftui.View {
	return swiftui.Text(text).
		Font(swiftui.FontCaption2).
		FontWeight(swiftui.WeightSemibold).
		ForegroundStyle(0.93, 0.95, 0.98, 1).
		AsView().
		Padding(7).
		BackgroundRoundedRect(0.16, 0.20, 0.26, 0.92, 10)
}

func headerStat(title, value string) swiftui.View {
	return swiftui.VStackSpaced(4,
		swiftui.HStack(
			swiftui.Text(title).
				Font(swiftui.FontCaption2).
				ForegroundStyleNamed("tertiary"),
			swiftui.Spacer(),
		),
		swiftui.HStack(
			swiftui.Text(value).
				Font(swiftui.FontCallout).
				FontWeight(swiftui.WeightSemibold).
				LineLimit(0),
			swiftui.Spacer(),
		),
	).Padding(10).
		BackgroundRoundedRect(0.12, 0.16, 0.22, 0.92, 12).
		Border(0.82, 0.86, 0.92, 0.08, 1).
		MaxFrame(-1, 0)
}

func section(title, subtitle string, cards ...swiftui.View) swiftui.View {
	children := []swiftui.Viewable{
		swiftui.VStackSpaced(4,
			swiftui.HStack(
				swiftui.Text(title).
					Font(swiftui.FontHeadline).
					FontWeight(swiftui.WeightSemibold),
				swiftui.Spacer(),
			),
			swiftui.HStack(
				swiftui.Text(subtitle).
					Font(swiftui.FontCaption).
					ForegroundStyleNamed("secondary").
					LineLimit(0),
				swiftui.Spacer(),
			),
		),
	}
	for _, card := range cards {
		children = append(children, card)
	}
	return swiftui.VStackSpaced(12, children...)
}

func featuredCard(title, summary, features string, body swiftui.View) swiftui.View {
	return swiftui.GroupBox(title, swiftui.VStackSpaced(12,
		cardSummary(summary),
		body,
		cardMeta(features),
	).Padding(14)).
		MaxFrame(-1, 0)
}

func compactCard(title, summary, features string, body swiftui.View) swiftui.View {
	return swiftui.GroupBox(title, swiftui.VStackSpaced(10,
		cardSummary(summary),
		body,
		cardMeta(features),
	).Padding(12)).
		MaxFrame(-1, 0)
}

func cardSummary(text string) swiftui.View {
	return swiftui.HStack(
		swiftui.Text(text).
			Font(swiftui.FontCallout).
			ForegroundStyleNamed("secondary").
			LineLimit(0),
		swiftui.Spacer(),
	)
}

func cardMeta(text string) swiftui.View {
	return swiftui.HStack(
		swiftui.Text(text).
			Font(swiftui.FontCaption2).
			ForegroundStyleNamed("tertiary").
			LineLimit(0),
		swiftui.Spacer(),
	)
}

func coverageBand(title string, lines ...string) swiftui.View {
	children := []swiftui.Viewable{
		swiftui.HStack(
			swiftui.Text(title).
				Font(swiftui.FontHeadline).
				FontWeight(swiftui.WeightSemibold),
			swiftui.Spacer(),
		),
	}
	for _, line := range lines {
		children = append(children, infoLine(line))
	}
	return swiftui.VStackSpaced(10, children...).
		Padding(14).
		BackgroundRoundedRect(0.08, 0.10, 0.14, 0.94, 18).
		Border(0.82, 0.86, 0.92, 0.10, 1)
}

func infoLine(text string) swiftui.View {
	return swiftui.HStack(
		swiftui.Image("checkmark.circle.fill").
			ForegroundStyle(green.R, green.G, green.B, 1),
		swiftui.Text(text).
			Font(swiftui.FontCaption).
			ForegroundStyleNamed("secondary").
			LineLimit(0),
		swiftui.Spacer(),
	)
}

func chartView(chart charts.ChartView, height float64) swiftui.View {
	version := swiftui.NewIntState(0)
	return swiftui.DynamicView(version, func(_ int) swiftui.View {
		return chart.Frame(chartWidth, height)
	}).OnAppear(func() {
		version.Set(1)
	})
}

func plotChrome(chart charts.ChartView, label string) charts.ChartView {
	return chart.ChartPlotStyle(
		charts.PlotBackgroundColor(swiftui.RGBA(0.07, 0.09, 0.12, 0.16)),
		charts.PlotBorder(swiftui.RGBA(0.88, 0.90, 0.94, 0.16), 1),
		charts.PlotBackgroundView(
			swiftui.ColorView(0.12, 0.15, 0.18, 1).
				Opacity(0.09),
		),
		charts.PlotOverlayView(plotBadge(label)),
	)
}

func plotBadge(label string) swiftui.View {
	return swiftui.VStack(
		swiftui.HStack(
			swiftui.Spacer(),
			swiftui.Text(label).
				Font(swiftui.FontCaption2).
				ForegroundStyleNamed("secondary").
				AsView().
				Padding(6).
				BackgroundRoundedRect(0.10, 0.13, 0.18, 0.86, 8),
		),
		swiftui.Spacer(),
	).MaxFrame(-1, -1)
}

func annotation(text string) swiftui.View {
	return swiftui.Text(text).
		Font(swiftui.FontCaption2).
		ForegroundStyleNamed("secondary").
		AsView().
		Padding(6).
		BackgroundRoundedRect(0.09, 0.12, 0.16, 0.94, 8).
		Border(0.83, 0.86, 0.91, 0.14, 1)
}

func serviceErrorsChart() charts.ChartView {
	data := []regionError{
		{Region: "us-east", Errors: 184},
		{Region: "us-west", Errors: 127},
		{Region: "eu-west", Errors: 96},
		{Region: "ap-south", Errors: 74},
	}
	order := make([]string, 0, len(data))
	xValues := make([]charts.Value, 0, len(data))
	marks := make([]charts.Mark, 0, len(data))
	for i, item := range data {
		order = append(order, item.Region)
		xValues = append(xValues, charts.CategoryValue("Region", item.Region))
		bar := charts.BarMark(
			charts.XString("Region", item.Region),
			charts.YInt("Errors", item.Errors),
		).ForegroundStyleBy("Region", item.Region).
			CornerRadius(7)
		if i == 0 {
			bar = bar.Annotation(charts.AnnotationTop, charts.AnnotationAlignmentCenter, annotation(fmt.Sprintf("%d errors", item.Errors)))
		}
		marks = append(marks, bar)
	}
	return plotChrome(
		charts.Chart(marks...).
			ChartXScaleDomain(charts.CategoryDomain(order...)).
			ChartXScaleType(charts.ScaleTypeCategory).
			ChartYScaleDomain(charts.IntegerDomain(0, 220)).
			ChartYScaleRange(charts.PlotDimensionRange(0, 12)).
			ChartForegroundStyleScale(
				charts.StyleScale("us-east", red),
				charts.StyleScale("us-west", orange),
				charts.StyleScale("eu-west", amber),
				charts.StyleScale("ap-south", cyan),
			).
			ChartLegend(charts.LegendHidden).
			ChartXAxis(charts.AxisMarks(
				charts.AxisValuesOption(charts.AxisValueList(xValues...)),
				charts.AxisLabels(),
				charts.AxisTicksStyled(charts.AxisTickLengthLabel(2), charts.Stroke(1), false),
			)).
			ChartYAxis(charts.AxisMarks(
				charts.AxisValuesOption(charts.AxisValueList(
					charts.IntegerValue("Errors", 0),
					charts.IntegerValue("Errors", 50),
					charts.IntegerValue("Errors", 100),
					charts.IntegerValue("Errors", 150),
					charts.IntegerValue("Errors", 200),
				)),
				charts.AxisGridLinesStyled(charts.Stroke(1, 3, 5), false),
				charts.AxisTicksStyled(charts.AxisTickLengthLongestLabel(3), charts.Stroke(1), false),
				charts.AxisLabels(),
			)).
			ChartXAxisLabel("region").
			ChartYAxisLabel("errors / 5 min", charts.AxisLabelSpacing(4)),
		"explicit category axis",
	)
}

func trainingLossChart() charts.ChartView {
	start := time.Date(2026, time.March, 16, 8, 0, 0, 0, time.Local)
	data := []lossPoint{
		{At: start.Add(0 * time.Minute), Train: 2.28, Valid: 2.42},
		{At: start.Add(10 * time.Minute), Train: 2.04, Valid: 2.17},
		{At: start.Add(20 * time.Minute), Train: 1.81, Valid: 1.93},
		{At: start.Add(30 * time.Minute), Train: 1.63, Valid: 1.71},
		{At: start.Add(40 * time.Minute), Train: 1.49, Valid: 1.56},
		{At: start.Add(50 * time.Minute), Train: 1.34, Valid: 1.42},
		{At: start.Add(60 * time.Minute), Train: 1.21, Valid: 1.29},
		{At: start.Add(70 * time.Minute), Train: 1.10, Valid: 1.17},
		{At: start.Add(80 * time.Minute), Train: 0.99, Valid: 1.08},
		{At: start.Add(90 * time.Minute), Train: 0.91, Valid: 0.98},
		{At: start.Add(100 * time.Minute), Train: 0.83, Valid: 0.92},
		{At: start.Add(110 * time.Minute), Train: 0.76, Valid: 0.86},
		{At: start.Add(120 * time.Minute), Train: 0.71, Valid: 0.81},
	}
	marks := make([]charts.Mark, 0, len(data)*3)
	for _, item := range data {
		marks = append(marks,
			charts.AreaMark(
				charts.XDate("Time", item.At, charts.TimeUnitMinute),
				charts.YFloat("Train", item.Train),
			).
				ForegroundStyle(swiftui.RGBA(cyan.R, cyan.G, cyan.B, 0.22)).
				Interpolation(charts.InterpolationCatmullRomAlpha(0.45)).
				AlignsMarkStylesWithPlotArea(true).
				Opacity(0.85).
				Blur(0.45),
			charts.LineMark(
				charts.XDate("Time", item.At, charts.TimeUnitMinute),
				charts.YFloat("Loss", item.Train),
			).
				ForegroundStyleBy("Series", "train").
				Interpolation(charts.InterpolationMonotone).
				LineStyle(charts.Stroke(2.8)),
			charts.LineMark(
				charts.XDate("Time", item.At, charts.TimeUnitMinute),
				charts.YFloat("Loss", item.Valid),
			).
				ForegroundStyleBy("Series", "valid").
				Interpolation(charts.InterpolationCardinalTension(0.15)).
				LineStyle(charts.Stroke(2, 6, 3)),
		)
	}
	last := data[len(data)-1]
	marks = append(marks,
		charts.PointMark(
			charts.XDate("Time", last.At, charts.TimeUnitMinute),
			charts.YFloat("Loss", last.Valid),
		).
			ForegroundStyleBy("Series", "valid").
			Symbol(charts.SymbolDiamond).
			SymbolSize(72).
			Annotation(charts.AnnotationTopTrailing, charts.AnnotationAlignmentTrailing, annotation("valid 0.81")),
	)
	return plotChrome(
		charts.Chart(marks...).
			ChartXScaleDomain(charts.TimeDomain(start, last.At)).
			ChartYScaleDomain(charts.NumberDomain(0.55, 2.55)).
			ChartYScaleRange(charts.PlotDimensionRange(8, 8)).
			ChartForegroundStyleScale(
				charts.StyleScale("train", blue),
				charts.StyleScale("valid", amber),
			).
			ChartLegend(charts.LegendVisible(charts.LegendPositionTop, charts.LegendAlignmentLeading, 8)).
			ChartXAxis(charts.AxisMarks(
				charts.AxisValuesOption(charts.TimeStrideCount(charts.TimeUnitMinute, 20)),
				charts.AxisTicksStyled(charts.AxisTickLengthLabel(2), charts.Stroke(1), false),
				charts.AxisDateLabels("HH:mm", false),
			)).
			ChartYAxis(charts.AxisMarks(
				charts.AxisValuesOption(charts.AutomaticAxisValuesCount(5)),
				charts.AxisGridLinesStyled(charts.Stroke(1, 3, 5), false),
				charts.AxisTicksStyled(charts.AxisTickLengthLongestLabel(2), charts.Stroke(1), false),
				charts.AxisLabels(),
			)).
			ChartXAxisLabel("wall clock").
			ChartYAxisLabel("cross-entropy").
			ChartScrollableAxes(charts.ScrollAxisHorizontal).
			ChartScrollPositionInitialXDate(start.Add(35*time.Minute)).
			ChartXVisibleDomainLength(55*60),
		"date scale + scroll window",
	)
}

func mixedTrainingChart() charts.ChartView {
	data := []throughputPoint{
		{Stage: "960", Throughput: 122, LossX100: 94},
		{Stage: "1080", Throughput: 129, LossX100: 83},
		{Stage: "1200", Throughput: 133, LossX100: 76},
		{Stage: "1320", Throughput: 139, LossX100: 69},
		{Stage: "1440", Throughput: 146, LossX100: 61},
		{Stage: "1560", Throughput: 151, LossX100: 55},
	}
	marks := make([]charts.Mark, 0, len(data)*3+1)
	for _, item := range data {
		marks = append(marks,
			charts.BarMark(
				charts.XString("Stage", item.Stage),
				charts.YFloat("Level", item.Throughput),
			).
				ForegroundStyle(teal).
				CornerRadius(6),
			charts.LineMark(
				charts.XString("Stage", item.Stage),
				charts.YFloat("Level", item.LossX100),
			).
				ForegroundStyle(amber).
				Interpolation(charts.InterpolationCardinalTension(0.2)).
				LineStyle(charts.Stroke(2.4)),
			charts.PointMark(
				charts.XString("Stage", item.Stage),
				charts.YFloat("Level", item.LossX100),
			).
				ForegroundStyle(amber).
				Symbol(charts.SymbolCircle).
				SymbolSize(54).
				Offset(0, -1).
				Shadow(swiftui.RGBA(0, 0, 0, 0.22), 6, 0, 3).
				ZIndex(3),
		)
	}
	marks = append(marks,
		charts.RuleMark(
			charts.YFloat("Gate", 65),
		).
			ForegroundStyle(red).
			LineStyle(charts.Stroke(1.5, 5, 4)).
			Annotation(charts.AnnotationTopLeading, charts.AnnotationAlignmentLeading, annotation("deploy gate: loss x100 <= 65")).
			ZIndex(4),
	)
	return plotChrome(
		charts.Chart(marks...).
			ChartXScaleDomain(charts.CategoryDomain("960", "1080", "1200", "1320", "1440", "1560")).
			ChartXScaleType(charts.ScaleTypeCategory).
			ChartYScaleDomain(charts.NumberDomain(0, 165)).
			ChartYScaleType(charts.ScaleTypeLinear).
			ChartForegroundStyleScale(
				charts.StyleScale("loss", amber),
			).
			ChartLegend(charts.LegendHidden).
			ChartXAxis(charts.AxisMarks(
				charts.AxisLabels(),
				charts.AxisTicks(),
			)).
			ChartYAxis(charts.AxisMarks(
				charts.AxisValuesOption(charts.NumericStrideRounded(25, true, true)),
				charts.AxisGridLinesStyled(charts.Stroke(1, 3, 5), false),
				charts.AxisTicksStyled(charts.AxisTickLengthLongestLabel(2), charts.Stroke(1), false),
				charts.AxisLabels(),
			)).
			ChartXAxisLabel("optimizer step (x10)").
			ChartYAxisLabel("throughput k tok/s and loss x100"),
		"bar + line + rule composition",
	)
}

func sectorShareChart() charts.ChartView {
	data := []sectorShare{
		{Label: "training", Value: 46},
		{Label: "serving", Value: 28},
		{Label: "evaluation", Value: 14},
		{Label: "retrieval", Value: 7},
		{Label: "cache", Value: 5},
	}
	marks := make([]charts.Mark, 0, len(data))
	for _, item := range data {
		mark := charts.SectorMark(
			charts.Angle(charts.NumberValue("Tokens", item.Value)),
			charts.InnerRadiusRatio(0.55),
			charts.AngularInset(2),
		).ForegroundStyleBy("Workload", item.Label).
			CornerRadius(5).
			Shadow(swiftui.RGBA(0, 0, 0, 0.10), 4, 0, 2)
		if item.Label == "training" {
			mark = mark.Annotation(charts.AnnotationOverlay, charts.AnnotationAlignmentCenter, annotation("46% training"))
		}
		marks = append(marks, mark)
	}
	return plotChrome(
		charts.Chart(marks...).
			ChartForegroundStyleScale(
				charts.StyleScale("training", blue),
				charts.StyleScale("serving", teal),
				charts.StyleScale("evaluation", amber),
				charts.StyleScale("retrieval", plum),
				charts.StyleScale("cache", muted),
			).
			ChartLegend(charts.LegendVisible(charts.LegendPositionTrailing, charts.LegendAlignmentTop, 8)),
		"sector marks",
	)
}

func benchmarkBarsChart() charts.ChartView {
	data := []benchCase{
		{Name: "router", Old: 248, New: 169},
		{Name: "decoder", Old: 181, New: 154},
		{Name: "kv-cache", Old: 138, New: 104},
		{Name: "tokenize", Old: 91, New: 97},
	}
	marks := make([]charts.Mark, 0, len(data)*2)
	for _, item := range data {
		marks = append(marks,
			charts.BarMark(
				charts.XString("Benchmark", item.Name),
				charts.YInt("ns/op", item.Old),
			).
				ForegroundStyleBy("Run", "old").
				PositionBy("Run", "old").
				Stacking(charts.MarkStackingUnstacked).
				CornerRadius(5),
			charts.BarMark(
				charts.XString("Benchmark", item.Name),
				charts.YInt("ns/op", item.New),
			).
				ForegroundStyleBy("Run", "new").
				PositionBy("Run", "new").
				Stacking(charts.MarkStackingUnstacked).
				CornerRadius(5),
		)
	}
	return plotChrome(
		charts.Chart(marks...).
			ChartXScaleDomain(charts.CategoryDomain("router", "decoder", "kv-cache", "tokenize")).
			ChartXScaleType(charts.ScaleTypeCategory).
			ChartYScaleDomain(charts.IntegerDomain(0, 280)).
			ChartYScaleRange(charts.PlotDimensionRange(6, 10)).
			ChartForegroundStyleScale(
				charts.StyleScale("old", muted),
				charts.StyleScale("new", blue),
			).
			ChartLegend(charts.LegendVisible(charts.LegendPositionTop, charts.LegendAlignmentLeading, 8)).
			ChartXAxis(charts.AxisMarks(
				charts.AxisValuesOption(charts.AxisValueList(
					charts.CategoryValue("Benchmark", "router"),
					charts.CategoryValue("Benchmark", "decoder"),
					charts.CategoryValue("Benchmark", "kv-cache"),
					charts.CategoryValue("Benchmark", "tokenize"),
				)),
				charts.AxisTicks(),
				charts.AxisLabels(),
			)).
			ChartYAxis(charts.AxisMarks(
				charts.AxisValuesOption(charts.NumericStride(40)),
				charts.AxisGridLinesStyled(charts.Stroke(1, 3, 5), false),
				charts.AxisLabels(),
			)).
			ChartXAxisLabel("benchmark").
			ChartYAxisLabel("ns/op"),
		"grouped bars",
	)
}

func benchmarkDeltaChart() charts.ChartView {
	data := []benchCase{
		{Name: "router", Old: 248, New: 169},
		{Name: "decoder", Old: 181, New: 154},
		{Name: "kv-cache", Old: 138, New: 104},
		{Name: "tokenize", Old: 91, New: 97},
	}
	marks := make([]charts.Mark, 0, len(data)+1)
	for _, item := range data {
		delta := 100 * (float64(item.New-item.Old) / float64(item.Old))
		severity := "flat"
		if delta < -10 {
			severity = "win"
		}
		if delta > 5 {
			severity = "regress"
		}
		point := charts.PointMark(
			charts.XString("Benchmark", item.Name),
			charts.YFloat("Delta", delta),
		).
			ForegroundStyleBy("Delta", severity).
			Symbol(charts.SymbolCircle).
			SymbolSize(62).
			Offset(0, -1)
		if item.Name == "tokenize" {
			point = point.Annotation(charts.AnnotationTop, charts.AnnotationAlignmentCenter, annotation(fmt.Sprintf("%.1f%% regression", delta)))
		}
		marks = append(marks, point)
	}
	marks = append(marks,
		charts.RuleMark(charts.YFloat("Zero", 0)).
			ForegroundStyle(muted).
			LineStyle(charts.Stroke(1.4, 4, 4)),
	)
	return plotChrome(
		charts.Chart(marks...).
			ChartXScaleDomain(charts.CategoryDomain("router", "decoder", "kv-cache", "tokenize")).
			ChartXScaleType(charts.ScaleTypeCategory).
			ChartYScaleDomain(charts.NumberDomain(-40, 20)).
			ChartYScaleType(charts.ScaleTypeSymmetricLog(3)).
			ChartForegroundStyleScale(
				charts.StyleScale("win", green),
				charts.StyleScale("flat", amber),
				charts.StyleScale("regress", red),
			).
			ChartLegend(charts.LegendHidden).
			ChartXAxis(charts.AxisMarks(
				charts.AxisLabels(),
			)).
			ChartYAxis(charts.AxisMarks(
				charts.AxisValuesOption(charts.AxisValueList(
					charts.NumberValue("Delta", -30),
					charts.NumberValue("Delta", -15),
					charts.NumberValue("Delta", 0),
					charts.NumberValue("Delta", 10),
				)),
				charts.AxisGridLinesStyled(charts.Stroke(1, 3, 5), false),
				charts.AxisTicksStyled(charts.AxisTickLengthLongestLabel(2), charts.Stroke(1), false),
				charts.AxisLabels(),
			)).
			ChartYAxisLabel("delta %"),
		"delta strip on symmetric-log",
	)
}

func benchmarkUncertaintyChart() charts.ChartView {
	data := []benchInterval{
		{Name: "router", Run: "old", Mean: 248, Low: 238, High: 259},
		{Name: "router", Run: "new", Mean: 169, Low: 162, High: 176},
		{Name: "decoder", Run: "old", Mean: 181, Low: 174, High: 188},
		{Name: "decoder", Run: "new", Mean: 154, Low: 149, High: 159},
		{Name: "kv-cache", Run: "old", Mean: 138, Low: 132, High: 144},
		{Name: "kv-cache", Run: "new", Mean: 104, Low: 100, High: 109},
		{Name: "tokenize", Run: "old", Mean: 91, Low: 87, High: 95},
		{Name: "tokenize", Run: "new", Mean: 97, Low: 92, High: 102},
	}
	marks := make([]charts.Mark, 0, len(data)*2)
	for _, item := range data {
		err := charts.ErrorBarMark(
			charts.XString("Benchmark", item.Name),
			charts.YFloatRange("ns/op", item.Low, item.High),
			charts.WidthRatio(0.34),
		).
			ForegroundStyleBy("Run", item.Run).
			PositionBy("Run", item.Run)
		point := charts.PointMark(
			charts.XString("Benchmark", item.Name),
			charts.YFloat("ns/op", item.Mean),
		).
			ForegroundStyleBy("Run", item.Run).
			PositionBy("Run", item.Run).
			SymbolBy("Run", item.Run).
			SymbolStroke(swiftui.RGB(0.96, 0.98, 1.0), 1.2).
			SymbolSize(76)
		if item.Name == "router" && item.Run == "new" {
			point = point.TextAnnotation("-31.9% wall-time", charts.AnnotationTop)
		}
		marks = append(marks, err, point)
	}
	return plotChrome(
		charts.Chart(marks...).
			ChartXScaleDomain(charts.CategoryDomain("router", "decoder", "kv-cache", "tokenize")).
			ChartXScaleType(charts.ScaleTypeCategory).
			ChartYScaleDomain(charts.NumberDomain(80, 280)).
			ChartForegroundStyleScale(
				charts.StyleScale("old", muted),
				charts.StyleScale("new", blue),
			).
			ChartSymbolScale(
				charts.SymbolScale("old", charts.SymbolCircle),
				charts.SymbolScale("new", charts.SymbolDiamond),
			).
			ChartLegend(charts.LegendVisible(charts.LegendPositionTop, charts.LegendAlignmentLeading, 8)).
			ChartXAxis(charts.AxisMarks(
				charts.AxisLabels(),
				charts.AxisTicks(),
			)).
			ChartYAxis(charts.AxisMarks(
				charts.AxisValuesOption(charts.NumericStride(40)),
				charts.AxisGridLinesStyled(charts.Stroke(1, 3, 5), false),
				charts.AxisValueLabels(charts.SuffixFormat(" ns/op", 0)),
			)).
			ChartXAxisLabel("benchmark").
			ChartYAxisLabel("mean with interval"),
		"error bars + formatted axis",
	)
}

func evaluationChart() charts.ChartView {
	data := []evalScore{
		{Task: "MMLU", Model: "Nova-2", Score: 83.2},
		{Task: "MMLU", Model: "Atlas-XL", Score: 79.1},
		{Task: "MMLU", Model: "Base", Score: 72.8},
		{Task: "GSM8K", Model: "Nova-2", Score: 91.4},
		{Task: "GSM8K", Model: "Atlas-XL", Score: 88.2},
		{Task: "GSM8K", Model: "Base", Score: 76.9},
		{Task: "HumanEval", Model: "Nova-2", Score: 81.0},
		{Task: "HumanEval", Model: "Atlas-XL", Score: 78.6},
		{Task: "HumanEval", Model: "Base", Score: 67.5},
		{Task: "DROP", Model: "Nova-2", Score: 72.5},
		{Task: "DROP", Model: "Atlas-XL", Score: 70.9},
		{Task: "DROP", Model: "Base", Score: 61.4},
		{Task: "GPQA", Model: "Nova-2", Score: 61.8},
		{Task: "GPQA", Model: "Atlas-XL", Score: 59.7},
		{Task: "GPQA", Model: "Base", Score: 48.6},
	}
	marks := make([]charts.Mark, 0, len(data))
	for _, item := range data {
		bar := charts.BarMark(
			charts.XString("Task", item.Task),
			charts.YFloat("Score", item.Score),
		).
			ForegroundStyleBy("Model", item.Model).
			PositionBy("Model", item.Model).
			Stacking(charts.MarkStackingUnstacked).
			CornerRadius(5)
		if item.Task == "GPQA" && item.Model == "Nova-2" {
			bar = bar.Annotation(charts.AnnotationTopTrailing, charts.AnnotationAlignmentTrailing, annotation("best frontier model"))
		}
		marks = append(marks, bar)
	}
	return plotChrome(
		charts.Chart(marks...).
			ChartXScaleDomain(charts.CategoryDomain("MMLU", "GSM8K", "HumanEval", "DROP", "GPQA")).
			ChartXScaleType(charts.ScaleTypeCategory).
			ChartYScaleDomain(charts.NumberDomain(40, 96)).
			ChartYScaleRange(charts.PlotDimensionRange(8, 8)).
			ChartForegroundStyleScale(
				charts.StyleScale("Nova-2", blue),
				charts.StyleScale("Atlas-XL", plum),
				charts.StyleScale("Base", muted),
			).
			ChartForegroundStyleScaleType(charts.ScaleTypeCategory).
			ChartLegend(charts.LegendVisible(charts.LegendPositionTop, charts.LegendAlignmentLeading, 8)).
			ChartXAxis(charts.AxisMarks(
				charts.AxisValuesOption(charts.AxisValueList(
					charts.CategoryValue("Task", "MMLU"),
					charts.CategoryValue("Task", "GSM8K"),
					charts.CategoryValue("Task", "HumanEval"),
					charts.CategoryValue("Task", "DROP"),
					charts.CategoryValue("Task", "GPQA"),
				)),
				charts.AxisLabels(),
			)).
			ChartYAxis(charts.AxisMarks(
				charts.AxisValuesOption(charts.NumericStrideRounded(10, true, true)),
				charts.AxisGridLinesStyled(charts.Stroke(1, 3, 5), false),
				charts.AxisLabels(),
			)).
			ChartXAxisLabel("evaluation task").
			ChartYAxisLabel("score %"),
		"style scale + grouped comparison",
	)
}

func heatmapChart() charts.ChartView {
	models := []string{"Atlas-XL", "Nova-2", "Reasoner", "Fast-Base"}
	tasks := []string{"MMLU", "GPQA", "Code", "Retrieval", "Safety"}
	scores := [][]float64{
		{79.1, 83.2, 81.7, 69.8},
		{59.7, 61.8, 63.9, 44.3},
		{78.6, 81.0, 74.2, 58.8},
		{72.1, 76.4, 68.9, 63.7},
		{74.8, 77.9, 70.4, 66.2},
	}
	marks := make([]charts.Mark, 0, len(models)*len(tasks)+1)
	best := matrixCell{}
	for y, task := range tasks {
		for x, model := range models {
			cell := matrixCell{
				Model: model,
				Task:  task,
				X:     float64(x + 1),
				Y:     float64(len(tasks) - y),
				Score: scores[y][x],
			}
			if cell.Score > best.Score {
				best = cell
			}
			marks = append(marks, charts.RectangleMark(
				charts.XFloatRange("Model", cell.X-0.42, cell.X+0.42),
				charts.YFloatRange("Task", cell.Y-0.42, cell.Y+0.42),
			).
				ForegroundStyle(scoreColor(cell.Score)).
				Opacity(0.93).
				CornerRadius(6))
		}
	}
	marks = append(marks,
		charts.PointMark(
			charts.XFloat("Model", best.X),
			charts.YFloat("Task", best.Y),
		).
			ForegroundStyle(swiftui.RGB(0.98, 0.99, 1.0)).
			Symbol(charts.SymbolPlus).
			SymbolSize(116).
			Shadow(swiftui.RGBA(0, 0, 0, 0.26), 8, 0, 2).
			ZIndex(5).
			Annotation(charts.AnnotationTopTrailing, charts.AnnotationAlignmentTrailing, annotation(fmt.Sprintf("%s leads %.1f", best.Model, best.Score))),
	)
	return plotChrome(
		charts.Chart(marks...).
			ChartXScaleDomain(charts.NumberDomain(0.5, 4.5)).
			ChartYScaleDomain(charts.NumberDomain(0.5, 5.5)).
			ChartXScaleRange(charts.PlotDimensionRange(14, 14)).
			ChartYScaleRange(charts.PlotDimensionRange(14, 14)).
			ChartXAxis(charts.AxisMarks(
				charts.AxisValuesOption(charts.AxisValueList(
					charts.NumberValue("Model", 1),
					charts.NumberValue("Model", 2),
					charts.NumberValue("Model", 3),
					charts.NumberValue("Model", 4),
				)),
				charts.AxisTicks(),
				charts.AxisLabels(),
			)).
			ChartYAxis(charts.AxisMarks(
				charts.AxisValuesOption(charts.AxisValueList(
					charts.NumberValue("Task", 1),
					charts.NumberValue("Task", 2),
					charts.NumberValue("Task", 3),
					charts.NumberValue("Task", 4),
					charts.NumberValue("Task", 5),
				)),
				charts.AxisTicks(),
				charts.AxisLabels(),
			)).
			ChartXAxisLabel("model index").
			ChartYAxisLabel("task index", charts.AxisLabelSpacing(4)),
		"rectangle grid",
	)
}

func latencyChart() charts.ChartView {
	start := time.Date(2026, time.March, 16, 10, 0, 0, 0, time.Local)
	data := []latencyPoint{
		{At: start.Add(0 * time.Minute), P50: 28, P95: 94, P99: 162},
		{At: start.Add(5 * time.Minute), P50: 29, P95: 98, P99: 174},
		{At: start.Add(10 * time.Minute), P50: 31, P95: 104, P99: 182},
		{At: start.Add(15 * time.Minute), P50: 32, P95: 110, P99: 191},
		{At: start.Add(20 * time.Minute), P50: 34, P95: 116, P99: 206},
		{At: start.Add(25 * time.Minute), P50: 35, P95: 124, P99: 228},
		{At: start.Add(30 * time.Minute), P50: 36, P95: 129, P99: 241},
		{At: start.Add(35 * time.Minute), P50: 38, P95: 137, P99: 258},
		{At: start.Add(40 * time.Minute), P50: 41, P95: 149, P99: 282},
		{At: start.Add(45 * time.Minute), P50: 40, P95: 143, P99: 264},
		{At: start.Add(50 * time.Minute), P50: 37, P95: 136, P99: 238},
		{At: start.Add(55 * time.Minute), P50: 35, P95: 130, P99: 216},
	}
	type seriesPoint struct {
		name  string
		value float64
	}
	marks := make([]charts.Mark, 0, len(data)*3+5)
	for _, item := range data {
		series := []seriesPoint{
			{name: "p50", value: item.P50},
			{name: "p95", value: item.P95},
			{name: "p99", value: item.P99},
		}
		for _, point := range series {
			style := charts.Stroke(2.1)
			if point.name == "p95" {
				style = charts.Stroke(2.3, 7, 3)
			}
			if point.name == "p99" {
				style = charts.Stroke(2.5)
			}
			marks = append(marks,
				charts.LineMark(
					charts.XDate("Time", item.At, charts.TimeUnitMinute),
					charts.YFloat("Latency", point.value),
				).
					ForegroundStyleBy("Series", point.name).
					LineStyle(style).
					Interpolation(charts.InterpolationMonotone),
			)
		}
	}
	last := data[len(data)-1]
	marks = append(marks,
		charts.PointMark(
			charts.XDate("Time", last.At, charts.TimeUnitMinute),
			charts.YFloat("Latency", last.P50),
		).
			ForegroundStyleBy("Series", "p50").
			Symbol(charts.SymbolCircle).
			SymbolSize(52).
			Shadow(swiftui.RGBA(0, 0, 0, 0.18), 6, 0, 3).
			ZIndex(5),
		charts.PointMark(
			charts.XDate("Time", last.At, charts.TimeUnitMinute),
			charts.YFloat("Latency", last.P95),
		).
			ForegroundStyleBy("Series", "p95").
			Symbol(charts.SymbolDiamond).
			SymbolSize(70).
			Shadow(swiftui.RGBA(0, 0, 0, 0.18), 6, 0, 3).
			ZIndex(5),
		charts.PointMark(
			charts.XDate("Time", last.At, charts.TimeUnitMinute),
			charts.YFloat("Latency", last.P99),
		).
			ForegroundStyleBy("Series", "p99").
			Symbol(charts.SymbolTriangle).
			SymbolSize(92).
			Shadow(swiftui.RGBA(0, 0, 0, 0.18), 6, 0, 3).
			ZIndex(5).
			Annotation(charts.AnnotationTopTrailing, charts.AnnotationAlignmentTrailing, annotation("p99 cooled to 216 ms")),
		charts.RuleMark(charts.YFloat("SLO", 180)).
			ForegroundStyle(amber).
			LineStyle(charts.Stroke(1.5, 4, 4)),
		charts.RuleMark(charts.YFloat("Page", 300)).
			ForegroundStyle(red).
			LineStyle(charts.Stroke(1.5, 2, 5)),
	)
	end := last.At
	return plotChrome(
		charts.Chart(marks...).
			ChartXScaleDomain(charts.TimeDomain(start, end)).
			ChartYScaleDomain(charts.NumberDomain(20, 360)).
			ChartYScaleType(charts.ScaleTypeLog).
			ChartForegroundStyleScale(
				charts.StyleScale("p50", teal),
				charts.StyleScale("p95", blue),
				charts.StyleScale("p99", red),
			).
			ChartLegend(charts.LegendVisible(charts.LegendPositionTop, charts.LegendAlignmentLeading, 8)).
			ChartXAxis(charts.AxisMarks(
				charts.AxisValuesOption(charts.TimeStrideCount(charts.TimeUnitMinute, 15)),
				charts.AxisTicksStyled(charts.AxisTickLengthLabel(2), charts.Stroke(1), false),
				charts.AxisDateLabels("HH:mm", false),
			)).
			ChartYAxis(charts.AxisMarks(
				charts.AxisValuesOption(charts.AxisValueList(
					charts.NumberValue("Latency", 25),
					charts.NumberValue("Latency", 50),
					charts.NumberValue("Latency", 100),
					charts.NumberValue("Latency", 200),
					charts.NumberValue("Latency", 300),
				)),
				charts.AxisGridLinesStyled(charts.Stroke(1, 3, 5), false),
				charts.AxisLabels(),
			)).
			ChartXAxisLabel("request time").
			ChartYAxisLabel("latency ms").
			ChartScrollableAxes(charts.ScrollAxisHorizontal).
			ChartScrollPositionInitialXDate(start.Add(20*time.Minute)).
			ChartXVisibleDomainLength(30*60),
		"log scale + scroll window",
	)
}

func latencySpreadChart() charts.ChartView {
	data := []latencySpread{
		{Service: "router", Min: 28, Q1: 41, Median: 57, Q3: 78, Max: 119},
		{Service: "decoder", Min: 34, Q1: 62, Median: 88, Q3: 131, Max: 192},
		{Service: "retrieval", Min: 22, Q1: 31, Median: 49, Q3: 73, Max: 118},
		{Service: "reranker", Min: 41, Q1: 78, Median: 109, Q3: 151, Max: 234},
	}
	marks := make([]charts.Mark, 0, len(data)+1)
	for _, item := range data {
		marks = append(marks, charts.BoxPlotMark(
			charts.YString("Service", item.Service),
			charts.Minimum(charts.NumberValue("Min", item.Min)),
			charts.Q1(charts.NumberValue("Q1", item.Q1)),
			charts.Median(charts.NumberValue("Median", item.Median)),
			charts.Q3(charts.NumberValue("Q3", item.Q3)),
			charts.Maximum(charts.NumberValue("Max", item.Max)),
			charts.HeightFixed(14),
		).
			ForegroundStyleBy("Service", item.Service).
			Opacity(0.92))
	}
	marks = append(marks,
		charts.ReferenceLineX(charts.NumberValue("SLO", 180), "p95 slo").
			ForegroundStyle(red).
			LineStyle(charts.Stroke(1.5, 4, 4)),
	)
	return plotChrome(
		charts.Chart(marks...).
			ChartXScaleDomain(charts.NumberDomain(0, 250)).
			ChartYScaleDomain(charts.CategoryDomain("router", "decoder", "retrieval", "reranker")).
			ChartYScaleType(charts.ScaleTypeCategory).
			ChartForegroundStyleScale(
				charts.StyleScale("router", teal),
				charts.StyleScale("decoder", blue),
				charts.StyleScale("retrieval", green),
				charts.StyleScale("reranker", amber),
			).
			ChartLegend(charts.LegendHidden).
			ChartXAxis(charts.AxisMarks(
				charts.AxisValuesOption(charts.NumericStride(50)),
				charts.AxisGridLinesStyled(charts.Stroke(1, 3, 5), false),
				charts.AxisValueLabels(charts.DurationFormat(charts.DurationUnitMillisecond, 0)),
			)).
			ChartYAxis(charts.AxisMarks(
				charts.AxisLabels(),
			)).
			ChartXAxisLabel("latency envelope").
			ChartYAxisLabel("service"),
		"box plot + reference line",
	)
}

func distributionChart() charts.ChartView {
	data := []bucketCount{
		{Label: "0-25", Count: 880},
		{Label: "25-50", Count: 610},
		{Label: "50-75", Count: 390},
		{Label: "75-100", Count: 260},
		{Label: "100-150", Count: 170},
		{Label: "150-200", Count: 94},
		{Label: "200-300", Count: 46},
		{Label: "300+", Count: 18},
	}
	marks := make([]charts.Mark, 0, len(data)+3)
	order := make([]string, 0, len(data))
	xvals := make([]charts.Value, 0, len(data))
	for _, item := range data {
		order = append(order, item.Label)
		xvals = append(xvals, charts.CategoryValue("Bucket", item.Label))
		marks = append(marks, charts.BarMark(
			charts.XString("Bucket", item.Label),
			charts.YInt("Count", item.Count),
		).
			ForegroundStyle(swiftui.RGBA(blue.R, blue.G, blue.B, 0.88)).
			CornerRadius(5))
	}
	marks = append(marks,
		charts.RuleMark(charts.XString("Percentile", "50-75")).
			ForegroundStyle(green).
			LineStyle(charts.Stroke(1.5, 4, 3)).
			TextAnnotation("p50", charts.AnnotationTop),
		charts.RuleMark(charts.XString("Percentile", "100-150")).
			ForegroundStyle(amber).
			LineStyle(charts.Stroke(1.5, 4, 3)).
			TextAnnotation("p95", charts.AnnotationTop),
		charts.RuleMark(charts.XString("Percentile", "200-300")).
			ForegroundStyle(red).
			LineStyle(charts.Stroke(1.5, 4, 3)).
			TextAnnotation("p99", charts.AnnotationTop),
	)
	return plotChrome(
		charts.Chart(marks...).
			ChartXScaleDomain(charts.CategoryDomain(order...)).
			ChartXScaleType(charts.ScaleTypeCategory).
			ChartYScaleDomain(charts.IntegerDomain(0, 950)).
			ChartYScaleType(charts.ScaleTypeSquareRoot).
			ChartXAxis(charts.AxisMarks(
				charts.AxisValuesOption(charts.AxisValueList(xvals...)),
				charts.AxisTicks(),
				charts.AxisLabels(),
			)).
			ChartYAxis(charts.AxisMarks(
				charts.AxisValuesOption(charts.AutomaticAxisValuesCount(5)),
				charts.AxisGridLinesStyled(charts.Stroke(1, 3, 5), false),
				charts.AxisLabels(),
			)).
			ChartXAxisLabel("latency bucket (ms)").
			ChartYAxisLabel("request count"),
		"square-root histogram view",
	)
}

func backlogChart() charts.ChartView {
	data := []backlogPoint{
		{Load: 1, Backlog: 14},
		{Load: 2, Backlog: 27},
		{Load: 3, Backlog: 46},
		{Load: 4, Backlog: 74},
		{Load: 5, Backlog: 113},
		{Load: 6, Backlog: 164},
		{Load: 7, Backlog: 211},
		{Load: 8, Backlog: 238},
	}
	marks := make([]charts.Mark, 0, len(data)*2+1)
	for _, item := range data {
		marks = append(marks,
			charts.LineMark(
				charts.XInt("Load", item.Load),
				charts.YFloat("Backlog", item.Backlog),
			).
				ForegroundStyle(plum).
				Interpolation(charts.InterpolationCardinalTension(0.22)).
				LineStyle(charts.Stroke(2.6)),
			charts.PointMark(
				charts.XInt("Load", item.Load),
				charts.YFloat("Backlog", item.Backlog),
			).
				ForegroundStyle(plum).
				Symbol(charts.SymbolAsterisk).
				SymbolSize(76),
		)
	}
	marks = append(marks,
		charts.RuleMark(charts.YFloat("Risk", 160)).
			ForegroundStyle(red).
			LineStyle(charts.Stroke(1.5, 3, 4)).
			Annotation(charts.AnnotationTopLeading, charts.AnnotationAlignmentLeading, annotation("pager queue threshold")),
	)
	return plotChrome(
		charts.Chart(marks...).
			ChartXScaleDomain(charts.IntegerDomain(1, 8)).
			ChartXScaleType(charts.ScaleTypeLinear).
			ChartYScaleDomain(charts.NumberDomain(0, 260)).
			ChartYScaleType(charts.ScaleTypePower(0.65)).
			ChartXScaleRange(charts.PlotDimensionRange(10, 12)).
			ChartYScaleRange(charts.PlotDimensionRange(8, 8)).
			ChartXAxis(charts.AxisMarks(
				charts.AxisValuesOption(charts.AutomaticAxisValuesCount(8)),
				charts.AxisTicksStyled(charts.AxisTickLengthLabel(2), charts.Stroke(1), false),
				charts.AxisLabels(),
			)).
			ChartYAxis(charts.AxisMarks(
				charts.AxisValuesOption(charts.AutomaticAxisValuesMinimumStride(40, 5)),
				charts.AxisGridLinesStyled(charts.Stroke(1, 3, 5), false),
				charts.AxisTicksStyled(charts.AxisTickLengthLongestLabel(2), charts.Stroke(1), false),
				charts.AxisLabels(),
			)).
			ChartXAxisLabel("load factor").
			ChartYAxisLabel("queued requests"),
		"power-law compression",
	)
}

func indexLegend() swiftui.View {
	return swiftui.VStackSpaced(6,
		swiftui.HStack(
			swiftui.Text("Model index: 1 Atlas-XL, 2 Nova-2, 3 Reasoner, 4 Fast-Base").
				Font(swiftui.FontCaption2).
				ForegroundStyleNamed("secondary").
				LineLimit(0),
			swiftui.Spacer(),
		),
		swiftui.HStack(
			swiftui.Text("Task index: 5 MMLU, 4 GPQA, 3 Code, 2 Retrieval, 1 Safety").
				Font(swiftui.FontCaption2).
				ForegroundStyleNamed("secondary").
				LineLimit(0),
			swiftui.Spacer(),
		),
	)
}

func scoreColor(score float64) swiftui.Color {
	t := clamp((score-40)/(90-40), 0, 1)
	r := mix(red.R, green.R, t)
	g := mix(red.G, green.G, t)
	b := mix(red.B, green.B, t)
	return swiftui.RGB(r, g, b)
}

func clamp(v, lo, hi float64) float64 {
	return math.Max(lo, math.Min(hi, v))
}

func mix(a, b, t float64) float64 {
	return a + (b-a)*t
}

func stringify(items []interface{}) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}
