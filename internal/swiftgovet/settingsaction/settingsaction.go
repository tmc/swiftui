// Package settingsaction reports misleading Save and Revert button handlers.
package settingsaction

import (
	"go/ast"
	"go/constant"
	"strings"

	"github.com/tmc/swiftui/internal/swiftgovet/swiftuiast"
	"golang.org/x/tools/go/analysis"
)

// Analyzer reports Save, Revert, and Apply button callbacks that only clear
// change-tracking state.
var Analyzer = &analysis.Analyzer{
	Name: "settingsaction",
	Doc:  "report Save, Revert, and Apply button callbacks that only clear change-tracking state",
	Run:  run,
}

func run(pass *analysis.Pass) (any, error) {
	callbacks := swiftuiast.IndexCallbacks(pass)

	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || !swiftuiast.IsSwiftUICall(pass, call, "Button") || len(call.Args) < 2 {
				return true
			}

			label, ok := swiftuiast.StringConst(pass, call.Args[0])
			if !ok || !isTrackedActionLabel(label) {
				return true
			}

			body, ok := swiftuiast.ResolveCallbackBody(pass, callbacks, call.Args[1])
			if !ok {
				return true
			}

			tracker, ok := onlyTrackerReset(pass, body)
			if !ok {
				return true
			}

			pass.Reportf(
				call.Args[0].Pos(),
				"%s button action only resets %s tracking state; persist or restore the underlying settings as well",
				label,
				tracker,
			)
			return true
		})
	}

	return nil, nil
}

func isTrackedActionLabel(label string) bool {
	switch strings.ToLower(strings.TrimSpace(label)) {
	case "apply", "revert", "save":
		return true
	default:
		return false
	}
}

func onlyTrackerReset(pass *analysis.Pass, body *ast.BlockStmt) (string, bool) {
	var tracker string
	for _, stmt := range body.List {
		if _, ok := stmt.(*ast.EmptyStmt); ok {
			continue
		}
		name, ok := trackerResetStmt(pass, stmt)
		if !ok {
			return "", false
		}
		if tracker == "" {
			tracker = name
		}
	}
	return tracker, tracker != ""
}

func trackerResetStmt(pass *analysis.Pass, stmt ast.Stmt) (string, bool) {
	expr, ok := stmt.(*ast.ExprStmt)
	if !ok {
		return "", false
	}
	call, ok := expr.X.(*ast.CallExpr)
	if !ok {
		return "", false
	}
	return trackerResetCall(pass, call)
}

func trackerResetCall(pass *analysis.Pass, call *ast.CallExpr) (string, bool) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || len(call.Args) == 0 {
		return "", false
	}
	switch sel.Sel.Name {
	case "Set", "SetAnimated", "SetAnimatedWith":
	default:
		return "", false
	}
	if !isZeroEquivalent(pass, call.Args[0]) {
		return "", false
	}
	tracker := trackerName(sel.X)
	if !looksLikeTracker(tracker) {
		return "", false
	}
	return tracker, true
}

func isZeroEquivalent(pass *analysis.Pass, expr ast.Expr) bool {
	tv, ok := pass.TypesInfo.Types[expr]
	if !ok || tv.Value == nil {
		return false
	}
	switch tv.Value.Kind() {
	case constant.Bool:
		return !constant.BoolVal(tv.Value)
	case constant.Int, constant.Float, constant.Complex:
		return constant.Sign(tv.Value) == 0
	case constant.String:
		return constant.StringVal(tv.Value) == ""
	default:
		return false
	}
}

func trackerName(expr ast.Expr) string {
	switch expr := expr.(type) {
	case *ast.Ident:
		return expr.Name
	case *ast.IndexExpr:
		return trackerName(expr.X)
	case *ast.ParenExpr:
		return trackerName(expr.X)
	case *ast.SelectorExpr:
		return expr.Sel.Name
	case *ast.StarExpr:
		return trackerName(expr.X)
	default:
		return ""
	}
}

func looksLikeTracker(name string) bool {
	name = strings.ToLower(name)
	for _, part := range []string{"change", "dirty", "modified", "unsaved"} {
		if strings.Contains(name, part) {
			return true
		}
	}
	return false
}
