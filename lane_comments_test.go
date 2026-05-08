package swiftui

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHandWrittenExportedTypesCarryLaneLabels(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("filepath.Glob() error = %v", err)
	}

	var missing []string
	fset := token.NewFileSet()
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("os.ReadFile(%q) error = %v", path, err)
		}
		if strings.Contains(string(data), "Code generated") {
			continue
		}
		file, err := parser.ParseFile(fset, path, data, parser.ParseComments)
		if err != nil {
			t.Fatalf("parser.ParseFile(%q) error = %v", path, err)
		}
		if file.Name == nil || file.Name.Name != "swiftui" {
			continue
		}
		for _, decl := range file.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.TYPE {
				continue
			}
			for _, spec := range gen.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok || !typeSpec.Name.IsExported() {
					continue
				}
				doc := ""
				if typeSpec.Doc != nil {
					doc = typeSpec.Doc.Text()
				} else if gen.Doc != nil {
					doc = gen.Doc.Text()
				}
				if hasLaneLabel(doc) {
					continue
				}
				missing = append(missing, path+":"+typeSpec.Name.Name)
			}
		}
	}

	if len(missing) != 0 {
		t.Fatalf("exported types missing lane labels: %s", strings.Join(missing, ", "))
	}
}

func hasLaneLabel(doc string) bool {
	return strings.Contains(doc, "Bridge surface.") ||
		strings.Contains(doc, "Curated surface.") ||
		strings.Contains(doc, "Runtime surface.")
}
