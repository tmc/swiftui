// Command swiftgovet provides project-specific vet checks for SwiftUI Go code.
package main

import (
	"github.com/tmc/swiftui/internal/swiftgovet/browseraction"
	"github.com/tmc/swiftui/internal/swiftgovet/settingsaction"
	"golang.org/x/tools/go/analysis/multichecker"
)

func main() {
	multichecker.Main(
		browseraction.Analyzer,
		settingsaction.Analyzer,
	)
}
