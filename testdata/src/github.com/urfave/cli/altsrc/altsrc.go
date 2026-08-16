// Package altsrc is a minimal stub of the urfave/cli v1 altsrc subpackage for
// analysistest fixtures. It is a REAL package of cli@v1.22.17 (12 Go files),
// and importing it pulls the v1 module into the build without naming v1's
// module path exactly.
package altsrc

// Source is a stand-in for a v1 altsrc input source type.
type Source struct{ Path string }
