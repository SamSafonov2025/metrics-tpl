// Package main provides a static analyzer that checks for:
// - Usage of built-in panic function
// - Calls to log.Fatal and os.Exit outside of main function in main package
package main

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

// ExitCheckAnalyzer is the custom static analyzer.
var ExitCheckAnalyzer = &analysis.Analyzer{
	Name: "exitcheck",
	Doc:  "checks for direct calls to os.Exit, log.Fatal, and panic outside main",
	Run:  run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	// Iterate through all files in the package
	for _, file := range pass.Files {
		// Check if we are in the main package
		isMainPackage := pass.Pkg.Name() == "main"

		// Inspect the AST
		ast.Inspect(file, func(node ast.Node) bool {
			// Look for function declarations
			if fn, ok := node.(*ast.FuncDecl); ok {
				isMainFunc := fn.Name.Name == "main" && isMainPackage

				// Inspect the function body
				if fn.Body != nil {
					ast.Inspect(fn.Body, func(n ast.Node) bool {
						if call, ok := n.(*ast.CallExpr); ok {
							checkCall(pass, call, isMainPackage, isMainFunc)
						}
						return true
					})
				}

				// Don't descend into function body again
				return false
			}

			return true
		})
	}
	return nil, nil
}

func checkCall(pass *analysis.Pass, call *ast.CallExpr, isMainPackage, insideMainFunc bool) {
	// Check for built-in panic
	if ident, ok := call.Fun.(*ast.Ident); ok {
		if ident.Name == "panic" {
			pass.Reportf(call.Pos(), "direct call to panic is not allowed")
		}
	}

	// Check for os.Exit and log.Fatal
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		if x, ok := sel.X.(*ast.Ident); ok {
			funcName := sel.Sel.Name
			pkgName := x.Name

			// Check for os.Exit
			if pkgName == "os" && funcName == "Exit" {
				// Allow os.Exit only in main function of main package
				if !isMainPackage || !insideMainFunc {
					pass.Reportf(call.Pos(), "os.Exit must only be called in main function of main package")
				}
			}

			// Check for log.Fatal variants
			if pkgName == "log" && (funcName == "Fatal" || funcName == "Fatalf" || funcName == "Fatalln") {
				// Allow log.Fatal only in main function of main package
				if !isMainPackage || !insideMainFunc {
					pass.Reportf(call.Pos(), "%s.%s must only be called in main function of main package", pkgName, funcName)
				}
			}
		}
	}
}
