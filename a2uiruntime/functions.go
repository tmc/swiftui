package a2uiruntime

import (
	"fmt"
	"os/exec"

	"github.com/tmc/swiftui/a2ui"
)

func executeClientFunction(fn *a2ui.FunctionCall, dm *a2ui.DataModel) error {
	if fn == nil {
		return nil
	}
	switch fn.Call {
	case "openUrl":
		raw, _ := fn.Args["url"].(string)
		if raw == "" {
			return fmt.Errorf("openUrl: missing url")
		}
		cmd := exec.Command("open", raw)
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("openUrl: %w", err)
		}
		return nil
	default:
		return fmt.Errorf("unsupported client function %q", fn.Call)
	}
}
