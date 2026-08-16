package main

import (
	cliv1 "github.com/urfave/cli"    // want `use github\.com/urfave/cli/v3;`
	cliv2 "github.com/urfave/cli/v2" // want `use github\.com/urfave/cli/v3;`
)

var _ = cliv1.App{}
var _ = cliv2.App{}

func main() {}
