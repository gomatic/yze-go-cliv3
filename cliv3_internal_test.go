package cliv3

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnquoteRejectsMalformedLiteral(t *testing.T) {
	t.Parallel()
	path, err := unquote("not-a-quoted-literal")
	require.ErrorIs(t, err, errUnquote)
	assert.Empty(t, path)
}

func TestUnquoteResolvesBothQuotingStyles(t *testing.T) {
	t.Parallel()
	for name, literal := range map[string]string{
		"interpreted": `"github.com/urfave/cli/v2"`,
		"raw":         "`github.com/urfave/cli/v2`",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			path, err := unquote(importLiteral(literal))
			require.NoError(t, err)
			assert.Equal(t, pathV2, path)
		})
	}
}

func TestIsLegacyImport(t *testing.T) {
	t.Parallel()
	for literal, legacy := range map[string]bool{
		"not-a-quoted-literal":                false, // unquotable literals cannot match
		`"github.com/urfave/cli"`:             true,  // v1, exact
		`"github.com/urfave/cli/v2"`:          true,  // v2, exact
		"`github.com/urfave/cli/v2`":          true,  // v2, raw-string quoting
		`"github.com/urfave/cli/v2/altsrc"`:   true,  // v2 subpackage
		`"github.com/urfave/cli/v20"`:         false, // shares the v2 prefix, not the module
		`"github.com/urfave/cli/v3"`:          false, // the sanctioned version
		`"github.com/urfave/cli-alt"`:         false, // shares the v1 prefix, not the module
		`"github.com/urfave/cli/community"`:   false, // v1 subpackages are not matched (v3 lives under the same prefix)
		`"example.com/github.com/urfave/cli"`: false,
	} {
		t.Run(literal, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, legacy, isLegacyImport(importLiteral(literal)))
		})
	}
}

// TestPathV1IsMatchedExactlyWhileV2MatchesItsSubpackages names the claim on
// pathV1 and pathV2. v1's module path is a PREFIX of both v2's and v3's — the
// versioned paths live beneath it — so a prefix match on v1 would report every
// v3 import as legacy, which is the opposite of what this analyzer is for. v2
// has no such problem and must match its subpackages, or an import of
// .../cli/v2/altsrc slips through.
func TestPathV1IsMatchedExactlyWhileV2MatchesItsSubpackages(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		path importPath
		why  string
		want bool
	}{
		{path: pathV1, want: true, why: "v1's module path itself"},
		{path: pathV2, want: true, why: "v2's module path itself"},
		{path: pathV2 + "/altsrc", want: true, why: "a v2 subpackage is still v2"},

		{path: "github.com/urfave/cli/v3", want: false, why: "v3 is the sanctioned version and lives beneath v1's path"},
		{path: "github.com/urfave/cli/v3/altsrc", want: false, why: "and so do its subpackages"},
		{path: "github.com/urfave/cli/altsrc", want: false, why: "v1 is matched exactly, not by prefix"},
		{path: "github.com/urfave/climate", want: false, why: "a path that merely starts with the same bytes"},
		{path: "", want: false, why: "an empty path"},
	} {
		assert.Equal(t, tc.want, isLegacyURFave(tc.path), "isLegacy(%q): %s", tc.path, tc.why)
	}
}

// TestErrUnquoteIsAssertableEvenThoughItNeverEscapes names errUnquote's claim.
// It exists so unquote's failure contract can be asserted rather than assumed —
// an unparseable import literal cannot occur in type-checked code, so without a
// named sentinel the branch would be unreachable AND unverifiable, which is the
// combination that lets a "cannot happen" quietly become "happens and is
// mishandled".
func TestErrUnquoteIsAssertableEvenThoughItNeverEscapes(t *testing.T) {
	t.Parallel()

	_, err := unquote(importLiteral(`"github.com/urfave/cli/v2"`))
	require.NoError(t, err, "a well-formed literal unquotes")

	// Note: strconv.Unquote accepts a RUNE literal ('x'), so that is not a
	// failure case — only genuinely unquotable input is.
	for _, bad := range []string{``, `no quotes`, `"unterminated`, `"bad escape: \q"`} {
		_, err := unquote(importLiteral(bad))
		assert.ErrorIs(t, err, errUnquote, "%q is not a Go string literal", bad)
	}
}
