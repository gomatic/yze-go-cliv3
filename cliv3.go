// Package cliv3 provides a go/analysis analyzer enforcing the gomatic CLI
// standard that command-line programs use urfave/cli v3 — never the legacy v1
// ("github.com/urfave/cli") or v2 ("github.com/urfave/cli/v2") import paths.
package cliv3

import (
	"go/ast"
	"strconv"
	"strings"

	errs "github.com/gomatic/go-error"
	goyze "github.com/gomatic/go-yze"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// message is the single diagnostic this analyzer emits.
const message = "use urfave/cli/v3; the legacy urfave/cli v1/v2 import path is forbidden by the gomatic CLI standard"

// errUnquote reports an import path literal that is not a valid Go string
// literal. It never escapes the analyzer (an unparseable import cannot occur in
// type-checked code); it exists so unquote's failure contract is assertable.
const errUnquote errs.Const = "import path literal cannot be unquoted"

// importPath is a Go import path as resolved from an *ast.ImportSpec literal
// (e.g. "github.com/urfave/cli/v2", without the source quoting).
type importPath string

// The legacy urfave/cli module paths: v1 is matched exactly (its subpackage
// namespace also contains v2/v3, so a prefix match would over-flag); v2 is
// matched as the module or any subpackage beneath it.
const (
	pathV1 importPath = "github.com/urfave/cli"
	pathV2 importPath = "github.com/urfave/cli/v2"
)

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
	if isLegacyImport(spec.Path.Value) {
		pass.Reportf(spec.Path.Pos(), message)
	}
}

// isLegacyImport reports whether an import path literal — in either quoting
// style — names a legacy urfave/cli package. A literal that cannot be unquoted
// cannot name one.
func isLegacyImport(literal string) bool {
	path, err := unquote(literal)
	return err == nil && isLegacyURFave(path)
}

// unquote resolves an import path literal, interpreted ("…") or raw (`…`), to
// the path it names.
func unquote(literal string) (importPath, error) {
	path, err := strconv.Unquote(literal)
	if err != nil {
		return "", errUnquote.With(err, literal)
	}
	return importPath(path), nil
}

// isLegacyURFave reports whether the import path is urfave/cli v1 (exact) or
// v2 — the module itself or any subpackage beneath it, matched at a path
// boundary so e.g. a hypothetical …/cli/v20 is not flagged.
func isLegacyURFave(path importPath) bool {
	return path == pathV1 || path == pathV2 || strings.HasPrefix(string(path), string(pathV2)+"/")
}
