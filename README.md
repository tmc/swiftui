# swiftui

[![Go Reference](https://pkg.go.dev/badge/github.com/tmc/swiftui.svg)](https://pkg.go.dev/github.com/tmc/swiftui)

Go bindings for Apple's SwiftUI framework on macOS. The root package exposes
core SwiftUI views, modifiers, and app lifecycle APIs, and subpackages such as
`avkit`, `quicklook`, and `spritekit` expose framework
overlays that return view pointers compatible with `swiftui.ViewFromPointer`.

Auto-generated from Apple developer documentation via
[applegen](https://github.com/tmc/appledocs/cmd/applegen).

## Requirements

- macOS 26 or later with the Swift toolchain available in `PATH`
- Xcode or Command Line Tools if the vendored bridge needs to be built

The vendored Swift bridge is loaded at runtime. If no prebuilt dylib is
available, the package runs `swift build` in `internal/swift/` automatically.

## Quick start

```go
package main

import (
	"runtime"

	"github.com/tmc/swiftui"
)

func init() { runtime.LockOSThread() }

func main() {
	swiftui.Run(swiftui.AppConfig{
		Title:  "Hello World",
		Width:  400,
		Height: 300,
	}, swiftui.Text("Hello from Go!").Padding(20).AsView())
}
```

Try the bundled example with:

```sh
go run ./examples/hello-world
```

## Disclaimer

This is not an official Apple product. Apple, macOS, and all related
frameworks are trademarks of Apple Inc. This project is an independent,
community-driven effort and is not affiliated with, endorsed by, or sponsored
by Apple Inc.

## License

[MIT](LICENSE)
