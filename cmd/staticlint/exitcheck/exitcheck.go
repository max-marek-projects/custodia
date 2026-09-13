// Package exitcheck implements analyzer that forbids os.Exit call in main function.
package exitcheck

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name: "exitcheck",
	Doc:  "checks that os.Exit is not called directly from main function of main package",
	Run:  run,
}

// isOsPackage reports whether the given identifier resolves to the standard "os" package,
// regardless of the import alias used (e.g. `import myos "os"`).
func isOsPackage(pass *analysis.Pass, ident *ast.Ident) bool {
	obj := pass.TypesInfo.Uses[ident]
	pkgName, ok := obj.(*types.PkgName)
	if !ok {
		return false
	}
	return pkgName.Imported().Path() == "os"
}

func run(pass *analysis.Pass) (interface{}, error) {
	if pass.Pkg.Name() != "main" {
		return nil, nil
	}

	for _, file := range pass.Files {
		// skip temporary files
		filename := pass.Fset.Position(file.Pos()).Filename
		if strings.Contains(filename, "go-build") {
			continue
		}
		ast.Inspect(file, func(n ast.Node) bool {
			fn, ok := n.(*ast.FuncDecl)
			if !ok {
				return true
			}
			if fn.Recv != nil || fn.Name.Name != "main" {
				return true
			}
			if fn.Body == nil {
				return true
			}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				if sel.Sel.Name != "Exit" {
					return true
				}
				ident, ok := sel.X.(*ast.Ident)
				if !ok {
					return true
				}
				if isOsPackage(pass, ident) {
					pass.Reportf(call.Pos(), "direct call to os.Exit in main function is forbidden")
				}
				return true
			})
			return true
		})
	}

	return nil, nil
}
