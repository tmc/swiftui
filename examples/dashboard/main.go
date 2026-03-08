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
		Title:  "Live Dashboard",
		Width:  750,
		Height: 600,
	}, swiftui.TabView(
		overviewTab(goroutines, heapMB, sysMB, numGC, chartVersion, autoRefreshState, lastUpdate),
		detailsTab(),
		controlsTab(spawnCount, autoRefreshState, refreshRateState),
	))
}

func overviewTab(goroutines, heapMB, sysMB, numGC, chartVersion, autoRefreshState, lastUpdate *swiftui.IntState) swiftui.View {
	return swiftui.ScrollView(
		swiftui.VStackSpaced(16,
			// Header.
			swiftui.HStack(
				swiftui.Text("Live Dashboard").
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

			// Heap history bar chart.
			swiftui.GroupBox("Heap History (last 10 readings)",
				swiftui.DynamicView(chartVersion, func(_ int) swiftui.View {
					return heapChart(200)
				}),
			).MaxFrame(-1, 0),
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
		Background(0.2, 0.2, 0.25, 0.5).
		CornerRadius(10)
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

func detailsTab() swiftui.View {
	return swiftui.NavigationStack(
		swiftui.List(
			swiftui.NavigationLink("Runtime Info", runtimeInfoPage()),
			swiftui.NavigationLink("Memory Details", memoryDetailsPage()),
			swiftui.NavigationLink("GC Stats", gcStatsPage()),
		).NavigationTitle("Details"),
	).TabItem("Details", "list.bullet.rectangle")
}

func runtimeInfoPage() swiftui.View {
	return swiftui.Form(
		swiftui.Section("Go Runtime",
			swiftui.VStack(
				infoRow("Version", runtime.Version()),
				infoRow("OS", runtime.GOOS),
				infoRow("Arch", runtime.GOARCH),
				infoRow("CPUs", fmt.Sprintf("%d", runtime.NumCPU())),
				infoRow("Goroutines", fmt.Sprintf("%d", runtime.NumGoroutine())),
			),
		),
	).NavigationTitle("Runtime Info")
}

func memoryDetailsPage() swiftui.View {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	return swiftui.Form(
		swiftui.Section("Heap",
			swiftui.VStack(
				infoRow("HeapAlloc", fmtBytes(ms.HeapAlloc)),
				infoRow("HeapSys", fmtBytes(ms.HeapSys)),
				infoRow("HeapIdle", fmtBytes(ms.HeapIdle)),
				infoRow("HeapInuse", fmtBytes(ms.HeapInuse)),
			),
		),
		swiftui.Section("Stack",
			swiftui.VStack(
				infoRow("StackInuse", fmtBytes(ms.StackInuse)),
				infoRow("StackSys", fmtBytes(ms.StackSys)),
			),
		),
	).NavigationTitle("Memory Details")
}

func gcStatsPage() swiftui.View {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	lastGC := "N/A"
	if ms.LastGC > 0 {
		lastGC = time.Unix(0, int64(ms.LastGC)).Format("15:04:05")
	}
	return swiftui.Form(
		swiftui.Section("Garbage Collection",
			swiftui.VStack(
				infoRow("NumGC", fmt.Sprintf("%d", ms.NumGC)),
				infoRow("PauseTotal", fmt.Sprintf("%.2f ms", float64(ms.PauseTotalNs)/1e6)),
				infoRow("LastGC", lastGC),
			),
		),
	).NavigationTitle("GC Stats")
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

func controlsTab(spawnCount *swiftui.IntState, autoRefreshState *swiftui.IntState, refreshRateState *swiftui.FloatState) swiftui.View {
	return swiftui.Form(
		swiftui.Section("Actions",
			swiftui.VStack(
				swiftui.Button("Force GC", func() {
					runtime.GC()
				}).ButtonStyle(swiftui.ButtonStyleBorderedProminent),

				swiftui.Button("Allocate Memory", func() {
					// Allocate ~10MB to cause a heap spike.
					_ = make([]byte, 10<<20)
				}).ButtonStyle(swiftui.ButtonStyleBordered),
			),
		),
		swiftui.Section("Goroutines",
			swiftui.VStack(
				swiftui.Stepper("Spawn Count", spawnCount, 1, 100, func() {}),
				swiftui.Button("Spawn Goroutines", func() {
					n := spawnCount.Get()
					for i := 0; i < n; i++ {
						go func() {
							time.Sleep(30 * time.Second)
						}()
					}
				}).ButtonStyle(swiftui.ButtonStyleBordered),
			),
		),
		swiftui.Section("Refresh",
			swiftui.VStack(
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
			),
		),
	).TabItem("Controls", "slider.horizontal.3")
}
