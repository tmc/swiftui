package settingsaction_test

import (
	"testing"

	"github.com/tmc/swiftui/internal/swiftgovet/settingsaction"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, settingsaction.Analyzer, "a")
}
