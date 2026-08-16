// Package cliv3 provides a go/analysis analyzer enforcing the gomatic CLI
// standard that Go programs use urfave/cli v3 — never the legacy v1 or v2.
//
// Five import path roots are KNOWN to serve legacy urfave/cli code. Each is
// reported as the module root itself and as every package beneath it, because a
// subpackage of a legacy module is legacy code and drags the legacy module into
// the build. Beneath every root, a first segment that is a Go major-version
// directory ("v2", "v3", …) is excluded, because those directories hold a
// different major that is judged on its own:
//
//   - "github.com/urfave/cli" — v1. That version-directory exclusion is the
//     whole of the exemption: any other package beneath v1 —
//     "github.com/urfave/cli/altsrc" is the one that exists in v1.22.17 — is v1
//     code and is reported.
//   - "github.com/urfave/cli/v2" — v2.
//   - "github.com/codegangsta/cli" — the name urfave/cli was renamed FROM, still
//     served by GitHub with 51 published versions. At v1.20.0 its tree is
//     byte-identical to github.com/urfave/cli@v1.20.0 and it resolves, builds
//     and links today.
//   - "gopkg.in/urfave/cli.v1" and "gopkg.in/codegangsta/cli.v1" — the same v1
//     code under gopkg.in, both resolving at v1.20.0 with the identical tree.
//
// The comparison of a path against a root is CASE-INSENSITIVE. Import paths are
// case-sensitive to the Go toolchain, but the hosts here are not: at any version
// published before go.mod existed there is no declared module path to disagree
// with, so "github.com/Urfave/cli@v1.20.0" resolves, builds and hands back the
// same *cli.App. Case is an unbounded family of spellings and folding it closes
// all of them at once. From v1.22.x onward the module's own go.mod rejects every
// variant spelling, which is why only the pre-module versions are reachable.
//
// THE ENUMERATION IS NOT CLOSED, AND THIS COMMENT DOES NOT CLAIM IT IS. "No path
// spelling escapes" is a negative universal over an unbounded set of host
// aliases, vanity domains, forks and local `replace` directives, and no finite
// list of roots closes it — the codegangsta family above was found by somebody
// remembering a rename, not by any instrument here. What is claimed is the bound:
// these five roots, folded for case, cover every spelling MEASURED to resolve
// against proxy.golang.org and hand back urfave/cli v1 or v2. What would actually
// close it is a decision made on something the import path cannot forge — the
// identity of the package the import resolves TO, which go/analysis offers as the
// resolved package object. That is the shape the rule should eventually take.
//
// The diagnostic names the package that REPLACES the one it reports, because a
// diagnostic naming a destination the author cannot reach leaves silencing the
// rule as the only move. A module root is replaced by "github.com/urfave/cli/v3";
// a subpackage is replaced by the module urfave extracted it into at the v3
// major, "github.com/urfave/cli-<name>/v3". urfave/cli/v3 carries no IMPORTABLE
// subpackage — the directories under v3.10.1 that hold Go files are all
// `package main` examples and scripts — and cli-altsrc/v3 and cli-docs/v3 are
// both published.
//
// The rule is about the import path a file names, so it holds in test files as
// much as in production files: a _test.go that builds a v2 App makes the module
// depend on v2 either way. This analyzer is therefore deliberately absent from
// the yze suite's source-only scope, unlike the four sibling CLI rules (cliapp,
// clidomain, cliflags, cliversion), which judge how the PRODUCTION program is
// wired and have a legitimate test-side idiom. There is no test idiom that
// needs v2. That absence is a fact about ANOTHER repository — gomatic/yze's
// registrations.go — which this package cannot see and no test here can pin; if
// cliv3 is ever added to that set, every _test.go finding this rule makes goes
// silent and nothing in this repo goes red.
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
	pathV1            importPath = "github.com/urfave/cli"
	pathV2            importPath = "github.com/urfave/cli/v2"
	pathGopkgV1       importPath = "gopkg.in/urfave/cli.v1"
	pathOriginal      importPath = "github.com/codegangsta/cli"
	pathGopkgOriginal importPath = "gopkg.in/codegangsta/cli.v1"

	pathV3 importPath = "github.com/urfave/cli/v3"
)

// legacyRoots are the module roots that serve legacy urfave/cli code. v2's root
// lies BENEATH v1's, so the listing puts the deeper root first — but that order
// is not what makes the answer right, and writing it down as the reason would be
// a justification the code does not rest on. What makes it right is that the
// major-version-directory exclusion is applied at EVERY root: beneath v1,
// "v2/altsrc" has the head "v2" and is skipped, so a v2 path is left for v2's own
// root whichever order the roots are tried in. That claim is driven by
// TestRootOrderDoesNotChangeTheVerdict rather than asserted here, and it was
// MEASURED before it was written: reversing the listing produced the identical
// sorted projection over four probe modules, 10 findings on each side.
//
// It is a function rather than a package-level slice because a slice var is
// mutable state any importer could rewrite.
func legacyRoots() []importPath {
	return []importPath{pathV2, pathGopkgV1, pathGopkgOriginal, pathOriginal, pathV1}
}

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
//
// It consults the import literal alone — not the file's name, not its
// directory, not the package clause, not the build. That is deliberate, and the
// suite has no instrument that would notice it changing. Adding a disjunct such
// as `|| strings.Contains(pass.Fset.File(spec.Path.Pos()).Name(), "_legacy/")`
// adds no statement, so coverage stays at 100.0%, and a fixture would have to
// sit at every path somebody might exempt. Measured: that mutant survives the
// whole suite and takes the fleet's only owned legacy findings to zero. This is
// the negative-universal-over-paths shape docs/s03.md describes, recorded here
// rather than papered over.
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
// legacy urfave/cli module root, and whether it names one. A first segment that
// is a major-version directory belongs to a different major and is judged on its
// own, so it disqualifies the root rather than the path — .../cli/v3 is not v1's
// subpackage "v3", it is v3.
func legacyURFave(path importPath) (subPackage, bool) {
	return resolve(legacyRoots(), path)
}

// resolve is legacyURFave against a given root list, so the claim that the
// verdict does not depend on the listing's order is something a test can drive
// rather than something this file asserts about itself.
func resolve(roots []importPath, path importPath) (subPackage, bool) {
	for _, root := range roots {
		sub, ok := beneath(path, root)
		if !ok || isMajorVersionDir(head(sub)) {
			continue
		}
		return sub, true
	}
	return "", false
}

// beneath resolves the subpackage an import path names beneath a module root:
// empty when the path IS the root, the remainder when it lies under it, and
// false when it is neither. The boundary is a whole segment, so .../cli/v20 does
// not lie beneath .../cli/v2, and the comparison folds case, so a host that
// serves the same repository under a different capitalisation is the same root.
func beneath(path, root importPath) (subPackage, bool) {
	if strings.EqualFold(string(path), string(root)) {
		return "", true
	}
	prefix := string(root) + "/"
	if len(path) <= len(prefix) || !strings.EqualFold(string(path)[:len(prefix)], prefix) {
		return "", false
	}
	return subPackage(string(path)[len(prefix):]), true
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
// one of them went — urfave/cli/v3 carries no importable subpackage. The
// segment is lowered because the root comparison folds case and a module path
// does not.
func v3Line(sub subPackage) importPath {
	if sub == "" {
		return pathV3
	}
	return importPath(extractedPrefix + strings.ToLower(string(head(sub))) + extractedSuffix)
}
