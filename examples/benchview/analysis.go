package main

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"math"
	"runtime"
	"strings"
	"sync"

	"golang.org/x/perf/benchfmt"
	"golang.org/x/perf/benchmath"
	"golang.org/x/perf/benchproc"
	"golang.org/x/perf/benchunit"
)

type analysisOptions struct {
	Filter     string
	Table      string
	Row        string
	Col        string
	Ignore     string
	Alpha      float64
	Confidence float64
	Format     string
}

const (
	formatDashboard = "dashboard"
	formatText      = "text"
	formatCSV       = "csv"
)

func defaultAnalysisOptions() analysisOptions {
	return analysisOptions{
		Filter:     "*",
		Table:      ".config",
		Row:        ".fullname",
		Col:        ".file",
		Ignore:     "",
		Alpha:      benchmath.DefaultThresholds.CompareAlpha,
		Confidence: 0.95,
		Format:     formatDashboard,
	}
}

type analysisTable struct {
	Opts         analysisTableOpts
	Unit         string
	Assumption   benchmath.Assumption
	Rows         []benchproc.Key
	Cols         []benchproc.Key
	Cells        map[analysisCellKey]*analysisTableCell
	Summary      map[benchproc.Key]*analysisTableSummary
	SummaryLabel string
}

type analysisCellKey struct {
	Row benchproc.Key
	Col benchproc.Key
}

type analysisTableCell struct {
	Sample     *benchmath.Sample
	Summary    benchmath.Summary
	Baseline   *analysisTableCell
	Comparison benchmath.Comparison
}

type analysisTableSummary struct {
	HasSummary bool
	Summary    float64
	HasRatio   bool
	Ratio      float64
	Warnings   []error
}

type analysisTableOpts struct {
	Confidence float64
	Thresholds *benchmath.Thresholds
	Units      benchfmt.UnitMetadataMap
}

type analysisTables struct {
	Tables []*analysisTable
	Keys   []benchproc.Key
}

type analysisBuilder struct {
	tableBy   *benchproc.Projection
	rowBy     *benchproc.Projection
	colBy     *benchproc.Projection
	residue   *benchproc.Projection
	unitField *benchproc.Field
	tables    map[benchproc.Key]*analysisBuilderTable
}

type analysisBuilderTable struct {
	rows  map[benchproc.Key]struct{}
	cols  map[benchproc.Key]struct{}
	cells map[analysisCellKey]*analysisBuilderCell
}

type analysisBuilderCell struct {
	values  []float64
	residue map[benchproc.Key]struct{}
}

func newAnalysisBuilder(tableBy, rowBy, colBy, residue *benchproc.Projection) *analysisBuilder {
	fields := tableBy.Fields()
	unitField := fields[len(fields)-1]
	if unitField.Name != ".unit" {
		panic("analysis table projection missing .unit")
	}
	return &analysisBuilder{
		tableBy:   tableBy,
		rowBy:     rowBy,
		colBy:     colBy,
		residue:   residue,
		unitField: unitField,
		tables:    make(map[benchproc.Key]*analysisBuilderTable),
	}
}

func (b *analysisBuilder) Add(result *benchfmt.Result) {
	tableKeys := b.tableBy.ProjectValues(result)
	rowKey := b.rowBy.Project(result)
	colKey := b.colBy.Project(result)
	residueKey := b.residue.Project(result)
	cellKey := analysisCellKey{Row: rowKey, Col: colKey}

	for unitI, tableKey := range tableKeys {
		table := b.tables[tableKey]
		if table == nil {
			table = &analysisBuilderTable{
				rows:  make(map[benchproc.Key]struct{}),
				cols:  make(map[benchproc.Key]struct{}),
				cells: make(map[analysisCellKey]*analysisBuilderCell),
			}
			b.tables[tableKey] = table
		}
		cell := table.cells[cellKey]
		if cell == nil {
			cell = &analysisBuilderCell{residue: make(map[benchproc.Key]struct{})}
			table.cells[cellKey] = cell
			table.rows[rowKey] = struct{}{}
			table.cols[colKey] = struct{}{}
		}
		cell.values = append(cell.values, result.Values[unitI].Value)
		cell.residue[residueKey] = struct{}{}
	}
}

func (b *analysisBuilder) ToTables(opts analysisTableOpts) *analysisTables {
	var keys []benchproc.Key
	for key := range b.tables {
		keys = append(keys, key)
	}
	benchproc.SortKeys(keys)

	limit := make(chan struct{}, 2*runtime.GOMAXPROCS(-1))
	var wg sync.WaitGroup
	var out []*analysisTable

	for _, key := range keys {
		raw := b.tables[key]
		unit := key.Get(b.unitField)
		table := &analysisTable{
			Opts:       opts,
			Unit:       unit,
			Assumption: opts.Units.GetAssumption(unit),
			Rows:       mapAnalysisKeys(raw.rows),
			Cols:       mapAnalysisKeys(raw.cols),
			Cells:      make(map[analysisCellKey]*analysisTableCell),
		}
		out = append(out, table)

		for cellKey, rawCell := range raw.cells {
			values := append([]float64(nil), rawCell.values...)
			table.Cells[cellKey] = &analysisTableCell{
				Sample: benchmath.NewSample(values, opts.Thresholds),
			}
		}

		if len(table.Cols) == 0 {
			continue
		}
		baselineCol := table.Cols[0]
		wg.Add(len(raw.cells))
		for cellKey, rawCell := range raw.cells {
			cell := table.Cells[cellKey]
			if cellKey.Col != baselineCol {
				if base, ok := table.Cells[analysisCellKey{Row: cellKey.Row, Col: baselineCol}]; ok {
					cell.Baseline = base
				}
			}
			limit <- struct{}{}
			go func(rawCell *analysisBuilderCell, cell *analysisTableCell) {
				analyzeCell(rawCell, cell, table.Assumption, opts.Confidence)
				<-limit
				wg.Done()
			}(rawCell, cell)
		}
	}
	wg.Wait()

	for _, table := range out {
		table.SummaryLabel = "geomean"
		table.Summary = make(map[benchproc.Key]*analysisTableSummary)
		if len(table.Cols) == 0 {
			continue
		}
		baseCol := table.Cols[0]
		nBase := 0
		for _, row := range table.Rows {
			if _, ok := table.Cells[analysisCellKey{Row: row, Col: baseCol}]; ok {
				nBase++
			}
		}
		for i, col := range table.Cols {
			summary := &analysisTableSummary{}
			table.Summary[col] = summary
			isBase := i == 0
			wg.Add(1)
			limit <- struct{}{}
			go func(table *analysisTable, col benchproc.Key, summary *analysisTableSummary, nBase int, isBase bool) {
				analyzeColumn(table, col, summary, nBase, isBase)
				<-limit
				wg.Done()
			}(table, col, summary, nBase, isBase)
		}
	}
	wg.Wait()

	return &analysisTables{Tables: out, Keys: keys}
}

func (t *analysisTable) rowScaler(row benchproc.Key) benchunit.Scaler {
	values := make([]float64, 0, len(t.Cols))
	for _, col := range t.Cols {
		cell, ok := t.Cells[analysisCellKey{Row: row, Col: col}]
		if ok {
			values = append(values, cell.Summary.Center)
		}
	}
	return benchunit.CommonScale(values, benchunit.ClassOf(t.Unit))
}

func (t *analysisTables) CSVPreview() string {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	for i, table := range t.Tables {
		if i > 0 {
			w.Write(nil)
		}
		key := t.Keys[i]
		for _, line := range keyLines(key, true) {
			w.Write([]string{line})
		}
		headers := []string{"row"}
		for colI, col := range table.Cols {
			colLabel := col.StringValues()
			if colLabel == "" {
				colLabel = fmt.Sprintf("col%d", colI+1)
			}
			headers = append(headers, colLabel+" "+table.Unit, "CI")
			if colI > 0 {
				headers = append(headers, "vs base", "P")
			}
		}
		w.Write(headers)
		for _, row := range table.Rows {
			record := []string{row.StringValues()}
			for colI, col := range table.Cols {
				cell, ok := table.Cells[analysisCellKey{Row: row, Col: col}]
				if !ok {
					if colI == 0 {
						record = append(record, "", "")
					} else {
						record = append(record, "", "", "", "")
					}
					continue
				}
				record = append(record, fmt.Sprintf("%g", cell.Summary.Center), cell.Summary.PctRangeString())
				if colI > 0 {
					if cell.Baseline == nil {
						record = append(record, "", "")
					} else {
						record = append(record,
							cell.Comparison.FormatDelta(cell.Baseline.Summary.Center, cell.Summary.Center),
							cell.Comparison.String(),
						)
					}
				}
			}
			w.Write(record)
		}
		if len(table.Rows) > 1 {
			record := []string{table.SummaryLabel}
			for colI, col := range table.Cols {
				summary := table.Summary[col]
				if summary != nil && summary.HasSummary {
					record = append(record, fmt.Sprintf("%g", summary.Summary))
				} else {
					record = append(record, "")
				}
				if colI == 0 {
					record = append(record, "")
					continue
				}
				if summary != nil && summary.HasRatio {
					record = append(record, "", fmt.Sprintf("%+.2f%%", (summary.Ratio-1)*100), "")
				} else {
					record = append(record, "", "", "")
				}
			}
			w.Write(record)
		}
	}
	w.Flush()
	return strings.TrimSpace(buf.String())
}

func (t *analysisTables) TextPreview() string {
	var buf strings.Builder
	for i, table := range t.Tables {
		if i > 0 {
			buf.WriteString("\n\n")
		}
		for _, line := range keyLines(t.Keys[i], false) {
			buf.WriteString(line)
			buf.WriteByte('\n')
		}
		headers := make([]string, 0, len(table.Cols)+1)
		headers = append(headers, "row")
		for _, col := range table.Cols {
			label := col.StringValues()
			if label == "" {
				label = "(baseline)"
			}
			headers = append(headers, label)
		}
		buf.WriteString(strings.Join(headers, " | "))
		buf.WriteByte('\n')
		for _, row := range table.Rows {
			fields := []string{row.StringValues()}
			for colI, col := range table.Cols {
				cell, ok := table.Cells[analysisCellKey{Row: row, Col: col}]
				if !ok {
					fields = append(fields, "n/a")
					continue
				}
				value := fmt.Sprintf("%g ± %s", cell.Summary.Center, cell.Summary.PctRangeString())
				if colI > 0 && cell.Baseline != nil {
					value += " " + cell.Comparison.FormatDelta(cell.Baseline.Summary.Center, cell.Summary.Center)
					value += " (" + cell.Comparison.String() + ")"
				}
				fields = append(fields, value)
			}
			buf.WriteString(strings.Join(fields, " | "))
			buf.WriteByte('\n')
		}
		if len(table.Rows) > 1 {
			fields := []string{table.SummaryLabel}
			for colI, col := range table.Cols {
				summary := table.Summary[col]
				value := "?"
				if summary != nil && summary.HasSummary {
					value = fmt.Sprintf("%g", summary.Summary)
				}
				if colI > 0 && summary != nil && summary.HasRatio {
					value += " " + fmt.Sprintf("%+.2f%%", (summary.Ratio-1)*100)
				}
				fields = append(fields, value)
			}
			buf.WriteString(strings.Join(fields, " | "))
		}
	}
	return strings.TrimSpace(buf.String())
}

func loadAnalysisTables(inputs []inputSpec, opts analysisOptions) (*analysisTables, []string, error) {
	filter, err := benchproc.NewFilter(opts.Filter)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing -filter: %w", err)
	}
	var parser benchproc.ProjectionParser
	var parseErr error
	parseProjection := func(name, expr string, withUnit bool) *benchproc.Projection {
		if withUnit {
			proj, _, err := parser.ParseWithUnit(expr, filter)
			if err != nil && parseErr == nil {
				parseErr = fmt.Errorf("parsing %s: %w", name, err)
			}
			return proj
		}
		proj, err := parser.Parse(expr, filter)
		if err != nil && parseErr == nil {
			parseErr = fmt.Errorf("parsing %s: %w", name, err)
		}
		return proj
	}
	tableBy := parseProjection("-table", opts.Table, true)
	rowBy := parseProjection("-row", opts.Row, false)
	colBy := parseProjection("-col", opts.Col, false)
	parseProjection("-ignore", opts.Ignore, false)
	residue := parser.Residue()
	if parseErr != nil {
		return nil, nil, parseErr
	}
	if opts.Alpha < 0 || opts.Alpha > 1 {
		return nil, nil, fmt.Errorf("-alpha must be in range [0, 1]")
	}
	if opts.Confidence < 0 || opts.Confidence > 1 {
		return nil, nil, fmt.Errorf("-confidence must be in range [0, 1]")
	}

	builder := newAnalysisBuilder(tableBy, rowBy, colBy, residue)
	units := make(benchfmt.UnitMetadataMap)
	var warnings []string

	for _, input := range inputs {
		if len(bytes.TrimSpace(input.Data)) == 0 {
			continue
		}
		var reader benchfmt.Reader
		reader.Reset(bytes.NewReader(input.Data), input.Label, ".file", input.Label)
		for reader.Scan() {
			switch rec := reader.Result().(type) {
			case *benchfmt.SyntaxError:
				warnings = append(warnings, rec.Error())
			case *benchfmt.Result:
				ok, err := filter.Apply(rec)
				if !ok {
					if err != nil {
						warnings = append(warnings, err.Error())
					}
					continue
				}
				builder.Add(rec)
			}
		}
		if err := reader.Err(); err != nil {
			return nil, warnings, err
		}
		for key, unit := range reader.Units() {
			units[key] = unit
		}
	}

	thresholds := benchmath.DefaultThresholds
	thresholds.CompareAlpha = opts.Alpha
	return builder.ToTables(analysisTableOpts{
		Confidence: opts.Confidence,
		Thresholds: &thresholds,
		Units:      units,
	}), warnings, nil
}

func mapAnalysisKeys(m map[benchproc.Key]struct{}) []benchproc.Key {
	keys := make([]benchproc.Key, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	benchproc.SortKeys(keys)
	return keys
}

func analyzeCell(raw *analysisBuilderCell, cell *analysisTableCell, assumption benchmath.Assumption, confidence float64) {
	cell.Summary = assumption.Summary(cell.Sample, confidence)
	if cell.Baseline != nil {
		cell.Comparison = assumption.Compare(cell.Baseline.Sample, cell.Sample)
	}
	fields := benchproc.NonSingularFields(mapAnalysisKeys(raw.residue))
	if len(fields) == 0 {
		return
	}
	var b strings.Builder
	b.WriteString("benchmarks vary in ")
	for i, field := range fields {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(field.Name)
	}
	cell.Sample.Warnings = append(cell.Sample.Warnings, errors.New(b.String()))
}

func analyzeColumn(table *analysisTable, col benchproc.Key, summary *analysisTableSummary, nBase int, isBase bool) {
	summaries := make([]float64, 0, len(table.Rows))
	ratios := make([]float64, 0, len(table.Rows))
	badRatio := false
	for _, row := range table.Rows {
		cell, ok := table.Cells[analysisCellKey{Row: row, Col: col}]
		if !ok {
			continue
		}
		summaries = append(summaries, cell.Summary.Center)
		if cell.Baseline == nil {
			continue
		}
		a, b := cell.Summary.Center, cell.Baseline.Summary.Center
		switch {
		case a == b:
			ratios = append(ratios, 1)
		case b == 0:
			badRatio = true
			ratios = append(ratios, 0)
		default:
			ratios = append(ratios, a/b)
		}
	}
	if !isBase && nBase != len(ratios) {
		summary.Warnings = append(summary.Warnings, fmt.Errorf("benchmark set differs from baseline; geomeans may not be comparable"))
	}
	if gm, ok := geoMean(summaries); ok {
		summary.HasSummary = true
		summary.Summary = gm
	} else {
		summary.Warnings = append(summary.Warnings, fmt.Errorf("summaries must be >0 to compute geomean"))
	}
	if !isBase && !badRatio {
		if gm, ok := geoMean(ratios); ok {
			summary.HasRatio = true
			summary.Ratio = gm
		} else {
			summary.Warnings = append(summary.Warnings, fmt.Errorf("ratios must be >0 to compute geomean"))
		}
	}
}

func geoMean(values []float64) (float64, bool) {
	if len(values) == 0 {
		return 0, false
	}
	sum := 0.0
	for _, value := range values {
		if value <= 0 {
			return 0, false
		}
		sum += math.Log(value)
	}
	return math.Exp(sum / float64(len(values))), true
}

func keyLines(key benchproc.Key, withNames bool) []string {
	if key.IsZero() {
		return nil
	}
	fields := key.Projection().FlattenedFields()
	lines := make([]string, 0, len(fields))
	for _, field := range fields {
		if field.Name == ".unit" {
			continue
		}
		value := key.Get(field)
		if value == "" {
			continue
		}
		if withNames {
			lines = append(lines, fmt.Sprintf("%s: %s", field.Name, value))
		} else {
			lines = append(lines, fmt.Sprintf("%s=%s", field.Name, value))
		}
	}
	return lines
}
