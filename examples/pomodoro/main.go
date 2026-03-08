// Command pomodoro demonstrates a Pomodoro technique timer built with SwiftUI
// from Go. A background goroutine drives countdown updates, while DynamicView
// rebuilds the UI reactively as state changes.
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

// Mode constants.
const (
	modeWork       = 0
	modeShortBreak = 1
	modeLongBreak  = 2
)

// Duration in seconds for each mode.
func modeDuration(mode int) int {
	switch mode {
	case modeShortBreak:
		return 5 * 60
	case modeLongBreak:
		return 15 * 60
	default:
		return 25 * 60
	}
}

func modeLabel(mode int) string {
	switch mode {
	case modeShortBreak:
		return "Short Break"
	case modeLongBreak:
		return "Long Break"
	default:
		return "Work"
	}
}

func main() {
	mode := swiftui.NewIntState(modeWork)
	seconds := swiftui.NewIntState(modeDuration(modeWork))
	progress := swiftui.NewFloatState(0)
	running := swiftui.NewIntState(0)
	sessions := swiftui.NewIntState(0) // completed work sessions

	var stop atomic.Bool

	advanceMode := func() {
		stop.Store(true)
		running.Set(0)
		cur := mode.Get()
		done := sessions.Get()
		if cur == modeWork {
			done++
			sessions.Set(done)
			if done%4 == 0 {
				mode.Set(modeLongBreak)
				seconds.Set(modeDuration(modeLongBreak))
			} else {
				mode.Set(modeShortBreak)
				seconds.Set(modeDuration(modeShortBreak))
			}
		} else {
			mode.Set(modeWork)
			seconds.Set(modeDuration(modeWork))
		}
		progress.Set(0)
	}

	startTimer := func() {
		if running.Get() == 1 {
			return
		}
		stop.Store(false)
		running.Set(1)
		total := modeDuration(mode.Get())
		go func() {
			ticker := time.NewTicker(time.Second)
			defer ticker.Stop()
			for range ticker.C {
				if stop.Load() {
					return
				}
				rem := seconds.Get() - 1
				if rem < 0 {
					advanceMode()
					return
				}
				seconds.Set(rem)
				progress.Set(1.0 - float64(rem)/float64(total))
			}
		}()
	}

	pauseTimer := func() {
		stop.Store(true)
		running.Set(0)
	}

	resetTimer := func() {
		stop.Store(true)
		running.Set(0)
		cur := mode.Get()
		seconds.Set(modeDuration(cur))
		progress.SetAnimated(0)
	}

	swiftui.Run(swiftui.AppConfig{
		Title:  "Pomodoro",
		Width:  400,
		Height: 550,
	}, swiftui.VStackSpaced(20,
		swiftui.Spacer(),

		// Mode label
		swiftui.DynamicView(mode, func(m int) swiftui.View {
			return swiftui.Text(modeLabel(m)).
				Font(swiftui.FontTitle2).
				FontWeight(swiftui.WeightSemibold).
				ForegroundStyle(modeColor(m)).
				AsView()
		}),

		// Circular progress ring
		swiftui.DynamicView(mode, func(m int) swiftui.View {
			r, g, b, a := modeColorComponents(m)
			return swiftui.ZStack(
				// Background ring
				swiftui.Circle().
					Stroke(0.85, 0.85, 0.85, 0.3, 8).
					Frame(200, 200).
					AsView(),
				// Foreground gauge
				swiftui.FloatGauge("", progress, 0, 1).
					Tint(r, g, b, a).
					Frame(170, 170),
				// Time remaining
				swiftui.DynamicView(seconds, func(secs int) swiftui.View {
					mins := secs / 60
					s := secs % 60
					return swiftui.Text(fmt.Sprintf("%02d:%02d", mins, s)).
						Font(swiftui.FontSystem(48)).
						FontWeight(swiftui.WeightBold).
						FontDesign(swiftui.DesignRounded).
						MonospacedDigit().
						AsView()
				}),
			)
		}),

		// Controls
		swiftui.DynamicView(running, func(state int) swiftui.View {
			if state == 0 {
				return swiftui.HStackSpaced(12,
					swiftui.Button("Start", func() {
						startTimer()
					}).ControlSize(swiftui.ControlSizeLarge).
						ButtonStyle(swiftui.ButtonStyleBorderedProminent),
					swiftui.Button("Reset", func() {
						resetTimer()
					}).ControlSize(swiftui.ControlSizeLarge).
						ButtonStyle(swiftui.ButtonStyleBordered),
					swiftui.Button("Skip", func() {
						advanceMode()
					}).ControlSize(swiftui.ControlSizeLarge).
						ButtonStyle(swiftui.ButtonStyleBordered),
				)
			}
			return swiftui.HStackSpaced(12,
				swiftui.Button("Pause", func() {
					pauseTimer()
				}).ControlSize(swiftui.ControlSizeLarge).
					ButtonStyle(swiftui.ButtonStyleBorderedProminent).
					Tint(0.9, 0.6, 0.1, 1.0),
				swiftui.Button("Reset", func() {
					resetTimer()
				}).ControlSize(swiftui.ControlSizeLarge).
					ButtonStyle(swiftui.ButtonStyleBordered),
				swiftui.Button("Skip", func() {
					advanceMode()
				}).ControlSize(swiftui.ControlSizeLarge).
					ButtonStyle(swiftui.ButtonStyleBordered),
			)
		}),

		// Session counter
		swiftui.DynamicView(sessions, func(done int) swiftui.View {
			cycle := done % 4
			dots := make([]swiftui.Viewable, 4)
			for i := range 4 {
				if i < cycle {
					dots[i] = swiftui.Circle().
						Fill(0.9, 0.3, 0.3, 1.0).
						Frame(12, 12)
				} else {
					dots[i] = swiftui.Circle().
						Stroke(0.5, 0.5, 0.5, 0.5, 1.5).
						Frame(12, 12)
				}
			}
			return swiftui.VStackSpaced(8,
				swiftui.Text(fmt.Sprintf("Session %d of 4", cycle+1)).
					Font(swiftui.FontCaption).
					ForegroundStyleNamed("secondary").
					AsView(),
				swiftui.HStackSpaced(8, dots...),
			)
		}),

		swiftui.Spacer(),
	).Padding(30))
}

func modeColorComponents(m int) (r, g, b, a float64) {
	switch m {
	case modeShortBreak:
		return 0.2, 0.8, 0.4, 1.0
	case modeLongBreak:
		return 0.3, 0.5, 0.9, 1.0
	default:
		return 0.9, 0.3, 0.3, 1.0
	}
}

func modeColor(m int) (r, g, b, a float64) {
	return modeColorComponents(m)
}
