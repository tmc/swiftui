# swiftui

[![Go Reference](https://pkg.go.dev/badge/github.com/tmc/swiftui.svg)](https://pkg.go.dev/github.com/tmc/swiftui)

Go bindings for Apple's SwiftUI framework on macOS. The root package exposes
core SwiftUI views, modifiers, and app lifecycle APIs, and subpackages such as
`avkit`, `quicklook`, and `spritekit` expose framework
overlays that return view pointers compatible with `swiftui.ViewFromPointer`.

Auto-generated from Apple developer documentation via an internal tool called applegen.

## Requirements

- macOS 26 or later with the Swift toolchain available in `PATH`
- Xcode or Command Line Tools if the vendored bridge needs to be built

The vendored Swift bridge is loaded at runtime. If no prebuilt dylib is
available, the package runs `swift build` in `internal/swift/` automatically.

The embedded `internal/embeddedbridge/libSwiftUIBridge.dylib` is a universal
binary. A plain `swift build` emits a single architecture, so rebuilding it by
hand needs `--arch arm64 --arch x86_64`, taking the result from
`.build/apple/Products/Release/` and confirming it with `lipo -info` before
copying it into place. Check the new dylib's exported symbols against the old
with `nm -gU`: symbols that go missing fall back to stubs at runtime rather
than failing to load.

TODO: the dylib-copying step in the generated `generate.go` cannot do this. Its
`find` glob matches only a lowercase `release` path segment, so it never sees
the universal output in `Release`, and `sort | tail` then selects `x86_64` from
what remains — silently downgrading the embedded dylib to one non-native
architecture. Fixing it means changing the template in appledocs
(`internal/swiftbridge/overlay_templates.go`, which emits this step for every
bridge, not just this one).

## Quick start

```go
package main

import (
	"log"
	"runtime"

	"github.com/tmc/swiftui"
)

func init() { runtime.LockOSThread() }

func main() {
	if err := swiftui.Run(swiftui.App{Windows: []swiftui.WindowConfig{{
		Title:  "Hello World",
		Width:  400,
		Height: 300,
		Root:   swiftui.Text("Hello from Go!").Padding(20).AsView(),
	}}}); err != nil {
		log.Fatal(err)
	}
}
```

Try the bundled example with:

```sh
go run ./examples/hello-world
```

## Scenes & multiple windows

A single `swiftui.App` describes every surface an app presents, so the same
`Run` call covers single-window, multi-window, menu-bar, and Settings apps.
Give each window a stable `ID` (required once there is more than one), and open
or focus one at runtime with `swiftui.OpenWindow`:

```go
app := swiftui.App{
	Windows: []swiftui.WindowConfig{
		{ID: "main", Title: "Main", Width: 520, Height: 360, Root: mainView},
		{ID: "inspector", Title: "Inspector", Width: 320, Height: 320, Root: inspectorView},
	},
	Settings: &swiftui.SettingsConfig{Title: "Settings", Root: settingsView},
}
swiftui.Run(app)

// Later, from a button callback, open or focus the inspector by id:
if err := swiftui.OpenWindow("inspector"); err != nil {
	log.Printf("open inspector: %v", err)
}
```

A menu-bar (status-bar) app sets `App.MenuBar` instead of (or alongside)
`Windows`; `swiftui.RunMenuBar` and `swiftui.RunWithMenuBar` are conveniences
for the common cases.

```sh
go run ./examples/multi-window
```

## Disclaimer

This is not an official Apple product. Apple, macOS, and all related
frameworks are trademarks of Apple Inc. This project is an independent,
community-driven effort and is not affiliated with, endorsed by, or sponsored
by Apple Inc.

## License

[MIT](LICENSE)
