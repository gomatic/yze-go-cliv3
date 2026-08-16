package main

import (
	_ "github.com/urfave/cli"    // want `use github\.com/urfave/cli/v3;`
	. "github.com/urfave/cli/v2" // want `use github\.com/urfave/cli/v3;`

	raw `github.com/urfave/cli/v2` // want `use github\.com/urfave/cli/v3;`

	v1altsrc "github.com/urfave/cli/altsrc"    // want `use github\.com/urfave/cli-altsrc/v3;`
	v2altsrc "github.com/urfave/cli/v2/altsrc" // want `use github\.com/urfave/cli-altsrc/v3;`
	gopkg "gopkg.in/urfave/cli.v1"             // want `use github\.com/urfave/cli/v3;`
	gopkgalt "gopkg.in/urfave/cli.v1/altsrc"   // want `use github\.com/urfave/cli-altsrc/v3;`

	safe "example.com/cli"
	cli20 "github.com/urfave/cli/v20"
	cli3 "github.com/urfave/cli/v3"
	climate "github.com/urfave/climate"
)

// Every legacy shape is flagged and every near-miss is silent. The three legacy
// roots are each present as the module root and as the one subpackage that
// exists in them (altsrc), and each subpackage names the module urfave
// extracted it into rather than a path inside urfave/cli/v3, which has none.
//
// The silent four each deviate from a flagged sibling in exactly one place:
// example.com/cli is a different module whose last segment is "cli";
// urfave/climate shares urfave/cli's bytes without ending at a segment
// boundary; urfave/cli/v20 is a major-version directory that is not v2; and
// urfave/cli/v3 is the sanctioned version, living beneath v1's path.
//
// DO NOT RUN gofmt ON THIS FILE. The raw-string import above is the only place
// the second quoting style is exercised against a real parse, and gofmt
// normalises it to an interpreted string — silently deleting the property the
// line exists to pin. `FMT_FILES` in the shared Makefile excludes testdata/ for
// exactly this reason. The same property is pinned again, out of a formatter's
// reach, by TestUnquoteResolvesBothQuotingStyles.
var (
	_ = App{}
	_ = raw.App{}
	_ = v1altsrc.Source{}
	_ = v2altsrc.Source{}
	_ = gopkg.App{}
	_ = gopkgalt.Source{}
	_ = safe.App{}
	_ = climate.App{}
	_ = cli20.App{}
	_ = cli3.Command{}
)

func main() {}
