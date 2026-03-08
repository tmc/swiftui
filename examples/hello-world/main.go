// Command hello-world is the simplest SwiftUI app from Go.
//
// Usage:
//
//	go run .
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
	}, swiftui.Text("Hello from Go!").
		Font(swiftui.FontLargeTitle).
		Padding(20).
		AsView())
}
