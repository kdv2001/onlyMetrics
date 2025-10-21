package osexit

import (
	"go/ast"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// CheckAnalyzer анализатор проверяющий использование os.Exit() в проекте
var CheckAnalyzer = &analysis.Analyzer{
	Name: "osexit",
	Doc:  "check os.Exit() usage",
	Run:  run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	for _, f := range pass.Files {
		ast.Inspect(f, func(node ast.Node) bool {
			switch n := node.(type) {
			case *ast.SelectorExpr:
				p, ok := n.X.(*ast.Ident)
				if !ok {
					break
				}
				if strings.EqualFold(p.Name, "os") &&
					strings.EqualFold(n.Sel.Name, "exit") {
					pass.Reportf(p.NamePos, "os exit usage prohibited")
				}
			}

			return true
		})
	}

	return nil, nil
}
