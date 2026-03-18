//go:build darwin
// +build darwin

// Command benchview renders a live benchstat comparison window.
//
// It reads a baseline benchmark file from the first argument and consumes
// benchmark output from standard input, updating the comparison as new results
// arrive.
//
// Usage:
//
//	go test -bench=. -count=10 | tee bench2.txt | benchview bench1.txt
package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"

	"github.com/tmc/swiftui"
	swcharts "github.com/tmc/swiftui/charts"
	"golang.org/x/perf/benchstat"
)

func init() { runtime.LockOSThread() }

type tableView struct {
	Metric      string
	Configs     []string
	OldNewDelta bool
	Compared    int
	Improved    int
	Regressed   int
	Unchanged   int
	Insight     string
	Domain      metricDomain
	Rows        []rowView
}

type rowView struct {
	Name       string
	Values     []string
	Stats      []metricStat
	MeanValues []float64
	SpreadPct  float64
	PctDelta   float64
	Delta      string
	Note       string
	Change     int
}

type metricStat struct {
	Min  float64
	Mean float64
	Max  float64
	OK   bool
}

type metricDomain struct {
	MaxValue float64
	MaxDelta float64
}

type highlightView struct {
	Title     string
	Metric    string
	Benchmark string
	Detail    string
	Change    int
}

type rowHighlight struct {
	Valid     bool
	Metric    string
	Benchmark string
	Detail    string
	Value     float64
	Change    int
}

type comparisonView struct {
	Tables         []tableView
	Inputs         int
	Rows           int
	Compared       int
	Improved       int
	Regressed      int
	Unchanged      int
	LiveResults    int
	Mode           string
	HasStreaming   bool
	StreamingLabel string
	Highlights     []highlightView
}

type displayPrefs struct {
	ShowSummaryCards bool
	ShowChangeBar    bool
	ShowHighlights   bool
	ShowAbsolute     bool
	ShowDelta        bool
	ShowRows         bool
	ShowInsights     bool
	CompactRows      bool
	RowLimit         int
}

type inputSpec struct {
	Label    string
	Path     string
	Stream   bool
	Explicit bool
	Data     []byte
}

var (
	viewMu      sync.Mutex
	currentView comparisonView
)

func main() {
	inputs, err := parseInputs(os.Args[1:], stdinAvailable())
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n\n", err)
		printUsage(os.Args[0])
		os.Exit(1)
	}
	if err := loadInputs(inputs); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	streamIndex := findStreamInput(inputs)
	windowWidth := benchviewWidth(len(inputs))

	updateTick := swiftui.NewIntState(0)
	status := swiftui.NewStringState(initialStatus(inputs))
	liveCount := swiftui.NewIntState(0)
	inputCount := swiftui.NewIntState(len(inputs))
	sourceSummary := swiftui.NewStringState(summarizeInputs(inputs))
	streamSummary := swiftui.NewStringState(streamLabel(inputs))
	showSummaryCards := swiftui.NewIntState(1)
	showChangeBar := swiftui.NewIntState(1)
	showHighlights := swiftui.NewIntState(1)
	showAbsolute := swiftui.NewIntState(1)
	showDelta := swiftui.NewIntState(1)
	showRows := swiftui.NewIntState(1)
	showInsights := swiftui.NewIntState(1)
	compactRows := swiftui.NewIntState(1)
	rowLimit := swiftui.NewIntState(5)
	showDisplayControls := swiftui.NewIntState(0)

	if streamIndex >= 0 {
		liveCount.Set(countBenchmarkLines(inputs[streamIndex].Data))
	}
	setView(buildComparison(inputs))

	if streamIndex >= 0 {
		go func() {
			scanner := bufio.NewScanner(os.Stdin)
			scanner.Buffer(make([]byte, 64*1024), 1024*1024)
			var incoming bytes.Buffer
			incoming.Write(inputs[streamIndex].Data)
			results := countBenchmarkLines(inputs[streamIndex].Data)
			nextRefresh := nextStreamRefresh(results)
			tick := updateTick.Get()
			refresh := func(streamStatus string) {
				inputs[streamIndex].Data = append(inputs[streamIndex].Data[:0], incoming.Bytes()...)
				setView(buildComparison(inputs))
				status.Set(streamStatus)
				tick++
				updateTick.Set(tick)
			}
			for scanner.Scan() {
				line := scanner.Text()
				incoming.WriteString(line)
				incoming.WriteByte('\n')
				if strings.HasPrefix(line, "Benchmark") {
					results++
					liveCount.Set(results)
					if results == 1 || results >= nextRefresh {
						refresh("Streaming benchmark input")
						nextRefresh = nextStreamRefresh(results)
					}
				}
			}
			if err := scanner.Err(); err != nil {
				refresh("Stopped reading stdin")
				streamSummary.Set(err.Error())
				return
			}
			refresh("Input complete")
		}()
	}

	bumpDisplay := func() {
		updateTick.Set(updateTick.Get() + 1)
	}

	swiftui.Run(swiftui.AppConfig{
		Title:  "Benchview",
		Width:  windowWidth,
		Height: 720,
	}, benchDashboardTab(
		updateTick,
		inputCount,
		sourceSummary,
		status,
		liveCount,
		streamSummary,
		showSummaryCards,
		showChangeBar,
		showHighlights,
		showAbsolute,
		showDelta,
		showRows,
		showInsights,
		compactRows,
		rowLimit,
		showDisplayControls,
		bumpDisplay,
	))
}

func buildComparison(inputs []inputSpec) comparisonView {
	collection := &benchstat.Collection{}
	view := comparisonView{
		Inputs: len(inputs),
		Mode:   comparisonMode(inputs),
	}
	for _, input := range inputs {
		if input.Stream {
			view.HasStreaming = true
			view.StreamingLabel = input.Label
			break
		}
	}
	for _, input := range inputs {
		if len(bytes.TrimSpace(input.Data)) == 0 {
			continue
		}
		collection.AddConfig(input.Label, input.Data)
	}

	tables := collection.Tables()
	var bestWin, bestLoss, noisiest rowHighlight
	for _, table := range tables {
		tv := tableView{
			Metric:      table.Metric,
			Configs:     append([]string(nil), table.Configs...),
			OldNewDelta: table.OldNewDelta,
		}
		for _, row := range table.Rows {
			rv := rowView{
				Name:       row.Benchmark,
				PctDelta:   row.PctDelta,
				Delta:      row.Delta,
				Note:       row.Note,
				Change:     row.Change,
				Values:     make([]string, len(table.Configs)),
				Stats:      make([]metricStat, len(table.Configs)),
				MeanValues: make([]float64, len(table.Configs)),
			}
			for i := range rv.Values {
				rv.Values[i] = "n/a"
				if i < len(row.Metrics) && row.Metrics[i] != nil {
					rv.Values[i] = row.Metrics[i].Format(row.Scaler)
					rv.Stats[i] = metricStat{
						Min:  row.Metrics[i].Min,
						Mean: row.Metrics[i].Mean,
						Max:  row.Metrics[i].Max,
						OK:   true,
					}
					rv.MeanValues[i] = row.Metrics[i].Mean
					tv.Domain.MaxValue = math.Max(tv.Domain.MaxValue, row.Metrics[i].Max)
					if row.Metrics[i].Mean > 0 {
						rv.SpreadPct = math.Max(rv.SpreadPct, (row.Metrics[i].Max-row.Metrics[i].Min)/row.Metrics[i].Mean)
					}
				}
			}
			if !table.OldNewDelta {
				rv.Delta = ""
				rv.Note = ""
			} else {
				tv.Domain.MaxDelta = math.Max(tv.Domain.MaxDelta, math.Abs(row.PctDelta))
			}
			tv.Rows = append(tv.Rows, rv)
			view.Rows++
			if table.OldNewDelta {
				view.Compared++
				tv.Compared++
				switch rv.Change {
				case 1:
					view.Improved++
					tv.Improved++
				case -1:
					view.Regressed++
					tv.Regressed++
				default:
					view.Unchanged++
					tv.Unchanged++
				}
			}
			if rv.SpreadPct > 0 && (!noisiest.Valid || rv.SpreadPct > noisiest.Value) {
				noisiest = rowHighlight{
					Valid:     true,
					Metric:    table.Metric,
					Benchmark: rv.Name,
					Detail:    fmt.Sprintf("%s spread", formatPercent(rv.SpreadPct)),
					Value:     rv.SpreadPct,
				}
			}
			if table.OldNewDelta && rv.Change == 1 && (!bestWin.Valid || math.Abs(rv.PctDelta) > bestWin.Value) {
				bestWin = rowHighlight{
					Valid:     true,
					Metric:    table.Metric,
					Benchmark: rv.Name,
					Detail:    formatDeltaPercent(rv.PctDelta),
					Value:     math.Abs(rv.PctDelta),
					Change:    1,
				}
			}
			if table.OldNewDelta && rv.Change == -1 && (!bestLoss.Valid || math.Abs(rv.PctDelta) > bestLoss.Value) {
				bestLoss = rowHighlight{
					Valid:     true,
					Metric:    table.Metric,
					Benchmark: rv.Name,
					Detail:    formatDeltaPercent(rv.PctDelta),
					Value:     math.Abs(rv.PctDelta),
					Change:    -1,
				}
			}
		}
		if len(tv.Rows) > 0 {
			sortTableRows(&tv)
			tv.Insight = metricInsight(tv)
			view.Tables = append(view.Tables, tv)
		}
	}

	if bestWin.Valid {
		view.Highlights = append(view.Highlights, highlightView{
			Title:     "Biggest Win",
			Metric:    bestWin.Metric,
			Benchmark: bestWin.Benchmark,
			Detail:    bestWin.Detail,
			Change:    bestWin.Change,
		})
	}
	if bestLoss.Valid {
		view.Highlights = append(view.Highlights, highlightView{
			Title:     "Biggest Loss",
			Metric:    bestLoss.Metric,
			Benchmark: bestLoss.Benchmark,
			Detail:    bestLoss.Detail,
			Change:    bestLoss.Change,
		})
	}
	if noisiest.Valid {
		view.Highlights = append(view.Highlights, highlightView{
			Title:     "Widest Spread",
			Metric:    noisiest.Metric,
			Benchmark: noisiest.Benchmark,
			Detail:    noisiest.Detail,
			Change:    0,
		})
	}

	for _, input := range inputs {
		if input.Stream {
			view.LiveResults = countBenchmarkLines(input.Data)
			break
		}
	}
	return view
}

func benchDashboardTab(
	updateTick *swiftui.IntState,
	inputCount *swiftui.IntState,
	sourceSummary *swiftui.StringState,
	status *swiftui.StringState,
	liveCount *swiftui.IntState,
	streamSummary *swiftui.StringState,
	showSummaryCards, showChangeBar, showHighlights, showAbsolute, showDelta, showRows, showInsights, compactRows, rowLimit *swiftui.IntState,
	showDisplayControls *swiftui.IntState,
	onChange func(),
) swiftui.View {
	content := swiftui.ScrollView(
		swiftui.VStackSpaced(14,
			swiftui.HStack(
				swiftui.VStackSpaced(4,
					swiftui.HStack(
						swiftui.Text("Benchview").
							Font(swiftui.FontTitle).
							FontWeight(swiftui.WeightBold),
						swiftui.Spacer(),
						swiftui.Button("Display", func() {
							if showDisplayControls.Get() == 0 {
								showDisplayControls.Set(1)
							} else {
								showDisplayControls.Set(0)
							}
						}).
							Padding(8).
							Background(1, 1, 1, 0.06).
							CornerRadius(999),
					),
					swiftui.HStack(
						swiftui.Text("Benchstat review across files and optional live stdin.").
							Font(swiftui.FontCaption).
							ForegroundStyleNamed("secondary"),
						swiftui.Spacer(),
					),
				).MaxFrame(-1, 0),
			),
			metaStrip(updateTick, inputCount, sourceSummary, status, liveCount, streamSummary),

			swiftui.DynamicView(updateTick, func(_ int) swiftui.View {
				view := snapshotView()
				prefs := currentDisplayPrefs(
					showSummaryCards,
					showChangeBar,
					showHighlights,
					showAbsolute,
					showDelta,
					showRows,
					showInsights,
					compactRows,
					rowLimit,
				)
				return renderDashboard(view, prefs)
			}),
		).Padding(20),
	)
	return content.Popover(showDisplayControls, displaySettingsPopover(
		updateTick,
		showSummaryCards,
		showChangeBar,
		showHighlights,
		showAbsolute,
		showDelta,
		showRows,
		showInsights,
		compactRows,
		rowLimit,
		onChange,
	))
}

func renderDashboard(view comparisonView, prefs displayPrefs) swiftui.View {
	sections := make([]swiftui.Viewable, 0, 4)
	if prefs.ShowSummaryCards || prefs.ShowChangeBar || prefs.ShowHighlights {
		sections = append(sections, overviewStrip(view, prefs))
	}
	sections = append(sections, renderTables(view, prefs))
	return swiftui.VStackSpaced(12, sections...)
}

func overviewStrip(view comparisonView, prefs displayPrefs) swiftui.View {
	lines := make([]swiftui.Viewable, 0, 4)
	tokens := make([]swiftui.Viewable, 0, 8)
	if prefs.ShowSummaryCards {
		tokens = append(tokens, compactStat("rows", fmt.Sprintf("%d", view.Rows), 0.58, 0.62, 0.82))
		if view.HasStreaming {
			tokens = append(tokens, compactStat("stream", fmt.Sprintf("%d", view.LiveResults), 0.95, 0.6, 0.2))
		}
	}
	if prefs.ShowChangeBar && view.Compared > 0 {
		tokens = append(tokens,
			compactStat("compared", fmt.Sprintf("%d", view.Compared), 0.35, 0.65, 1.0),
			compactStat("wins", fmt.Sprintf("%d", view.Improved), 0.3, 0.8, 0.4),
			compactStat("losses", fmt.Sprintf("%d", view.Regressed), 0.9, 0.5, 0.2),
			compactStat("flat", fmt.Sprintf("%d", view.Unchanged), 0.58, 0.62, 0.82),
		)
	}
	if len(tokens) > 0 {
		lines = append(lines, swiftui.HStackSpaced(8, append(tokens, swiftui.Spacer())...))
	}
	lines = append(lines,
		swiftui.HStack(
			swiftui.Text(modeSummary(view)).
				Font(swiftui.FontCaption).
				ForegroundStyleNamed("secondary"),
			swiftui.Spacer(),
		),
	)
	if prefs.ShowChangeBar {
		if view.Compared == 0 {
			lines = append(lines,
				swiftui.HStack(
					swiftui.Text("Deltas appear when benchstat can compare exactly two populated inputs.").
						Font(swiftui.FontCaption2).
						ForegroundStyleNamed("tertiary"),
					swiftui.Spacer(),
				),
			)
		} else {
			lines = append(lines, stackedChangeBar(view.Improved, view.Regressed, view.Unchanged, 560))
		}
	}
	if prefs.ShowHighlights && len(view.Highlights) > 0 {
		for _, highlight := range view.Highlights {
			lines = append(lines, highlightLine(highlight))
		}
	}
	return swiftui.VStackSpaced(6, lines...).
		Padding(10).
		Background(1, 1, 1, 0.03).
		CornerRadius(10).
		MaxFrame(-1, 0)
}

func metaStrip(
	updateTick *swiftui.IntState,
	inputCount *swiftui.IntState,
	sourceSummary *swiftui.StringState,
	status *swiftui.StringState,
	liveCount *swiftui.IntState,
	streamSummary *swiftui.StringState,
) swiftui.View {
	return swiftui.DynamicView(updateTick, func(_ int) swiftui.View {
		tokens := []swiftui.Viewable{
			compactStat("inputs", fmt.Sprintf("%d", inputCount.Get()), 0.58, 0.62, 0.82),
		}
		if liveCount.Get() > 0 {
			tokens = append(tokens, compactStat("live", fmt.Sprintf("%d", liveCount.Get()), 0.95, 0.6, 0.2))
		}
		tokens = append(tokens, compactStat("status", status.Get(), 0.35, 0.65, 1.0))
		return swiftui.VStackSpaced(5,
			swiftui.HStackSpaced(8, append(tokens, swiftui.Spacer())...),
			swiftui.HStack(
				swiftui.Text(sourceSummary.Get()).
					Font(swiftui.FontCaption).
					ForegroundStyleNamed("secondary").
					LineLimit(1),
				swiftui.Spacer(),
				swiftui.Text(streamSummary.Get()).
					Font(swiftui.FontCaption2).
					ForegroundStyleNamed("tertiary").
					LineLimit(1),
			),
		).Padding(10).
			Background(1, 1, 1, 0.03).
			CornerRadius(10)
	})
}

func summaryStrip(view comparisonView) swiftui.View {
	tokens := []swiftui.Viewable{
		compactStat("rows", fmt.Sprintf("%d", view.Rows), 0.58, 0.62, 0.82),
	}
	if view.Compared > 0 {
		tokens = append(tokens,
			compactStat("compared", fmt.Sprintf("%d", view.Compared), 0.35, 0.65, 1.0),
			compactStat("wins", fmt.Sprintf("%d", view.Improved), 0.3, 0.8, 0.4),
			compactStat("losses", fmt.Sprintf("%d", view.Regressed), 0.9, 0.5, 0.2),
			compactStat("flat", fmt.Sprintf("%d", view.Unchanged), 0.58, 0.62, 0.82),
		)
	}
	if view.HasStreaming {
		tokens = append(tokens, compactStat("stream", fmt.Sprintf("%d", view.LiveResults), 0.95, 0.6, 0.2))
	}
	return swiftui.GroupBox("Summary",
		swiftui.VStackSpaced(6,
			swiftui.HStackSpaced(8, append(tokens, swiftui.Spacer())...),
			swiftui.HStack(
				swiftui.Text(modeSummary(view)).
					Font(swiftui.FontCaption).
					ForegroundStyleNamed("secondary"),
				swiftui.Spacer(),
			),
		).Padding(10),
	).MaxFrame(-1, 0)
}

func currentDisplayPrefs(
	showSummaryCards, showChangeBar, showHighlights, showAbsolute, showDelta, showRows, showInsights, compactRows, rowLimit *swiftui.IntState,
) displayPrefs {
	return displayPrefs{
		ShowSummaryCards: showSummaryCards.Get() != 0,
		ShowChangeBar:    showChangeBar.Get() != 0,
		ShowHighlights:   showHighlights.Get() != 0,
		ShowAbsolute:     showAbsolute.Get() != 0,
		ShowDelta:        showDelta.Get() != 0,
		ShowRows:         showRows.Get() != 0,
		ShowInsights:     showInsights.Get() != 0,
		CompactRows:      compactRows.Get() != 0,
		RowLimit:         rowLimit.Get(),
	}
}

func changeOverview(view comparisonView) swiftui.View {
	if view.Compared == 0 {
		return swiftui.GroupBox("Change",
			swiftui.VStackSpaced(6,
				swiftui.Text("Comparison deltas appear when benchstat has exactly two populated inputs for a metric.").
					Font(swiftui.FontCaption).
					ForegroundStyleNamed("secondary"),
			).Padding(10),
		).MaxFrame(-1, 0)
	}
	return swiftui.GroupBox("Change",
		swiftui.VStackSpaced(6,
			swiftui.HStack(
				swiftui.Text(fmt.Sprintf("%d compared rows", view.Compared)).
					Font(swiftui.FontCaption).
					FontWeight(swiftui.WeightSemibold).
					ForegroundStyleNamed("secondary"),
				swiftui.Spacer(),
				changeCountPill("Wins", view.Improved, 0.3, 0.8, 0.4),
				changeCountPill("Losses", view.Regressed, 0.9, 0.5, 0.2),
				changeCountPill("Flat", view.Unchanged, 0.58, 0.62, 0.82),
			),
			stackedChangeBar(view.Improved, view.Regressed, view.Unchanged, 560),
		).Padding(10),
	).MaxFrame(-1, 0)
}

func highlightOverview(view comparisonView) swiftui.View {
	if len(view.Highlights) == 0 {
		return swiftui.GroupBox("Highlights",
			swiftui.VStackSpaced(6,
				swiftui.HStack(
					swiftui.Text(modeSummary(view)).
						Font(swiftui.FontCaption).
						ForegroundStyleNamed("secondary"),
					swiftui.Spacer(),
				),
			).Padding(10),
		).MaxFrame(-1, 0)
	}
	lines := make([]swiftui.Viewable, 0, len(view.Highlights)+1)
	lines = append(lines,
		swiftui.HStack(
			swiftui.Text(modeSummary(view)).
				Font(swiftui.FontCaption).
				ForegroundStyleNamed("secondary"),
			swiftui.Spacer(),
		),
	)
	for _, highlight := range view.Highlights {
		lines = append(lines, highlightLine(highlight))
	}
	return swiftui.GroupBox("Highlights",
		swiftui.VStackSpaced(5, lines...).Padding(10),
	).MaxFrame(-1, 0)
}

func renderTables(view comparisonView, prefs displayPrefs) swiftui.View {
	if len(view.Tables) == 0 {
		return swiftui.GroupBox("Comparison",
			swiftui.VStackSpaced(8,
				swiftui.Text("Waiting for benchmark samples.").
					Font(swiftui.FontCallout),
				swiftui.Text("Provide one or more files, and use `-` or a pipe to stream stdin into the view.").
					Font(swiftui.FontCaption).
					ForegroundStyleNamed("secondary"),
			).Padding(16),
		).MaxFrame(-1, 0)
	}

	groups := make([]swiftui.Viewable, 0, len(view.Tables))
	for _, table := range view.Tables {
		table := table
		displayTable := table
		if prefs.RowLimit > 0 && len(displayTable.Rows) > prefs.RowLimit {
			displayTable.Rows = append([]rowView(nil), displayTable.Rows[:prefs.RowLimit]...)
		}
		rows := make([]swiftui.Viewable, 0, len(displayTable.Rows)+4)
		rows = append(rows, metricSummary(displayTable, prefs))
		if len(displayTable.Rows) != len(table.Rows) {
			rows = append(rows,
				swiftui.HStack(
					swiftui.Text(fmt.Sprintf("Showing %d of %d rows", len(displayTable.Rows), len(table.Rows))).
						Font(swiftui.FontCaption2).
						ForegroundStyleNamed("secondary"),
					swiftui.Spacer(),
				),
			)
		}
		if prefs.ShowRows {
			rows = append(rows, swiftui.Divider())
			rows = append(rows, comparisonHeaderRow(displayTable))
			for _, row := range displayTable.Rows {
				rows = append(rows, swiftui.Divider())
				rows = append(rows, comparisonRow(displayTable, row, prefs))
			}
		}
		groups = append(groups, swiftui.GroupBox(displayTable.Metric,
			swiftui.VStackSpaced(8, rows...).Padding(12),
		).MaxFrame(-1, 0))
	}
	return swiftui.VStackSpaced(12, groups...)
}

func metricSummary(table tableView, prefs displayPrefs) swiftui.View {
	views := []swiftui.Viewable{
		swiftui.HStack(
			swiftui.Text(metricSummaryTitle(table)).
				Font(swiftui.FontCallout).
				FontWeight(swiftui.WeightSemibold),
			swiftui.Spacer(),
		),
		configChipRow(table.Configs),
	}
	if prefs.ShowInsights {
		views = append(views,
			swiftui.HStack(
				swiftui.Text(table.Insight).
					Font(swiftui.FontCaption).
					ForegroundStyleNamed("secondary"),
				swiftui.Spacer(),
			),
		)
	}
	if prefs.ShowAbsolute {
		if chart := benchmarkMetricChart(table); chart.Pointer() != 0 {
			views = append(views, chart)
		}
	}
	if prefs.ShowDelta {
		if chart := benchmarkDeltaChart(table); chart.Pointer() != 0 {
			views = append(views, chart)
		}
	}
	if table.OldNewDelta && prefs.ShowChangeBar {
		views = append(views,
			stackedChangeBar(table.Improved, table.Regressed, table.Unchanged, 260),
			swiftui.HStackSpaced(10,
				changeCountPill("Wins", table.Improved, 0.3, 0.8, 0.4),
				changeCountPill("Losses", table.Regressed, 0.9, 0.5, 0.2),
				changeCountPill("Flat", table.Unchanged, 0.58, 0.62, 0.82),
			),
		)
	}
	return swiftui.VStackSpaced(8, views...)
}

func comparisonHeaderRow(table tableView) swiftui.View {
	views := []swiftui.Viewable{headerText("Benchmark", 0)}
	width := valueColumnWidth(len(table.Configs))
	for _, config := range table.Configs {
		views = append(views, headerText(config, width))
	}
	if table.OldNewDelta {
		views = append(views, headerText("Delta", 90))
		views = append(views, headerText("Note", 110))
	}
	return swiftui.HStackSpaced(12, views...)
}

func comparisonRow(table tableView, row rowView, prefs displayPrefs) swiftui.View {
	r, g, b := changeColor(row.Change)
	views := []swiftui.Viewable{
		benchmarkCell(table, row, prefs),
	}
	width := valueColumnWidth(len(table.Configs))
	for _, value := range row.Values {
		views = append(views, monoText(value, width))
	}
	if table.OldNewDelta {
		delta := row.Delta
		if delta == "" {
			delta = "n/a"
		}
		note := row.Note
		if note == "" {
			note = " "
		}
		views = append(views,
			swiftui.Text(delta).
				Font(swiftui.FontCaption).
				FontWeight(swiftui.WeightSemibold).
				MonospacedDigit().
				ForegroundStyle(r, g, b, 1.0).
				Frame(90, 0),
			swiftui.Text(note).
				Font(swiftui.FontCaption).
				ForegroundStyleNamed("secondary").
				Frame(110, 0),
		)
	}
	padding := 10.0
	if prefs.CompactRows {
		padding = 6
	}
	return swiftui.HStackSpaced(12, views...).
		Padding(padding).
		Background(r, g, b, rowTint(table, row)).
		CornerRadius(10)
}

func displaySettingsPopover(
	updateTick *swiftui.IntState,
	showSummaryCards, showChangeBar, showHighlights, showAbsolute, showDelta, showRows, showInsights, compactRows, rowLimit *swiftui.IntState,
	onChange func(),
) swiftui.View {
	return swiftui.ScrollView(
		swiftui.VStackSpaced(10,
			swiftui.HStack(
				swiftui.Text("Display").
					Font(swiftui.FontHeadline).
					FontWeight(swiftui.WeightSemibold),
				swiftui.Spacer(),
			),
			swiftui.GroupBox("Sections",
				swiftui.VStackSpaced(6,
					swiftui.Toggle("Summary", showSummaryCards, onChange),
					swiftui.Toggle("Change", showChangeBar, onChange),
					swiftui.Toggle("Highlights", showHighlights, onChange),
					swiftui.Toggle("Insights", showInsights, onChange),
				).Padding(10),
			),
			swiftui.GroupBox("Metrics",
				swiftui.VStackSpaced(6,
					swiftui.Toggle("Absolute", showAbsolute, onChange),
					swiftui.Toggle("Delta", showDelta, onChange),
					swiftui.Toggle("Rows", showRows, onChange),
					swiftui.Toggle("Compact", compactRows, onChange),
					swiftui.Stepper("Rows per metric", rowLimit, 3, 12, onChange),
				).Padding(10),
			),
			swiftui.GroupBox("Current",
				swiftui.DynamicView(updateTick, func(_ int) swiftui.View {
					prefs := currentDisplayPrefs(
						showSummaryCards,
						showChangeBar,
						showHighlights,
						showAbsolute,
						showDelta,
						showRows,
						showInsights,
						compactRows,
						rowLimit,
					)
					return swiftui.VStackSpaced(6,
						settingsLine("Summary", onOffLabel(prefs.ShowSummaryCards)),
						settingsLine("Change", onOffLabel(prefs.ShowChangeBar)),
						settingsLine("Highlights", onOffLabel(prefs.ShowHighlights)),
						settingsLine("Absolute", onOffLabel(prefs.ShowAbsolute)),
						settingsLine("Delta", onOffLabel(prefs.ShowDelta)),
						settingsLine("Rows", onOffLabel(prefs.ShowRows)),
						settingsLine("Density", map[bool]string{true: "Compact", false: "Comfortable"}[prefs.CompactRows]),
						settingsLine("Per metric", fmt.Sprintf("%d", prefs.RowLimit)),
					).Padding(10)
				}),
			),
		).Padding(14),
	)
}

func headerText(label string, width float64) swiftui.View {
	view := swiftui.Text(label).
		Font(swiftui.FontCaption).
		FontWeight(swiftui.WeightSemibold).
		ForegroundStyleNamed("secondary")
	if width > 0 {
		return view.Frame(width, 0).AsView()
	}
	return view.MaxFrame(-1, 0)
}

func monoText(value string, width float64) swiftui.View {
	return swiftui.ZStack(
		swiftui.RoundedRectangle(8).
			Fill(1, 1, 1, 0.04).
			Frame(width, 30).
			AsView(),
		swiftui.Text(value).
			Font(swiftui.FontCaption).
			MonospacedDigit().
			Frame(width-12, 0).
			AsView(),
	)
}

func compactStat(label, value string, r, g, b float64) swiftui.View {
	return swiftui.HStackSpaced(5,
		swiftui.Text(label).
			Font(swiftui.FontCaption2).
			ForegroundStyleNamed("secondary"),
		swiftui.Text(value).
			Font(swiftui.FontCaption).
			FontWeight(swiftui.WeightSemibold).
			MonospacedDigit().
			LineLimit(1),
	).Padding(6).
		Background(r, g, b, 0.14).
		CornerRadius(999)
}

func highlightLine(highlight highlightView) swiftui.View {
	r, g, b := changeColor(highlight.Change)
	return swiftui.HStackSpaced(8,
		swiftui.Text(highlight.Title).
			Font(swiftui.FontCaption2).
			FontWeight(swiftui.WeightSemibold).
			ForegroundStyle(r, g, b, 1.0),
		swiftui.Text(highlight.Benchmark).
			Font(swiftui.FontCaption).
			FontWeight(swiftui.WeightSemibold).
			LineLimit(1),
		swiftui.Text(highlight.Detail).
			Font(swiftui.FontCaption).
			MonospacedDigit(),
		swiftui.Text(highlight.Metric).
			Font(swiftui.FontCaption2).
			ForegroundStyleNamed("secondary").
			LineLimit(1),
		swiftui.Spacer(),
	)
}

func stringCard(icon, label, value, note string, r, g, b float64) swiftui.View {
	return swiftui.VStackSpaced(6,
		swiftui.HStack(
			swiftui.Image(icon).
				ForegroundStyle(r, g, b, 1.0).
				ImageScale(swiftui.ImageScaleSmall),
			swiftui.Spacer(),
		),
		swiftui.HStack(
			swiftui.Text(value).
				Font(swiftui.FontTitle3).
				FontWeight(swiftui.WeightBold).
				LineLimit(1),
			swiftui.Spacer(),
		),
		swiftui.HStack(
			swiftui.Text(label).
				Font(swiftui.FontCaption).
				ForegroundStyleNamed("secondary"),
			swiftui.Spacer(),
		),
		swiftui.HStack(
			swiftui.Text(note).
				Font(swiftui.FontCaption2).
				ForegroundStyleNamed("tertiary"),
			swiftui.Spacer(),
		),
	).Padding(12).
		Background(0.2, 0.2, 0.25, 0.42).
		CornerRadius(12).
		MaxFrame(-1, 0)
}

func intCard(icon, label string, state *swiftui.IntState, note string, r, g, b float64) swiftui.View {
	return swiftui.VStackSpaced(6,
		swiftui.HStack(
			swiftui.Image(icon).
				ForegroundStyle(r, g, b, 1.0).
				ImageScale(swiftui.ImageScaleSmall),
			swiftui.Spacer(),
		),
		swiftui.HStack(
			swiftui.TextFrom(state).
				Font(swiftui.FontTitle3).
				FontWeight(swiftui.WeightBold).
				MonospacedDigit(),
			swiftui.Spacer(),
		),
		swiftui.HStack(
			swiftui.Text(label).
				Font(swiftui.FontCaption).
				ForegroundStyleNamed("secondary"),
			swiftui.Spacer(),
		),
		swiftui.HStack(
			swiftui.Text(note).
				Font(swiftui.FontCaption2).
				ForegroundStyleNamed("tertiary"),
			swiftui.Spacer(),
		),
	).Padding(12).
		Background(0.2, 0.2, 0.25, 0.42).
		CornerRadius(12).
		MaxFrame(-1, 0)
}

func stateStringCard(icon, label string, state *swiftui.StringState, note string, r, g, b float64) swiftui.View {
	return swiftui.VStackSpaced(6,
		swiftui.HStack(
			swiftui.Image(icon).
				ForegroundStyle(r, g, b, 1.0).
				ImageScale(swiftui.ImageScaleSmall),
			swiftui.Spacer(),
		),
		swiftui.HStack(
			swiftui.TextFromString(state).
				Font(swiftui.FontCallout).
				FontWeight(swiftui.WeightSemibold).
				LineLimit(1),
			swiftui.Spacer(),
		),
		swiftui.HStack(
			swiftui.Text(label).
				Font(swiftui.FontCaption).
				ForegroundStyleNamed("secondary"),
			swiftui.Spacer(),
		),
		swiftui.HStack(
			swiftui.Text(note).
				Font(swiftui.FontCaption2).
				ForegroundStyleNamed("tertiary"),
			swiftui.Spacer(),
		),
	).Padding(12).
		Background(0.2, 0.2, 0.25, 0.42).
		CornerRadius(12).
		MaxFrame(-1, 0)
}

func changeColor(change int) (float64, float64, float64) {
	switch change {
	case 1:
		return 0.3, 0.8, 0.4
	case -1:
		return 0.9, 0.5, 0.2
	default:
		return 0.62, 0.66, 0.74
	}
}

func rowTint(table tableView, row rowView) float64 {
	if !table.OldNewDelta {
		return 0.03
	}
	switch row.Change {
	case 1, -1:
		return 0.08
	default:
		return 0.04
	}
}

func benchmarkMetricChart(table tableView) swiftui.View {
	if len(table.Rows) == 0 {
		return swiftui.ViewFromPointer(0)
	}
	if len(table.Configs) == 1 {
		return benchmarkSingleInputChart(table)
	}
	marks := make([]swcharts.Mark, 0, len(table.Rows)*len(table.Configs)*2)
	styles := make([]swcharts.StyleScaleEntry, 0, len(table.Configs))
	symbols := make([]swcharts.SymbolScaleEntry, 0, len(table.Configs))
	benchmarkOrder := make([]string, 0, len(table.Rows))
	maxValue := 0.0
	hasData := false
	for i, config := range table.Configs {
		styles = append(styles, swcharts.StyleScale(config, chartColor(i)))
		symbols = append(symbols, swcharts.SymbolScale(config, chartSymbol(i)))
	}
	for _, row := range table.Rows {
		benchmarkOrder = append(benchmarkOrder, row.Name)
		for i, config := range table.Configs {
			if i >= len(row.Stats) || !row.Stats[i].OK {
				continue
			}
			stat := row.Stats[i]
			hasData = true
			maxValue = math.Max(maxValue, stat.Max)
			marks = append(marks,
				swcharts.ErrorBarMark(
					swcharts.XFloatRange("Spread", stat.Min, stat.Max),
					swcharts.YString("Benchmark", row.Name),
					swcharts.HeightFixed(10),
				).
					PositionBy("Input", config).
					ForegroundStyleBy("Input", config).
					ExcludeFromLegend().
					Opacity(0.55).
					AccessibilityHidden(true),
				swcharts.PointMark(
					swcharts.XFloat("Mean", stat.Mean),
					swcharts.YString("Benchmark", row.Name),
				).
					PositionBy("Input", config).
					ForegroundStyleBy("Input", config).
					SymbolBy("Input", config).
					SymbolDiameter(10).
					SymbolStroke(swiftui.RGBA(0.12, 0.12, 0.14, 0.9), 1).
					Opacity(0.95).
					AccessibilityLabel(fmt.Sprintf("%s %s", row.Name, config)).
					AccessibilityValue(row.Values[i]),
			)
		}
	}
	if !hasData {
		return swiftui.ViewFromPointer(0)
	}
	if table.Domain.MaxValue > maxValue {
		maxValue = table.Domain.MaxValue
	}
	if maxValue <= 0 {
		maxValue = 1
	}
	height := 90.0 + float64(len(table.Rows))*30.0
	if height < 180 {
		height = 180
	}
	if height > 320 {
		height = 320
	}
	chart := swcharts.Chart(marks...).
		ChartForegroundStyleScale(styles...).
		ChartSymbolScale(symbols...).
		ChartLegend(metricLegend(table)).
		ChartXScaleDomain(swcharts.NumberDomain(0, maxValue*1.08)).
		ChartYScaleDomain(swcharts.CategoryDomain(benchmarkOrder...)).
		ChartXAxis(swcharts.AxisMarks(
			swcharts.AxisGridLines(),
			swcharts.AxisValueLabels(swcharts.CompactFormat(1)),
		)).
		ChartYAxis(swcharts.AxisMarks(
			swcharts.AxisLabels(),
		)).
		ChartXAxisLabel(table.Metric).
		ChartPlotStyle(
			swcharts.PlotBackgroundColor(swiftui.RGBA(1, 1, 1, 0.03)),
			swcharts.PlotBorder(swiftui.RGBA(1, 1, 1, 0.08), 1),
		)
	return chart.Frame(-1, height)
}

func benchmarkSingleInputChart(table tableView) swiftui.View {
	config := table.Configs[0]
	color := chartColor(0)
	marks := make([]swcharts.Mark, 0, len(table.Rows)*2)
	benchmarkOrder := make([]string, 0, len(table.Rows))
	maxValue := table.Domain.MaxValue
	if maxValue <= 0 {
		maxValue = 1
	}
	for i, row := range table.Rows {
		benchmarkOrder = append(benchmarkOrder, row.Name)
		if len(row.Stats) == 0 || !row.Stats[0].OK {
			continue
		}
		stat := row.Stats[0]
		marks = append(marks,
			swcharts.ErrorBarMark(
				swcharts.XFloatRange("Spread", stat.Min, stat.Max),
				swcharts.YString("Benchmark", row.Name),
				swcharts.HeightFixed(10),
			).
				ForegroundStyle(color).
				Opacity(0.55).
				AccessibilityHidden(true),
		)
		point := swcharts.PointMark(
			swcharts.XFloat("Mean", stat.Mean),
			swcharts.YString("Benchmark", row.Name),
		).
			ForegroundStyle(color).
			Symbol(swcharts.SymbolCircle).
			SymbolDiameter(10).
			AccessibilityLabel(fmt.Sprintf("%s %s", row.Name, config)).
			AccessibilityValue(row.Values[0])
		if shouldAnnotateSingleInput(table, i) {
			point = point.TextAnnotation(row.Values[0], swcharts.AnnotationTrailing).
				AnnotationOffset(8, 0).
				AnnotationOverflow(swcharts.AnnotationOverflowFitPlot, swcharts.AnnotationOverflowFitPlot)
		}
		marks = append(marks, point)
	}
	height := 90.0 + float64(len(table.Rows))*28.0
	if height < 180 {
		height = 180
	}
	if height > 320 {
		height = 320
	}
	return swcharts.Chart(marks...).
		ChartLegend(swcharts.LegendHidden).
		ChartXScaleDomain(swcharts.NumberDomain(0, maxValue*1.08)).
		ChartYScaleDomain(swcharts.CategoryDomain(benchmarkOrder...)).
		ChartXAxis(swcharts.AxisMarks(
			swcharts.AxisGridLines(),
			swcharts.AxisValueLabels(swcharts.CompactFormat(1)),
		)).
		ChartYAxis(swcharts.AxisMarks(
			swcharts.AxisLabels(),
		)).
		ChartXAxisLabel(table.Metric).
		ChartPlotStyle(
			swcharts.PlotBackgroundColor(swiftui.RGBA(1, 1, 1, 0.03)),
			swcharts.PlotBorder(swiftui.RGBA(1, 1, 1, 0.08), 1),
		).
		Frame(-1, height)
}

func benchmarkDeltaChart(table tableView) swiftui.View {
	if !table.OldNewDelta || len(table.Rows) == 0 {
		return swiftui.ViewFromPointer(0)
	}
	marks := make([]swcharts.Mark, 0, len(table.Rows)+1)
	benchmarkOrder := make([]string, 0, len(table.Rows))
	maxAbs := 0.0
	for _, row := range table.Rows {
		benchmarkOrder = append(benchmarkOrder, row.Name)
		value := row.PctDelta
		maxAbs = math.Max(maxAbs, math.Abs(value))
		start, end := 0.0, value
		if value < 0 {
			start, end = value, 0
		}
		mark := swcharts.RangeBarMark(
			swcharts.XFloatRange("Delta", start, end),
			swcharts.YString("Benchmark", row.Name),
		).
			ForegroundStyle(deltaChartColor(row.Change)).
			Opacity(0.92).
			CornerRadius(5).
			AccessibilityLabel(fmt.Sprintf("%s delta", row.Name)).
			AccessibilityValue(formatDeltaPercent(row.PctDelta))
		if shouldAnnotateDelta(table, row) {
			position := swcharts.AnnotationTrailing
			offset := 8.0
			if row.PctDelta < 0 {
				position = swcharts.AnnotationLeading
				offset = -8
			}
			mark = mark.TextAnnotation(formatDeltaPercent(row.PctDelta), position).
				AnnotationOffset(offset, 0).
				AnnotationOverflow(swcharts.AnnotationOverflowFitPlot, swcharts.AnnotationOverflowFitPlot)
		}
		marks = append(marks, mark)
	}
	if table.Domain.MaxDelta > maxAbs {
		maxAbs = table.Domain.MaxDelta
	}
	if maxAbs == 0 {
		maxAbs = 0.01
	}
	marks = append(marks,
		swcharts.ReferenceLineX(swcharts.NumberValue("Baseline", 0), "baseline").
			ForegroundStyle(swiftui.RGBA(1, 1, 1, 0.35)).
			LineStyle(swcharts.Stroke(1, 3, 3)).
			AccessibilityHidden(true),
	)
	height := 90.0 + float64(len(table.Rows))*26.0
	if height < 170 {
		height = 170
	}
	if height > 300 {
		height = 300
	}
	return swiftui.VStackSpaced(6,
		swiftui.HStack(
			swiftui.Text("Delta vs baseline").
				Font(swiftui.FontCaption).
				FontWeight(swiftui.WeightSemibold).
				ForegroundStyleNamed("secondary"),
			swiftui.Spacer(),
		),
		swcharts.Chart(marks...).
			ChartLegend(swcharts.LegendHidden).
			ChartXScaleDomain(swcharts.NumberDomain(-maxAbs*1.15, maxAbs*1.15)).
			ChartYScaleDomain(swcharts.CategoryDomain(benchmarkOrder...)).
			ChartXAxis(swcharts.AxisMarks(
				swcharts.AxisGridLines(),
				swcharts.AxisValueLabels(swcharts.PercentFormat(0)),
			)).
			ChartYAxis(swcharts.AxisMarks(
				swcharts.AxisLabels(),
			)).
			ChartXAxisLabel("delta %").
			ChartPlotStyle(
				swcharts.PlotBackgroundColor(swiftui.RGBA(1, 1, 1, 0.02)),
				swcharts.PlotBorder(swiftui.RGBA(1, 1, 1, 0.08), 1),
			).
			Frame(-1, height),
	)
}

func metricSummaryTitle(table tableView) string {
	if len(table.Configs) == 1 {
		return fmt.Sprintf("%d rows with spread and mean", len(table.Rows))
	}
	if !table.OldNewDelta {
		return fmt.Sprintf("%d inputs compared side by side", len(table.Configs))
	}
	return fmt.Sprintf("%d comparable rows across %d inputs", table.Compared, len(table.Configs))
}

func metricInsight(table tableView) string {
	if table.OldNewDelta {
		parts := []string{
			fmt.Sprintf("%d wins", table.Improved),
			fmt.Sprintf("%d losses", table.Regressed),
		}
		if highlight := tableTopDelta(table, 1); highlight.Valid {
			parts = append(parts, fmt.Sprintf("largest win %s %s", highlight.Benchmark, highlight.Detail))
		}
		if highlight := tableTopDelta(table, -1); highlight.Valid {
			parts = append(parts, fmt.Sprintf("largest loss %s %s", highlight.Benchmark, highlight.Detail))
		}
		if widest := tableWidestSpread(table); widest.Valid {
			parts = append(parts, fmt.Sprintf("widest spread %s %s", widest.Benchmark, widest.Detail))
		}
		return strings.Join(parts, " • ")
	}
	if widest := tableWidestSpread(table); widest.Valid {
		return fmt.Sprintf("%d configs side by side; widest spread %s %s", len(table.Configs), widest.Benchmark, widest.Detail)
	}
	return fmt.Sprintf("%d configs side by side", len(table.Configs))
}

func chartColor(i int) swiftui.Color {
	palette := []swiftui.Color{
		swiftui.RGB(0.35, 0.65, 1.0),
		swiftui.RGB(0.95, 0.6, 0.2),
		swiftui.RGB(0.3, 0.8, 0.4),
		swiftui.RGB(0.88, 0.42, 0.52),
		swiftui.RGB(0.62, 0.66, 0.9),
		swiftui.RGB(0.5, 0.78, 0.78),
	}
	return palette[i%len(palette)]
}

func chartSymbol(i int) swcharts.SymbolKind {
	symbols := []swcharts.SymbolKind{
		swcharts.SymbolCircle,
		swcharts.SymbolDiamond,
		swcharts.SymbolTriangle,
		swcharts.SymbolSquare,
		swcharts.SymbolPentagon,
		swcharts.SymbolPlus,
	}
	return symbols[i%len(symbols)]
}

func deltaChartColor(change int) swiftui.Color {
	switch change {
	case 1:
		return swiftui.RGB(0.3, 0.8, 0.4)
	case -1:
		return swiftui.RGB(0.9, 0.5, 0.2)
	default:
		return swiftui.RGB(0.62, 0.66, 0.82)
	}
}

func benchmarkCell(table tableView, row rowView, prefs displayPrefs) swiftui.View {
	title := swiftui.VStackSpaced(3,
		swiftui.HStack(
			swiftui.Text(row.Name).
				Font(swiftui.FontCallout).
				FontWeight(swiftui.WeightSemibold).
				LineLimit(1),
			swiftui.Spacer(),
		),
	)
	if row.SpreadPct > 0 && !prefs.CompactRows {
		title = swiftui.VStackSpaced(3,
			swiftui.HStack(
				swiftui.Text(row.Name).
					Font(swiftui.FontCallout).
					FontWeight(swiftui.WeightSemibold).
					LineLimit(1),
				swiftui.Spacer(),
			),
			swiftui.HStack(
				swiftui.Text(fmt.Sprintf("spread %s", formatPercent(row.SpreadPct))).
					Font(swiftui.FontCaption2).
					ForegroundStyleNamed("secondary"),
				swiftui.Spacer(),
			),
		)
	}
	views := []swiftui.Viewable{title.MaxFrame(-1, 0)}
	if table.OldNewDelta {
		views = append(views, outcomePill(row.Change))
	}
	return swiftui.HStackSpaced(10, views...).MaxFrame(-1, 0)
}

func settingsLine(label, value string) swiftui.View {
	return swiftui.HStack(
		swiftui.Text(label).
			ForegroundStyleNamed("secondary"),
		swiftui.Spacer(),
		swiftui.Text(value).
			Font(swiftui.FontCaption).
			FontWeight(swiftui.WeightSemibold),
	)
}

func onOffLabel(v bool) string {
	if v {
		return "On"
	}
	return "Off"
}

func outcomePill(change int) swiftui.View {
	label := "steady"
	r, g, b := 0.58, 0.62, 0.82
	switch change {
	case 1:
		label = "win"
		r, g, b = 0.3, 0.8, 0.4
	case -1:
		label = "loss"
		r, g, b = 0.9, 0.5, 0.2
	}
	return swiftui.HStack(
		swiftui.Text(label).
			Font(swiftui.FontCaption2).
			FontWeight(swiftui.WeightSemibold),
	).Padding(6).
		Background(r, g, b, 0.18).
		CornerRadius(999)
}

func configChipRow(configs []string) swiftui.View {
	chips := make([]swiftui.Viewable, 0, len(configs))
	for _, config := range configs {
		chips = append(chips, configChip(config))
	}
	return swiftui.HStackSpaced(8, chips...)
}

func configChip(label string) swiftui.View {
	r, g, b := 0.35, 0.65, 1.0
	if label == "stdin" {
		r, g, b = 0.95, 0.6, 0.2
	}
	return swiftui.HStack(
		swiftui.Text(label).
			Font(swiftui.FontCaption2).
			FontWeight(swiftui.WeightSemibold).
			LineLimit(1),
	).Padding(6).
		Background(r, g, b, 0.18).
		CornerRadius(999)
}

func changeCountPill(label string, count int, r, g, b float64) swiftui.View {
	return swiftui.HStackSpaced(4,
		swiftui.Text(fmt.Sprintf("%d", count)).
			Font(swiftui.FontCaption).
			FontWeight(swiftui.WeightBold).
			MonospacedDigit(),
		swiftui.Text(label).
			Font(swiftui.FontCaption2).
			ForegroundStyleNamed("secondary"),
	).Padding(6).
		Background(r, g, b, 0.14).
		CornerRadius(999)
}

func stackedChangeBar(improved, regressed, unchanged int, width float64) swiftui.View {
	total := improved + regressed + unchanged
	if total == 0 {
		return swiftui.RoundedRectangle(999).
			Fill(1, 1, 1, 0.06).
			Frame(width, 10).
			AsView()
	}
	segments := make([]swiftui.Viewable, 0, 3)
	for _, segment := range []struct {
		count int
		r     float64
		g     float64
		b     float64
	}{
		{count: improved, r: 0.3, g: 0.8, b: 0.4},
		{count: unchanged, r: 0.58, g: 0.62, b: 0.82},
		{count: regressed, r: 0.9, g: 0.5, b: 0.2},
	} {
		if segment.count == 0 {
			continue
		}
		segments = append(segments,
			swiftui.Rectangle().
				Fill(segment.r, segment.g, segment.b, 0.9).
				Frame(width*float64(segment.count)/float64(total), 10).
				AsView(),
		)
	}
	return swiftui.HStackSpaced(0, segments...).
		Background(1, 1, 1, 0.04).
		CornerRadius(999)
}

func highlightCard(highlight highlightView) swiftui.View {
	r, g, b := changeColor(highlight.Change)
	return swiftui.VStackSpaced(6,
		swiftui.HStack(
			swiftui.Text(highlight.Title).
				Font(swiftui.FontCaption).
				FontWeight(swiftui.WeightSemibold).
				ForegroundStyle(r, g, b, 1.0),
			swiftui.Spacer(),
		),
		swiftui.HStack(
			swiftui.Text(highlight.Benchmark).
				Font(swiftui.FontCallout).
				FontWeight(swiftui.WeightSemibold).
				LineLimit(1),
			swiftui.Spacer(),
		),
		swiftui.HStack(
			swiftui.Text(highlight.Metric).
				Font(swiftui.FontCaption2).
				ForegroundStyleNamed("secondary"),
			swiftui.Spacer(),
		),
		swiftui.HStack(
			swiftui.Text(highlight.Detail).
				Font(swiftui.FontCaption2).
				MonospacedDigit(),
			swiftui.Spacer(),
		),
	).Padding(10).
		Background(r, g, b, 0.12).
		CornerRadius(12).
		MaxFrame(-1, 0)
}

func modeSummary(view comparisonView) string {
	switch view.Mode {
	case "single":
		return "Single input: mean and spread only."
	case "single-live":
		return fmt.Sprintf("Streaming single input on %s.", view.StreamingLabel)
	case "compare":
		return "Two-input compare: spread above, deltas below."
	case "stream-compare":
		return fmt.Sprintf("Streaming comparison on %s.", view.StreamingLabel)
	default:
		return "Multi-input side-by-side comparison."
	}
}

func comparisonMode(inputs []inputSpec) string {
	streaming := false
	for _, input := range inputs {
		if input.Stream {
			streaming = true
			break
		}
	}
	switch {
	case len(inputs) <= 1 && streaming:
		return "single-live"
	case len(inputs) <= 1:
		return "single"
	case streaming:
		return "stream-compare"
	case len(inputs) == 2:
		return "compare"
	default:
		return "multi"
	}
}

func metricLegend(table tableView) swcharts.LegendConfig {
	if len(table.Configs) <= 1 {
		return swcharts.LegendHidden
	}
	return swcharts.LegendCompact(swcharts.LegendPositionTop, swcharts.LegendAlignmentLeading)
}

func shouldAnnotateSingleInput(table tableView, index int) bool {
	if len(table.Rows) <= 4 {
		return true
	}
	return index < 3
}

func shouldAnnotateDelta(table tableView, row rowView) bool {
	if row.PctDelta == 0 || table.Domain.MaxDelta == 0 {
		return false
	}
	if row.Change != 0 && math.Abs(row.PctDelta) >= table.Domain.MaxDelta*0.6 {
		return true
	}
	return math.Abs(row.PctDelta) >= table.Domain.MaxDelta*0.9
}

func tableTopDelta(table tableView, change int) rowHighlight {
	var best rowHighlight
	for _, row := range table.Rows {
		if row.Change != change {
			continue
		}
		value := math.Abs(row.PctDelta)
		if !best.Valid || value > best.Value {
			best = rowHighlight{
				Valid:     true,
				Metric:    table.Metric,
				Benchmark: row.Name,
				Detail:    formatDeltaPercent(row.PctDelta),
				Value:     value,
				Change:    change,
			}
		}
	}
	return best
}

func tableWidestSpread(table tableView) rowHighlight {
	var widest rowHighlight
	for _, row := range table.Rows {
		if row.SpreadPct <= 0 {
			continue
		}
		if !widest.Valid || row.SpreadPct > widest.Value {
			widest = rowHighlight{
				Valid:     true,
				Metric:    table.Metric,
				Benchmark: row.Name,
				Detail:    fmt.Sprintf("%s spread", formatPercent(row.SpreadPct)),
				Value:     row.SpreadPct,
			}
		}
	}
	return widest
}

func sortTableRows(table *tableView) {
	sort.SliceStable(table.Rows, func(i, j int) bool {
		a := table.Rows[i]
		b := table.Rows[j]
		if table.OldNewDelta {
			aSig := a.Change != 0
			bSig := b.Change != 0
			if aSig != bSig {
				return aSig
			}
			aDelta := math.Abs(a.PctDelta)
			bDelta := math.Abs(b.PctDelta)
			if aDelta != bDelta {
				return aDelta > bDelta
			}
		}
		if a.SpreadPct != b.SpreadPct {
			return a.SpreadPct > b.SpreadPct
		}
		return a.Name < b.Name
	})
}

func formatPercent(value float64) string {
	return fmt.Sprintf("%.1f%%", value*100)
}

func formatSignedPercent(value float64) string {
	return fmt.Sprintf("%+.1f%%", value*100)
}

func formatDeltaPercent(value float64) string {
	if math.Abs(value) <= 1 {
		return fmt.Sprintf("%+.1f%%", value*100)
	}
	return fmt.Sprintf("%+.1f%%", value)
}

func setView(view comparisonView) {
	viewMu.Lock()
	defer viewMu.Unlock()
	currentView = view
}

func snapshotView() comparisonView {
	viewMu.Lock()
	defer viewMu.Unlock()
	return currentView
}

func printUsage(argv0 string) {
	name := filepath.Base(argv0)
	fmt.Fprintf(os.Stderr, "usage: %s [label=]path|- ...\n", name)
	fmt.Fprintf(os.Stderr, "examples:\n")
	fmt.Fprintf(os.Stderr, "  %s bench-old.txt bench-new.txt\n", name)
	fmt.Fprintf(os.Stderr, "  go test -bench=. | %s bench-old.txt\n", name)
	fmt.Fprintf(os.Stderr, "  go test -bench=. | %s old=bench-old.txt live=-\n", name)
}

func parseInputs(args []string, stdinPiped bool) ([]inputSpec, error) {
	if len(args) == 0 && !stdinPiped {
		return nil, errors.New("benchview needs at least one file or piped stdin")
	}

	var (
		inputs    []inputSpec
		seen      = make(map[string]bool)
		hasStream bool
	)
	for _, arg := range args {
		spec, err := parseInput(arg)
		if err != nil {
			return nil, err
		}
		if spec.Stream {
			if hasStream {
				return nil, errors.New("benchview accepts at most one stdin input")
			}
			hasStream = true
		}
		label, err := reserveLabel(spec.Label, spec.Explicit, seen)
		if err != nil {
			return nil, err
		}
		spec.Label = label
		inputs = append(inputs, spec)
	}
	if stdinPiped && !hasStream {
		label, err := reserveLabel("stdin", false, seen)
		if err != nil {
			return nil, err
		}
		inputs = append(inputs, inputSpec{
			Label:  label,
			Path:   "-",
			Stream: true,
		})
	}
	if len(inputs) == 0 {
		return nil, errors.New("benchview needs at least one input")
	}
	return inputs, nil
}

func parseInput(arg string) (inputSpec, error) {
	if arg == "" {
		return inputSpec{}, errors.New("empty input")
	}
	spec := inputSpec{Path: arg}
	if i := strings.Index(arg, "="); i > 0 {
		spec.Label = arg[:i]
		spec.Path = arg[i+1:]
		spec.Explicit = true
		if spec.Label == "" || spec.Path == "" {
			return inputSpec{}, fmt.Errorf("invalid input %q", arg)
		}
	}
	if spec.Path == "-" {
		spec.Stream = true
		if spec.Label == "" {
			spec.Label = "stdin"
		}
		return spec, nil
	}
	if spec.Label == "" {
		spec.Label = filepath.Base(spec.Path)
		if spec.Label == "." || spec.Label == string(filepath.Separator) || spec.Label == "" {
			spec.Label = spec.Path
		}
	}
	return spec, nil
}

func reserveLabel(label string, explicit bool, seen map[string]bool) (string, error) {
	if !seen[label] {
		seen[label] = true
		return label, nil
	}
	if explicit {
		return "", fmt.Errorf("duplicate input label %q", label)
	}
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s-%d", label, i)
		if seen[candidate] {
			continue
		}
		seen[candidate] = true
		return candidate, nil
	}
}

func loadInputs(inputs []inputSpec) error {
	for i := range inputs {
		if inputs[i].Stream {
			continue
		}
		data, err := os.ReadFile(inputs[i].Path)
		if err != nil {
			return fmt.Errorf("read %s: %w", inputs[i].Path, err)
		}
		inputs[i].Data = data
	}
	return nil
}

func findStreamInput(inputs []inputSpec) int {
	for i, input := range inputs {
		if input.Stream {
			return i
		}
	}
	return -1
}

func summarizeInputs(inputs []inputSpec) string {
	if len(inputs) == 0 {
		return "no inputs"
	}
	labels := make([]string, 0, len(inputs))
	for _, input := range inputs {
		labels = append(labels, input.Label)
	}
	if len(labels) <= 3 {
		return strings.Join(labels, ", ")
	}
	return fmt.Sprintf("%s, +%d more", strings.Join(labels[:3], ", "), len(labels)-3)
}

func streamLabel(inputs []inputSpec) string {
	for _, input := range inputs {
		if input.Stream {
			return input.Label
		}
	}
	return "none"
}

func initialStatus(inputs []inputSpec) string {
	if findStreamInput(inputs) >= 0 {
		return "Waiting for stdin"
	}
	if len(inputs) == 1 {
		return "Loaded 1 input"
	}
	return fmt.Sprintf("Loaded %d inputs", len(inputs))
}

func benchviewWidth(n int) float64 {
	width := 620 + 140*n
	if width < 920 {
		return 920
	}
	if width > 1600 {
		return 1600
	}
	return float64(width)
}

func valueColumnWidth(n int) float64 {
	switch {
	case n >= 6:
		return 84
	case n >= 4:
		return 96
	default:
		return 120
	}
}

func stdinAvailable() bool {
	info, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice == 0
}

func countBenchmarkLines(data []byte) int {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	n := 0
	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), "Benchmark") {
			n++
		}
	}
	return n
}

func nextStreamRefresh(results int) int {
	switch {
	case results < 1:
		return 1
	case results < 4:
		return results + 1
	default:
		return results + 4
	}
}
