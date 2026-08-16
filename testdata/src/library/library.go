// Package library is NOT package main. It exists because every other fixture
// carrying a want is package main, and a rule keyed on the package clause would
// otherwise be invisible to this suite while silencing every non-main package in
// the fleet — which is where the urfave imports actually live.
package library

import (
	cliv2 "github.com/urfave/cli/v2" // want `use github\.com/urfave/cli/v3;`
)

// App returns the legacy v2 application type.
func App() *cliv2.App { return &cliv2.App{} }
