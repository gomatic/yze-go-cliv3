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
