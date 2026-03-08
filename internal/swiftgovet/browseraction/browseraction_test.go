package browseraction_test

import (
	"testing"

	"github.com/tmc/swiftui/internal/swiftgovet/browseraction"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, browseraction.Analyzer, "a")
}
