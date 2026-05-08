//go:build darwin

// Command autoresearchbench-viz renders a native SwiftUI live visualizer for
// AutoResearchBench swarm data.
//
// v0 reads /tmp/ensue-swarm/{chip}/{ram}/{type}/*.json once at startup and
// renders a horizon-chart timeline: per-agent RectangleMark lanes plus
// PointMark event glyphs, with native Swift Charts horizontal scrubbing.
//
// Usage:
//
//	go run ./examples/autoresearchbench-viz
//	go run ./examples/autoresearchbench-viz /path/to/swarm-dir
package main

import (
	"fmt"
	"log"
	"os"
	"runtime"

	"github.com/tmc/swiftui"
)

func init() { runtime.LockOSThread() }

const (
	defaultSwarmDir = "/tmp/ensue-swarm"
	windowWidth     = 1360
	windowHeight    = 720
	chartWidth      = 1280
	chartHeight     = 420
)

var (
	colMuted    = swiftui.RGB(0.46, 0.50, 0.56)
	colCard     = swiftui.RGBA(1, 1, 1, 0.90)
	colCardEdge = swiftui.RGBA(0.22, 0.28, 0.34, 0.08)
)

func main() {
	dir := defaultSwarmDir
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}

	events, err := loadSwarm(dir)
	if err != nil {
		log.Fatalf("autoresearchbench-viz: load %s: %v", dir, err)
	}
	if len(events) == 0 {
		log.Fatalf("autoresearchbench-viz: no events found under %s", dir)
	}

	agents := uniqueAgents(events)
	start, end := timeBounds(events)

	hero := headerHero(
		"AutoResearchBench Swarm Timeline",
		fmt.Sprintf("%d events · %d agents · %s → %s",
			len(events), len(agents),
			start.Format("Jan 2 15:04"), end.Format("Jan 2 15:04")),
		"Scroll horizontally to scrub; 1-hour visible window; zoom not enabled in v0",
		fmt.Sprintf("Source: %s", dir),
	)

	chart := panel(
		"Horizon timeline",
		"Per-agent activity lanes with event glyphs (circle=result, triangle=insight, diamond=claim, pentagon=best, square=baseline)",
		timelineChart(events, agents).Frame(chartWidth, chartHeight),
	)

	root := swiftui.VStackSpaced(16, hero, chart).
		Padding(20).
		BackgroundStyle("windowBackground")

	swiftui.Run(swiftui.AppConfig{
		Title:  "AutoResearchBench Viz",
		Width:  windowWidth,
		Height: windowHeight,
	}, root)
}

func headerHero(title, subtitle string, lines ...string) swiftui.View {
	items := []swiftui.Viewable{
		swiftui.HStack(
			swiftui.Text(title).
				Font(swiftui.FontLargeTitle).
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
	}
	for _, line := range lines {
		items = append(items, swiftui.HStack(
			swiftui.Text("• "+line).
				Font(swiftui.FontCaption).
				ForegroundStyle(colMuted.R, colMuted.G, colMuted.B, 1),
			swiftui.Spacer(),
		))
	}
	return swiftui.VStackSpaced(6, items...).Padding(18).
		Background(colCard.R, colCard.G, colCard.B, colCard.A).
		CornerRadius(20).
		Shadow(0.08, 0.11, 0.15, 0.08, 14, 0, 8)
}

func panel(title, subtitle string, content swiftui.View) swiftui.View {
	return swiftui.VStackSpaced(10,
		swiftui.HStack(
			swiftui.VStackSpaced(3,
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
			).MaxFrame(-1, 0),
		),
		content,
	).Padding(16).
		Background(colCard.R, colCard.G, colCard.B, colCard.A).
		CornerRadius(18).
		Shadow(0.08, 0.11, 0.15, 0.08, 12, 0, 6).
		Border(colCardEdge.R, colCardEdge.G, colCardEdge.B, colCardEdge.A, 1)
}
