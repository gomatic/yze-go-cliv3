// Package library is NOT package main, and its legacy import carries NO local
// name. Both are dimensions every other fixture used to hold constant: every
// fixture carrying an expectation was package main, and every legacy import in
// the corpus carried an explicit name (`_`, `.`, or an alias). A rule keyed on
// either —
// `pass.Pkg.Name() == "main"`, or `spec.Name == nil` — adds no statement to any
// existing condition, so statement coverage cannot see it, and it silences most
// of the fleet: all four legacy urfave imports in owned code are unnamed, and
// non-main packages are where CLI code actually lives.
package library

import (
	"github.com/urfave/cli/v2" // want `use github\.com/urfave/cli/v3;`
)

// App returns the legacy v2 application type.
func App() *cli.App { return &cli.App{} }
