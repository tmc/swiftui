//go:build darwin
// +build darwin

// Command system-monitor displays live Go runtime metrics using SwiftUI gauges
// and progress bars, updated every two seconds from a background goroutine.
//
// Usage:
//
//	go run .
package main

import (
	"fmt"
	"log"
	"runtime"
	"time"

	"github.com/tmc/swiftui"
)

func init() { runtime.LockOSThread() }

func main() {
	goroutines := swiftui.NewIntState(runtime.NumGoroutine())
	heapMB := swiftui.NewFloatState(0)
	sysMB := swiftui.NewFloatState(0)
	numGC := swiftui.NewIntState(0)

	// Percentages for gauges (0.0–1.0).
	heapPct := swiftui.NewFloatState(0)
	sysPct := swiftui.NewFloatState(0)
	gcPct := swiftui.NewFloatState(0)

	// Read initial stats.
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	heapMB.Set(float64(ms.HeapAlloc) / (1 << 20))
	sysMB.Set(float64(ms.Sys) / (1 << 20))
	numGC.Set(int(ms.NumGC))

	// Background goroutine for periodic updates.
	go func() {
		tick := time.NewTicker(2 * time.Second)
		defer tick.Stop()
		for range tick.C {
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			goroutines.Set(runtime.NumGoroutine())
			heap := float64(m.HeapAlloc) / (1 << 20)
			sys := float64(m.Sys) / (1 << 20)
			heapMB.Set(heap)
			sysMB.Set(sys)
			numGC.Set(int(m.NumGC))
			// Gauge percentages: heap relative to sys, sys capped at 256 MB, GC at 100 cycles.
			if sys > 0 {
				heapPct.Set(heap / sys)
			}
			sysPct.Set(clamp(sys / 256.0))
			gcPct.Set(clamp(float64(m.NumGC) / 100.0))
		}
	}()
	if err := swiftui.Run(swiftui.WithWindow(swiftui.AppConfig{
		Title:  "System Monitor",
		Width:  520,
		Height: 480,
	}, swiftui.ScrollView(
		swiftui.VStackSpaced(16,
			// Header
			swiftui.HStack(
				swiftui.Label("System Monitor", "waveform.path.ecg").
					Font(swiftui.FontTitle).
					FontWeight(swiftui.WeightBold),
				swiftui.Spacer(),
			),

			// Gauges row
			swiftui.GroupBox("Runtime Gauges",
				swiftui.HStackSpaced(24,
					gaugeView("Goroutines", goroutines, 0.35, 0.65, 1.0),
					circleGauge("Heap", heapPct, "memorychip.fill", 0.9, 0.5, 0.2),
					circleGauge("Sys", sysPct, "cpu", 0.3, 0.7, 1.0),
					circleGauge("GC", gcPct, "arrow.triangle.2.circlepath", 0.3, 0.8, 0.4),
				).Padding(12),
			).MaxFrame(-1, 0),

			// Detail cards
			swiftui.GroupBox("Memory",
				swiftui.VStackSpaced(12,
					metricRow("Heap Alloc", "memorychip.fill", heapMB, "MB", 256),
					metricRow("Sys Memory", "cpu", sysMB, "MB", 256),
				).Padding(8),
			).MaxFrame(-1, 0),

			swiftui.GroupBox("Scheduling",
				swiftui.VStackSpaced(12,
					swiftui.HStack(
						swiftui.Label("Goroutines", "arrow.triangle.branch").
							Font(swiftui.FontBody),
						swiftui.Spacer(),
						swiftui.TextFrom(goroutines).
							Font(swiftui.FontBody).
							FontWeight(swiftui.WeightSemibold).
							MonospacedDigit(),
					),
					swiftui.HStack(
						swiftui.Label("CPUs", "square.grid.3x3.fill").
							Font(swiftui.FontBody),
						swiftui.Spacer(),
						swiftui.Text(fmt.Sprintf("%d", runtime.NumCPU())).
							Font(swiftui.FontBody).
							FontWeight(swiftui.WeightSemibold).
							MonospacedDigit(),
					),
					swiftui.HStack(
						swiftui.Label("GC Cycles", "arrow.triangle.2.circlepath").
							Font(swiftui.FontBody),
						swiftui.Spacer(),
						swiftui.TextFrom(numGC).
							Font(swiftui.FontBody).
							FontWeight(swiftui.WeightSemibold).
							MonospacedDigit(),
					),
				).Padding(8),
			).MaxFrame(-1, 0),

			// Force GC button
			swiftui.HStack(
				swiftui.Spacer(),
				swiftui.Button("Force GC", func() {
					runtime.GC()
				}).ButtonStyle(swiftui.ButtonStyleBorderedProminent).
					ControlSize(swiftui.ControlSizeLarge),
				swiftui.Spacer(),
			),
		).Padding(24),
	))); err != nil {
		log.Fatal(err)
	}
}

func clamp(v float64) float64 {
	if v > 1 {
		return 1
	}
	if v < 0 {
		return 0
	}
	return v
}

// gaugeView shows a goroutine count as a circular gauge with DynamicView.
func gaugeView(label string, state *swiftui.IntState, r, g, b float64) swiftui.View {
	return swiftui.VStackSpaced(4,
		swiftui.DynamicView(state, func(v int) swiftui.View {
			pct := clamp(float64(v) / 50.0) // scale: 50 goroutines = full
			return swiftui.ZStack(
				swiftui.Circle().
					Stroke(swiftui.RGBA(r, g, b, 0.15), 8).
					Frame(64, 64).
					AsView(),
				swiftui.Circle().
					Fill(swiftui.RGBA(r, g, b, pct*0.3)).
					Frame(64, 64).
					AsView(),
				swiftui.VStack(
					swiftui.Text(fmt.Sprintf("%d", v)).
						Font(swiftui.FontTitle2).
						FontWeight(swiftui.WeightBold).
						MonospacedDigit().
						ForegroundStyle(swiftui.RGBA(r, g, b, 1.0)),
				),
			)
		}),
		swiftui.Text(label).
			Font(swiftui.FontCaption2).
			ForegroundStyleNamed("secondary"),
	)
}

// circleGauge shows a float percentage as a circular gauge.
func circleGauge(label string, pct *swiftui.FloatState, icon string, r, g, b float64) swiftui.View {
	return swiftui.VStackSpaced(4,
		swiftui.ZStack(
			swiftui.Circle().
				Stroke(swiftui.RGBA(r, g, b, 0.15), 8).
				Frame(64, 64).
				AsView(),
			swiftui.FloatGauge(label, pct, 0, 1).
				Frame(0, 0), // hidden; drives the state
			swiftui.VStack(
				swiftui.Image(icon).
					ForegroundStyle(swiftui.RGBA(r, g, b, 1.0)).
					ImageScale(swiftui.ImageScaleSmall),
			),
		),
		swiftui.Text(label).
			Font(swiftui.FontCaption2).
			ForegroundStyleNamed("secondary"),
	)
}

// metricRow shows a labeled progress bar with a float value.
func metricRow(label, icon string, state *swiftui.FloatState, unit string, total float64) swiftui.View {
	return swiftui.HStack(
		swiftui.Image(icon).
			ForegroundStyleNamed("secondary").
			ImageScale(swiftui.ImageScaleSmall).
			Frame(16, 0),
		swiftui.Text(label).
			Font(swiftui.FontBody).
			Frame(90, 0),
		swiftui.FloatProgressView(state, total).
			Tint(swiftui.RGBA(0.35, 0.65, 1.0, 1.0)),
	)
}
