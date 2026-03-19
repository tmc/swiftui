//go:build darwin
// +build darwin

// Command timer demonstrates Go goroutines driving reactive SwiftUI updates.
//
// A background goroutine ticks every second, updating elapsed time and progress
// state. Start, stop, and reset buttons control the timer. The key insight:
// State.Set() is safe to call from any goroutine, so a time.Ticker goroutine
// can update the UI in real time without synchronization.
//
// Usage:
//
//	go run .
package main

import (
	"fmt"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/tmc/swiftui"
)

func init() { runtime.LockOSThread() }

func main() {
	elapsed := swiftui.NewIntState(0)
	progress := swiftui.NewFloatState(0)
	running := swiftui.NewIntState(0) // 0=stopped, 1=running

	var stop atomic.Bool

	startTicker := func() {
		if !stop.CompareAndSwap(true, false) && running.Get() == 1 {
			return
		}
		running.Set(1)
		go func() {
			ticker := time.NewTicker(time.Second)
			defer ticker.Stop()
			for range ticker.C {
				if stop.Load() {
					return
				}
				secs := elapsed.Get() + 1
				elapsed.Set(secs)
				progress.SetAnimated(float64(secs) / 60.0)
			}
		}()
	}

	stopTicker := func() {
		stop.Store(true)
		running.Set(0)
	}

	resetTimer := func() {
		stop.Store(true)
		running.Set(0)
		elapsed.Set(0)
		progress.SetAnimated(0)
	}

	swiftui.Run(swiftui.AppConfig{
		Title:  "Timer",
		Width:  380,
		Height: 430,
	}, swiftui.VStackSpaced(16,
		swiftui.HStack(
			swiftui.VStackSpaced(4,
				swiftui.HStack(
					swiftui.Text("Timer").
						Font(swiftui.FontTitle3).
						FontWeight(swiftui.WeightBold),
					swiftui.Spacer(),
				),
				swiftui.HStack(
					swiftui.Text("A one-minute reactive timer driven by a Go ticker.").
						Font(swiftui.FontCaption).
						ForegroundStyleNamed("secondary"),
					swiftui.Spacer(),
				),
			).MaxFrame(-1, 0),
			swiftui.DynamicView(running, func(state int) swiftui.View {
				label := "Idle"
				r, g, b := 0.55, 0.58, 0.62
				if state == 1 {
					label = "Running"
					r, g, b = 0.3, 0.8, 0.4
				}
				return swiftui.Label(label, "circle.fill").
					Font(swiftui.FontCaption).
					ForegroundStyle(r, g, b, 1.0).
					AsView()
			}),
		),

		swiftui.VStackSpaced(10,
			swiftui.ZStack(
				swiftui.Circle().
					Fill(0.2, 0.45, 0.9, 0.12).
					Frame(210, 210).
					AsView(),
				swiftui.DynamicView(elapsed, func(secs int) swiftui.View {
					mins := secs / 60
					s := secs % 60
					return swiftui.Text(fmt.Sprintf("%02d:%02d", mins, s)).
						Font(swiftui.FontSystem(64)).
						FontWeight(swiftui.WeightBold).
						FontDesign(swiftui.DesignRounded).
						MonospacedDigit().
						AsView()
				}),
			),
			swiftui.FloatProgressView(progress, 1.0).
				Tint(0.2, 0.6, 1.0, 1.0),
			swiftui.DynamicView(elapsed, func(secs int) swiftui.View {
				if secs == 0 {
					return swiftui.Text("Ready for a fresh run").
						Font(swiftui.FontCaption).
						ForegroundStyleNamed("secondary").
						AsView()
				}
				return swiftui.Text(fmt.Sprintf("%d seconds elapsed", secs)).
					Font(swiftui.FontCaption).
					ForegroundStyleNamed("secondary").
					AsView()
			}),
		).Padding(16).
			Background(0.18, 0.19, 0.22, 0.45).
			CornerRadius(18),

		swiftui.HStackSpaced(12,
			timerStatCard("Target", "60s"),
			swiftui.DynamicView(elapsed, func(secs int) swiftui.View {
				return timerStatCard("Remaining", fmt.Sprintf("%ds", max(0, 60-secs)))
			}),
		),

		swiftui.DynamicView(running, func(state int) swiftui.View {
			if state == 0 {
				return swiftui.HStackSpaced(12,
					swiftui.Button("Start", func() {
						startTicker()
					}).ControlSize(swiftui.ControlSizeLarge).
						ButtonStyle(swiftui.ButtonStyleBorderedProminent),
					swiftui.Button("Reset", func() {
						resetTimer()
					}).ControlSize(swiftui.ControlSizeLarge).
						ButtonStyle(swiftui.ButtonStyleBordered),
				)
			}
			return swiftui.HStackSpaced(12,
				swiftui.Button("Stop", func() {
					stopTicker()
				}).ControlSize(swiftui.ControlSizeLarge).
					ButtonStyle(swiftui.ButtonStyleBorderedProminent).
					Tint(0.9, 0.3, 0.3, 1.0),
				swiftui.Button("Reset", func() {
					resetTimer()
				}).ControlSize(swiftui.ControlSizeLarge).
					ButtonStyle(swiftui.ButtonStyleBordered),
			)
		}),
	).Padding(30))
}

func timerStatCard(label, value string) swiftui.View {
	return swiftui.VStackSpaced(4,
		swiftui.HStack(
			swiftui.Text(label).
				Font(swiftui.FontCaption).
				ForegroundStyleNamed("secondary"),
			swiftui.Spacer(),
		),
		swiftui.HStack(
			swiftui.Text(value).
				Font(swiftui.FontCallout).
				FontWeight(swiftui.WeightSemibold).
				MonospacedDigit(),
			swiftui.Spacer(),
		),
	).Padding(12).
		Background(0.18, 0.19, 0.22, 0.45).
		CornerRadius(14)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
