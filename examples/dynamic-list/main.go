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
	"fmt"
	"runtime"
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

	swiftui.Run(swiftui.AppConfig{
		Title:  "Todo List",
		Width:  400,
		Height: 500,
	}, swiftui.VStack(
		// Input row
		swiftui.HStackSpaced(8,
			swiftui.TextField("New item...", input, func() {
				addItem(input, count)
			}),
			swiftui.Button("Add", func() {
				addItem(input, count)
			}).ButtonStyle(swiftui.ButtonStyleBorderedProminent),
		).Padding(12),

		swiftui.Divider(),

		// Dynamic list that rebuilds when count changes
		swiftui.ScrollView(
			swiftui.DynamicView(count, func(n int) swiftui.View {
				if n == 0 {
					return swiftui.VStack(
						swiftui.Spacer(),
						swiftui.Image("tray").
							ForegroundStyleNamed("secondary").
							ImageScale(swiftui.ImageScaleLarge),
						swiftui.Text("No items yet").
							ForegroundStyleNamed("secondary"),
						swiftui.Spacer(),
					).Padding(40)
				}
				mu.Lock()
				snapshot := make([]string, len(items))
				copy(snapshot, items)
				mu.Unlock()

				var rows []swiftui.Viewable
				for i, item := range snapshot {
					rows = append(rows,
						swiftui.HStack(
							swiftui.Text(fmt.Sprintf("%d. %s", i+1, item)).
								Font(swiftui.FontBody),
							swiftui.Spacer(),
						).Padding(8),
					)
				}
				return swiftui.VStack(rows...)
			}),
		),

		swiftui.Divider(),

		// Footer
		swiftui.HStack(
			swiftui.TextFrom(count).
				Font(swiftui.FontCaption).
				ForegroundStyleNamed("secondary"),
			swiftui.Text(" items").
				Font(swiftui.FontCaption).
				ForegroundStyleNamed("secondary"),
			swiftui.Spacer(),
			swiftui.Button("Clear All", func() {
				mu.Lock()
				items = nil
				mu.Unlock()
				count.Set(0)
			}).ButtonStyle(swiftui.ButtonStyleBorderless).
				ForegroundStyle(1, 0.3, 0.3, 1),
		).Padding(12),
	))
}

func addItem(input *swiftui.StringState, count *swiftui.IntState) {
	text := input.Get()
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
