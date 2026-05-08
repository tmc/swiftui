//go:build darwin

// Command commands demonstrates menu bar commands, keyboard shortcuts,
// dynamic enable/disable, submenus, and lifecycle hooks.
//
// This is a minimal quickstart showing just menus and shortcuts.
// See examples/scenes for the flagship app-shell command story and
// examples/note-pad for a smaller stateful command demo.
//
// Usage:
//
//	go run .
package main

import (
	"fmt"
	"runtime"

	"github.com/tmc/swiftui"
)

func init() { runtime.LockOSThread() }

func main() {
	status := swiftui.NewStringState("Ready.")

	fileMenu := swiftui.CommandMenu{
		Title: "File",
		Items: []swiftui.CommandItem{
			{
				Title:    "New",
				Shortcut: swiftui.KeyboardShortcut{Key: "n", Modifiers: swiftui.ModCommand},
				Action:   func(swiftui.CommandContext) { status.Set("New document created.") },
			},
			{
				Title:    "Open…",
				Shortcut: swiftui.KeyboardShortcut{Key: "o", Modifiers: swiftui.ModCommand},
				Action:   func(swiftui.CommandContext) { status.Set("Open dialog shown.") },
			},
			swiftui.Separator(),
			{
				Title:    "Save",
				Shortcut: swiftui.KeyboardShortcut{Key: "s", Modifiers: swiftui.ModCommand},
				Action:   func(swiftui.CommandContext) { status.Set("Saved.") },
			},
		},
	}

	toolsMenu := swiftui.CommandMenu{
		Title: "Tools",
		Items: []swiftui.CommandItem{
			{
				Title:    "Run Build",
				Shortcut: swiftui.KeyboardShortcut{Key: "b", Modifiers: swiftui.ModCommand},
				Action:   func(swiftui.CommandContext) { status.Set("Build started…") },
			},
			swiftui.Separator(),
			{
				Title: "Export",
				Children: []swiftui.CommandItem{
					{Title: "Export as PDF", Action: func(swiftui.CommandContext) { status.Set("Exporting PDF…") }},
					{Title: "Export as HTML", Action: func(swiftui.CommandContext) { status.Set("Exporting HTML…") }},
				},
			},
		},
	}

	lifecycle := swiftui.AppLifecycle{
		OnLaunched:  func() { fmt.Println("commands: launched") },
		OnTerminate: func() { fmt.Println("commands: terminating") },
	}

	content := swiftui.VStack(
		swiftui.Text("Commands Demo").Font(swiftui.FontTitle).AsView(),
		swiftui.Spacer().Frame(0, 12),
		swiftui.Text("Use File and Tools menus above.").
			ForegroundStyleNamed("secondary").AsView(),
		swiftui.Spacer(),
		swiftui.TextFromString(status).
			Font(swiftui.FontCaption).
			ForegroundStyleNamed("secondary").AsView(),
	).Padding(24)

	if err := swiftui.RunScenes(
		swiftui.Window("commands-demo", swiftui.AppConfig{Title: "Commands Demo", Width: 480, Height: 320}, content),
		swiftui.WithCommands(swiftui.Commands(fileMenu, swiftui.StandardEditMenu(), toolsMenu)),
		swiftui.WithLifecycle(lifecycle),
	); err != nil {
		panic(err)
	}
}
