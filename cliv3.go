// Package cliv3 provides a go/analysis analyzer enforcing the gomatic CLI
// standard that command-line programs use urfave/cli v3 — never the legacy v1
// ("github.com/urfave/cli") or v2 ("github.com/urfave/cli/v2") import paths.
package cliv3

import (
	"go/ast"

	goyze "github.com/gomatic/go-yze"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// message is the single diagnostic this analyzer emits.
const message = "use urfave/cli/v3; the legacy urfave/cli v1/v2 import path is forbidden by the gomatic CLI standard"

// importPath is a Go import path as it appears in source: the raw quoted string
// literal of an *ast.ImportSpec (e.g. `"github.com/urfave/cli/v2"`).
type importPath string

// Analyzer reports imports of the legacy urfave/cli v1/v2 packages.
var Analyzer = &analysis.Analyzer{
	Name:     "cliv3",
	Doc:      "reports imports of the legacy urfave/cli v1/v2, which the gomatic CLI standard forbids in favor of v3",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

// Registration declares this analyzer to the yze framework.
var Registration = goyze.Registration{
	Name:       "cliv3",
	Categories: []goyze.Category{"cli"},
	URL:        "https://docs.gomatic.dev/yze/cliv3",
	Analyzer:   Analyzer,
}

// run reports every import of a legacy urfave/cli path.
func run(pass *analysis.Pass) (any, error) {
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	insp.Preorder([]ast.Node{(*ast.ImportSpec)(nil)}, func(n ast.Node) {
		check(pass, n.(*ast.ImportSpec))
	})
	return nil, nil
}

// check flags the import when its path is a legacy urfave/cli version.
func check(pass *analysis.Pass, spec *ast.ImportSpec) {
	if isLegacyURFave(importPath(spec.Path.Value)) {
		pass.Reportf(spec.Path.Pos(), message)
	}
}

// isLegacyURFave reports whether the quoted import path is urfave/cli v1 or v2.
// The path is matched as the raw quoted literal (e.g. `"github.com/urfave/cli/v2"`),
// avoiding an unquote whose error branch is unreachable for valid Go.
func isLegacyURFave(quoted importPath) bool {
	switch quoted {
	case `"github.com/urfave/cli"`, `"github.com/urfave/cli/v2"`:
		return true
	default:
		return false
	}
}
