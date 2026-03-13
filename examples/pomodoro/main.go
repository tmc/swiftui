//go:build darwin
// +build darwin

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
	var autoStartToken atomic.Uint64
	var startTimer func()
	cancelAutoStart := func() {
		autoStartToken.Add(1)
	}
	updateMenuLabel := func(secs int) {
		swiftui.UpdateMenuBarLabelStyled(
			fmt.Sprintf("%02d:%02d", secs/60, secs%60),
			swiftui.MenuBarLabelStyle{
				MonospacedDigits: true,
				Animate:          true,
			},
		)
	}

	// advanceMode transitions to the next stage with an optional sound and
	// auto-start. When autoStart is true the next timer begins immediately.
	advanceMode := func(autoStart bool) {
		cancelAutoStart()
		stop.Store(true)
		running.Set(0)
		cur := mode.Get()
		done := sessions.Get()
		if cur == modeWork {
			done++
			sessions.Set(done)
			swiftui.PlaySystemSound("Glass") // subtle chime for work→break
			if done%4 == 0 {
				mode.Set(modeLongBreak)
				seconds.Set(modeDuration(modeLongBreak))
			} else {
				mode.Set(modeShortBreak)
				seconds.Set(modeDuration(modeShortBreak))
			}
		} else {
			swiftui.PlaySystemSound("Blow") // deeper tone for break→work
			mode.Set(modeWork)
			seconds.Set(modeDuration(modeWork))
		}
		progress.Set(0)
		newMode := mode.Get()
		dur := modeDuration(newMode)
		updateMenuLabel(dur)
		if autoStart {
			// Small delay so the user sees the transition before it restarts.
			token := autoStartToken.Add(1)
			time.AfterFunc(500*time.Millisecond, func() {
				if autoStartToken.Load() != token {
					return
				}
				startTimer()
			})
		}
	}

	startTimer = func() {
		cancelAutoStart()
		if running.Get() == 1 {
			return
		}
		swiftui.PlaySystemSound("Tink")
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
					advanceMode(true)
					return
				}
				seconds.Set(rem)
				progress.Set(1.0 - float64(rem)/float64(total))
				updateMenuLabel(rem)
			}
		}()
	}

	pauseTimer := func() {
		cancelAutoStart()
		stop.Store(true)
		running.Set(0)
	}

	resetTimer := func() {
		cancelAutoStart()
		stop.Store(true)
		running.Set(0)
		cur := mode.Get()
		dur := modeDuration(cur)
		seconds.Set(dur)
		progress.SetAnimated(0)
		updateMenuLabel(dur)
	}

	swiftui.RunMenuBar(swiftui.MenuBarConfig{
		Label:        "25:00",
		SystemImage:  "timer",
		Width:        280,
		Height:       340,
		OpenOnLaunch: true,
	}, swiftui.VStackSpaced(16,
		// Mode label
		swiftui.DynamicView(mode, func(m int) swiftui.View {
			return swiftui.Text(modeLabel(m)).
				Font(swiftui.FontTitle3).
				FontWeight(swiftui.WeightSemibold).
				ForegroundStyle(modeColor(m)).
				AsView()
		}),

		// Timer ring with countdown
		swiftui.ZStack(
			// Track ring
			swiftui.Circle().
				Stroke(0.5, 0.5, 0.5, 0.3, 6).
				Frame(160, 160).
				AsView(),
			// Time remaining
			swiftui.DynamicView(seconds, func(secs int) swiftui.View {
				mins := secs / 60
				s := secs % 60
				return swiftui.Text(fmt.Sprintf("%02d:%02d", mins, s)).
					Font(swiftui.FontSystem(36)).
					FontWeight(swiftui.WeightBold).
					FontDesign(swiftui.DesignRounded).
					MonospacedDigit().
					AsView()
			}),
		),

		// Controls
		swiftui.DynamicView(running, func(state int) swiftui.View {
			if state == 0 {
				return swiftui.HStackSpaced(10,
					swiftui.Button("Start", func() {
						startTimer()
					}).ButtonStyle(swiftui.ButtonStyleBorderedProminent),
					swiftui.Button("Reset", func() {
						resetTimer()
					}).ButtonStyle(swiftui.ButtonStyleBordered),
					swiftui.Button("Skip", func() {
						advanceMode(false)
					}).ButtonStyle(swiftui.ButtonStyleBordered),
				)
			}
			return swiftui.HStackSpaced(10,
				swiftui.Button("Pause", func() {
					pauseTimer()
				}).ButtonStyle(swiftui.ButtonStyleBorderedProminent).
					Tint(0.9, 0.6, 0.1, 1.0),
				swiftui.Button("Reset", func() {
					resetTimer()
				}).ButtonStyle(swiftui.ButtonStyleBordered),
				swiftui.Button("Skip", func() {
					advanceMode(false)
				}).ButtonStyle(swiftui.ButtonStyleBordered),
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
						Frame(10, 10)
				} else {
					dots[i] = swiftui.Circle().
						Stroke(0.5, 0.5, 0.5, 0.5, 1.5).
						Frame(10, 10)
				}
			}
			return swiftui.VStackSpaced(6,
				swiftui.Text(fmt.Sprintf("Session %d of 4", cycle+1)).
					Font(swiftui.FontCaption).
					ForegroundStyleNamed("secondary").
					AsView(),
				swiftui.HStackSpaced(6, dots...),
			)
		}),
	).Padding(20))
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
