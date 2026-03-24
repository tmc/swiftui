//go:build darwin
// +build darwin

// Command dashboard demonstrates a multi-tab live dashboard with real-time
// Go runtime metrics, navigation, and interactive controls.
//
// It displays goroutine counts, heap usage, GC cycles, and a bar chart of
// recent heap readings. A background goroutine updates metrics periodically.
//
// Usage:
//
//	go run .
package main

import (
	"fmt"
	"math"
	"runtime"
	"sync"
	"time"

	"github.com/tmc/swiftui"
)

func init() { runtime.LockOSThread() }

const maxHistory = 10

var (
	mu          sync.Mutex
	heapHistory []float64
	autoRefresh = true
	refreshRate = 2.0 // seconds
)

func main() {
	// State for overview tab.
	goroutines := swiftui.NewIntState(runtime.NumGoroutine())
	heapMB := swiftui.NewIntState(0)
	sysMB := swiftui.NewIntState(0)
	numGC := swiftui.NewIntState(0)
	chartVersion := swiftui.NewIntState(0)

	// State for controls tab.
	spawnCount := swiftui.NewIntState(10)
	autoRefreshState := swiftui.NewIntState(1)
	refreshRateState := swiftui.NewFloatState(2.0)
	lastUpdate := swiftui.NewIntState(0)

	// Read initial stats.
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	heap := float64(ms.HeapAlloc) / (1 << 20)
	heapMB.Set(int(heap))
	sysMB.Set(int(ms.Sys / (1 << 20)))
	numGC.Set(int(ms.NumGC))
	mu.Lock()
	heapHistory = append(heapHistory, heap)
	mu.Unlock()
	lastUpdate.Set(int(time.Now().Unix()))

	// Background data goroutine.
	go func() {
		for {
			mu.Lock()
			rate := refreshRate
			running := autoRefresh
			mu.Unlock()

			if running {
				var m runtime.MemStats
				runtime.ReadMemStats(&m)
				h := float64(m.HeapAlloc) / (1 << 20)

				goroutines.Set(runtime.NumGoroutine())
				heapMB.Set(int(h))
				sysMB.Set(int(m.Sys / (1 << 20)))
				numGC.Set(int(m.NumGC))
				lastUpdate.Set(int(time.Now().Unix()))

				mu.Lock()
				heapHistory = append(heapHistory, h)
				if len(heapHistory) > maxHistory {
					heapHistory = heapHistory[len(heapHistory)-maxHistory:]
				}
				mu.Unlock()

				chartVersion.Set(chartVersion.Get() + 1)
			}

			time.Sleep(time.Duration(rate*1000) * time.Millisecond)
		}
	}()

	swiftui.Run(swiftui.AppConfig{
		Title:  "Dashboard",
		Width:  750,
		Height: 650,
	}, swiftui.TabView(
		overviewTab(goroutines, heapMB, sysMB, numGC, chartVersion, autoRefreshState, lastUpdate),
		detailsTab(goroutines, heapMB, sysMB, numGC, chartVersion, lastUpdate),
		controlsTab(spawnCount, autoRefreshState, refreshRateState, goroutines, heapMB, sysMB, numGC, chartVersion),
	))
}

func overviewTab(goroutines, heapMB, sysMB, numGC, chartVersion, autoRefreshState, lastUpdate *swiftui.IntState) swiftui.View {
	return swiftui.ScrollView(
		swiftui.VStackSpaced(16,
			// Header.
			swiftui.HStack(
				swiftui.Text("Dashboard").
					Font(swiftui.FontTitle).
					FontWeight(swiftui.WeightBold),
				swiftui.Spacer(),
				swiftui.DynamicView(autoRefreshState, func(v int) swiftui.View {
					if v != 0 {
						return swiftui.HStack(
							swiftui.ProgressSpinning().ControlSize(swiftui.ControlSizeSmall),
							swiftui.DynamicView(lastUpdate, func(ts int) swiftui.View {
								t := time.Unix(int64(ts), 0)
								return swiftui.Text(t.Format("15:04:05")).
									Font(swiftui.FontCaption).
									ForegroundStyleNamed("secondary").
									MonospacedDigit().AsView()
							}),
						)
					}
					return swiftui.Text("Paused").
						Font(swiftui.FontCaption).
						ForegroundStyleNamed("secondary").AsView()
				}),
			),

			// Metric cards.
			swiftui.HStackSpaced(12,
				metricCard("bolt.fill", "Goroutines", goroutines, 0.35, 0.65, 1.0),
				metricCard("memorychip.fill", "Heap MB", heapMB, 0.9, 0.5, 0.2),
				metricCard("cpu", "Sys MB", sysMB, 0.3, 0.7, 1.0),
				metricCard("arrow.triangle.2.circlepath", "GC Cycles", numGC, 0.3, 0.8, 0.4),
			),

			swiftui.GroupBox("Heap History (last 10 readings)",
				swiftui.DynamicView(chartVersion, func(_ int) swiftui.View {
					return heapChart(125)
				}),
			).MaxFrame(-1, 0),

			swiftui.HStackSpaced(12,
				swiftui.GroupBox("Runtime Signals",
					swiftui.DynamicView(chartVersion, func(_ int) swiftui.View {
						return runtimeSignalsPanel(
							goroutines.Get(),
							heapMB.Get(),
							sysMB.Get(),
							numGC.Get(),
						)
					}),
				).MaxFrame(-1, 0),

				swiftui.GroupBox("Session",
					swiftui.DynamicView(chartVersion, func(_ int) swiftui.View {
						return sessionPanel(autoRefreshState.Get() != 0, lastUpdate.Get())
					}),
				).MaxFrame(-1, 0),
			),
		).Padding(24),
	).TabItem("Overview", "gauge.with.dots.needle.33percent")
}

func metricCard(icon string, label string, state *swiftui.IntState, r, g, b float64) swiftui.View {
	return swiftui.VStackSpaced(6,
		swiftui.HStack(
			swiftui.Image(icon).
				ForegroundStyle(r, g, b, 1.0).
				ImageScale(swiftui.ImageScaleSmall),
			swiftui.Spacer(),
		),
		swiftui.HStack(
			swiftui.TextFrom(state).
				Font(swiftui.FontTitle2).
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
	).Padding(12).
		BackgroundStyle("regularMaterial").
		CornerRadius(10)
}

func runtimeSignalsPanel(goroutines, heapMB, sysMB, numGC int) swiftui.View {
	heapRatio := 0.0
	if sysMB > 0 {
		heapRatio = float64(heapMB) / float64(sysMB)
	}
	targetGoroutines := runtime.NumCPU() * 4
	if targetGoroutines < 8 {
		targetGoroutines = 8
	}
	goroutineRatio := clamp(float64(goroutines) / float64(targetGoroutines))
	gcRatio := clamp(float64(numGC) / 12.0)

	return swiftui.VStackSpaced(10,
		signalRow(
			"Heap Pressure",
			"memorychip.fill",
			fmt.Sprintf("%d%% of sys", int(heapRatio*100+0.5)),
			heapRatio,
			0.9, 0.5, 0.2,
		),
		signalRow(
			"Scheduler Load",
			"bolt.fill",
			fmt.Sprintf("%d goroutines", goroutines),
			goroutineRatio,
			0.35, 0.65, 1.0,
		),
		signalRow(
			"GC Activity",
			"arrow.triangle.2.circlepath",
			fmt.Sprintf("%d cycles", numGC),
			gcRatio,
			0.3, 0.8, 0.4,
		),
	).Padding(8)
}

func signalRow(label, icon, value string, ratio float64, r, g, b float64) swiftui.View {
	return swiftui.VStackSpaced(6,
		swiftui.HStack(
			swiftui.Label(label, icon).
				Font(swiftui.FontCallout),
			swiftui.Spacer(),
			swiftui.Text(value).
				Font(swiftui.FontCaption).
				FontWeight(swiftui.WeightSemibold).
				MonospacedDigit().
				ForegroundStyleNamed("secondary"),
		),
		swiftui.ProgressLinear(clamp(ratio), 1.0).
			Tint(r, g, b, 1.0),
	)
}

func sessionPanel(autoRefresh bool, lastUpdate int) swiftui.View {
	modeLabel := "Paused"
	modeIcon := "pause.circle.fill"
	modeColor := [3]float64{0.9, 0.5, 0.2}
	if autoRefresh {
		modeLabel = "Automatic"
		modeIcon = "play.circle.fill"
		modeColor = [3]float64{0.3, 0.8, 0.4}
	}

	lastSample := "Waiting for sample"
	if lastUpdate > 0 {
		lastSample = time.Unix(int64(lastUpdate), 0).Format("15:04:05")
	}

	trend, tr, tg, tb := heapTrend()

	return swiftui.VStackSpaced(8,
		noteRow("clock.arrow.circlepath", "Cadence", fmt.Sprintf("Every %.1fs", refreshRate), 0.35, 0.65, 1.0),
		noteRow(modeIcon, "Mode", modeLabel, modeColor[0], modeColor[1], modeColor[2]),
		noteRow("clock", "Last Sample", lastSample, 0.6, 0.6, 0.75),
		noteRow("chart.line.uptrend.xyaxis", "Trend", trend, tr, tg, tb),
	).Padding(8)
}

func noteRow(icon, label, value string, r, g, b float64) swiftui.View {
	return swiftui.HStack(
		swiftui.Image(icon).
			ForegroundStyle(r, g, b, 1.0).
			ImageScale(swiftui.ImageScaleSmall).
			Frame(16, 0),
		swiftui.Text(label).
			Font(swiftui.FontCallout).
			ForegroundStyleNamed("secondary"),
		swiftui.Spacer(),
		swiftui.Text(value).
			Font(swiftui.FontCaption).
			FontWeight(swiftui.WeightMedium).
			MonospacedDigit(),
	)
}

func heapTrend() (string, float64, float64, float64) {
	mu.Lock()
	history := make([]float64, len(heapHistory))
	copy(history, heapHistory)
	mu.Unlock()

	if len(history) < 2 {
		return "Collecting window", 0.6, 0.6, 0.75
	}

	delta := history[len(history)-1] - history[0]
	if math.Abs(delta) < 0.1 {
		return "Stable window", 0.35, 0.65, 1.0
	}
	if delta > 0 {
		return fmt.Sprintf("+%.1f MB in window", delta), 0.9, 0.5, 0.2
	}
	return fmt.Sprintf("%.1f MB in window", delta), 0.3, 0.8, 0.4
}

func clamp(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func heapChart(maxHeight float64) swiftui.View {
	mu.Lock()
	history := make([]float64, len(heapHistory))
	copy(history, heapHistory)
	mu.Unlock()

	if len(history) == 0 {
		return swiftui.Text("No data yet").
			ForegroundStyleNamed("secondary").
			Padding(20).AsView()
	}

	maxVal := 0.0
	for _, v := range history {
		if v > maxVal {
			maxVal = v
		}
	}
	if maxVal == 0 {
		maxVal = 1
	}

	var columns []swiftui.Viewable
	for i, v := range history {
		ratio := v / maxVal
		h := math.Max(ratio*maxHeight, 4)
		columns = append(columns, swiftui.VStack(
			swiftui.Spacer(),
			swiftui.Text(fmt.Sprintf("%.1f", v)).
				Font(swiftui.FontCaption2).
				ForegroundStyleNamed("secondary").
				MonospacedDigit(),
			swiftui.RoundedRectangle(4).
				Fill(0.35, 0.65, 1.0, 0.85).
				Frame(30, h).
				AsView(),
			swiftui.Text(fmt.Sprintf("%d", i+1)).
				Font(swiftui.FontCaption2).
				ForegroundStyleNamed("tertiary"),
		).MaxFrame(-1, 0))
	}
	return swiftui.HStackSpaced(4, columns...).
		MaxFrame(-1, 0).
		Padding(12)
}

func detailsTab(goroutines, heapMB, sysMB, numGC, chartVersion, lastUpdate *swiftui.IntState) swiftui.View {
	return swiftui.ScrollView(
		swiftui.VStackSpaced(16,
			swiftui.VStackSpaced(4,
				swiftui.HStack(
					swiftui.Text("Runtime Details").
						Font(swiftui.FontTitle2).
						FontWeight(swiftui.WeightBold),
					swiftui.Spacer(),
					swiftui.DynamicView(lastUpdate, func(ts int) swiftui.View {
						t := time.Unix(int64(ts), 0)
						return swiftui.Text("Updated " + t.Format("15:04:05")).
							Font(swiftui.FontCaption).
							ForegroundStyleNamed("secondary").
							MonospacedDigit().
							AsView()
					}),
				),
				swiftui.HStack(
					swiftui.Text("A compact view of runtime health, memory layout, and GC behavior.").
						Font(swiftui.FontCallout).
						ForegroundStyleNamed("secondary"),
					swiftui.Spacer(),
				),
			),

			swiftui.HStackSpaced(12,
				swiftui.GroupBox("Runtime Snapshot",
					swiftui.DynamicView(chartVersion, func(_ int) swiftui.View {
						return swiftui.VStackSpaced(10,
							infoRow("Version", runtime.Version()),
							infoRow("OS / Arch", runtime.GOOS+"/"+runtime.GOARCH),
							infoRow("CPUs", fmt.Sprintf("%d", runtime.NumCPU())),
							infoRow("Goroutines", fmt.Sprintf("%d", goroutines.Get())),
							infoRow("Refresh", fmt.Sprintf("%.1fs", refreshRate)),
						).Padding(10)
					}),
				).MaxFrame(-1, 0),
				swiftui.GroupBox("Pressure Signals",
					swiftui.DynamicView(chartVersion, func(_ int) swiftui.View {
						return runtimeSignalsPanel(
							goroutines.Get(),
							heapMB.Get(),
							sysMB.Get(),
							numGC.Get(),
						)
					}),
				).MaxFrame(-1, 0),
			),

			swiftui.HStackSpaced(12,
				swiftui.GroupBox("Memory Layout",
					swiftui.DynamicView(chartVersion, func(_ int) swiftui.View {
						var ms runtime.MemStats
						runtime.ReadMemStats(&ms)
						return swiftui.VStackSpaced(10,
							infoRow("Heap Alloc", fmtBytes(ms.HeapAlloc)),
							infoRow("Heap Sys", fmtBytes(ms.HeapSys)),
							infoRow("Heap Idle", fmtBytes(ms.HeapIdle)),
							infoRow("Heap In Use", fmtBytes(ms.HeapInuse)),
							infoRow("Stack In Use", fmtBytes(ms.StackInuse)),
							infoRow("Stack Sys", fmtBytes(ms.StackSys)),
						).Padding(10)
					}),
				).MaxFrame(-1, 0),
				swiftui.GroupBox("Garbage Collection",
					swiftui.DynamicView(chartVersion, func(_ int) swiftui.View {
						var ms runtime.MemStats
						runtime.ReadMemStats(&ms)
						lastGC := "N/A"
						if ms.LastGC > 0 {
							lastGC = time.Unix(0, int64(ms.LastGC)).Format("15:04:05")
						}
						trend, _, _, _ := heapTrend()
						return swiftui.VStackSpaced(10,
							infoRow("Cycles", fmt.Sprintf("%d", ms.NumGC)),
							infoRow("Pause Total", fmt.Sprintf("%.2f ms", float64(ms.PauseTotalNs)/1e6)),
							infoRow("Last GC", lastGC),
							infoRow("Heap Trend", trend),
							infoRow("Next GC Goal", fmtBytes(ms.NextGC)),
						).Padding(10)
					}),
				).MaxFrame(-1, 0),
			),

			swiftui.GroupBox("Environment",
				swiftui.VStackSpaced(10,
					infoRow("Toolchain", runtime.Version()),
					infoRow("Process Model", "Go runtime bridged into SwiftUI"),
					infoRow("Update Loop", "Background sampler + goroutine-safe state"),
					infoRow("Primary Signals", "Heap, scheduler load, system bytes, GC"),
				).Padding(10),
			).MaxFrame(-1, 0),
		).Padding(24),
	).TabItem("Details", "list.bullet.rectangle")
}

func infoRow(label, value string) swiftui.View {
	return swiftui.HStack(
		swiftui.Text(label).
			ForegroundStyleNamed("secondary"),
		swiftui.Spacer(),
		swiftui.Text(value).
			FontWeight(swiftui.WeightMedium).
			MonospacedDigit(),
	)
}

func fmtBytes(b uint64) string {
	mb := float64(b) / (1 << 20)
	return fmt.Sprintf("%.2f MB", mb)
}

func controlsTab(spawnCount *swiftui.IntState, autoRefreshState *swiftui.IntState, refreshRateState *swiftui.FloatState, goroutines, heapMB, sysMB, numGC, chartVersion *swiftui.IntState) swiftui.View {
	return swiftui.ScrollView(
		swiftui.VStackSpaced(16,
			swiftui.VStackSpaced(4,
				swiftui.HStack(
					swiftui.Text("Controls").
						Font(swiftui.FontTitle2).
						FontWeight(swiftui.WeightBold),
					swiftui.Spacer(),
				),
				swiftui.HStack(
					swiftui.Text("Drive load, tune the refresh loop, and watch the runtime respond in the other tabs.").
						Font(swiftui.FontCallout).
						ForegroundStyleNamed("secondary"),
					swiftui.Spacer(),
				),
			),

			swiftui.HStackSpaced(12,
				swiftui.DynamicView(chartVersion, func(_ int) swiftui.View {
					return metricCard("bolt.fill", "Goroutines", goroutines, 0.35, 0.65, 1.0)
				}),
				swiftui.DynamicView(chartVersion, func(_ int) swiftui.View {
					return metricCard("memorychip.fill", "Heap MB", heapMB, 0.9, 0.5, 0.2)
				}),
				swiftui.DynamicView(chartVersion, func(_ int) swiftui.View {
					return metricCard("cpu", "Sys MB", sysMB, 0.3, 0.7, 1.0)
				}),
				swiftui.DynamicView(chartVersion, func(_ int) swiftui.View {
					return metricCard("arrow.triangle.2.circlepath", "GC Cycles", numGC, 0.3, 0.8, 0.4)
				}),
			),

			swiftui.HStackSpaced(12,
				swiftui.GroupBox("Load Injection",
					swiftui.VStackSpaced(12,
						swiftui.Text("Create visible movement in the sampler by forcing collection, allocating heap, or adding temporary workers.").
							Font(swiftui.FontCaption).
							ForegroundStyleNamed("secondary"),
						swiftui.Button("Force GC", func() {
							runtime.GC()
						}).
							ButtonStyle(swiftui.ButtonStyleBorderedProminent).
							ControlSize(swiftui.ControlSizeLarge),
						swiftui.Button("Allocate 10 MB", func() {
							_ = make([]byte, 10<<20)
						}).
							ButtonStyle(swiftui.ButtonStyleBordered),
						swiftui.Stepper("Spawn Count", spawnCount, 1, 100, func() {}),
						swiftui.Button("Spawn Goroutines", func() {
							n := spawnCount.Get()
							for i := 0; i < n; i++ {
								go func() {
									time.Sleep(30 * time.Second)
								}()
							}
						}).
							ButtonStyle(swiftui.ButtonStyleBordered),
					).Padding(10),
				).MaxFrame(-1, 0),

				swiftui.GroupBox("Refresh Loop",
					swiftui.VStackSpaced(12,
						swiftui.Toggle("Auto-Refresh", autoRefreshState, func() {
							mu.Lock()
							autoRefresh = autoRefreshState.Get() != 0
							mu.Unlock()
						}),
						swiftui.FloatSlider("Refresh Rate (s)", refreshRateState, 0.5, 5.0, func() {
							mu.Lock()
							refreshRate = refreshRateState.Get()
							mu.Unlock()
						}),
						swiftui.DynamicView(autoRefreshState, func(v int) swiftui.View {
							mode := "Paused"
							if v != 0 {
								mode = "Running"
							}
							return swiftui.VStackSpaced(10,
								infoRow("Mode", mode),
								infoRow("Cadence", fmt.Sprintf("%.1fs", refreshRateState.Get())),
								infoRow("Recommended Flow", "Force load, then inspect Overview"),
							).Padding(10)
						}),
					).Padding(10),
				).MaxFrame(-1, 0),
			),

			swiftui.GroupBox("Suggested Checks",
				swiftui.VStackSpaced(10,
					infoRow("1", "Force GC and confirm cycles rise without a heap spike."),
					infoRow("2", "Allocate memory and watch Heap Pressure grow in Overview."),
					infoRow("3", "Spawn workers and confirm Scheduler Load reacts immediately."),
				).Padding(10),
			).MaxFrame(-1, 0),
		).Padding(24),
	).TabItem("Controls", "slider.horizontal.3")
}
