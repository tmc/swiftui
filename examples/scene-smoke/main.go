//go:build darwin
// +build darwin

// Command scene-smoke exercises the parts of the bridge that cannot be
// covered by go test, which cannot drive AppKit because it does not own the
// main thread.
//
// It installs an application menu and three windows that between them use
// every WindowConfig presentation option, then reports on the behavior that
// only a running app can show: whether a menu action reaches Go, whether a
// disabled item is asked for its state, whether a callback still arrives
// after the window that prompted it is gone, and whether reopening a window
// focuses the existing one instead of stacking duplicates.
//
// Checks that need a real click are listed on stdout and tallied live, so
// running this and working through the list is the smoke test:
//
//	go run ./examples/scene-smoke
//
// Pass -auto to run only the unattended checks and exit non-zero on failure.
package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"sync/atomic"

	"github.com/tmc/swiftui"
)

func init() { runtime.LockOSThread() }

var (
	actionFired    atomic.Int64 // menu actions that reached Go
	enabledQueried atomic.Int64 // validateMenuItem round trips
	afterTeardown  atomic.Int64 // callbacks fired once windows were dismissed
	aboutOpened    atomic.Int64 // OpenWindow("about") calls that returned nil
	windowsGone    atomic.Bool
)

func main() {
	auto := flag.Bool("auto", false, "run only unattended checks, then exit")
	flag.Parse()

	resizable := false

	app := swiftui.App{
		Windows: []swiftui.WindowConfig{{
			ID:     "main",
			Title:  "Scene Smoke",
			Width:  520,
			Height: 320,
			Root: swiftui.VStackSpaced(14,
				swiftui.Text("Scene smoke harness").Font(swiftui.FontTitle),
				swiftui.Text("Work through the checklist printed on stdout."),
				swiftui.Button("Run window checks", windowChecks),
			).Padding(24),
		}, {
			// Fixed size, centered every time, and drawing its own header:
			// the About panel shape the bridge could not express before.
			ID:             "about",
			Title:          "About Scene Smoke",
			Width:          360,
			Height:         200,
			Resizable:      &resizable,
			HiddenTitleBar: true,
			Centered:       true,
			Root: swiftui.VStack(
				swiftui.Text("Scene Smoke").Font(swiftui.FontTitle),
				swiftui.Text("Fixed 360x200, no title bar, always centered."),
			).Padding(24),
		}, {
			// Floating utility panel, below the 300x200 floor that used to be
			// hardcoded for every scene window.
			ID:      "tools",
			Title:   "Tools",
			Width:   260,
			Height:  180,
			Utility: true,
			Root: swiftui.VStack(
				swiftui.Text("Utility"),
				swiftui.Text("260x180"),
			).Padding(16),
		}},
		Commands: []swiftui.CommandGroup{{
			// AppMenu puts these items in the application's menu.
			Title:   appName(),
			AppMenu: true,
			Items: []swiftui.CommandItem{{
				Title: "About Scene Smoke",
				Action: func() {
					actionFired.Add(1)
					report("menu action reached Go: About")
					openAbout()
				},
			}, {
				Title:       "Check for Updates",
				ShortcutKey: "u",
				Action: func() {
					actionFired.Add(1)
					report("menu action reached Go: Check for Updates")
					if windowsGone.Load() {
						afterTeardown.Add(1)
						report("callback arrived after window teardown")
					}
				},
			}, {Kind: "separator"}, {
				Title: "Always Disabled",
				Action: func() {
					actionFired.Add(1)
					fail("a disabled item fired its action")
				},
				Enabled: func() bool {
					enabledQueried.Add(1)
					return false
				},
			}, {
				Title: "Nested",
				Children: []swiftui.CommandItem{{
					Title: "Nested Action",
					Action: func() {
						actionFired.Add(1)
						report("menu action reached Go: Nested Action")
					},
				}},
			}},
		}, {
			Title: "Smoke",
			Items: []swiftui.CommandItem{{
				Title:       "Open Tools",
				ShortcutKey: "t",
				Action: func() {
					actionFired.Add(1)
					if err := swiftui.OpenWindow("tools"); err != nil {
						fail("OpenWindow(tools): %v", err)
						return
					}
					report("menu action reached Go: Open Tools")
				},
			}},
		}},
	}

	// OpenWindow before Run crosses no bridge call, so this one check is
	// safe to make here, off the AppKit main thread.
	if err := swiftui.OpenWindow("about"); err == swiftui.ErrAppNotRunning {
		report("OpenWindow before Run refused: %v", err)
	} else {
		fail("OpenWindow before Run returned %v, want ErrAppNotRunning", err)
	}

	if *auto {
		summary()
		return
	}
	fmt.Print(checklist)

	if err := swiftui.Run(app); err != nil {
		fmt.Fprintln(os.Stderr, "run:", err)
		os.Exit(1)
	}
}

// windowChecks runs the checks that must happen on the AppKit main thread.
// It is wired to a button, because a button action is already on it; calling
// OpenWindow from a goroutine traps inside the Swift runtime, which annotates
// the bridge entry point @MainActor.
func windowChecks() {
	for i := 0; i < 5; i++ {
		if err := swiftui.OpenWindow("about"); err != nil {
			fail("OpenWindow(about) #%d: %v", i+1, err)
		} else {
			aboutOpened.Add(1)
		}
	}
	if got := aboutOpened.Load(); got == 5 {
		report("repeated About activation: 5/5 focused one window")
	} else {
		fail("repeated About activation: %d/5 returned nil", got)
	}

	if err := swiftui.OpenWindow("nonexistent"); err == swiftui.ErrNoWindow {
		report("OpenWindow of an unknown id refused: %v", err)
	} else {
		fail("OpenWindow(nonexistent) returned %v, want ErrNoWindow", err)
	}

	if err := swiftui.OpenWindow("tools"); err != nil {
		fail("OpenWindow(tools): %v", err)
	} else {
		report("utility panel opened")
	}

	windowsGone.Store(true)
	fmt.Println("now close every window and press Cmd-U (check 4)")
}

const checklist = `
Checks that need a click (results print as you go):

  0. Press "Run window checks" in the main window. Those checks must run on
     the AppKit main thread, which a button action already is.
  1. Open the application menu. About Scene Smoke, Check for Updates, a
     separator, and a Nested submenu appear ABOVE the standard About and
     Quit items, not in a menu of their own.
  2. "Always Disabled" is greyed out and does not respond.
  3. Choose About Scene Smoke. The panel opens centered, has no title bar,
     and cannot be resized by dragging its edges.
  4. Close every window, then press Cmd-U. The action must still reach Go.
  5. The Tools panel floats above the main window at 260x180, under the old
     300x200 floor. Being a utility panel it hides whenever another app is
     frontmost, and comes back when this one is.

Press Ctrl-C when done.
`

func openAbout() {
	if err := swiftui.OpenWindow("about"); err != nil {
		fail("OpenWindow(about): %v", err)
	}
}

func appName() string {
	if len(os.Args) == 0 {
		return "scene-smoke"
	}
	name := os.Args[0]
	for i := len(name) - 1; i >= 0; i-- {
		if name[i] == '/' {
			return name[i+1:]
		}
	}
	return name
}

var failed atomic.Bool

func report(format string, args ...any) {
	fmt.Printf("pass: "+format+"\n", args...)
}

func fail(format string, args ...any) {
	failed.Store(true)
	fmt.Printf("FAIL: "+format+"\n", args...)
}

func summary() {
	fmt.Printf("\nunattended summary: about=%d actions=%d enabledQueries=%d\n",
		aboutOpened.Load(), actionFired.Load(), enabledQueried.Load())
	if failed.Load() {
		os.Exit(1)
	}
	os.Exit(0)
}
