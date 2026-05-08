//go:build darwin

// Command menu-bar demonstrates a menu bar app with commands and lifecycle hooks.
//
// It runs a status-item-only app (no dock icon) that shows a clipboard history
// popover. The app menu bar includes custom commands with keyboard shortcuts,
// dynamic enable/disable, and lifecycle logging.
//
// Usage:
//
//	go run .
package main

import (
	"fmt"
	"log"
	"runtime"
	"sync"
	"time"

	"github.com/tmc/swiftui"
)

func init() { runtime.LockOSThread() }

const maxHistory = 10

func main() {
	var mu sync.Mutex
	historyCount := swiftui.NewIntState(0)
	statusText := swiftui.NewStringState("Ready")

	// Simulated clipboard history entries.
	var history []string
	addEntry := func(text string) {
		mu.Lock()
		history = append([]string{text}, history...)
		if len(history) > maxHistory {
			history = history[:maxHistory]
		}
		n := len(history)
		mu.Unlock()
		historyCount.Set(n)
		statusText.Set(fmt.Sprintf("Copied: %s", text))
	}

	// Seed with a few entries.
	addEntry("https://github.com/tmc/swiftui")
	addEntry("go build ./...")
	addEntry("Hello, world!")

	clearAll := func() {
		mu.Lock()
		history = history[:0]
		mu.Unlock()
		historyCount.Set(0)
		statusText.Set("History cleared")
	}

	// Custom commands.
	clipMenu := swiftui.CommandMenu{
		Title: "Clipboard",
		Items: []swiftui.CommandItem{
			{
				Title:    "Add Timestamp",
				Shortcut: swiftui.KeyboardShortcut{Key: "t", Modifiers: swiftui.ModCommand | swiftui.ModShift},
				Action: func(swiftui.CommandContext) {
					addEntry(time.Now().Format("2006-01-02 15:04:05"))
				},
			},
			{
				Title:    "Add Note",
				Shortcut: swiftui.KeyboardShortcut{Key: "n", Modifiers: swiftui.ModCommand | swiftui.ModShift},
				Action: func(swiftui.CommandContext) {
					mu.Lock()
					n := len(history) + 1
					mu.Unlock()
					addEntry(fmt.Sprintf("Note #%d", n))
				},
			},
			swiftui.Separator(),
			{
				Title:    "Clear History",
				Shortcut: swiftui.KeyboardShortcut{Key: "k", Modifiers: swiftui.ModCommand | swiftui.ModShift},
				Action: func(swiftui.CommandContext) {
					clearAll()
				},
				Enabled: func() bool {
					mu.Lock()
					defer mu.Unlock()
					return len(history) > 0
				},
			},
		},
	}

	lifecycle := swiftui.AppLifecycle{
		OnLaunched: func() {
			log.Println("menu-bar: launched")
		},
		OnActivate: func() {
			log.Println("menu-bar: activated")
		},
		OnResignActive: func() {
			log.Println("menu-bar: resigned active")
		},
		OnTerminate: func() {
			log.Println("menu-bar: terminating")
		},
	}

	popover := swiftui.VStackSpaced(10,
		swiftui.HStack(
			swiftui.Label("Clipboard History", "doc.on.clipboard").
				Font(swiftui.FontHeadline).
				FontWeight(swiftui.WeightSemibold).
				AsView(),
			swiftui.Spacer(),
			swiftui.DynamicView(historyCount, func(n int) swiftui.View {
				return swiftui.Text(fmt.Sprintf("%d items", n)).
					Font(swiftui.FontCaption).
					ForegroundStyleNamed("secondary").
					AsView()
			}),
		),

		swiftui.Divider(),

		swiftui.DynamicView(historyCount, func(_ int) swiftui.View {
			mu.Lock()
			snapshot := make([]string, len(history))
			copy(snapshot, history)
			mu.Unlock()

			if len(snapshot) == 0 {
				return swiftui.Text("No items").
					Font(swiftui.FontBody).
					ForegroundStyleNamed("secondary").
					AsView().
					Frame(0, 60)
			}
			items := make([]swiftui.Viewable, len(snapshot))
			for i, entry := range snapshot {
				idx := i
				items[i] = swiftui.HStack(
					swiftui.Text(entry).
						Font(swiftui.FontBody).
						LineLimit(1).
						AsView(),
					swiftui.Spacer(),
					swiftui.Text(fmt.Sprintf("#%d", idx+1)).
						Font(swiftui.FontCaption2).
						ForegroundStyleNamed("tertiary").
						AsView(),
				).Padding(4)
			}
			return swiftui.VStackSpaced(2, items...)
		}),

		swiftui.Divider(),

		swiftui.HStack(
			swiftui.TextFromString(statusText).
				Font(swiftui.FontCaption).
				ForegroundStyleNamed("secondary").
				AsView(),
			swiftui.Spacer(),
			swiftui.Button("Clear", func() {
				clearAll()
			}).ButtonStyle(swiftui.ButtonStyleBordered).
				ControlSize(swiftui.ControlSizeSmall),
		),
	).Padding(14)

	err := swiftui.RunScenes(
		swiftui.MenuBarExtra(swiftui.MenuBarConfig{
			Label:        "Clips",
			SystemImage:  "doc.on.clipboard",
			Width:        320,
			Height:       360,
			OpenOnLaunch: true,
		}, popover),
		swiftui.WithCommands(swiftui.Commands(
			clipMenu,
			swiftui.StandardEditMenu(),
			swiftui.StandardWindowMenu(),
		)),
		swiftui.WithLifecycle(lifecycle),
	)
	if err != nil {
		log.Fatal(err)
	}
}
