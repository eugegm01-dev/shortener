// Command staticlint is a custom multichecker for the URL shortener project.
//
// It combines several static analysis tools:
//   - Standard analyzers: printf, shadow, structtag, composite, nilfunc
//   - All SA class analyzers from staticcheck.io (critical style/error checks)
//   - One ST class analyzer from staticcheck: ST1000 (missing package comment)
//   - Two additional third‑party analyzers:
//   - github.com/timakin/bodyclose – checks that HTTP response bodies are closed
//   - github.com/gostaticanalysis/nilerr – checks for nil error returns that are not handled
//   - A custom analyzer that forbids direct os.Exit calls inside the main function of the main package.
//
// Usage:
//
//	go build -o staticlint ./cmd/staticlint && ./staticlint ./...
//
// The multichecker returns a non‑zero exit code if any issues are found.
package main

import (
	"go/ast"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"golang.org/x/tools/go/analysis/passes/composite"
	"golang.org/x/tools/go/analysis/passes/nilfunc"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/shadow"
	"golang.org/x/tools/go/analysis/passes/structtag"

	"github.com/gostaticanalysis/nilerr"
	"github.com/timakin/bodyclose/passes/bodyclose"
	"honnef.co/go/tools/staticcheck"
	"honnef.co/go/tools/stylecheck"
)

// noExitAnalyzer prohibits direct calls to os.Exit in the main function of the main package.
var noExitAnalyzer = &analysis.Analyzer{
	Name: "noexit",
	Doc:  "disallow direct os.Exit calls in main function of main package",
	Run:  runNoExit,
}

func runNoExit(pass *analysis.Pass) (interface{}, error) {
	if pass.Pkg.Name() != "main" {
		return nil, nil
	}

	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			funcDecl, ok := n.(*ast.FuncDecl)
			if !ok || funcDecl.Name.Name != "main" {
				return true
			}
			ast.Inspect(funcDecl.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				if ident, ok := sel.X.(*ast.Ident); ok && ident.Name == "os" && sel.Sel.Name == "Exit" {
					pass.Reportf(call.Pos(), "direct call to os.Exit in main function is forbidden")
				}
				return true
			})
			return false
		})
	}
	return nil, nil
}

func main() {
	// 1. Standard analyzers
	standardAnalyzers := []*analysis.Analyzer{
		printf.Analyzer,
		shadow.Analyzer,
		structtag.Analyzer,
		composite.Analyzer,
		nilfunc.Analyzer,
	}

	// 2. All SA class analyzers from staticcheck
	var saAnalyzers []*analysis.Analyzer
	for _, a := range staticcheck.Analyzers {
		if strings.HasPrefix(a.Analyzer.Name, "SA") {
			saAnalyzers = append(saAnalyzers, a.Analyzer)
		}
	}

	// 3. One analyzer from ST class (ST1000 – package comment)
	var stAnalyzers []*analysis.Analyzer
	for _, a := range stylecheck.Analyzers {
		if a.Analyzer.Name == "ST1000" {
			stAnalyzers = append(stAnalyzers, a.Analyzer)
			break
		}
	}

	// 4. Third‑party analyzers
	externalAnalyzers := []*analysis.Analyzer{
		bodyclose.Analyzer,
		nilerr.Analyzer,
	}

	// 5. Custom analyzer
	customAnalyzers := []*analysis.Analyzer{
		noExitAnalyzer,
	}

	// Combine all
	allAnalyzers := make([]*analysis.Analyzer, 0)
	allAnalyzers = append(allAnalyzers, standardAnalyzers...)
	allAnalyzers = append(allAnalyzers, saAnalyzers...)
	allAnalyzers = append(allAnalyzers, stAnalyzers...)
	allAnalyzers = append(allAnalyzers, externalAnalyzers...)
	allAnalyzers = append(allAnalyzers, customAnalyzers...)

	multichecker.Main(allAnalyzers...)
}
