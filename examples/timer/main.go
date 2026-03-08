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
		Width:  340,
		Height: 400,
	}, swiftui.VStackSpaced(16,
		swiftui.Spacer(),

		// Elapsed time display driven by DynamicView
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

		// Progress toward 60 seconds
		swiftui.FloatProgressView(progress, 1.0).
			Tint(0.2, 0.6, 1.0, 1.0),

		// Status label
		swiftui.DynamicView(elapsed, func(secs int) swiftui.View {
			if secs == 0 {
				return swiftui.Text("Ready").
					Font(swiftui.FontCaption).
					ForegroundStyleNamed("secondary").
					AsView()
			}
			return swiftui.Text(fmt.Sprintf("%d seconds elapsed", secs)).
				Font(swiftui.FontCaption).
				ForegroundStyleNamed("secondary").
				AsView()
		}),

		swiftui.Spacer(),

		// Control buttons
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

		swiftui.Spacer(),
	).Padding(30))
}
