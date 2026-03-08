// Package browseraction reports browser controls wired to no-op callbacks.
package browseraction

import (
	"go/ast"
	"strings"

	"github.com/tmc/swiftui/internal/swiftgovet/swiftuiast"
	"golang.org/x/tools/go/analysis"
)

// Analyzer reports browser action buttons that do nothing when pressed.
var Analyzer = &analysis.Analyzer{
	Name: "browseraction",
	Doc:  "report browser action buttons with empty callbacks",
	Run:  run,
}

func run(pass *analysis.Pass) (any, error) {
	callbacks := swiftuiast.IndexCallbacks(pass)

	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			label, callback, ok := browserActionButton(pass, call)
			if !ok {
				return true
			}

			body, ok := swiftuiast.ResolveCallbackBody(pass, callbacks, callback)
			if !ok || !swiftuiast.IsNoOpBody(body) {
				return true
			}

			pass.Reportf(
				call.Args[0].Pos(),
				"browser action button %q has an empty callback; wire it to navigation or sharing behavior, or remove the control",
				label,
			)
			return true
		})
	}

	return nil, nil
}

func browserActionButton(pass *analysis.Pass, call *ast.CallExpr) (string, ast.Expr, bool) {
	if len(call.Args) < 2 {
		return "", nil, false
	}
	if !swiftuiast.IsSwiftUICall(pass, call, "Button") && !swiftuiast.IsSwiftUICall(pass, call, "ButtonWithImage") {
		return "", nil, false
	}
	label, ok := swiftuiast.StringConst(pass, call.Args[0])
	if !ok || !isBrowserActionLabel(label) {
		return "", nil, false
	}
	return label, call.Args[1], true
}

func isBrowserActionLabel(label string) bool {
	switch strings.ToLower(strings.TrimSpace(label)) {
	case "arrow.clockwise", "back", "chevron.left", "chevron.right", "forward", "home", "house", "reload", "safari", "share", "square.and.arrow.up":
		return true
	default:
		return false
	}
}
