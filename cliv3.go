// Package cliv3 provides a go/analysis analyzer enforcing the gomatic CLI
// standard that Go programs use urfave/cli v3 — never the legacy v1 or v2.
//
// Three import path roots name legacy urfave/cli code. Each is reported as the
// module root itself and as every package beneath it, because a subpackage of a
// legacy module is legacy code and drags the legacy module into the build:
//
//   - "github.com/urfave/cli" — v1. Its module path is also the PREFIX of every
//     later major, so a path whose first segment beneath it is a major-version
//     directory ("v2", "v3", …) is not v1 code and is judged on its own. That
//     exclusion covers the version directories and nothing else: any other
//     package beneath v1 — "github.com/urfave/cli/altsrc" is the one that
//     exists in v1.22.17 — is v1 code and is reported.
//   - "github.com/urfave/cli/v2" — v2, whose namespace holds no major-version
//     directories, so every path beneath it is v2 code.
//   - "gopkg.in/urfave/cli.v1" — the same v1 code served under a second live
//     module path, which resolves today and hands back the same *cli.App. There
//     is no gopkg.in v2: gopkg.in/urfave/cli.v2 declares its module path as
//     github.com/urfave/cli/v2 and will not resolve under the gopkg.in path.
//
// The diagnostic names the package that REPLACES the one it reports, because a
// diagnostic naming a destination the author cannot reach leaves silencing the
// rule as the only move. A module root is replaced by "github.com/urfave/cli/v3";
// a subpackage is replaced by the module urfave extracted it into at the v3
// major, "github.com/urfave/cli-<name>/v3" — urfave/cli/v3 carries no Go
// subpackages of its own, and cli-altsrc/v3 and cli-docs/v3 are both published.
//
// The rule is about the import path a file names, so it holds in test files as
// much as in production files: a _test.go that builds a v2 App makes the module
// depend on v2 either way. This analyzer is therefore deliberately absent from
// the yze suite's source-only scope, unlike the four sibling CLI rules (cliapp,
// clidomain, cliflags, cliversion), which judge how the PRODUCTION program is
// wired and have a legitimate test-side idiom. There is no test idiom that
// needs v2.
//
// The analyzer registers no flags and offers no suggested fix.
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

// messageFormat is the diagnostic this analyzer emits. Its first clause names
// the v3-line package that REPLACES the reported one, so the remedy the
// diagnostic prescribes is one the author can actually take.
const messageFormat = "use %s; the legacy urfave/cli v1/v2 import path is forbidden by the gomatic CLI standard"

// errUnquote reports an import path literal that is not a valid Go string
// literal. It never escapes the analyzer (an unparseable import cannot occur in
// type-checked code); it exists so unquote's failure contract is assertable.
const errUnquote errs.Const = "import path literal cannot be unquoted"

// importPath is a Go import path as resolved from an *ast.ImportSpec literal
// (e.g. "github.com/urfave/cli/v2", without the source quoting).
type importPath string

// The legacy urfave/cli module roots, and the sanctioned replacement for a
// legacy module root.
const (
	pathV1      importPath = "github.com/urfave/cli"
	pathV2      importPath = "github.com/urfave/cli/v2"
	pathGopkgV1 importPath = "gopkg.in/urfave/cli.v1"

	pathV3 importPath = "github.com/urfave/cli/v3"
)

// extractedPrefix and extractedSuffix bracket the module urfave extracted a
// legacy subpackage into at the v3 major: .../cli/v2/altsrc became
// github.com/urfave/cli-altsrc/v3.
const (
	extractedPrefix = "github.com/urfave/cli-"
	extractedSuffix = "/v3"
)

// decimalDigits is the cutset a major-version directory's digits are drawn
// from, so the test is a Trim rather than a rune callback — a callback would
// have to take a bare rune to satisfy strings.IndexFunc.
const decimalDigits = "0123456789"

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

// check flags the import when its path names a legacy urfave/cli package,
// naming the v3-line package that replaces it.
func check(pass *analysis.Pass, spec *ast.ImportSpec) {
	sub, legacy := legacyImport(importLiteral(spec.Path.Value))
	if !legacy {
		return
	}
	pass.Reportf(spec.Path.Pos(), messageFormat, v3Line(sub))
}

// importLiteral is an import path literal as written in source, in either
// quoting style ("…" or `…`).
type importLiteral string

// subPackage is the path beneath a legacy urfave/cli module root — empty for
// the module root itself, "altsrc" for .../cli/v2/altsrc.
type subPackage string

// pathSegment is one slash-delimited segment of an import path.
type pathSegment string

// legacyImport resolves an import path literal — in either quoting style — to
// the subpackage it names beneath a legacy urfave/cli module root, and whether
// it names one at all. A literal that cannot be unquoted cannot name one.
func legacyImport(literal importLiteral) (subPackage, bool) {
	path, err := unquote(literal)
	if err != nil {
		return "", false
	}
	return legacyURFave(path)
}

// unquote resolves an import path literal, interpreted ("…") or raw (`…`), to
// the path it names.
func unquote(literal importLiteral) (importPath, error) {
	path, err := strconv.Unquote(string(literal))
	if err != nil {
		return "", errUnquote.With(err, literal)
	}
	return importPath(path), nil
}

// legacyURFave resolves an import path to the subpackage it names beneath a
// legacy urfave/cli module root, and whether it names one. v2 and the gopkg.in
// v1 alias are tested first: v2's root lies beneath v1's, so testing v1 first
// would attribute every v2 path to v1 and prescribe the wrong replacement.
func legacyURFave(path importPath) (subPackage, bool) {
	if sub, ok := beneath(path, pathV2); ok {
		return sub, true
	}
	if sub, ok := beneath(path, pathGopkgV1); ok {
		return sub, true
	}
	sub, ok := beneath(path, pathV1)
	return sub, ok && !isMajorVersionDir(head(sub))
}

// beneath resolves the subpackage an import path names beneath a module root:
// empty when the path IS the root, the remainder when it lies under it, and
// false when it is neither — the boundary is a whole segment, so
// .../cli/v20 does not lie beneath .../cli/v2.
func beneath(path, root importPath) (subPackage, bool) {
	if path == root {
		return "", true
	}
	rest, ok := strings.CutPrefix(string(path), string(root)+"/")
	return subPackage(rest), ok
}

// head is a subpackage's first path segment, which is the whole of it when it
// has no separator.
func head(sub subPackage) pathSegment {
	first, _, _ := strings.Cut(string(sub), "/")
	return pathSegment(first)
}

// isMajorVersionDir reports whether a path segment is a Go module
// major-version directory — "v" followed by one or more digits.
func isMajorVersionDir(segment pathSegment) bool {
	digits, ok := strings.CutPrefix(string(segment), "v")
	if !ok || digits == "" {
		return false
	}
	return strings.Trim(digits, decimalDigits) == ""
}

// v3Line names the v3-line package that replaces a legacy urfave/cli package.
// A module root is replaced by urfave/cli/v3 itself; a subpackage is replaced
// by the module urfave extracted it into at the v3 major, which is where every
// one of them went — urfave/cli/v3 carries no Go subpackages at all.
func v3Line(sub subPackage) importPath {
	if sub == "" {
		return pathV3
	}
	return importPath(extractedPrefix + string(head(sub)) + extractedSuffix)
}
