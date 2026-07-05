package main

import (
	_ "github.com/urfave/cli"    // want `use urfave/cli/v3`
	. "github.com/urfave/cli/v2" // want `use urfave/cli/v3`

	raw `github.com/urfave/cli/v2`           // want `use urfave/cli/v3`
	altsrc "github.com/urfave/cli/v2/altsrc" // want `use urfave/cli/v3`

	safe "example.com/cli"
	cli20 "github.com/urfave/cli/v20"
	cli3 "github.com/urfave/cli/v3"
)

// The dot-imported, raw-string-imported, and subpackage v2 imports are all
// flagged; the unrelated, hypothetical-v20, and v3 imports are not.
var (
	_ = App{}
	_ = raw.App{}
	_ = altsrc.Source{}
	_ = safe.App{}
	_ = cli20.App{}
	_ = cli3.Command{}
)

func main() {}
