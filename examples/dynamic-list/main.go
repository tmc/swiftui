//go:build darwin
// +build darwin

// Command dynamic-list demonstrates reactive list updates with SwiftUI from Go.
//
// It maintains a list of items that can be added via a text field and cleared
// with a button. The list rebuilds automatically when the item count changes.
//
// Usage:
//
//	go run .
package main

import (
	"log"
	"runtime"
	"strings"
	"sync"

	"github.com/tmc/swiftui"
)

func init() { runtime.LockOSThread() }

var (
	mu    sync.Mutex
	items []string
)

func main() {
	items = []string{"Buy groceries", "Write tests", "Review PR"}
	input := swiftui.NewStringState("")
	count := swiftui.NewIntState(len(items))
	if err := swiftui.Run(swiftui.WithWindow(swiftui.AppConfig{
		Title:  "Todo List",
		Width:  500,
		Height: 600,
	}, swiftui.VStackSpaced(12,
		swiftui.VStackSpaced(4,
			swiftui.HStack(
				swiftui.Text("Todo List").
					Font(swiftui.FontTitle2).
					FontWeight(swiftui.WeightBold),
				swiftui.Spacer(),
				swiftui.Label("Synced", "checkmark.circle.fill").
					Font(swiftui.FontCaption).
					ForegroundStyle(swiftui.RGBA(0.3, 0.8, 0.4, 1.0)),
			),
			swiftui.HStack(
				swiftui.Text("A simple reactive list with inline add and clear actions.").
					Font(swiftui.FontCallout).
					ForegroundStyleNamed("secondary"),
				swiftui.Spacer(),
			),
		),
		swiftui.HStackSpaced(8,
			swiftui.TextField("New item...", input, func() {
				addItem(input, count)
			}).TextFieldStyle(swiftui.TextFieldStyleRoundedBorder),
			swiftui.Button("Add", func() {
				addItem(input, count)
			}).ButtonStyle(swiftui.ButtonStyleBorderedProminent),
		),
		swiftui.DynamicView(count, func(n int) swiftui.View {
			return summaryBar(n)
		}),
		swiftui.GroupBox("Items",
			swiftui.ScrollView(
				swiftui.DynamicView(count, func(n int) swiftui.View {
					if n == 0 {
						return swiftui.VStackSpaced(8,
							swiftui.Spacer(),
							swiftui.Image("tray").
								ForegroundStyleNamed("secondary").
								ImageScale(swiftui.ImageScaleLarge),
							swiftui.Text("Nothing queued").
								Font(swiftui.FontBody).
								ForegroundStyleNamed("secondary"),
							swiftui.Text("Add a task above to repopulate the list.").
								Font(swiftui.FontCaption).
								ForegroundStyleNamed("tertiary"),
							swiftui.Spacer(),
						).Padding(36)
					}
					mu.Lock()
					snapshot := make([]string, len(items))
					copy(snapshot, items)
					mu.Unlock()

					rows := make([]swiftui.Viewable, 0, len(snapshot))
					for i, item := range snapshot {
						rows = append(rows, todoRow(i+1, item))
					}
					return swiftui.VStackSpaced(8, rows...).Padding(8)
				}),
			).Frame(440, 320),
		).MaxFrame(-1, 0),
		swiftui.HStack(
			swiftui.Text("Keep the list small and explicit. This example is about state, not data modeling.").
				Font(swiftui.FontCaption).
				ForegroundStyleNamed("secondary"),
			swiftui.Spacer(),
			swiftui.Button("Clear All", func() {
				mu.Lock()
				items = nil
				mu.Unlock()
				count.Set(0)
			}).
				ButtonStyle(swiftui.ButtonStyleBordered).
				ForegroundStyle(swiftui.RGBA(1, 0.35, 0.35, 1)),
		),
	).Padding(20))); err != nil {
		log.Fatal(err)
	}
}

func summaryBar(n int) swiftui.View {
	label := "Open items"
	if n == 1 {
		label = "Open item"
	}
	return swiftui.HStackSpaced(12,
		listStatCard("Count", pluralCount(n, label)),
		listStatCard("Status", statusLabel(n)),
	)
}

func todoRow(index int, item string) swiftui.View {
	return swiftui.HStackSpaced(10,
		swiftui.ZStack(
			swiftui.Circle().
				Fill(swiftui.RGBA(0.3, 0.6, 1.0, 0.16)).
				Frame(24, 24).
				AsView(),
			swiftui.Text(string(rune('0'+index))).
				Font(swiftui.FontCaption).
				FontWeight(swiftui.WeightBold),
		),
		swiftui.Text(item).
			Font(swiftui.FontBody),
		swiftui.Spacer(),
	).Padding(10).
		Background(swiftui.RGBA(0.18, 0.19, 0.22, 0.55)).
		CornerRadius(10)
}

func listStatCard(label, value string) swiftui.View {
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
				FontWeight(swiftui.WeightSemibold),
			swiftui.Spacer(),
		),
	).Padding(12).
		Background(swiftui.RGBA(0.18, 0.19, 0.22, 0.55)).
		CornerRadius(10)
}

func pluralCount(n int, label string) string {
	return strings.TrimSpace(strings.Join([]string{itoa(n), label}, " "))
}

func statusLabel(n int) string {
	if n == 0 {
		return "Cleared"
	}
	if n < 4 {
		return "On track"
	}
	return "Busy"
}

func addItem(input *swiftui.StringState, count *swiftui.IntState) {
	text := strings.TrimSpace(input.Get())
	if text == "" {
		return
	}
	mu.Lock()
	items = append(items, text)
	n := len(items)
	mu.Unlock()
	input.Set("")
	count.Set(n)
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	return string(buf[i:])
}
