// Package swiftuiast provides small helpers for SwiftUI-specific analyzers.
package swiftuiast

import (
	"go/ast"
	"go/constant"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

// IndexCallbacks records function bodies that can be referenced by button
// callbacks.
func IndexCallbacks(pass *analysis.Pass) map[types.Object]*ast.BlockStmt {
	callbacks := make(map[types.Object]*ast.BlockStmt)
	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.AssignStmt:
				for i, lhs := range n.Lhs {
					if i >= len(n.Rhs) {
						break
					}
					ident, ok := lhs.(*ast.Ident)
					if !ok {
						continue
					}
					lit, ok := n.Rhs[i].(*ast.FuncLit)
					if !ok {
						continue
					}
					if obj := pass.TypesInfo.Defs[ident]; obj != nil {
						callbacks[obj] = lit.Body
					}
				}
			case *ast.FuncDecl:
				if n.Name == nil || n.Body == nil {
					return true
				}
				if obj := pass.TypesInfo.Defs[n.Name]; obj != nil {
					callbacks[obj] = n.Body
				}
			case *ast.ValueSpec:
				for i, name := range n.Names {
					if i >= len(n.Values) {
						break
					}
					lit, ok := n.Values[i].(*ast.FuncLit)
					if !ok {
						continue
					}
					if obj := pass.TypesInfo.Defs[name]; obj != nil {
						callbacks[obj] = lit.Body
					}
				}
			}
			return true
		})
	}
	return callbacks
}

// ResolveCallbackBody resolves a direct or named callback expression to its
// function body.
func ResolveCallbackBody(pass *analysis.Pass, callbacks map[types.Object]*ast.BlockStmt, expr ast.Expr) (*ast.BlockStmt, bool) {
	switch expr := expr.(type) {
	case *ast.FuncLit:
		return expr.Body, expr.Body != nil
	case *ast.Ident:
		obj := pass.TypesInfo.ObjectOf(expr)
		if obj == nil {
			return nil, false
		}
		body, ok := callbacks[obj]
		return body, ok
	default:
		return nil, false
	}
}

// StringConst resolves a compile-time string constant.
func StringConst(pass *analysis.Pass, expr ast.Expr) (string, bool) {
	tv, ok := pass.TypesInfo.Types[expr]
	if !ok || tv.Value == nil || tv.Value.Kind() != constant.String {
		return "", false
	}
	return constant.StringVal(tv.Value), true
}

// IsNoOpBody reports whether body has no observable effect.
func IsNoOpBody(body *ast.BlockStmt) bool {
	if body == nil {
		return false
	}
	for _, stmt := range body.List {
		switch stmt := stmt.(type) {
		case *ast.EmptyStmt:
			continue
		case *ast.ReturnStmt:
			if len(stmt.Results) == 0 {
				continue
			}
		}
		return false
	}
	return true
}

// IsSwiftUICall reports whether call invokes the named function from the
// github.com/tmc/swiftui package.
func IsSwiftUICall(pass *analysis.Pass, call *ast.CallExpr, name string) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != name {
		return false
	}
	pkgIdent, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	pkg, ok := pass.TypesInfo.Uses[pkgIdent].(*types.PkgName)
	if !ok {
		return false
	}
	return pkg.Imported().Path() == "github.com/tmc/swiftui"
}
