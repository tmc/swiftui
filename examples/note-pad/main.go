//go:build darwin

// Command note-pad demonstrates a note-taking app with app commands,
// accessibility identifiers, and lifecycle hooks.
//
// It exercises the T1 surface: File/Edit/View menus with keyboard shortcuts,
// dynamic enable/disable based on note count, accessibility identifiers on
// all interactive elements, and lifecycle logging.
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

type note struct {
	title   string
	created time.Time
}

func main() {
	var mu sync.Mutex
	var notes []note
	noteCount := swiftui.NewIntState(0)
	revision := swiftui.NewIntState(0)
	statusText := swiftui.NewStringState("Welcome to Note Pad")

	bump := func() { revision.Set(revision.Get() + 1) }

	addNote := func() {
		mu.Lock()
		n := len(notes) + 1
		notes = append(notes, note{
			title:   fmt.Sprintf("Note %d", n),
			created: time.Now(),
		})
		count := len(notes)
		mu.Unlock()
		noteCount.Set(count)
		statusText.Set(fmt.Sprintf("Created note %d", n))
		bump()
	}

	deleteLastNote := func() {
		mu.Lock()
		if len(notes) == 0 {
			mu.Unlock()
			return
		}
		removed := notes[len(notes)-1].title
		notes = notes[:len(notes)-1]
		count := len(notes)
		mu.Unlock()
		noteCount.Set(count)
		statusText.Set(fmt.Sprintf("Deleted %s", removed))
		bump()
	}

	clearNotes := func() {
		mu.Lock()
		notes = notes[:0]
		mu.Unlock()
		noteCount.Set(0)
		statusText.Set("All notes cleared")
		bump()
	}

	hasNotes := func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(notes) > 0
	}

	// File menu.
	fileMenu := swiftui.CommandMenu{
		Title: "File",
		Items: []swiftui.CommandItem{
			{
				Title:    "New Note",
				Shortcut: swiftui.KeyboardShortcut{Key: "n", Modifiers: swiftui.ModCommand},
				Action:   func(swiftui.CommandContext) { addNote() },
			},
			swiftui.Separator(),
			{
				Title:    "Delete Last Note",
				Shortcut: swiftui.KeyboardShortcut{Key: "d", Modifiers: swiftui.ModCommand},
				Enabled:  hasNotes,
				Action:   func(swiftui.CommandContext) { deleteLastNote() },
			},
			{
				Title:   "Delete All Notes",
				Enabled: hasNotes,
				Action:  func(swiftui.CommandContext) { clearNotes() },
			},
		},
	}

	// View menu with display options.
	viewMenu := swiftui.CommandMenu{
		Title: "View",
		Items: []swiftui.CommandItem{
			{
				Title: "Sort By",
				Children: []swiftui.CommandItem{
					{
						Title: "Date Created",
						Action: func(swiftui.CommandContext) {
							statusText.Set("Sorted by date (already default)")
						},
					},
					{
						Title: "Title",
						Action: func(swiftui.CommandContext) {
							mu.Lock()
							// Simple alphabetical sort.
							for i := range notes {
								for j := i + 1; j < len(notes); j++ {
									if notes[j].title < notes[i].title {
										notes[i], notes[j] = notes[j], notes[i]
									}
								}
							}
							mu.Unlock()
							statusText.Set("Sorted by title")
							bump()
						},
					},
				},
			},
		},
	}

	lifecycle := swiftui.AppLifecycle{
		OnLaunched:     func() { log.Println("note-pad: launched") },
		OnActivate:     func() { log.Println("note-pad: activated") },
		OnResignActive: func() { log.Println("note-pad: resigned active") },
		OnTerminate:    func() { log.Println("note-pad: terminating") },
	}

	content := swiftui.VStackSpaced(12,
		// Header with AX identifiers.
		swiftui.HStack(
			swiftui.Label("Note Pad", "note.text").
				Font(swiftui.FontTitle).
				FontWeight(swiftui.WeightBold).
				AsView().
				AccessibilityIdentifier("header-title"),
			swiftui.Spacer(),
			swiftui.DynamicView(noteCount, func(n int) swiftui.View {
				return swiftui.Text(fmt.Sprintf("%d notes", n)).
					Font(swiftui.FontCallout).
					ForegroundStyleNamed("secondary").
					AsView().
					AccessibilityIdentifier("note-count")
			}),
		),

		swiftui.Divider(),

		// Note list.
		swiftui.DynamicView(revision, func(_ int) swiftui.View {
			mu.Lock()
			snapshot := make([]note, len(notes))
			copy(snapshot, notes)
			mu.Unlock()

			if len(snapshot) == 0 {
				return swiftui.VStack(
					swiftui.Spacer(),
					swiftui.Image("note.text").
						ImageScale(swiftui.ImageScaleLarge).
						ForegroundStyleNamed("secondary"),
					swiftui.Text("No notes yet").
						Font(swiftui.FontTitle3).
						ForegroundStyleNamed("secondary").
						AsView(),
					swiftui.Text("Press ⌘N to create a note").
						Font(swiftui.FontCaption).
						ForegroundStyleNamed("tertiary").
						AsView(),
					swiftui.Spacer(),
				).AccessibilityIdentifier("empty-state")
			}

			items := make([]swiftui.Viewable, len(snapshot))
			for i, n := range snapshot {
				items[i] = swiftui.HStack(
					swiftui.VStack(
						swiftui.Text(n.title).
							Font(swiftui.FontBody).
							FontWeight(swiftui.WeightMedium),
						swiftui.Text(n.created.Format("3:04 PM")).
							Font(swiftui.FontCaption).
							ForegroundStyleNamed("secondary"),
					),
					swiftui.Spacer(),
				).Padding(8).
					AccessibilityIdentifier(fmt.Sprintf("note-%d", i))
			}
			return swiftui.ScrollView(
				swiftui.LazyVStack(items...),
			).AccessibilityIdentifier("note-list")
		}),

		swiftui.Divider(),

		// Status bar.
		swiftui.HStack(
			swiftui.TextFromString(statusText).
				Font(swiftui.FontCaption).
				ForegroundStyleNamed("secondary").
				AsView().
				AccessibilityIdentifier("status-text"),
			swiftui.Spacer(),
			swiftui.Button("New Note", func() {
				addNote()
			}).ButtonStyle(swiftui.ButtonStyleBorderedProminent).
				ControlSize(swiftui.ControlSizeSmall).
				AccessibilityIdentifier("new-note-button"),
		),
	).Padding(20)

	err := swiftui.RunScenes(
		swiftui.Window("note-pad", swiftui.AppConfig{
			Title:  "Note Pad",
			Width:  460,
			Height: 520,
		}, content),
		swiftui.WithCommands(swiftui.Commands(
			fileMenu,
			swiftui.StandardEditMenu(),
			viewMenu,
			swiftui.StandardWindowMenu(),
			swiftui.StandardHelpMenu(),
		)),
		swiftui.WithLifecycle(lifecycle),
	)
	if err != nil {
		log.Fatal(err)
	}
}
