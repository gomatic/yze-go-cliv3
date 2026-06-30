package main

import (
	_ "github.com/urfave/cli"    // want `use urfave/cli/v3`
	. "github.com/urfave/cli/v2" // want `use urfave/cli/v3`

	safe "example.com/cli"
	cli3 "github.com/urfave/cli/v3"
)

// dot-imported v2 App is in scope; the unrelated and v3 imports are not flagged.
var (
	_ = App{}
	_ = safe.App{}
	_ = cli3.Command{}
)

func main() {}
