package charts

import (
	"testing"
	"time"

	"github.com/tmc/swiftui"
)

func TestErrorBarMarkSpec(t *testing.T) {
	spec := ErrorBarMark(
		XFloatRange("CI", 1.2, 1.8),
		YString("Series", "kept"),
		HeightFixed(8),
	).toSpec()

	if spec.Kind != int32(markGroup) {
		t.Fatalf("kind = %d, want %d", spec.Kind, markGroup)
	}
	if len(spec.Children) != 3 {
		t.Fatalf("children = %d, want 3", len(spec.Children))
	}
	if spec.Children[0].Kind != int32(markRule) {
		t.Fatalf("first child kind = %d, want %d", spec.Children[0].Kind, markRule)
	}
	if spec.Children[1].Kind != int32(markRectangle) || spec.Children[2].Kind != int32(markRectangle) {
		t.Fatalf("cap mark kinds = %d,%d, want rectangle,rectangle", spec.Children[1].Kind, spec.Children[2].Kind)
	}
}

func TestBoxPlotMarkSpec(t *testing.T) {
	spec := BoxPlotMark(
		YString("Benchmark", "Encode/small"),
		Minimum(NumberValue("Min", 10)),
		Q1(NumberValue("Q1", 12)),
		Median(NumberValue("Median", 14)),
		Q3(NumberValue("Q3", 16)),
		Maximum(NumberValue("Max", 19)),
	).toSpec()

	if spec.Kind != int32(markGroup) {
		t.Fatalf("kind = %d, want %d", spec.Kind, markGroup)
	}
	if len(spec.Children) != 5 {
		t.Fatalf("children = %d, want 5", len(spec.Children))
	}
}

func TestAxisValueLabelsSpec(t *testing.T) {
	spec := Chart(
		PointMark(
			XFloat("Step", 1),
			YFloat("Loss", 0.42),
		),
	).ChartXAxis(AxisMarks(
		AxisGridLines(),
		AxisValueLabels(PercentFormat(1)),
	)).builder.toSpec()

	if spec.XAxis == nil {
		t.Fatal("xAxis = nil")
	}
	if got, want := spec.XAxis.ValueFormat.Kind, int32(valueFormatPercent); got != want {
		t.Fatalf("value format kind = %d, want %d", got, want)
	}
	if got, want := spec.XAxis.ValueFormat.Precision, 1; got != want {
		t.Fatalf("value format precision = %d, want %d", got, want)
	}
}

func TestPointMarkFidelityModsSpec(t *testing.T) {
	spec := PointMark(
		XFloat("Run", 7),
		YFloat("Score", 0.91),
	).
		Symbol(SymbolCircle).
		SymbolStroke(swiftui.RGB(1, 1, 1), 1.5).
		AnnotationOverflow(AnnotationOverflowFitPlot, AnnotationOverflowDisabled).
		AnnotationOffset(6, -4).
		AnnotationDeclutterLatest(3).
		AccessibilityLabel("kept experiment").
		AccessibilityValue("0.91").
		AccessibilityHidden(true).
		toSpec()

	var kinds []int32
	for _, mod := range spec.Mods {
		kinds = append(kinds, mod.Kind)
	}
	if len(kinds) == 0 {
		t.Fatal("mods = 0, want symbol/accessibility mods")
	}

	var foundStroke, foundOverflow, foundOffset, foundDeclutter, foundA11yLabel, foundA11yValue, foundA11yHidden bool
	for _, mod := range spec.Mods {
		switch mod.Kind {
		case int32(modSymbolStroke):
			foundStroke = true
			if mod.FloatV != 1.5 {
				t.Fatalf("symbol stroke width = %v, want 1.5", mod.FloatV)
			}
		case int32(modAnnotationOverflow):
			foundOverflow = true
		case int32(modAnnotationOffset):
			foundOffset = true
			if mod.FloatX != 6 || mod.FloatY != -4 {
				t.Fatalf("annotation offset = (%v,%v), want (6,-4)", mod.FloatX, mod.FloatY)
			}
		case int32(modAnnotationDeclutterLatest):
			foundDeclutter = true
			if mod.IntV != 3 {
				t.Fatalf("annotation declutter limit = %d, want 3", mod.IntV)
			}
		case int32(modAccessibilityLabel):
			foundA11yLabel = true
			if mod.Value != "kept experiment" {
				t.Fatalf("accessibility label = %q, want kept experiment", mod.Value)
			}
		case int32(modAccessibilityValue):
			foundA11yValue = true
			if mod.Value != "0.91" {
				t.Fatalf("accessibility value = %q, want 0.91", mod.Value)
			}
		case int32(modAccessibilityHidden):
			foundA11yHidden = true
			if mod.IntV != 1 {
				t.Fatalf("accessibility hidden = %d, want 1", mod.IntV)
			}
		}
	}
	if !foundStroke {
		t.Fatal("missing symbol stroke mod")
	}
	if !foundOverflow {
		t.Fatal("missing annotation overflow mod")
	}
	if !foundOffset {
		t.Fatal("missing annotation offset mod")
	}
	if !foundDeclutter {
		t.Fatal("missing annotation declutter mod")
	}
	if !foundA11yLabel {
		t.Fatal("missing accessibility label mod")
	}
	if !foundA11yValue {
		t.Fatal("missing accessibility value mod")
	}
	if !foundA11yHidden {
		t.Fatal("missing accessibility hidden mod")
	}
}

func TestExcludeFromLegendResolvesScaleBackedStyle(t *testing.T) {
	spec := Chart(
		PointMark(
			XFloat("Step", 1),
			YFloat("Score", 0.8),
		).ForegroundStyleBy("Series", "kept").ExcludeFromLegend(),
	).ChartForegroundStyleScale(
		StyleScale("kept", swiftui.RGB(0.2, 0.7, 0.3)),
	).builder.toSpec()

	if len(spec.Marks) != 1 {
		t.Fatalf("marks = %d, want 1", len(spec.Marks))
	}
	var hasFixed, hasLegendKey bool
	for _, mod := range spec.Marks[0].Mods {
		switch mod.Kind {
		case int32(modForegroundStyle):
			hasFixed = true
		case int32(modForegroundStyleBy):
			hasLegendKey = true
		}
	}
	if !hasFixed {
		t.Fatal("missing fixed foreground style after legend exclusion")
	}
	if hasLegendKey {
		t.Fatal("unexpected foreground-style-by mod after legend exclusion")
	}
}

func TestAnnotationDeclutterByDistance(t *testing.T) {
	spec := Chart(
		PointMark(
			XFloat("Step", 1),
			YFloat("Score", 0.50),
		).TextAnnotation("older", AnnotationTop).
			AnnotationDeclutterByDistance(0.2, 0.1),
		PointMark(
			XFloat("Step", 1.05),
			YFloat("Score", 0.54),
		).TextAnnotation("latest", AnnotationTop).
			AnnotationDeclutterByDistance(0.2, 0.1),
	).builder.toSpec()

	if len(spec.Marks) != 2 {
		t.Fatalf("marks = %d, want 2", len(spec.Marks))
	}
	var olderHasAnnotation, latestHasAnnotation bool
	for _, mod := range spec.Marks[0].Mods {
		if mod.Kind == int32(modAnnotation) {
			olderHasAnnotation = true
		}
	}
	for _, mod := range spec.Marks[1].Mods {
		if mod.Kind == int32(modAnnotation) {
			latestHasAnnotation = true
		}
	}
	if olderHasAnnotation {
		t.Fatal("older mark annotation unexpectedly preserved")
	}
	if !latestHasAnnotation {
		t.Fatal("latest mark annotation unexpectedly removed")
	}
}

func TestAnnotationCollisionKeepPriority(t *testing.T) {
	spec := Chart(
		PointMark(
			XFloat("Step", 1.00),
			YFloat("Score", 0.50),
		).TextAnnotation("newer", AnnotationTop).
			AnnotationCollision(AnnotationCollisionKeepPriority).
			AnnotationPriority(1),
		PointMark(
			XFloat("Step", 1.00),
			YFloat("Score", 0.50),
		).TextAnnotation("older-priority", AnnotationTop).
			AnnotationCollision(AnnotationCollisionKeepPriority).
			AnnotationPriority(3),
	).ChartXScaleDomain(NumberDomain(0.9, 1.1)).
		ChartYScaleDomain(NumberDomain(0.48, 0.52)).
		builder.toSpec()

	if len(spec.Marks) != 2 {
		t.Fatalf("marks = %d, want 2", len(spec.Marks))
	}
	var newerHasAnnotation, olderHasAnnotation bool
	for _, mod := range spec.Marks[0].Mods {
		if mod.Kind == int32(modAnnotation) {
			newerHasAnnotation = true
		}
	}
	for _, mod := range spec.Marks[1].Mods {
		if mod.Kind == int32(modAnnotation) {
			olderHasAnnotation = true
		}
	}
	if newerHasAnnotation {
		t.Fatal("newer lower-priority annotation unexpectedly preserved")
	}
	if !olderHasAnnotation {
		t.Fatal("older higher-priority annotation unexpectedly removed")
	}
}

func TestAnnotationOffsetChartCreatesAnchor(t *testing.T) {
	spec := Chart(
		PointMark(
			XFloat("Step", 10),
			YFloat("Score", 0.7),
		).TextAnnotation("kept", AnnotationTop).
			AnnotationOffsetChart(0.5, 0.1),
	).builder.toSpec()

	if len(spec.Marks) != 2 {
		t.Fatalf("marks = %d, want 2", len(spec.Marks))
	}
	var sourceHasAnnotation bool
	for _, mod := range spec.Marks[0].Mods {
		if mod.Kind == int32(modAnnotation) {
			sourceHasAnnotation = true
		}
	}
	if sourceHasAnnotation {
		t.Fatal("source mark annotation unexpectedly preserved")
	}
	var anchorHasAnnotation, anchorHidden bool
	if got := len(spec.Marks[1].Dims); got < 2 {
		t.Fatalf("anchor dims = %d, want at least 2", got)
	}
	if got, want := spec.Marks[1].Dims[0].Value.Number, 10.5; got != want {
		t.Fatalf("anchor x = %v, want %v", got, want)
	}
	if got, want := spec.Marks[1].Dims[1].Value.Number, 0.8; got < want-1e-9 || got > want+1e-9 {
		t.Fatalf("anchor y = %v, want %v", got, want)
	}
	for _, mod := range spec.Marks[1].Mods {
		switch mod.Kind {
		case int32(modAnnotation):
			anchorHasAnnotation = true
		case int32(modAccessibilityHidden):
			anchorHidden = mod.IntV == 1
		}
	}
	if !anchorHasAnnotation {
		t.Fatal("anchor annotation missing")
	}
	if !anchorHidden {
		t.Fatal("anchor accessibility hidden mod missing")
	}
}

func TestSymbolSizeConvenience(t *testing.T) {
	d := PointMark(
		XFloat("Step", 1),
		YFloat("Score", 0.5),
	).SymbolDiameter(10).toSpec()
	r := PointMark(
		XFloat("Step", 2),
		YFloat("Score", 0.6),
	).SymbolRadius(5).toSpec()

	var diamArea, radiusArea float64
	for _, mod := range d.Mods {
		if mod.Kind == int32(modSymbolSize) {
			diamArea = mod.FloatV
		}
	}
	for _, mod := range r.Mods {
		if mod.Kind == int32(modSymbolSize) {
			radiusArea = mod.FloatV
		}
	}
	if diamArea == 0 || radiusArea == 0 {
		t.Fatal("missing symbol size mods")
	}
	if diamArea != radiusArea {
		t.Fatalf("diameter area %v != radius area %v, want equal", diamArea, radiusArea)
	}
}

func TestAxisTickHelpers(t *testing.T) {
	cfg := AxisMarks(
		AxisTickCount(5),
		AxisTickMinimumStride(0.1, 4),
		AxisTickStrideRounded(0.25, true, false),
	)
	spec := cfg.toSpec()
	if got, want := spec.Values.Kind, int32(axisValuesNumericStride); got != want {
		t.Fatalf("axis values kind = %d, want %d", got, want)
	}
	if got, want := spec.Values.Step, 0.25; got != want {
		t.Fatalf("axis values step = %v, want %v", got, want)
	}
	if !spec.Values.RoundLowerBound || spec.Values.RoundUpperBound {
		t.Fatalf("axis rounding = (%v,%v), want (true,false)", spec.Values.RoundLowerBound, spec.Values.RoundUpperBound)
	}
}

func TestAxisStyleSpec(t *testing.T) {
	spec := Chart(
		PointMark(XFloat("Step", 1), YFloat("Score", 0.5)),
	).ChartXAxisStyle(
		AxisStyleForeground(swiftui.RGB(0.3, 0.4, 0.5)),
		AxisStyleOpacity(0.85),
	).ChartYAxisStyle(
		AxisStyleBackgroundColor(swiftui.RGBA(0.1, 0.1, 0.1, 0.05)),
	).builder.toSpec()

	if spec.XAxisStyle == nil || len(spec.XAxisStyle.Entries) != 2 {
		t.Fatalf("x axis style = %#v, want 2 entries", spec.XAxisStyle)
	}
	if got, want := spec.XAxisStyle.Entries[0].Kind, int32(axisStyleForegroundColor); got != want {
		t.Fatalf("x axis style first kind = %d, want %d", got, want)
	}
	if got, want := spec.XAxisStyle.Entries[1].Opacity, 0.85; got != want {
		t.Fatalf("x axis opacity = %v, want %v", got, want)
	}
	if spec.YAxisStyle == nil || len(spec.YAxisStyle.Entries) != 1 {
		t.Fatalf("y axis style = %#v, want 1 entry", spec.YAxisStyle)
	}
	if got, want := spec.YAxisStyle.Entries[0].Kind, int32(axisStyleBackgroundColor); got != want {
		t.Fatalf("y axis style kind = %d, want %d", got, want)
	}
}

func TestPlotInsetSpec(t *testing.T) {
	spec := Chart(
		PointMark(XFloat("Step", 1), YFloat("Score", 0.5)),
	).ChartPlotStyle(
		PlotInset(6, 10, 8, 12),
	).builder.toSpec()

	if len(spec.Plot) != 1 {
		t.Fatalf("plot styles = %d, want 1", len(spec.Plot))
	}
	entry := spec.Plot[0]
	if got, want := entry.Kind, int32(plotStyleInset); got != want {
		t.Fatalf("plot style kind = %d, want %d", got, want)
	}
	if entry.Top != 6 || entry.Right != 10 || entry.Bottom != 8 || entry.Left != 12 {
		t.Fatalf("plot inset = (%v,%v,%v,%v), want (6,10,8,12)", entry.Top, entry.Right, entry.Bottom, entry.Left)
	}
}

func TestVectorizedPlotsUseVectorizedKinds(t *testing.T) {
	point := PointPlot(
		PlotPoint(NumberValue("Step", 1), NumberValue("Score", 0.5)),
	).toSpec()
	if got, want := point.Kind, int32(markPointPlot); got != want {
		t.Fatalf("point plot kind = %d, want %d", got, want)
	}
	if got, want := len(point.PointData), 1; got != want {
		t.Fatalf("point plot data = %d, want %d", got, want)
	}

	rect := RectanglePlot(
		PlotRectangleXY(NumberValue("X", 1), NumberValue("Y", 2), MarkDimensionFixed(8), MarkDimensionInset(2)),
	).toSpec()
	if got, want := rect.Kind, int32(markRectanglePlot); got != want {
		t.Fatalf("rectangle plot kind = %d, want %d", got, want)
	}
	if got, want := len(rect.RectangleData), 1; got != want {
		t.Fatalf("rectangle plot data = %d, want %d", got, want)
	}

	rule := RulePlot(
		PlotRuleXRange(NumberValue("Start", 1), NumberValue("End", 3), NumberValue("Y", 2)),
	).toSpec()
	if got, want := rule.Kind, int32(markRulePlot); got != want {
		t.Fatalf("rule plot kind = %d, want %d", got, want)
	}
	if got, want := len(rule.RuleData), 1; got != want {
		t.Fatalf("rule plot data = %d, want %d", got, want)
	}
}

func TestInteractionSpecs(t *testing.T) {
	x := NewOptionalNumberState(0, false)
	y := NewOptionalNumberState(0, false)
	xr := NewNumberRangeState(0, 0, false)
	scrollX := NewNumberState(2)
	scrollY := NewNumberState(5)
	defer x.Release()
	defer y.Release()
	defer xr.Release()
	defer scrollX.Release()
	defer scrollY.Release()

	spec := Chart(
		PointMark(XFloat("Step", 1), YFloat("Score", 0.5)),
	).ChartXSelection(x).
		ChartYSelection(y).
		ChartXSelectionRange(xr).
		ChartOverlay(CrosshairOverlay(x, y, swiftui.RGB(0.2, 0.4, 0.6), 1)).
		ChartBackground(SelectionBandBackgroundX(xr, swiftui.RGBA(0.2, 0.4, 0.6, 0.15))).
		ChartGesture(DragXSelectionGesture(x, 3)).
		ChartScrollTargetBehavior(ValueAlignedXYScrollTarget(10, 5, nil, nil, ValueAlignedLimitAlways)).
		ChartScrollPositionX(scrollX).
		ChartScrollPositionY(scrollY).
		ChartXVisibleDomain(VisibleDomainNumeric(20)).
		ChartYVisibleDomain(VisibleDomainTime(2 * time.Minute)).
		builder.toSpec()

	if spec.XSelection == nil || spec.YSelection == nil || spec.XSelectionRange == nil {
		t.Fatal("missing serialized selection state")
	}
	if spec.ScrollTarget == nil || spec.ScrollTarget.Kind != int32(scrollTargetBehaviorValueAlignedXYUnit) {
		t.Fatalf("scroll target = %#v, want value-aligned xy", spec.ScrollTarget)
	}
	if spec.XScrollBinding == nil || spec.YScrollBinding == nil {
		t.Fatal("missing serialized scroll bindings")
	}
	if spec.XVisibleLength == nil || spec.XVisibleLength.Number != 20 {
		t.Fatalf("x visible length = %#v, want 20", spec.XVisibleLength)
	}
	if spec.YVisibleLength == nil || spec.YVisibleLength.Kind != visibleDomainTime {
		t.Fatalf("y visible length = %#v, want time length", spec.YVisibleLength)
	}
	if got := len(spec.Overlays); got != 1 {
		t.Fatalf("overlays = %d, want 1", got)
	}
	if got := len(spec.Backgrounds); got != 1 {
		t.Fatalf("backgrounds = %d, want 1", got)
	}
	if spec.Gesture == nil || spec.Gesture.Kind != int32(proxyGestureDragXValue) {
		t.Fatalf("gesture = %#v, want drag x value", spec.Gesture)
	}
}

func TestInitialYScrollHelpers(t *testing.T) {
	date := time.Unix(1_700_000_000, 0)
	numberSpec := Chart().ChartScrollPositionInitialYNumber(7).builder.toSpec()
	if numberSpec.ScrollPosition == nil || numberSpec.ScrollPosition.Axis != int32(scrollPositionAxisY) || numberSpec.ScrollPosition.Number != 7 {
		t.Fatalf("numeric y scroll position = %#v, want y=7", numberSpec.ScrollPosition)
	}
	dateSpec := Chart().ChartScrollPositionInitialYDate(date).builder.toSpec()
	if dateSpec.ScrollPosition == nil || dateSpec.ScrollPosition.Axis != int32(scrollPositionAxisY) || dateSpec.ScrollPosition.Time != date.UnixMilli() {
		t.Fatalf("date y scroll position = %#v, want y date", dateSpec.ScrollPosition)
	}
	lengthSpec := Chart().ChartYVisibleDomainLength(12).builder.toSpec()
	if lengthSpec.YVisibleLength == nil || lengthSpec.YVisibleLength.Number != 12 {
		t.Fatalf("y visible length = %#v, want 12", lengthSpec.YVisibleLength)
	}
}

func TestReferenceLineYSpec(t *testing.T) {
	spec := ReferenceLineY(NumberValue("SLO", 180), "slo 180 ms").
		ForegroundStyle(swiftui.RGB(1, 0, 0)).
		LineStyle(Stroke(1.5, 4, 4)).
		toSpec()

	if got, want := spec.Kind, int32(markRule); got != want {
		t.Fatalf("kind = %d, want %d", got, want)
	}

	var foundAnnotation, foundLineStyle bool
	for _, mod := range spec.Mods {
		switch mod.Kind {
		case int32(modAnnotation):
			foundAnnotation = true
			if mod.ViewPtr == "" {
				t.Fatal("annotation view ptr = empty, want non-empty")
			}
		case int32(modLineStyle):
			foundLineStyle = true
			if mod.FloatV != 1.5 {
				t.Fatalf("line style width = %v, want 1.5", mod.FloatV)
			}
		}
	}
	if !foundAnnotation {
		t.Fatal("missing annotation mod")
	}
	if !foundLineStyle {
		t.Fatal("missing line style mod")
	}
}
