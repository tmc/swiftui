package main

import (
	"strings"
	"testing"
)

func TestParseInputsImplicitStdinOnly(t *testing.T) {
	inputs, err := parseInputs(nil, true)
	if err != nil {
		t.Fatalf("parseInputs: %v", err)
	}
	if len(inputs) != 1 {
		t.Fatalf("len(inputs) = %d, want 1", len(inputs))
	}
	if !inputs[0].Stream {
		t.Fatalf("inputs[0].Stream = false, want true")
	}
	if got, want := inputs[0].Label, "stdin"; got != want {
		t.Fatalf("inputs[0].Label = %q, want %q", got, want)
	}
}

func TestParseInputsImplicitStdinWithFiles(t *testing.T) {
	inputs, err := parseInputs([]string{"bench-old.txt", "bench-new.txt"}, true)
	if err != nil {
		t.Fatalf("parseInputs: %v", err)
	}
	if len(inputs) != 3 {
		t.Fatalf("len(inputs) = %d, want 3", len(inputs))
	}
	if got, want := inputs[0].Label, "bench-old.txt"; got != want {
		t.Fatalf("inputs[0].Label = %q, want %q", got, want)
	}
	if got, want := inputs[1].Label, "bench-new.txt"; got != want {
		t.Fatalf("inputs[1].Label = %q, want %q", got, want)
	}
	if !inputs[2].Stream {
		t.Fatalf("inputs[2].Stream = false, want true")
	}
}

func TestBuildComparisonMultipleInputs(t *testing.T) {
	inputs := []inputSpec{
		{
			Label: "old",
			Data:  []byte("goos: darwin\npkg: example\nBenchmarkFoo-8\t1\t100 ns/op\nBenchmarkBar-8\t1\t200 ns/op\n"),
		},
		{
			Label: "new",
			Data:  []byte("goos: darwin\npkg: example\nBenchmarkFoo-8\t1\t90 ns/op\nBenchmarkBar-8\t1\t210 ns/op\n"),
		},
		{
			Label: "trial",
			Data:  []byte("goos: darwin\npkg: example\nBenchmarkFoo-8\t1\t95 ns/op\nBenchmarkBar-8\t1\t205 ns/op\n"),
		},
	}

	view := buildComparison(inputs)
	if got, want := view.Inputs, 3; got != want {
		t.Fatalf("view.Inputs = %d, want %d", got, want)
	}
	if len(view.Tables) == 0 {
		t.Fatal("len(view.Tables) = 0, want > 0")
	}
	table := view.Tables[0]
	if got, want := len(table.Configs), 3; got != want {
		t.Fatalf("len(table.Configs) = %d, want %d", got, want)
	}
	if !table.HasComparisons {
		t.Fatal("table.HasComparisons = false, want true for baseline comparisons")
	}
	if len(table.Rows) == 0 {
		t.Fatal("len(table.Rows) = 0, want > 0")
	}
	if got, want := len(table.Rows[0].Cells), 3; got != want {
		t.Fatalf("len(table.Rows[0].Cells) = %d, want %d", got, want)
	}
	if got, want := view.Mode, "multi"; got != want {
		t.Fatalf("view.Mode = %q, want %q", got, want)
	}
	if got := table.Insight; got == "" {
		t.Fatal("table.Insight = empty, want non-empty")
	}
}

func TestBuildComparisonHighlightsAndMode(t *testing.T) {
	inputs := []inputSpec{
		{
			Label: "old",
			Data: []byte("goos: darwin\npkg: example\n" +
				"BenchmarkFast-8\t1\t100 ns/op\n" +
				"BenchmarkFast-8\t1\t101 ns/op\n" +
				"BenchmarkFast-8\t1\t99 ns/op\n" +
				"BenchmarkFast-8\t1\t100 ns/op\n" +
				"BenchmarkFast-8\t1\t101 ns/op\n" +
				"BenchmarkFast-8\t1\t99 ns/op\n" +
				"BenchmarkSlow-8\t1\t100 ns/op\n" +
				"BenchmarkSlow-8\t1\t101 ns/op\n" +
				"BenchmarkSlow-8\t1\t99 ns/op\n" +
				"BenchmarkSlow-8\t1\t100 ns/op\n" +
				"BenchmarkSlow-8\t1\t101 ns/op\n" +
				"BenchmarkSlow-8\t1\t99 ns/op\n"),
		},
		{
			Label: "new",
			Data: []byte("goos: darwin\npkg: example\n" +
				"BenchmarkFast-8\t1\t80 ns/op\n" +
				"BenchmarkFast-8\t1\t81 ns/op\n" +
				"BenchmarkFast-8\t1\t79 ns/op\n" +
				"BenchmarkFast-8\t1\t80 ns/op\n" +
				"BenchmarkFast-8\t1\t81 ns/op\n" +
				"BenchmarkFast-8\t1\t79 ns/op\n" +
				"BenchmarkSlow-8\t1\t140 ns/op\n" +
				"BenchmarkSlow-8\t1\t141 ns/op\n" +
				"BenchmarkSlow-8\t1\t139 ns/op\n" +
				"BenchmarkSlow-8\t1\t140 ns/op\n" +
				"BenchmarkSlow-8\t1\t141 ns/op\n" +
				"BenchmarkSlow-8\t1\t139 ns/op\n"),
		},
		{
			Label:  "stdin",
			Path:   "-",
			Stream: true,
			Data: []byte("goos: darwin\npkg: example\n" +
				"BenchmarkFast-8\t1\t82 ns/op\n" +
				"BenchmarkSlow-8\t1\t138 ns/op\n"),
		},
	}

	view := buildComparison(inputs[:2])
	if got, want := view.Mode, "compare"; got != want {
		t.Fatalf("view.Mode = %q, want %q", got, want)
	}
	if len(view.Highlights) == 0 {
		t.Fatal("len(view.Highlights) = 0, want > 0")
	}
	table := view.Tables[0]
	if !table.HasComparisons {
		t.Fatal("table.HasComparisons = false, want true")
	}
	if got := table.Domain.MaxValue; got <= 0 {
		t.Fatalf("table.Domain.MaxValue = %f, want > 0", got)
	}
	if got := table.Domain.MaxDelta; got <= 0 {
		t.Fatalf("table.Domain.MaxDelta = %f, want > 0", got)
	}
	if got := table.Insight; got == "" {
		t.Fatal("table.Insight = empty, want non-empty")
	}

	streamView := buildComparison(inputs)
	if got, want := streamView.Mode, "stream-compare"; got != want {
		t.Fatalf("streamView.Mode = %q, want %q", got, want)
	}
	if !streamView.HasStreaming {
		t.Fatal("streamView.HasStreaming = false, want true")
	}
	if got, want := streamView.StreamingLabel, "stdin"; got != want {
		t.Fatalf("streamView.StreamingLabel = %q, want %q", got, want)
	}
	if got := streamView.LiveResults; got != 2 {
		t.Fatalf("streamView.LiveResults = %d, want 2", got)
	}
}

func TestMetricInsight(t *testing.T) {
	table := tableView{
		Metric:         "ns/op",
		Configs:        []string{"old", "new"},
		HasComparisons: true,
		Improved:       1,
		Regressed:      1,
		Rows: []rowView{
			{
				Name:      "BenchmarkFast",
				SpreadPct: 0.04,
				Cells: []valueCellView{
					{},
					{HasDelta: true, Change: 1, DeltaPct: -0.20},
				},
			},
			{
				Name:      "BenchmarkSlow",
				SpreadPct: 0.08,
				Cells: []valueCellView{
					{},
					{HasDelta: true, Change: -1, DeltaPct: 0.40},
				},
			},
		},
	}

	got := metricInsight(table)
	if got == "" {
		t.Fatal("metricInsight(table) = empty, want non-empty")
	}
	if want := "largest win BenchmarkFast -0.20%"; !containsAll(got, "1 wins", "1 losses", want, "largest loss BenchmarkSlow +0.40%", "widest spread BenchmarkSlow 8.0% spread") {
		t.Fatalf("metricInsight(table) = %q, want required substrings", got)
	}
}

func TestSortTableRows(t *testing.T) {
	table := tableView{
		HasComparisons: true,
		Rows: []rowView{
			{Name: "steady", Change: 0, MaxDelta: 0.01, SpreadPct: 0.09},
			{Name: "small-win", Change: 1, MaxDelta: 0.10, SpreadPct: 0.02},
			{Name: "big-loss", Change: -1, MaxDelta: 0.35, SpreadPct: 0.01},
		},
	}

	sortTableRows(&table)

	got := []string{table.Rows[0].Name, table.Rows[1].Name, table.Rows[2].Name}
	want := []string{"big-loss", "small-win", "steady"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("row order = %v, want %v", got, want)
		}
	}
}

func TestMetricSummaryTitleSingleInput(t *testing.T) {
	table := tableView{
		Configs:      []string{"stdin"},
		SummaryLabel: "median",
		Rows:         []rowView{{Name: "BenchmarkFoo"}, {Name: "BenchmarkBar"}},
	}

	if got, want := metricSummaryTitle(table), "2 rows with median and confidence interval"; got != want {
		t.Fatalf("metricSummaryTitle(table) = %q, want %q", got, want)
	}
}

func TestBenchmarkLabelParts(t *testing.T) {
	gotLabel, gotMeta := benchmarkLabelParts("BenchmarkMLXGenerate/model=Qwen3.5-4B-4bit/language=Python/Generation-16")
	if got, want := gotLabel, "Python Generation 16"; got != want {
		t.Fatalf("benchmarkLabelParts label = %q, want %q", got, want)
	}
	if got, want := gotMeta, "Qwen3.5-4B"; got != want {
		t.Fatalf("benchmarkLabelParts meta = %q, want %q", got, want)
	}
	if got, want := chartRowLabel("BenchmarkMLXGenerate/model=Qwen3.5-4B-4bit/language=Python/Generation-16"), "Py Gen 16"; got != want {
		t.Fatalf("chartRowLabel = %q, want %q", got, want)
	}
}

func TestCompactTableTitle(t *testing.T) {
	title := ".config=goos=darwin goarch=arm64 pkg=github.com/tmc/mlx-go-lm/benchmarks/perfdelta cpu=Apple-M4-Max"
	if got, want := compactTableTitle(title), "perfdelta • Apple-M4-Max"; got != want {
		t.Fatalf("compactTableTitle = %q, want %q", got, want)
	}
}

func TestBuildComparisonWithProjectionAndFilter(t *testing.T) {
	inputs := []inputSpec{
		{
			Label: "old",
			Data: []byte("goos: darwin\npkg: example\n" +
				"BenchmarkFoo/size=4k-8\t1\t100 ns/op\n" +
				"BenchmarkFoo/size=8k-8\t1\t200 ns/op\n"),
		},
		{
			Label: "new",
			Data: []byte("goos: darwin\npkg: example\n" +
				"BenchmarkFoo/size=4k-8\t1\t90 ns/op\n" +
				"BenchmarkFoo/size=8k-8\t1\t210 ns/op\n"),
		},
	}

	opts := defaultAnalysisOptions()
	opts.Row = "/size"
	opts.Filter = "/size:4k"
	view := buildComparisonWithOptions(inputs, opts)
	if view.Error != "" {
		t.Fatalf("view.Error = %q, want empty", view.Error)
	}
	if got, want := len(view.Tables), 1; got != want {
		t.Fatalf("len(view.Tables) = %d, want %d", got, want)
	}
	if got, want := len(view.Tables[0].Rows), 1; got != want {
		t.Fatalf("len(view.Tables[0].Rows) = %d, want %d", got, want)
	}
	if got, want := view.Tables[0].Rows[0].Name, "4k"; got != want {
		t.Fatalf("row name = %q, want %q", got, want)
	}
}

func TestNextStreamRefresh(t *testing.T) {
	tests := []struct {
		results int
		want    int
	}{
		{results: 0, want: 1},
		{results: 1, want: 2},
		{results: 2, want: 3},
		{results: 3, want: 4},
		{results: 4, want: 8},
		{results: 8, want: 12},
	}
	for _, tt := range tests {
		if got := nextStreamRefresh(tt.results); got != tt.want {
			t.Fatalf("nextStreamRefresh(%d) = %d, want %d", tt.results, got, tt.want)
		}
	}
}

func containsAll(s string, subs ...string) bool {
	for _, sub := range subs {
		if !strings.Contains(s, sub) {
			return false
		}
	}
	return true
}
