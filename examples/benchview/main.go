//go:build darwin

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
	"strconv"
	"strings"
	"sync"

	"github.com/tmc/swiftui"
	swcharts "github.com/tmc/swiftui/charts"
	"golang.org/x/perf/benchmath"
	"golang.org/x/perf/benchproc"
	"golang.org/x/perf/benchunit"
)

func init() { runtime.LockOSThread() }

type tableView struct {
	Title          string
	TableKey       string
	Metric         string
	SummaryLabel   string
	Configs        []string
	HasComparisons bool
	Compared       int
	Improved       int
	Regressed      int
	Unchanged      int
	Insight        string
	Domain         metricDomain
	Rows           []rowView
	Summary        summaryRowView
	WarningCount   int
	WarningPreview string
}

type rowView struct {
	Name      string
	Cells     []valueCellView
	SpreadPct float64
	MaxDelta  float64
	Change    int
	Warning   string
}

type valueCellView struct {
	Value    string
	Range    string
	Stat     metricStat
	Delta    string
	DeltaPct float64
	Note     string
	Change   int
	Warning  string
	HasValue bool
	HasDelta bool
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

type summaryRowView struct {
	Label string
	Cells []summaryCellView
}

type summaryCellView struct {
	Value    string
	Delta    string
	Warning  string
	HasValue bool
	HasDelta bool
}

type highlightView struct {
	Title     string
	Metric    string
	Benchmark string
	Config    string
	Detail    string
	Change    int
}

type rowHighlight struct {
	Valid     bool
	Metric    string
	Benchmark string
	Config    string
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
	Warnings       []string
	Error          string
	Options        analysisOptions
	Preview        string
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
	optionError := swiftui.NewStringState("")
	filterQuery := swiftui.NewStringState(defaultAnalysisOptions().Filter)
	tableProjection := swiftui.NewStringState(defaultAnalysisOptions().Table)
	rowProjection := swiftui.NewStringState(defaultAnalysisOptions().Row)
	colProjection := swiftui.NewStringState(defaultAnalysisOptions().Col)
	ignoreProjection := swiftui.NewStringState(defaultAnalysisOptions().Ignore)
	alphaInput := swiftui.NewStringState(trimFloat(defaultAnalysisOptions().Alpha))
	confidenceInput := swiftui.NewStringState(trimFloat(defaultAnalysisOptions().Confidence))
	formatMode := swiftui.NewIntState(formatIndex(defaultAnalysisOptions().Format))
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
	var optsMu sync.Mutex
	appliedOptions := defaultAnalysisOptions()
	getOptions := func() analysisOptions {
		optsMu.Lock()
		defer optsMu.Unlock()
		return appliedOptions
	}
	setOptions := func(opts analysisOptions) {
		optsMu.Lock()
		appliedOptions = opts
		optsMu.Unlock()
	}
	rebuild := func() {
		setView(buildComparisonWithOptions(inputs, getOptions()))
		updateTick.Set(updateTick.Get() + 1)
	}
	applyOptions := func() {
		opts, err := parseAnalysisOptions(
			filterQuery.Get(),
			tableProjection.Get(),
			rowProjection.Get(),
			colProjection.Get(),
			ignoreProjection.Get(),
			alphaInput.Get(),
			confidenceInput.Get(),
			formatFromIndex(formatMode.Get()),
		)
		if err != nil {
			optionError.Set(err.Error())
			return
		}
		optionError.Set("")
		setOptions(opts)
		rebuild()
	}
	resetOptions := func() {
		opts := defaultAnalysisOptions()
		filterQuery.Set(opts.Filter)
		tableProjection.Set(opts.Table)
		rowProjection.Set(opts.Row)
		colProjection.Set(opts.Col)
		ignoreProjection.Set(opts.Ignore)
		alphaInput.Set(trimFloat(opts.Alpha))
		confidenceInput.Set(trimFloat(opts.Confidence))
		formatMode.Set(formatIndex(opts.Format))
		optionError.Set("")
		setOptions(opts)
		rebuild()
	}

	if streamIndex >= 0 {
		liveCount.Set(countBenchmarkLines(inputs[streamIndex].Data))
	}
	setView(buildComparisonWithOptions(inputs, getOptions()))

	reloadFiles := func() {
		for i := range inputs {
			if inputs[i].Stream {
				continue
			}
			data, err := os.ReadFile(inputs[i].Path)
			if err != nil {
				status.Set(fmt.Sprintf("reload error: %v", err))
				return
			}
			inputs[i].Data = data
		}
		setView(buildComparisonWithOptions(inputs, getOptions()))
		status.Set("Reloaded")
		updateTick.Set(updateTick.Get() + 1)
	}

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
				setView(buildComparisonWithOptions(inputs, getOptions()))
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
		optionError,
		filterQuery,
		tableProjection,
		rowProjection,
		colProjection,
		ignoreProjection,
		alphaInput,
		confidenceInput,
		formatMode,
		bumpDisplay,
		reloadFiles,
		applyOptions,
		resetOptions,
	))
}

func buildComparison(inputs []inputSpec) comparisonView {
	return buildComparisonWithOptions(inputs, defaultAnalysisOptions())
}

func buildComparisonWithOptions(inputs []inputSpec, opts analysisOptions) comparisonView {
	view := comparisonView{
		Inputs:  len(inputs),
		Mode:    comparisonMode(inputs),
		Options: opts,
	}
	for _, input := range inputs {
		if input.Stream {
			view.HasStreaming = true
			view.StreamingLabel = input.Label
			break
		}
	}
	tables, warnings, err := loadAnalysisTables(inputs, opts)
	view.Warnings = warnings
	if err != nil {
		view.Error = err.Error()
		return view
	}
	switch opts.Format {
	case formatText:
		view.Preview = tables.TextPreview()
	case formatCSV:
		view.Preview = tables.CSVPreview()
	}
	var bestWin, bestLoss, noisiest rowHighlight
	for tableIndex, table := range tables.Tables {
		tableKey := tables.Keys[tableIndex]
		title := tableLabel(tableKey)
		tv := tableView{
			Title:          title,
			TableKey:       tableKey.String(),
			Metric:         table.Unit,
			SummaryLabel:   table.Assumption.SummaryLabel(),
			Configs:        make([]string, len(table.Cols)),
			HasComparisons: len(table.Cols) > 1,
			Summary: summaryRowView{
				Label: table.SummaryLabel,
				Cells: make([]summaryCellView, len(table.Cols)),
			},
		}
		for i, col := range table.Cols {
			tv.Configs[i] = keyLabel(col, fmt.Sprintf("col%d", i+1))
		}
		for _, row := range table.Rows {
			scaler := table.rowScaler(row)
			rv := rowView{
				Name:  keyLabel(row, row.StringValues()),
				Cells: make([]valueCellView, len(table.Cols)),
			}
			rowBetter := 0.0
			rowWarningParts := make([]string, 0, 1)
			for i, col := range table.Cols {
				cell, ok := table.Cells[analysisCellKey{Row: row, Col: col}]
				if !ok {
					continue
				}
				cv := valueCellView{
					HasValue: true,
					Value:    scaler.Format(cell.Summary.Center),
					Range:    cell.Summary.PctRangeString(),
					Stat: metricStat{
						Min:  cell.Summary.Lo,
						Mean: cell.Summary.Center,
						Max:  cell.Summary.Hi,
						OK:   true,
					},
					Warning: joinWarnings(cell.Sample.Warnings, cell.Summary.Warnings),
				}
				if cv.Warning != "" {
					rowWarningParts = append(rowWarningParts, cv.Warning)
					tv.WarningCount++
					if tv.WarningPreview == "" {
						tv.WarningPreview = cv.Warning
					}
				}
				tv.Domain.MaxValue = math.Max(tv.Domain.MaxValue, cell.Summary.Hi)
				if spreadPct := summarySpread(cell.Summary); spreadPct > rv.SpreadPct {
					rv.SpreadPct = spreadPct
				}
				if cell.Baseline != nil {
					cv.HasDelta = true
					cv.Delta = cell.Comparison.FormatDelta(cell.Baseline.Summary.Center, cell.Summary.Center)
					cv.Note = cell.Comparison.String()
					if cell.Baseline.Summary.Center != 0 {
						cv.DeltaPct = ((cell.Summary.Center / cell.Baseline.Summary.Center) - 1) * 100
						tv.Domain.MaxDelta = math.Max(tv.Domain.MaxDelta, math.Abs(cv.DeltaPct))
						rv.MaxDelta = math.Max(rv.MaxDelta, math.Abs(cv.DeltaPct))
					}
					cv.Change = comparisonChange(table.Opts.Units.GetBetter(table.Unit), cell.Comparison, cv.DeltaPct, cv.Delta)
					tv.Compared++
					view.Compared++
					switch cv.Change {
					case 1:
						tv.Improved++
						view.Improved++
					case -1:
						tv.Regressed++
						view.Regressed++
					default:
						tv.Unchanged++
						view.Unchanged++
					}
					if math.Abs(cv.DeltaPct) >= rowBetter {
						rowBetter = math.Abs(cv.DeltaPct)
						rv.Change = cv.Change
					}
					if cv.Change == 1 && (!bestWin.Valid || math.Abs(cv.DeltaPct) > bestWin.Value) {
						bestWin = rowHighlight{
							Valid:     true,
							Metric:    table.Unit,
							Benchmark: rv.Name,
							Config:    tv.Configs[i],
							Detail:    formatDeltaPercent(cv.DeltaPct),
							Value:     math.Abs(cv.DeltaPct),
							Change:    1,
						}
					}
					if cv.Change == -1 && (!bestLoss.Valid || math.Abs(cv.DeltaPct) > bestLoss.Value) {
						bestLoss = rowHighlight{
							Valid:     true,
							Metric:    table.Unit,
							Benchmark: rv.Name,
							Config:    tv.Configs[i],
							Detail:    formatDeltaPercent(cv.DeltaPct),
							Value:     math.Abs(cv.DeltaPct),
							Change:    -1,
						}
					}
				}
				rv.Cells[i] = cv
			}
			rv.Warning = strings.Join(uniqueStrings(rowWarningParts), " • ")
			tv.Rows = append(tv.Rows, rv)
			view.Rows++
			if rv.SpreadPct > 0 && (!noisiest.Valid || rv.SpreadPct > noisiest.Value) {
				noisiest = rowHighlight{
					Valid:     true,
					Metric:    table.Unit,
					Benchmark: rv.Name,
					Detail:    fmt.Sprintf("%s spread", formatPercent(rv.SpreadPct)),
					Value:     rv.SpreadPct,
				}
			}
		}
		summaryScale := summaryScaler(table)
		for i, col := range table.Cols {
			summary := table.Summary[col]
			if summary == nil {
				continue
			}
			cell := summaryCellView{
				Warning: joinWarnings(summary.Warnings),
			}
			if summary.HasSummary {
				cell.HasValue = true
				cell.Value = summaryScale.Format(summary.Summary)
			}
			if i > 0 && summary.HasRatio {
				cell.HasDelta = true
				cell.Delta = fmt.Sprintf("%+.2f%%", (summary.Ratio-1)*100)
			}
			if cell.Warning != "" {
				tv.WarningCount++
				if tv.WarningPreview == "" {
					tv.WarningPreview = cell.Warning
				}
			}
			tv.Summary.Cells[i] = cell
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
	optionError *swiftui.StringState,
	filterQuery, tableProjection, rowProjection, colProjection, ignoreProjection, alphaInput, confidenceInput *swiftui.StringState,
	formatMode *swiftui.IntState,
	onChange func(),
	onReload func(),
	onApplyOptions func(),
	onResetOptions func(),
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
						swiftui.Button("Reload", func() {
							onReload()
						}).
							Padding(8).
							Background(1, 1, 1, 0.06).
							CornerRadius(999),
						swiftui.Button("Options", func() {
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
	return content.Popover(showDisplayControls, optionsPopover(
		updateTick,
		optionError,
		filterQuery,
		tableProjection,
		rowProjection,
		colProjection,
		ignoreProjection,
		alphaInput,
		confidenceInput,
		formatMode,
		showSummaryCards,
		showChangeBar,
		showHighlights,
		showAbsolute,
		showDelta,
		showRows,
		showInsights,
		compactRows,
		rowLimit,
		onApplyOptions,
		onResetOptions,
		onChange,
	))
}

func renderDashboard(view comparisonView, prefs displayPrefs) swiftui.View {
	sections := make([]swiftui.Viewable, 0, 4)
	if prefs.ShowSummaryCards || prefs.ShowChangeBar || prefs.ShowHighlights {
		sections = append(sections, overviewStrip(view, prefs))
	}
	if view.Error != "" {
		sections = append(sections, errorPanel(view.Error))
	} else if view.Options.Format == formatDashboard {
		sections = append(sections, renderTables(view, prefs))
	} else {
		sections = append(sections, previewPanel(view))
	}
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
	lines = append(lines, analysisOptionStrip(view.Options, len(view.Warnings)))
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
					swiftui.Text("Deltas appear when benchstat can compare populated columns against the first baseline column.").
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
		return swiftui.HStackSpaced(8,
			append(tokens,
				swiftui.Text(sourceSummary.Get()).
					Font(swiftui.FontCaption).
					ForegroundStyleNamed("secondary").
					LineLimit(1).
					AsView(),
				swiftui.Spacer(),
				swiftui.Text(streamSummary.Get()).
					Font(swiftui.FontCaption2).
					ForegroundStyleNamed("tertiary").
					LineLimit(1).
					AsView(),
			)...,
		).Padding(8).
			Background(1, 1, 1, 0.03).
			CornerRadius(10)
	})
}

func analysisOptionStrip(opts analysisOptions, warnings int) swiftui.View {
	tokens := []swiftui.Viewable{
		compactStat("filter", opts.Filter, 0.42, 0.58, 0.94),
		compactStat("table", opts.Table, 0.62, 0.62, 0.82),
		compactStat("row", opts.Row, 0.62, 0.62, 0.82),
		compactStat("col", opts.Col, 0.62, 0.62, 0.82),
		compactStat("α", trimFloat(opts.Alpha), 0.35, 0.65, 1.0),
		compactStat("ci", formatConfidence(opts.Confidence), 0.35, 0.65, 1.0),
	}
	if opts.Ignore != "" {
		tokens = append(tokens, compactStat("ignore", opts.Ignore, 0.62, 0.62, 0.82))
	}
	if opts.Format != formatDashboard {
		tokens = append(tokens, compactStat("view", strings.ToUpper(opts.Format), 0.95, 0.6, 0.2))
	}
	if warnings > 0 {
		tokens = append(tokens, compactStat("warnings", fmt.Sprintf("%d", warnings), 0.9, 0.5, 0.2))
	}
	return swiftui.HStackSpaced(8, append(tokens, swiftui.Spacer())...)
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
			rows = append(rows, comparisonHeaderRow(displayTable))
			for _, row := range displayTable.Rows {
				rows = append(rows, swiftui.Divider())
				rows = append(rows, comparisonRow(displayTable, row, prefs))
			}
			if displayTable.Summary.Label != "" && len(displayTable.Summary.Cells) > 0 {
				rows = append(rows, swiftui.Divider())
				rows = append(rows, summaryRow(displayTable))
			}
		}
		groups = append(groups, swiftui.GroupBox(tableGroupTitle(displayTable),
			swiftui.VStackSpaced(6, rows...).Padding(10),
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
			configChipRow(table.Configs),
		),
		swiftui.HStack(
			swiftui.Text(metricSubtitle(table)).
				Font(swiftui.FontCaption2).
				ForegroundStyleNamed("secondary").
				LineLimit(1),
			swiftui.Spacer(),
		),
	}
	if prefs.ShowInsights {
		views = append(views,
			swiftui.HStack(
				swiftui.Text(table.Insight).
					Font(swiftui.FontCaption).
					ForegroundStyleNamed("secondary").
					LineLimit(1),
				swiftui.Spacer(),
			),
		)
	}
	if prefs.ShowAbsolute && shouldShowCharts(table) {
		if chart := benchmarkMetricChart(table); chart.Pointer() != 0 {
			views = append(views, chart)
		}
	}
	if prefs.ShowDelta && shouldShowCharts(table) {
		if chart := benchmarkDeltaChart(table); chart.Pointer() != 0 {
			views = append(views, chart)
		}
	}
	return swiftui.VStackSpaced(5, views...)
}

func comparisonHeaderRow(table tableView) swiftui.View {
	views := []swiftui.Viewable{headerText("Benchmark", 0)}
	width := valueColumnWidth(len(table.Configs))
	for _, config := range table.Configs {
		views = append(views, headerText(config, width))
	}
	return swiftui.HStackSpaced(12, views...)
}

func comparisonRow(table tableView, row rowView, prefs displayPrefs) swiftui.View {
	r, g, b := changeColor(row.Change)
	views := []swiftui.Viewable{
		benchmarkCell(table, row, prefs),
	}
	width := valueColumnWidth(len(table.Configs))
	for _, cell := range row.Cells {
		views = append(views, metricCellView(cell, width))
	}
	padding := 10.0
	if prefs.CompactRows {
		padding = 5
	}
	return swiftui.HStackSpaced(12, views...).
		Padding(padding).
		Background(r, g, b, rowTint(table, row)).
		CornerRadius(10)
}

func summaryRow(table tableView) swiftui.View {
	views := []swiftui.Viewable{
		swiftui.Text(table.Summary.Label).
			Font(swiftui.FontCaption).
			FontWeight(swiftui.WeightSemibold).
			ForegroundStyleNamed("secondary").
			MaxFrame(-1, 0),
	}
	width := valueColumnWidth(len(table.Configs))
	for _, cell := range table.Summary.Cells {
		views = append(views, summaryCellViewBox(cell, width))
	}
	return swiftui.HStackSpaced(12, views...).
		Padding(8).
		Background(1, 1, 1, 0.03).
		CornerRadius(10)
}

func optionsPopover(
	updateTick *swiftui.IntState,
	optionError, filterQuery, tableProjection, rowProjection, colProjection, ignoreProjection, alphaInput, confidenceInput *swiftui.StringState,
	formatMode, showSummaryCards, showChangeBar, showHighlights, showAbsolute, showDelta, showRows, showInsights, compactRows, rowLimit *swiftui.IntState,
	onApplyOptions func(),
	onResetOptions func(),
	onChange func(),
) swiftui.View {
	return swiftui.ScrollView(
		swiftui.VStackSpaced(10,
			swiftui.HStack(
				swiftui.Text("Benchstat").
					Font(swiftui.FontHeadline).
					FontWeight(swiftui.WeightSemibold),
				swiftui.Spacer(),
			),
			swiftui.GroupBox("Analysis",
				swiftui.VStackSpaced(8,
					labeledField("Filter", filterQuery),
					labeledField("Table", tableProjection),
					labeledField("Row", rowProjection),
					labeledField("Col", colProjection),
					labeledField("Ignore", ignoreProjection),
					labeledField("Alpha", alphaInput),
					labeledField("Confidence", confidenceInput),
					swiftui.PickerSegmented("View", formatMode, segmentedOptions("Dashboard", "Text", "CSV"), func() {}).
						MaxFrame(-1, 0),
					swiftui.HStackSpaced(8,
						swiftui.Button("Apply", onApplyOptions).
							Padding(8).
							Background(0.35, 0.65, 1.0, 0.18).
							CornerRadius(999),
						swiftui.Button("Defaults", onResetOptions).
							Padding(8).
							Background(1, 1, 1, 0.06).
							CornerRadius(999),
						swiftui.Spacer(),
					),
					swiftui.DynamicView(updateTick, func(_ int) swiftui.View {
						message := optionError.Get()
						if message == "" {
							return swiftui.Text("Defaults mirror benchstat: .config / .fullname / .file / * / α 0.05 / CI 95%.").
								Font(swiftui.FontCaption2).
								ForegroundStyleNamed("secondary").
								AsView()
						}
						return swiftui.Text(message).
							Font(swiftui.FontCaption2).
							ForegroundStyle(0.9, 0.5, 0.2, 1.0).
							AsView()
					}),
				).Padding(10),
			),
			swiftui.GroupBox("Layout",
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
						settingsLine("View", []string{"Dashboard", "Text", "CSV"}[clampIndex(formatMode.Get(), 3)]),
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

func labeledField(label string, state *swiftui.StringState) swiftui.View {
	return swiftui.VStackSpaced(4,
		swiftui.HStack(
			swiftui.Text(label).
				Font(swiftui.FontCaption).
				ForegroundStyleNamed("secondary"),
			swiftui.Spacer(),
		),
		swiftui.TextField(label, state, func() {}).
			MaxFrame(-1, 0),
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
	label, detail := benchmarkLabelParts(highlight.Benchmark)
	if detail == "" {
		detail = highlight.Metric
	} else {
		detail = detail + " • " + highlight.Metric
	}
	return swiftui.HStackSpaced(8,
		swiftui.Text(highlight.Title).
			Font(swiftui.FontCaption2).
			FontWeight(swiftui.WeightSemibold).
			ForegroundStyle(r, g, b, 1.0),
		swiftui.Text(label).
			Font(swiftui.FontCaption).
			FontWeight(swiftui.WeightSemibold).
			LineLimit(1),
		swiftui.Text(highlight.Detail).
			Font(swiftui.FontCaption).
			MonospacedDigit(),
		swiftui.Text(detail).
			Font(swiftui.FontCaption2).
			ForegroundStyleNamed("secondary").
			LineLimit(1),
		swiftui.Spacer(),
	)
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
	if !table.HasComparisons {
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
		label := chartRowLabel(row.Name)
		benchmarkOrder = append(benchmarkOrder, label)
		for i, config := range table.Configs {
			if i >= len(row.Cells) || !row.Cells[i].HasValue || !row.Cells[i].Stat.OK {
				continue
			}
			stat := row.Cells[i].Stat
			hasData = true
			maxValue = math.Max(maxValue, stat.Max)
			marks = append(marks,
				swcharts.ErrorBarMark(
					swcharts.XFloatRange("Spread", stat.Min, stat.Max),
					swcharts.YString("Benchmark", label),
					swcharts.HeightFixed(10),
				).
					PositionBy("Input", config).
					ForegroundStyleBy("Input", config).
					ExcludeFromLegend().
					Opacity(0.55).
					AccessibilityHidden(true),
				swcharts.PointMark(
					swcharts.XFloat("Mean", stat.Mean),
					swcharts.YString("Benchmark", label),
				).
					PositionBy("Input", config).
					ForegroundStyleBy("Input", config).
					SymbolBy("Input", config).
					SymbolDiameter(10).
					SymbolStroke(swiftui.RGBA(0.12, 0.12, 0.14, 0.9), 1).
					Opacity(0.95).
					AccessibilityLabel(fmt.Sprintf("%s %s", row.Name, config)).
					AccessibilityValue(row.Cells[i].Value),
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
	if chart.Pointer() == 0 {
		return swiftui.ViewFromPointer(0)
	}
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
		label := chartRowLabel(row.Name)
		benchmarkOrder = append(benchmarkOrder, label)
		if len(row.Cells) == 0 || !row.Cells[0].HasValue || !row.Cells[0].Stat.OK {
			continue
		}
		stat := row.Cells[0].Stat
		marks = append(marks,
			swcharts.ErrorBarMark(
				swcharts.XFloatRange("Spread", stat.Min, stat.Max),
				swcharts.YString("Benchmark", label),
				swcharts.HeightFixed(10),
			).
				ForegroundStyle(color).
				Opacity(0.55).
				AccessibilityHidden(true),
		)
		point := swcharts.PointMark(
			swcharts.XFloat("Mean", stat.Mean),
			swcharts.YString("Benchmark", label),
		).
			ForegroundStyle(color).
			Symbol(swcharts.SymbolCircle).
			SymbolDiameter(10).
			AccessibilityLabel(fmt.Sprintf("%s %s", row.Name, config)).
			AccessibilityValue(row.Cells[0].Value)
		if shouldAnnotateSingleInput(table, i) {
			point = point.TextAnnotation(row.Cells[0].Value, swcharts.AnnotationTrailing).
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
	chart := swcharts.Chart(marks...).
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
		)
	if chart.Pointer() == 0 {
		return swiftui.ViewFromPointer(0)
	}
	return chart.Frame(-1, height)
}

func benchmarkDeltaChart(table tableView) swiftui.View {
	if !table.HasComparisons || len(table.Rows) == 0 {
		return swiftui.ViewFromPointer(0)
	}
	marks := make([]swcharts.Mark, 0, len(table.Rows)*len(table.Configs))
	styles := make([]swcharts.StyleScaleEntry, 0, max(0, len(table.Configs)-1))
	benchmarkOrder := make([]string, 0, len(table.Rows))
	maxAbs := 0.0
	for i := 1; i < len(table.Configs); i++ {
		styles = append(styles, swcharts.StyleScale(table.Configs[i], chartColor(i)))
	}
	for _, row := range table.Rows {
		label := chartRowLabel(row.Name)
		benchmarkOrder = append(benchmarkOrder, label)
		for i := 1; i < len(row.Cells); i++ {
			cell := row.Cells[i]
			if !cell.HasDelta {
				continue
			}
			maxAbs = math.Max(maxAbs, math.Abs(cell.DeltaPct))
			start, end := 0.0, cell.DeltaPct
			if cell.DeltaPct < 0 {
				start, end = cell.DeltaPct, 0
			}
			mark := swcharts.RangeBarMark(
				swcharts.XFloatRange("Delta", start, end),
				swcharts.YString("Benchmark", label),
			).
				PositionBy("Input", table.Configs[i]).
				ForegroundStyleBy("Input", table.Configs[i]).
				Opacity(0.9).
				CornerRadius(5).
				AccessibilityLabel(fmt.Sprintf("%s %s delta", row.Name, table.Configs[i])).
				AccessibilityValue(formatDeltaPercent(cell.DeltaPct))
			if shouldAnnotateDelta(table, cell) {
				position := swcharts.AnnotationTrailing
				offset := 8.0
				if cell.DeltaPct < 0 {
					position = swcharts.AnnotationLeading
					offset = -8
				}
				mark = mark.TextAnnotation(formatDeltaPercent(cell.DeltaPct), position).
					AnnotationOffset(offset, 0).
					AnnotationOverflow(swcharts.AnnotationOverflowFitPlot, swcharts.AnnotationOverflowFitPlot)
			}
			marks = append(marks, mark)
		}
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
	chart := swcharts.Chart(marks...).
		ChartForegroundStyleScale(styles...).
		ChartLegend(metricLegend(table)).
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
		)
	if chart.Pointer() == 0 {
		return swiftui.ViewFromPointer(0)
	}
	return swiftui.VStackSpaced(6,
		swiftui.HStack(
			swiftui.Text("Delta vs baseline").
				Font(swiftui.FontCaption).
				FontWeight(swiftui.WeightSemibold).
				ForegroundStyleNamed("secondary"),
			swiftui.Spacer(),
		),
		chart.Frame(-1, height),
	)
}

func metricSummaryTitle(table tableView) string {
	if len(table.Configs) == 1 {
		return fmt.Sprintf("%d rows with %s and confidence interval", len(table.Rows), table.SummaryLabel)
	}
	if !table.HasComparisons {
		return fmt.Sprintf("%d inputs compared side by side", len(table.Configs))
	}
	return fmt.Sprintf("%d comparisons against baseline across %d inputs", table.Compared, len(table.Configs))
}

func metricInsight(table tableView) string {
	if table.HasComparisons {
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
		if table.WarningCount > 0 {
			parts = append(parts, fmt.Sprintf("%d warnings", table.WarningCount))
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

func benchmarkCell(table tableView, row rowView, prefs displayPrefs) swiftui.View {
	titleLabel, detailLabel := benchmarkLabelParts(row.Name)
	title := swiftui.HStack(
		swiftui.Text(titleLabel).
			Font(swiftui.FontCaption).
			FontWeight(swiftui.WeightSemibold).
			LineLimit(1),
		swiftui.Spacer(),
	)
	if !prefs.CompactRows {
		parts := make([]string, 0, 3)
		if detailLabel != "" {
			parts = append(parts, detailLabel)
		}
		if row.SpreadPct > 0 {
			parts = append(parts, fmt.Sprintf("%s spread", formatPercent(row.SpreadPct)))
		}
		if row.Warning != "" {
			parts = append(parts, row.Warning)
		}
		if len(parts) > 0 {
			title = swiftui.VStackSpaced(2,
				title,
				swiftui.HStack(
					swiftui.Text(strings.Join(parts, " • ")).
						Font(swiftui.FontCaption2).
						ForegroundStyleNamed("secondary").
						LineLimit(1),
					swiftui.Spacer(),
				),
			)
		}
	}
	views := []swiftui.Viewable{title.MaxFrame(-1, 0)}
	if table.HasComparisons {
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
	return swiftui.HStackSpaced(6, chips...)
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
	).Padding(5).
		Background(r, g, b, 0.18).
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

func modeSummary(view comparisonView) string {
	switch view.Mode {
	case "single":
		return "Single input: summary and confidence interval."
	case "single-live":
		return fmt.Sprintf("Streaming single input on %s.", view.StreamingLabel)
	case "compare":
		return "Two-input compare against the first column baseline."
	case "stream-compare":
		return fmt.Sprintf("Streaming comparison on %s.", view.StreamingLabel)
	default:
		return "Multi-input comparison against the first column baseline."
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

func metricCellView(cell valueCellView, width float64) swiftui.View {
	if !cell.HasValue {
		return monoText("n/a", width)
	}
	top := []swiftui.Viewable{
		swiftui.Text(cell.Value).
			Font(swiftui.FontCaption).
			FontWeight(swiftui.WeightSemibold).
			MonospacedDigit().
			AsView(),
	}
	if cell.HasDelta {
		r, g, b := changeColor(cell.Change)
		top = append(top,
			swiftui.Text(cell.Delta).
				Font(swiftui.FontCaption2).
				FontWeight(swiftui.WeightSemibold).
				MonospacedDigit().
				ForegroundStyle(r, g, b, 1.0).
				AsView(),
		)
	}
	meta := []string{cell.Range}
	if cell.HasDelta && cell.Note != "" {
		meta = append(meta, cell.Note)
	}
	if cell.Warning != "" {
		meta = append(meta, cell.Warning)
	}
	return swiftui.VStackSpaced(1,
		swiftui.HStackSpaced(6, append(top, swiftui.Spacer())...),
		swiftui.HStack(
			swiftui.Text(strings.Join(meta, " • ")).
				Font(swiftui.FontCaption2).
				ForegroundStyleNamed("secondary").
				LineLimit(1),
			swiftui.Spacer(),
		),
	).
		Padding(6).
		Background(1, 1, 1, 0.025).
		CornerRadius(6).
		Frame(width, 0)
}

func summaryCellViewBox(cell summaryCellView, width float64) swiftui.View {
	if !cell.HasValue && !cell.HasDelta {
		return monoText(" ", width)
	}
	top := []swiftui.Viewable{}
	if cell.HasValue {
		top = append(top,
			swiftui.Text(cell.Value).
				Font(swiftui.FontCaption).
				FontWeight(swiftui.WeightSemibold).
				MonospacedDigit().
				AsView(),
		)
	}
	meta := make([]string, 0, 2)
	if cell.HasDelta {
		r, g, b := changeColor(changeFromDeltaLabel(cell.Delta))
		top = append(top,
			swiftui.Text(cell.Delta).
				Font(swiftui.FontCaption2).
				FontWeight(swiftui.WeightSemibold).
				MonospacedDigit().
				ForegroundStyle(r, g, b, 1.0).
				AsView(),
		)
	}
	if cell.Warning != "" {
		meta = append(meta, cell.Warning)
	}
	views := []swiftui.Viewable{
		swiftui.HStackSpaced(6, append(top, swiftui.Spacer())...),
	}
	if len(meta) > 0 {
		views = append(views,
			swiftui.HStack(
				swiftui.Text(strings.Join(meta, " • ")).
					Font(swiftui.FontCaption2).
					ForegroundStyleNamed("secondary").
					LineLimit(1),
				swiftui.Spacer(),
			),
		)
	}
	return swiftui.VStackSpaced(1, views...).
		Padding(6).
		Background(1, 1, 1, 0.025).
		CornerRadius(6).
		Frame(width, 0)
}

func tableGroupTitle(table tableView) string {
	if table.Title != "" {
		return compactTableTitle(table.Title)
	}
	return table.Metric
}

func metricSubtitle(table tableView) string {
	parts := []string{table.SummaryLabel, table.Metric}
	if table.WarningCount > 0 {
		parts = append(parts, fmt.Sprintf("%d warn", table.WarningCount))
	}
	return strings.Join(parts, " • ")
}

func previewPanel(view comparisonView) swiftui.View {
	title := "Benchstat Preview"
	if view.Options.Format == formatCSV {
		title = "CSV Preview"
	} else if view.Options.Format == formatText {
		title = "Text Preview"
	}
	preview := view.Preview
	if preview == "" {
		preview = "No output."
	}
	return swiftui.GroupBox(title,
		swiftui.VStackSpaced(8,
			swiftui.HStack(
				swiftui.Text("The preview follows the current benchstat filter and projection settings.").
					Font(swiftui.FontCaption).
					ForegroundStyleNamed("secondary"),
				swiftui.Spacer(),
			),
			swiftui.Text(preview).
				Font(swiftui.FontCaption).
				LineLimit(0),
		).Padding(12),
	).MaxFrame(-1, 0)
}

func errorPanel(message string) swiftui.View {
	return swiftui.GroupBox("Analysis Error",
		swiftui.VStackSpaced(8,
			swiftui.Text(message).
				Font(swiftui.FontCallout).
				ForegroundStyle(0.9, 0.5, 0.2, 1.0),
			swiftui.Text("Adjust the benchstat options and apply again.").
				Font(swiftui.FontCaption).
				ForegroundStyleNamed("secondary"),
		).Padding(12),
	).MaxFrame(-1, 0)
}

func summaryScaler(table *analysisTable) benchunit.Scaler {
	values := make([]float64, 0, len(table.Cols))
	for _, col := range table.Cols {
		summary := table.Summary[col]
		if summary != nil && summary.HasSummary {
			values = append(values, summary.Summary)
		}
	}
	return benchunit.CommonScale(values, benchunit.ClassOf(table.Unit))
}

func keyLabel(key benchproc.Key, fallback string) string {
	label := key.StringValues()
	if label != "" {
		return label
	}
	if fallback != "" {
		return fallback
	}
	return "value"
}

func tableLabel(key benchproc.Key) string {
	lines := keyLines(key, false)
	if len(lines) == 0 {
		return key.Get(key.Projection().FlattenedFields()[len(key.Projection().FlattenedFields())-1])
	}
	return strings.Join(lines, " • ")
}

func summarySpread(summary benchmath.Summary) float64 {
	if summary.Center == 0 {
		return 0
	}
	return math.Max(summary.Hi/summary.Center-1, 1-summary.Lo/summary.Center)
}

func comparisonChange(better int, cmp benchmath.Comparison, deltaPct float64, deltaLabel string) int {
	if deltaLabel == "~" || deltaLabel == "?" || better == 0 || deltaPct == 0 {
		return 0
	}
	if cmp.P > cmp.Alpha {
		return 0
	}
	if deltaPct*float64(better) > 0 {
		return 1
	}
	return -1
}

func joinWarnings(groups ...[]error) string {
	parts := make([]string, 0, 4)
	for _, group := range groups {
		for _, err := range group {
			if err == nil {
				continue
			}
			parts = append(parts, err.Error())
		}
	}
	return strings.Join(uniqueStrings(parts), "; ")
}

func uniqueStrings(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	seen := make(map[string]bool, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

func changeFromDeltaLabel(delta string) int {
	switch {
	case strings.HasPrefix(delta, "-"):
		return 1
	case strings.HasPrefix(delta, "+"):
		return -1
	default:
		return 0
	}
}

func shouldShowCharts(table tableView) bool {
	if len(table.Rows) == 0 {
		return false
	}
	limit := 6
	if len(table.Configs) == 1 {
		limit = 8
	}
	return len(table.Rows) <= limit
}

func chartRowLabel(name string) string {
	label, _ := benchmarkLabelParts(name)
	return abbreviateBenchmarkLabel(label)
}

func benchmarkLabelParts(name string) (string, string) {
	raw := strings.TrimPrefix(name, "Benchmark")
	if raw == "" {
		return name, ""
	}
	parts := strings.Split(raw, "/")
	base := parts[0]
	if len(parts) == 1 {
		base = trimBenchmarkSuffix(base)
	}
	attrs := make(map[string]string)
	segments := make([]string, 0, len(parts))
	for _, part := range parts[1:] {
		if key, value, ok := strings.Cut(part, "="); ok {
			attrs[key] = value
			continue
		}
		segments = append(segments, part)
	}
	op := cleanBenchmarkSegment(base, false)
	if len(segments) > 0 {
		op = cleanBenchmarkSegment(segments[len(segments)-1], false)
	}
	titleParts := make([]string, 0, 2)
	if language := attrs["language"]; language != "" {
		titleParts = append(titleParts, language)
	}
	if op != "" {
		titleParts = append(titleParts, op)
	}
	title := strings.Join(titleParts, " ")
	if title == "" {
		title = cleanBenchmarkSegment(base, false)
	}
	meta := make([]string, 0, 4)
	for _, key := range []string{"model", "size", "mode"} {
		if value := attrs[key]; value != "" {
			meta = append(meta, compactAttrValue(key, value))
		}
	}
	if len(segments) > 1 {
		for _, segment := range segments[:len(segments)-1] {
			meta = append(meta, cleanBenchmarkSegment(segment, false))
		}
	}
	return title, strings.Join(meta, " • ")
}

func abbreviateBenchmarkLabel(label string) string {
	replacer := strings.NewReplacer(
		"Python", "Py",
		"Generation", "Gen",
	)
	return replacer.Replace(label)
}

func compactTableTitle(title string) string {
	parts := strings.Split(title, " • ")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		switch {
		case strings.HasPrefix(part, ".config="):
			config := strings.TrimPrefix(part, ".config=")
			for _, field := range strings.Fields(config) {
				switch {
				case strings.HasPrefix(field, "pkg="):
					out = append(out, filepath.Base(strings.TrimPrefix(field, "pkg=")))
				case strings.HasPrefix(field, "cpu="):
					out = append(out, strings.TrimPrefix(field, "cpu="))
				case strings.HasPrefix(field, "goos="), strings.HasPrefix(field, "goarch="):
				default:
					out = append(out, field)
				}
			}
		case strings.HasPrefix(part, "pkg="):
			out = append(out, filepath.Base(strings.TrimPrefix(part, "pkg=")))
		case strings.HasPrefix(part, "cpu="):
			out = append(out, strings.TrimPrefix(part, "cpu="))
		case strings.HasPrefix(part, "goos="), strings.HasPrefix(part, "goarch="):
		default:
			out = append(out, part)
		}
	}
	out = uniqueStrings(out)
	if len(out) == 0 {
		return title
	}
	return strings.Join(out, " • ")
}

func trimBenchmarkSuffix(name string) string {
	if i := strings.LastIndex(name, "-"); i >= 0 && allDigits(name[i+1:]) {
		return name[:i]
	}
	return name
}

func allDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func cleanBenchmarkSegment(segment string, short bool) string {
	segment = strings.ReplaceAll(segment, "-", " ")
	segment = strings.ReplaceAll(segment, "_", " ")
	segment = strings.TrimSpace(segment)
	if !short {
		return segment
	}
	return abbreviateBenchmarkLabel(segment)
}

func compactAttrValue(key, value string) string {
	switch key {
	case "model":
		return strings.TrimSuffix(value, "-4bit")
	default:
		return value
	}
}

func shouldAnnotateDelta(table tableView, cell valueCellView) bool {
	if cell.DeltaPct == 0 || table.Domain.MaxDelta == 0 {
		return false
	}
	if cell.Change != 0 && math.Abs(cell.DeltaPct) >= table.Domain.MaxDelta*0.6 {
		return true
	}
	return math.Abs(cell.DeltaPct) >= table.Domain.MaxDelta*0.9
}

func tableTopDelta(table tableView, change int) rowHighlight {
	var best rowHighlight
	for _, row := range table.Rows {
		for i, cell := range row.Cells {
			if !cell.HasDelta || cell.Change != change {
				continue
			}
			value := math.Abs(cell.DeltaPct)
			if !best.Valid || value > best.Value {
				best = rowHighlight{
					Valid:     true,
					Metric:    table.Metric,
					Benchmark: row.Name,
					Config:    table.Configs[i],
					Detail:    formatDeltaPercent(cell.DeltaPct),
					Value:     value,
					Change:    change,
				}
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
		if table.HasComparisons {
			aSig := a.Change != 0
			bSig := b.Change != 0
			if aSig != bSig {
				return aSig
			}
			aDelta := a.MaxDelta
			bDelta := b.MaxDelta
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

func formatDeltaPercent(value float64) string {
	return fmt.Sprintf("%+.2f%%", value)
}

func trimFloat(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}

func formatConfidence(v float64) string {
	return fmt.Sprintf("%.0f%%", v*100)
}

func formatIndex(format string) int {
	switch format {
	case formatText:
		return 1
	case formatCSV:
		return 2
	default:
		return 0
	}
}

func formatFromIndex(index int) string {
	switch index {
	case 1:
		return formatText
	case 2:
		return formatCSV
	default:
		return formatDashboard
	}
}

func clampIndex(v, n int) int {
	if v < 0 {
		return 0
	}
	if v >= n {
		return n - 1
	}
	return v
}

func segmentedOptions(labels ...string) swiftui.View {
	children := make([]swiftui.Viewable, 0, len(labels))
	for i, label := range labels {
		children = append(children, swiftui.Text(label).AsView().Tag(int32(i)))
	}
	return swiftui.VStack(children...)
}

func parseAnalysisOptions(filter, table, row, col, ignore, alpha, confidence, format string) (analysisOptions, error) {
	opts := defaultAnalysisOptions()
	if strings.TrimSpace(filter) != "" {
		opts.Filter = strings.TrimSpace(filter)
	}
	if strings.TrimSpace(table) != "" {
		opts.Table = strings.TrimSpace(table)
	}
	if strings.TrimSpace(row) != "" {
		opts.Row = strings.TrimSpace(row)
	}
	if strings.TrimSpace(col) != "" {
		opts.Col = strings.TrimSpace(col)
	}
	opts.Ignore = strings.TrimSpace(ignore)
	if strings.TrimSpace(alpha) != "" {
		v, err := strconv.ParseFloat(strings.TrimSpace(alpha), 64)
		if err != nil {
			return analysisOptions{}, fmt.Errorf("invalid alpha: %w", err)
		}
		opts.Alpha = v
	}
	if strings.TrimSpace(confidence) != "" {
		v, err := strconv.ParseFloat(strings.TrimSpace(confidence), 64)
		if err != nil {
			return analysisOptions{}, fmt.Errorf("invalid confidence: %w", err)
		}
		opts.Confidence = v
	}
	opts.Format = format
	return opts, nil
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
		return 80
	case n >= 4:
		return 92
	default:
		return 108
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
